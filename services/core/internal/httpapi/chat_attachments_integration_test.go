package httpapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/objectstorage"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestChatAttachmentsPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
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
	st := store.NewWithPDF(pool, objects, nil)
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	register := func(role domain.Role) (domain.User, string) {
		u, token, e := st.RegisterUser(t.Context(), id()+"@example.test", "Chat attachments fixture", "Password123!", role, "test", "127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		return u, token
	}
	tutor, ts := register(domain.RoleTutor)
	student, ss := register(domain.RoleStudent)
	peer, ps := register(domain.RoleStudent)
	other, osession := register(domain.RoleTutor)
	connect := func(u domain.User) string {
		_, token, e := st.CreateInvitation(t.Context(), tutor, u.Email, "")
		if e != nil {
			t.Fatal(e)
		}
		if _, e = st.AcceptInvitation(t.Context(), u, token); e != nil {
			t.Fatal(e)
		}
		items, e := st.ListConversations(t.Context(), u)
		if e != nil || len(items) != 1 {
			t.Fatal(e, items)
		}
		return items[0].ID
	}
	conv, otherConv := connect(student), connect(peer)
	srv := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{}))
	defer srv.Close()
	call := func(session, method, path string, body any, key string, want int, out any) ([]byte, http.Header) {
		t.Helper()
		data, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest(method, srv.URL+path, bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		if session != "" {
			req.AddCookie(&http.Cookie{Name: store.SessionCookieName, Value: session})
		}
		res, e := srv.Client().Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s=%d want%d %s", method, path, res.StatusCode, want, raw)
		}
		if out != nil {
			if e = json.Unmarshal(raw, out); e != nil {
				t.Fatal(e)
			}
		}
		return raw, res.Header
	}
	base := "/api/v1/messages/conversations/" + conv
	upload := func(name string, data []byte, nonce string, want int) domain.ChatFile {
		t.Helper()
		var f domain.ChatFile
		call(ts, "POST", base+"/files", map[string]any{"filename": name, "data": data, "request_id": nonce}, "", want, &f)
		return f
	}
	var pngBytes bytes.Buffer
	if err = png.Encode(&pngBytes, image.NewNRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	source := append(pngBytes.Bytes(), []byte("<script>trailing payload</script>")...)
	nonce := id()
	file := upload("Схема.png", source, nonce, 201)
	same := upload("Схема.png", source, nonce, 201)
	if file.ID != same.ID || file.Size < 1 {
		t.Fatal("upload replay", file, same)
	}
	raw, _ := call(ts, "GET", base+"/files/"+file.ID, nil, "", 200, nil)
	if bytes.Contains(raw, []byte("script")) {
		t.Fatal("original payload leaked")
	}
	call(ss, "GET", base+"/files/"+file.ID, nil, "", 404, nil)
	upload("Другое.png", source, nonce, 409)
	upload("Схема.png", []byte("changed"), nonce, 400)
	for _, name := range []string{"x.html", "x.svg", "x.pdf", "x.zip", "x.exe", "../x.txt"} {
		upload(name, []byte("arbitrary content"), id(), 400)
	}
	upload("x.txt", []byte{0xff}, id(), 400)
	upload("x.txt", []byte(strings.Repeat("x", (4<<20)+1)), id(), 400)
	call(ts, "POST", base+"/files", map[string]any{"filename": "x.txt", "data": []byte("text"), "request_id": id(), "owner_id": student.ID}, "", 400, nil)
	for _, session := range []string{"", ps, osession} {
		want := 404
		if session == "" {
			want = 401
		}
		call(session, "GET", base+"/files/"+file.ID, nil, "", want, nil)
		call(session, "POST", base+"/files", map[string]any{"filename": "x.txt", "data": []byte("text"), "request_id": id()}, "", want, nil)
	}
	var sent, again domain.Message
	messageNonce := id()
	input := domain.MessageInput{Body: "Схема к уроку", FileID: file.ID}
	call(ts, "POST", base, input, messageNonce, 201, &sent)
	call(ts, "POST", base, input, messageNonce, 201, &again)
	if sent.ID != again.ID || sent.Attachment == nil || sent.Attachment.File == nil || sent.Attachment.File.ID != file.ID {
		t.Fatal("attachment replay", sent, again)
	}
	call(ts, "POST", base, domain.MessageInput{Body: "changed", FileID: file.ID}, messageNonce, 409, nil)
	call(ts, "POST", base, input, id(), 404, nil)
	call(ss, "POST", base, input, id(), 404, nil)
	call(ts, "POST", "/api/v1/messages/conversations/"+otherConv, input, id(), 404, nil)
	call(ts, "DELETE", base+"/files/"+file.ID, nil, "", 404, nil)
	raw, headers := call(ss, "GET", base+"/files/"+file.ID, nil, "", 200, nil)
	if headers.Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(headers.Get("Content-Disposition"), "attachment;") || headers.Get("Content-Security-Policy") != "sandbox; default-src 'none'" || headers.Get("Cache-Control") != "no-store" || len(raw) != int(file.Size) {
		t.Fatal("unsafe download headers", headers)
	}
	var messages struct {
		Items []domain.Message `json:"messages"`
	}
	list, _ := call(ss, "GET", base, nil, "", 200, &messages)
	if len(messages.Items) != 1 || messages.Items[0].Attachment == nil || bytes.Contains(list, []byte("object_key")) || bytes.Contains(list, []byte("source_hash")) {
		t.Fatal("message files metadata", string(list))
	}
	pending := upload("Заметки.txt", []byte("<b>Текст</b>\r\nПример"), id(), 201)
	call(ss, "DELETE", base+"/files/"+pending.ID, nil, "", 404, nil)
	call(ts, "DELETE", base+"/files/"+pending.ID, nil, "", 200, nil)
	call(ts, "GET", base+"/files/"+pending.ID, nil, "", 404, nil)
	expired := upload("Истёк.txt", []byte("expired"), id(), 201)
	var objectKey string
	if err = pool.QueryRow(t.Context(), `update chat_files set expires_at=clock_timestamp()-interval '1 second' where id=$1 returning object_key`, expired.ID).Scan(&objectKey); err != nil {
		t.Fatal(err)
	}
	call(ts, "GET", base+"/files/"+expired.ID, nil, "", 404, nil)
	call(ts, "POST", base, domain.MessageInput{FileID: expired.ID}, id(), 404, nil)
	upload("Следующий.txt", []byte("next"), id(), 201)
	if _, err = objects.Open(t.Context(), objectKey); !os.IsNotExist(err) {
		t.Fatal("expired object not cleaned", err)
	}
	// The per-owner quota is checked under an advisory lock, independent of dialog.
	if _, err = pool.Exec(t.Context(), `update chat_files set size_bytes=4194304 where owner_id=$1`, tutor.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 11; i++ {
		upload("quota.txt", []byte("quota"), id(), 201)
	}
	if _, err = pool.Exec(t.Context(), `update chat_files set size_bytes=4194304 where owner_id=$1`, tutor.ID); err != nil {
		t.Fatal(err)
	}
	upload("blocked.txt", []byte("quota"), id(), 422)

	assignment, err := st.CreateAssignment(t.Context(), tutor, student.ID, "Дроби", "Математика", "Дроби", "Решите 1/2 + 1/2", time.Now().Add(24*time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := st.CreateAssignment(t.Context(), tutor, peer.ID, "Чужое задание", "Математика", "Дроби", "Private prompt", time.Now().Add(24*time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	call(ts, "POST", base, domain.MessageInput{AssignmentID: wrong.ID}, id(), 404, nil)
	assignmentNonce := id()
	if _, err = st.SendChatMessage(t.Context(), tutor, conv, domain.MessageInput{AssignmentID: assignment.ID}, assignmentNonce); err != nil {
		t.Fatal("send assignment", err)
	}
	call(ts, "POST", base, domain.MessageInput{AssignmentID: assignment.ID}, assignmentNonce, 201, &sent)
	preview, _ := call(ss, "GET", base+"/assignments/"+assignment.ID, nil, "", 200, nil)
	if !bytes.Contains(preview, []byte("1/2")) || bytes.Contains(preview, []byte("answer")) || bytes.Contains(preview, []byte("feedback")) {
		t.Fatal("unsafe assignment preview", string(preview))
	}
	call(ps, "GET", base+"/assignments/"+assignment.ID, nil, "", 404, nil)
	call(ss, "GET", base+"/assignments/"+wrong.ID, nil, "", 404, nil)
	var references struct {
		Items []domain.ChatAttachment `json:"items"`
	}
	call(ss, "GET", base+"/references?kind=assignment&q=Дроби", nil, "", 200, &references)
	if len(references.Items) != 1 || references.Items[0].ID != assignment.ID {
		t.Fatal(references)
	}
	call(ss, "GET", base+"/references?kind=assignment&offset=20", nil, "", 200, &references)
	if len(references.Items) != 0 {
		t.Fatal("pagination", references)
	}
	lesson := func(author domain.User) domain.LearningLesson {
		t.Helper()
		course, e := st.CreateCourse(t.Context(), author, domain.CourseInput{Title: "Chat course", Track: "school", GradeMin: 5, GradeMax: 6, Difficulty: 1, Visibility: "linked_students"})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = st.SetCourseStatus(t.Context(), author, course.ID, "published"); e != nil {
			t.Fatal(e)
		}
		topic, e := st.CreateTopic(t.Context(), author, course.ID, domain.TopicInput{Title: "Fractions"})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = st.SetTopicStatus(t.Context(), author, topic.ID, "published"); e != nil {
			t.Fatal(e)
		}
		l, e := st.CreateLearningLesson(t.Context(), author, course.ID, domain.LearningLessonInput{Title: "Published fractions " + id(), Content: "Learn fractions", TopicID: topic.ID, Track: "school", GradeMin: 5, GradeMax: 6, Difficulty: 1})
		if e != nil {
			t.Fatal(e)
		}
		l, e = st.PublishLearningLesson(t.Context(), author, l.ID)
		if e != nil {
			t.Fatal(e)
		}
		return l
	}
	material, private := lesson(tutor), lesson(other)
	call(ts, "POST", base, domain.MessageInput{LessonID: private.ID}, id(), 404, nil)
	call(ts, "POST", base, domain.MessageInput{LessonID: material.ID}, id(), 201, &sent)
	call(ss, "GET", base+"/references?kind=lesson&q="+url.QueryEscape(material.Version.Title), nil, "", 200, &references)
	if len(references.Items) != 1 || references.Items[0].ID != material.ID {
		t.Fatal("private lesson leak", references)
	}
	// Draft edits cannot replace the published title in chat attachments.
	draft := material.Version.LearningLessonInput
	draft.Title = "Secret draft title"
	if _, err = st.UpdateLearningLesson(t.Context(), tutor, material.ID, draft); err != nil {
		t.Fatal(err)
	}
	list, _ = call(ss, "GET", base, nil, "", 200, &messages)
	if bytes.Contains(list, []byte("Secret draft")) {
		t.Fatal("draft leaked")
	}
	if _, err = st.ArchiveLearningLesson(t.Context(), tutor, material.ID); err != nil {
		t.Fatal(err)
	}
	messages.Items = nil
	call(ss, "GET", base, nil, "", 200, &messages)
	for _, m := range messages.Items {
		if m.ID == sent.ID && (m.Attachment == nil || m.Attachment.Available || m.Attachment.ID != "" || m.Attachment.Title != "") {
			t.Fatal("archived reference remains available", m)
		}
	}
	if _, err = st.SubmitAssignment(t.Context(), student, assignment.ID, "1"); err != nil {
		t.Fatal(err)
	}
	if _, err = st.GradeAssignment(t.Context(), tutor, assignment.ID, 10, "Good", false); err != nil {
		t.Fatal(err)
	}
	for _, actor := range []domain.User{student, tutor} {
		notes, e := st.ListNotifications(t.Context(), actor)
		if e != nil {
			t.Fatal(e)
		}
		found := false
		for _, n := range notes {
			if n.Href == "/tasks?task="+assignment.ID {
				found = true
			}
		}
		if !found {
			t.Fatal("assignment notification deep link missing", actor.ID)
		}
	}
	if _, err = pool.Exec(t.Context(), `update assignments set status='archived' where id=$1`, assignment.ID); err != nil {
		t.Fatal(err)
	}
	call(ss, "GET", base+"/assignments/"+assignment.ID, nil, "", 404, nil)
	if _, err = pool.Exec(t.Context(), `update relations set status='ended' where tutor_id=$1 and student_id=$2`, tutor.ID, student.ID); err != nil {
		t.Fatal(err)
	}
	call(ss, "GET", base+"/files/"+file.ID, nil, "", 404, nil)
	call(ts, "GET", base+"/references?kind=lesson", nil, "", 404, nil)
}
