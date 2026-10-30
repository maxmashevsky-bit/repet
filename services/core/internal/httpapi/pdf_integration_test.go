package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/objectstorage"
	"tutor-platform/services/core/internal/pdfextract"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

type testPDFExtractor struct{ data []byte }

type cancelPDFPut struct {
	objectstorage.ObjectStorage
	cancel context.CancelFunc
	key    string
}

func (s *cancelPDFPut) Put(ctx context.Context, r io.Reader, limit int64) (objectstorage.Object, error) {
	obj, err := s.ObjectStorage.Put(ctx, r, limit)
	if err == nil {
		s.key = obj.Key
		s.cancel()
	}
	return obj, err
}

func (e testPDFExtractor) Extract(context.Context, []byte) (pdfextract.Result, error) {
	return pdfextract.Result{Pages: []pdfextract.Page{{Number: 1, Text: "1. Calculate 5 + 2.\n2. Find 9 - 1.\n", PNG: e.data}, {Number: 2, PNG: e.data, OCRRequired: true}}, Figures: []pdfextract.Figure{{Number: 0, Page: 1, PNG: e.data}, {Number: 1, Page: 1, PNG: e.data}}, Warnings: []string{"Check the original"}}, nil
}
func TestPDFImportsPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = store.Migrate(t.Context(), pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	objects, err := objectstorage.NewFilesystem(filepath.Join(t.TempDir(), "objects"))
	if err != nil {
		t.Fatal(err)
	}
	defer objects.Close()
	var picture bytes.Buffer
	if err = png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	st := store.NewWithPDF(pool, objects, testPDFExtractor{picture.Bytes()})
	app := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20, CORSAllowedOrigins: []string{"http://app.test"}}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{}))
	defer app.Close()
	uuid := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	register := func(role domain.Role) (domain.User, *http.Cookie) {
		u, token, e := st.RegisterUser(t.Context(), uuid()+"@example.test", "PDF test", "password123", role, "test", "127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		return u, &http.Cookie{Name: store.SessionCookieName, Value: token}
	}
	tutor, tc := register(domain.RoleTutor)
	student, sc := register(domain.RoleStudent)
	_, other := register(domain.RoleTutor)
	_, peer := register(domain.RoleStudent)
	_, token, err := st.CreateInvitation(t.Context(), tutor, student.Email, "")
	if err != nil {
		t.Fatal(err)
	}
	rel, err := st.AcceptInvitation(t.Context(), student, token)
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path string, input any, cookie *http.Cookie, want int) []byte {
		t.Helper()
		data, e := json.Marshal(input)
		if e != nil {
			t.Fatal(e)
		}
		r, e := http.NewRequestWithContext(t.Context(), method, app.URL+path, bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		res, e := app.Client().Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		out, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s = %d, want %d: %s", method, path, res.StatusCode, want, out)
		}
		return out
	}
	decode := func(data []byte, out any) {
		t.Helper()
		if e := json.Unmarshal(data, out); e != nil {
			t.Fatal(e)
		}
	}
	in := domain.PDFImportInput{RequestID: uuid(), Filename: "own.pdf", DataBase64: base64.StdEncoding.EncodeToString([]byte("%PDF-1.7 synthetic")), RightsStatus: "own", License: "Own demo", SourceLabel: "Self authored"}
	call("POST", "/api/v1/pdf-imports", in, nil, 401)
	call("POST", "/api/v1/pdf-imports", in, sc, 403)
	validJSON, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var injected map[string]any
	decode(validJSON, &injected)
	injected["role"] = "tutor"
	call("POST", "/api/v1/pdf-imports", injected, tc, 400)
	var source, again domain.PDFImport
	payload := call("POST", "/api/v1/pdf-imports", in, tc, 202)
	decode(payload, &source)
	if strings.Contains(string(payload), "source_key") || source.Status != "queued" {
		t.Fatal("unsafe source", string(payload))
	}
	decode(call("POST", "/api/v1/pdf-imports", in, tc, 202), &again)
	if again.ID != source.ID {
		t.Fatal("upload replay")
	}
	in.SourceLabel = "Changed"
	call("POST", "/api/v1/pdf-imports", in, tc, 409)
	invalid := in
	invalid.RequestID = uuid()
	invalid.Filename = "../bad.pdf"
	call("POST", "/api/v1/pdf-imports", invalid, tc, 400)
	invalid.Filename = "bad.pdf"
	invalid.DataBase64 = base64.StdEncoding.EncodeToString([]byte("<script>"))
	call("POST", "/api/v1/pdf-imports", invalid, tc, 400)
	path := "/api/v1/pdf-imports/" + source.ID
	call("GET", path, nil, other, 404)
	call("GET", path, nil, sc, 403)
	call("GET", path+"/source", nil, other, 404)
	if string(call("GET", path+"/source", nil, tc, 200)) != "%PDF-1.7 synthetic" {
		t.Fatal("original changed")
	}
	worked, err := st.ProcessPDFImport(t.Context())
	if err != nil || !worked {
		t.Fatal(worked, err)
	}
	decode(call("GET", path, nil, tc, 200), &source)
	if source.Status != "ocr_required" || len(source.Pages) != 2 || len(source.Assets) != 4 || !strings.Contains(source.Pages[0].OriginalText, "5 + 2") {
		t.Fatalf("extraction %+v", source)
	}
	if worked, err = st.ProcessPDFImport(t.Context()); err != nil || worked {
		t.Fatal("job repeated", worked, err)
	}
	var history domain.CatalogPage[domain.PDFImport]
	decode(call("GET", "/api/v1/pdf-imports?limit=1&offset=0", nil, tc, 200), &history)
	if history.Total != 1 || len(history.Items) != 1 || len(history.Items[0].Pages) != 0 || len(history.Items[0].Assets) != 0 {
		t.Fatal("history must be paginated metadata only", history)
	}
	decode(call("GET", "/api/v1/pdf-imports?limit=1&offset=1", nil, tc, 200), &history)
	if history.Total != 1 || len(history.Items) != 0 {
		t.Fatal("history offset", history)
	}
	decode(call("GET", "/api/v1/pdf-imports", nil, other, 200), &history)
	if history.Total != 0 || len(history.Items) != 0 {
		t.Fatal("foreign import in history")
	}
	pageID := source.Pages[0].ImageID
	call("GET", "/api/v1/pdf-assets/"+pageID, nil, tc, 200)
	call("GET", "/api/v1/pdf-assets/"+pageID, nil, sc, 404)
	call("GET", "/api/v1/pdf-assets/"+pageID, nil, other, 404)
	crop := map[string]int{"page_number": 1, "x": 1, "y": 1, "width": 2, "height": 2}
	call("POST", path+"/figures", crop, other, 404)
	call("POST", path+"/figures", crop, sc, 403)
	badCrop := map[string]int{"page_number": 1, "x": 3, "y": 3, "width": 2, "height": 2}
	call("POST", path+"/figures", badCrop, tc, 400)
	var cropped domain.PDFImport
	decode(call("POST", path+"/figures", crop, tc, 201), &cropped)
	if len(cropped.Assets) != 5 {
		t.Fatal("crop was not saved")
	}
	call("PUT", path+"/pages/2", map[string]string{"text": ""}, tc, 400)
	for _, pg := range source.Pages {
		call("PUT", path+"/pages/"+string(rune('0'+pg.Number)), map[string]string{"text": "Checked page"}, other, 404)
	}
	var course domain.Course
	decode(call("POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "PDF", Track: "school", GradeMin: 5, GradeMax: 7, Difficulty: 1, Visibility: "linked_students"}, tc, 201), &course)
	var topic domain.Topic
	decode(call("POST", "/api/v1/catalog/courses/"+course.ID+"/topics", domain.TopicInput{Title: "Addition"}, tc, 201), &topic)
	lessonIn := domain.LearningLessonInput{TopicID: topic.ID, Title: "PDF exercises", Track: "school", GradeMin: 5, GradeMax: 7, Difficulty: 1}
	var lesson domain.LearningLesson
	decode(call("POST", path+"/lessons", map[string]any{"course_id": course.ID, "lesson": lessonIn}, tc, 201), &lesson)
	lp := "/api/v1/catalog/lessons/" + lesson.ID
	call("POST", "/api/v1/catalog/courses/"+course.ID+"/publish", map[string]any{}, tc, 200)
	call("POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, tc, 200)
	call("POST", lp+"/publish", map[string]any{}, tc, 422)
	var figures []string
	for _, a := range source.Assets {
		if a.Kind == "figure" {
			figures = append(figures, a.ID)
		}
	}
	exIn := domain.ExerciseInput{Kind: "numeric", Prompt: "1. Calculate 5 + 2.", Options: []string{}, Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: "7", Explanation: "PRIVATE PDF SOLUTION", Hints: []string{"Add two"}}, Media: []domain.ExerciseMedia{{AssetID: figures[0], Alt: "Triangle"}}}
	var ex domain.Exercise
	decode(call("POST", lp+"/exercises", exIn, tc, 201), &ex)
	exIn.Media = []domain.ExerciseMedia{{AssetID: pageID, Alt: "Full page"}}
	call("POST", lp+"/exercises", exIn, tc, 404)
	exIn.Media = []domain.ExerciseMedia{{AssetID: figures[1], Alt: "Second triangle"}}
	exIn.Position = 1
	var secondExercise domain.Exercise
	decode(call("POST", lp+"/exercises", exIn, tc, 201), &secondExercise)
	review := func(want int) {
		t.Helper()
		var state struct {
			Revision int64 `json:"revision"`
		}
		decode(call("GET", lp+"/pdf-source", nil, tc, 200), &state)
		call("POST", lp+"/pdf-review", map[string]any{"revision": state.Revision, "confirmed": true}, tc, want)
	}
	review(422)
	call("PUT", path+"/pages/1", map[string]string{"text": source.Pages[0].OriginalText}, tc, 200)
	call("PUT", path+"/pages/2", map[string]string{"text": "3. Explain the triangle."}, tc, 200)
	review(200)
	var previousReview struct {
		Revision int64 `json:"revision"`
	}
	decode(call("GET", lp+"/pdf-source", nil, tc, 200), &previousReview)
	call("PUT", path+"/pages/1", map[string]string{"text": source.Pages[0].OriginalText}, tc, 200)
	call("POST", lp+"/pdf-review", map[string]any{"revision": previousReview.Revision, "confirmed": true}, tc, 422)
	call("POST", lp+"/publish", map[string]any{}, tc, 422)
	review(200)
	call("PUT", lp, lessonIn, tc, 200)
	call("POST", lp+"/publish", map[string]any{}, tc, 422)
	review(200)
	decode(call("POST", lp+"/publish", map[string]any{}, tc, 200), &lesson)
	publishedID := lesson.Version.ID
	public := call("GET", lp, nil, sc, 200)
	if strings.Contains(string(public), "PRIVATE") || strings.Contains(string(public), "answer_key") {
		t.Fatal("private key leak")
	}
	call("GET", lp+"/pdf-source", nil, sc, 403)
	call("GET", path+"/source", nil, sc, 403)
	call("GET", "/api/v1/pdf-assets/"+figures[0], nil, sc, 200)
	call("GET", "/api/v1/pdf-assets/"+figures[0], nil, peer, 404)
	call("GET", "/api/v1/pdf-assets/"+pageID, nil, sc, 404)
	ass, err := st.CreateMaterialAssignment(t.Context(), tutor, domain.MaterialAssignmentInput{RequestID: uuid(), StudentID: student.ID, LessonID: lesson.ID, ExerciseIDs: []string{ex.ID}, DueAt: time.Now().Add(time.Hour), Mode: "practice"})
	if err != nil {
		t.Fatal(err)
	}
	call("POST", lp+"/learning-attempts", domain.LearningStartInput{RequestID: uuid(), Mode: "practice"}, sc, 201)
	// Replacing the current publication must not break images in an already pinned run.
	exIn.Media = nil
	call("PUT", lp+"/exercises/"+ex.ID, exIn, tc, 200)
	call("PUT", lp+"/exercises/"+secondExercise.ID, exIn, tc, 200)
	review(200)
	call("POST", lp+"/publish", map[string]any{}, tc, 200)
	call("GET", "/api/v1/pdf-assets/"+figures[0], nil, sc, 200)
	call("GET", "/api/v1/pdf-assets/"+figures[1], nil, sc, 200)
	if _, err = st.ArchiveLearningLesson(t.Context(), tutor, lesson.ID); err != nil {
		t.Fatal(err)
	}
	call("GET", "/api/v1/pdf-assets/"+figures[0], nil, sc, 200)
	call("GET", "/api/v1/pdf-assets/"+figures[1], nil, sc, 404)
	if _, err = pool.Exec(t.Context(), `update relations set status='ended' where id=$1`, rel.ID); err != nil {
		t.Fatal(err)
	}
	call("GET", "/api/v1/pdf-assets/"+figures[0], nil, sc, 404)
	if _, err = pool.Exec(t.Context(), `update exercises set media='[]' where version_id=$1`, publishedID); err == nil {
		t.Fatal("published media mutable")
	}
	if _, err = pool.Exec(t.Context(), `update pdf_pages set original_text='changed' where import_id=$1`, source.ID); err == nil {
		t.Fatal("source text mutable")
	}
	if _, err = pool.Exec(t.Context(), `update pdf_assets set object_key='changed' where id=$1`, figures[0]); err == nil {
		t.Fatal("source image mutable")
	}
	_ = ass
	t.Run("concurrent replay and expired lease", func(t *testing.T) {
		replay := in
		replay.RequestID = uuid()
		var wg sync.WaitGroup
		results := make([]domain.PDFImport, 2)
		errs := make([]error, 2)
		for i := range results {
			wg.Add(1)
			go func() { defer wg.Done(); results[i], errs[i] = st.ImportPDF(t.Context(), tutor, replay) }()
		}
		wg.Wait()
		if errs[0] != nil || errs[1] != nil || results[0].ID != results[1].ID {
			t.Fatal(results, errs)
		}
		jobID := results[0].ID
		if _, err := pool.Exec(t.Context(), `update pdf_imports set status='extracting',extraction_attempts=1,lease_token=$2,lease_until=now()-interval '1 second' where id=$1`, jobID, uuid()); err != nil {
			t.Fatal(err)
		}
		if worked, err := st.ProcessPDFImport(t.Context()); err != nil || !worked {
			t.Fatal(worked, err)
		}
		var count int
		if err := pool.QueryRow(t.Context(), `select count(*) from pdf_assets where import_id=$1`, jobID).Scan(&count); err != nil || count != 4 {
			t.Fatal(count, err)
		}
	})
	t.Run("cancelled upload removes the uncommitted original", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		files := &cancelPDFPut{ObjectStorage: objects, cancel: cancel}
		cancelledStore := store.NewWithPDF(pool, files, testPDFExtractor{picture.Bytes()})
		input := in
		input.RequestID = uuid()
		if _, err := cancelledStore.ImportPDF(ctx, tutor, input); err == nil {
			t.Fatal("cancelled upload succeeded")
		}
		if files.key == "" {
			t.Fatal("cancellation did not occur after storing bytes")
		}
		if body, err := objects.Open(t.Context(), files.key); err == nil {
			body.Close()
			t.Fatal("cancelled upload left an unreferenced object")
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	})
}
