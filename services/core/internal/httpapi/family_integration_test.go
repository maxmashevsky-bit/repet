package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
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
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestFamilyAccessPostgres(t *testing.T) {
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
	srv := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), store.New(pool), auth.DevProvider{}))
	defer srv.Close()
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	call := func(c *http.Client, method, path string, body any, want int, out any) []byte {
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
		res, e := c.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, e := io.ReadAll(res.Body)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s=%d want%d: %s", method, path, res.StatusCode, want, raw)
		}
		if out != nil {
			if e = json.Unmarshal(raw, out); e != nil {
				t.Fatal(e)
			}
		}
		return raw
	}
	reg := func(role string) (*http.Client, domain.User) {
		j, e := cookiejar.New(nil)
		if e != nil {
			t.Fatal(e)
		}
		c := &http.Client{Transport: srv.Client().Transport, Jar: j}
		var out struct {
			User domain.User `json:"user"`
		}
		call(c, "POST", "/api/v1/auth/register", map[string]any{"email": id() + "@example.test", "display_name": "Family " + role, "role": role, "password": "FamilyTesting123!", "accept_terms": true}, 201, &out)
		return c, out.User
	}
	tc, tutor := reg("tutor")
	sc, student := reg("student")
	pc, parent := reg("student")
	foreign, foreignParent := reg("student")
	foreignTutor, _ := reg("tutor")
	var teaching struct {
		Token string `json:"token"`
	}
	call(tc, "POST", "/api/v1/invitations", map[string]any{"student_email": student.Email}, 201, &teaching)
	var relation domain.Relation
	call(sc, "POST", "/api/v1/invitations/"+teaching.Token+"/accept", map[string]any{}, 201, &relation)
	reportPath := "/api/v1/family/children/" + student.ID
	call(pc, "GET", reportPath, nil, 404, nil)
	call(sc, "GET", reportPath, nil, 404, nil)
	call(tc, "GET", reportPath, nil, 404, nil)
	call(tc, "POST", "/api/v1/family/invitations", domain.GuardianInviteInput{GuardianEmail: parent.Email, ConfirmShare: true}, 403, nil)
	call(sc, "POST", "/api/v1/family/invitations", map[string]any{"guardian_email": parent.Email, "confirm_share": true, "guardian_id": parent.ID}, 400, nil)
	call(sc, "POST", "/api/v1/family/invitations", domain.GuardianInviteInput{GuardianEmail: parent.Email}, 400, nil)
	var invite struct {
		Link      domain.GuardianLink `json:"link"`
		AcceptURL string              `json:"accept_url"`
	}
	call(sc, "POST", "/api/v1/family/invitations", domain.GuardianInviteInput{GuardianEmail: strings.ToUpper(parent.Email), ConfirmShare: true}, 201, &invite)
	u, e := url.Parse(invite.AcceptURL)
	if e != nil {
		t.Fatal(e)
	}
	token := strings.TrimPrefix(u.Fragment, "token=")
	if len(token) < 32 || u.RawQuery != "" {
		t.Fatal("token must be in fragment", u)
	}
	var hash string
	if e = pool.QueryRow(t.Context(), `select token_hash from guardian_links where id=$1`, invite.Link.ID).Scan(&hash); e != nil || hash == token || hash != platform.TokenHash(token) {
		t.Fatal("token not hashed", e)
	}
	var links domain.CatalogPage[domain.GuardianLink]
	raw := call(sc, "GET", "/api/v1/family/links", nil, 200, &links)
	if links.Total != 1 || strings.Contains(string(raw), token) || strings.Contains(string(raw), hash) {
		t.Fatal("secret or missing link", string(raw))
	}
	call(foreign, "POST", "/api/v1/family/invitations/preview", map[string]string{"token": token}, 404, nil)
	call(foreign, "POST", "/api/v1/family/invitations/accept", map[string]string{"token": token}, 404, nil)
	call(pc, "POST", "/api/v1/family/invitations/preview", map[string]string{"token": token}, 200, nil)
	call(pc, "GET", reportPath, nil, 404, nil)
	for range 2 {
		call(pc, "POST", "/api/v1/family/invitations/accept", map[string]string{"token": token}, 200, nil)
	}
	childB, childBUser := reg("student")
	var inviteB struct {
		AcceptURL string `json:"accept_url"`
	}
	call(childB, "POST", "/api/v1/family/invitations", domain.GuardianInviteInput{GuardianEmail: foreignParent.Email, ConfirmShare: true}, 201, &inviteB)
	otherURL, parseErr := url.Parse(inviteB.AcceptURL)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	call(foreign, "POST", "/api/v1/family/invitations/accept", map[string]string{"token": strings.TrimPrefix(otherURL.Fragment, "token=")}, 200, nil)
	call(pc, "GET", "/api/v1/family/children/"+childBUser.ID, nil, 404, nil)
	call(foreign, "GET", reportPath, nil, 404, nil)
	var report domain.FamilyReport
	if _, e = store.New(pool).FamilyReport(t.Context(), parent, student.ID, domain.FamilyFilter{AnalyticsFilter: domain.AnalyticsFilter{From: time.Now().Add(-time.Hour), To: time.Now().Add(time.Hour)}}); e != nil {
		t.Fatal("family projection", e)
	}
	call(pc, "GET", reportPath, nil, 200, &report)
	if report.Student.ID != student.ID || report.Summary.Attempts != 0 || report.Events.Total != 0 {
		t.Fatal(report)
	}
	for _, q := range []string{"lesson_offset=-1", "event_offset=100001", "from=bad", "role=tutor", "to=2020-01-01&to=2021-01-01"} {
		call(pc, "GET", reportPath+"?"+q, nil, 400, nil)
	}
	var lesson domain.Lesson
	call(tc, "POST", "/api/v1/lessons", map[string]any{"relation_id": relation.ID, "title": "Academic lesson", "starts_at": time.Now().Add(time.Hour), "ends_at": time.Now().Add(2 * time.Hour)}, 201, &lesson)
	var assignment domain.Assignment
	call(tc, "POST", "/api/v1/assignments", map[string]any{"student_id": student.ID, "title": "Academic homework", "subject": "Математика", "topic": "Дроби", "body": "Private assignment body", "due_at": time.Now().Add(24 * time.Hour), "max_score": 10}, 201, &assignment)
	call(sc, "POST", "/api/v1/assignments/"+assignment.ID+"/submit", map[string]string{"answer": "PRIVATE_STUDENT_ANSWER"}, 200, nil)
	call(tc, "POST", "/api/v1/assignments/"+assignment.ID+"/grade", map[string]any{"score": 8, "feedback": "PRIVATE_TEACHER_FEEDBACK"}, 200, nil)
	var conversations struct {
		Items []domain.Conversation `json:"conversations"`
	}
	call(sc, "GET", "/api/v1/messages/conversations", nil, 200, &conversations)
	call(sc, "POST", "/api/v1/messages/conversations/"+conversations.Items[0].ID, map[string]any{"body": "PRIVATE_CHAT_MESSAGE"}, 201, nil)
	noteIn := domain.GuardianNoteInput{StudentID: student.ID, Body: "Разрешённый родителю комментарий", RequestID: id()}
	var note, replay domain.GuardianNote
	call(foreignTutor, "POST", "/api/v1/family/notes", noteIn, 404, nil)
	call(pc, "POST", "/api/v1/family/notes", noteIn, 403, nil)
	call(tc, "POST", "/api/v1/family/notes", noteIn, 201, &note)
	call(tc, "POST", "/api/v1/family/notes", noteIn, 201, &replay)
	if note.ID != replay.ID {
		t.Fatal("note replay")
	}
	noteIn.Body = "Changed"
	call(tc, "POST", "/api/v1/family/notes", noteIn, 409, nil)
	raw = call(pc, "GET", reportPath, nil, 200, &report)
	for _, secret := range []string{"PRIVATE_CHAT_MESSAGE", "PRIVATE_STUDENT_ANSWER", "PRIVATE_TEACHER_FEEDBACK", "Private assignment body", "token_hash", "password_hash"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("private leak", secret)
		}
	}
	if report.Lessons.Total != 1 || report.Assignments.Total != 1 || report.Assignments.Items[0].Score == nil || *report.Assignments.Items[0].Score != 8 || report.Notes.Total != 1 || report.Events.Total != 5 || report.Unread != 5 {
		t.Fatal("academic projection/events", report)
	}
	call(pc, "POST", reportPath+"/read", map[string]any{}, 200, nil)
	call(pc, "GET", reportPath, nil, 200, &report)
	if report.Unread != 0 {
		t.Fatal("read watermark", report.Unread)
	}
	call(foreign, "GET", reportPath, nil, 404, nil)
	call(pc, "GET", "/api/v1/lessons/"+lesson.ID, nil, 404, nil)
	call(pc, "GET", "/api/v1/assignments/"+assignment.ID, nil, 404, nil)
	call(pc, "GET", "/api/v1/messages/conversations/"+conversations.Items[0].ID, nil, 404, nil)
	call(pc, "GET", "/api/v1/learning-analytics/"+student.ID, nil, 404, nil)
	call(foreignTutor, "POST", "/api/v1/family/notes/"+note.ID+"/withdraw", map[string]any{}, 404, nil)
	call(tc, "POST", "/api/v1/family/notes/"+note.ID+"/withdraw", map[string]any{}, 200, nil)
	call(pc, "GET", reportPath, nil, 200, &report)
	if report.Notes.Total != 0 {
		t.Fatal("withdrawn note")
	}
	// Native results are deliberately a projection, not a request made with a forged student actor.
	var course domain.Course
	call(tc, "POST", "/api/v1/catalog/courses", domain.CourseInput{Title: "Family own course", Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1, Visibility: "linked_students"}, 201, &course)
	var topic domain.Topic
	call(tc, "POST", "/api/v1/catalog/courses/"+course.ID+"/topics", domain.TopicInput{Title: "Family topic"}, 201, &topic)
	call(tc, "POST", "/api/v1/catalog/courses/"+course.ID+"/publish", map[string]any{}, 200, nil)
	call(tc, "POST", "/api/v1/catalog/topics/"+topic.ID+"/publish", map[string]any{}, 200, nil)
	var material domain.LearningLesson
	call(tc, "POST", "/api/v1/catalog/courses/"+course.ID+"/lessons", domain.LearningLessonInput{Title: "Family native", TopicID: topic.ID, Track: "school", GradeMin: 5, GradeMax: 11, Difficulty: 1}, 201, &material)
	lp := "/api/v1/catalog/lessons/" + material.ID
	var ex domain.Exercise
	call(tc, "POST", lp+"/exercises", domain.ExerciseInput{Kind: "numeric", Prompt: "3 + 4", Difficulty: 1, AnswerKey: domain.AnswerKeyInput{Answer: "7", Explanation: "PRIVATE_NATIVE_SOLUTION"}}, 201, &ex)
	call(tc, "POST", lp+"/publish", map[string]any{}, 200, nil)
	var state domain.LearningState
	call(sc, "POST", lp+"/learning-attempts", domain.LearningStartInput{RequestID: id()}, 201, &state)
	value := "7"
	call(sc, "POST", lp+"/exercises/"+ex.ID+"/attempts", domain.ExerciseAttemptInput{RequestID: id(), VersionID: state.Version.ID, LearningAttemptID: state.Attempt.ID, Answer: domain.ExerciseAnswer{Value: &value}}, 201, nil)
	call(sc, "POST", lp+"/learning-attempts/"+state.Attempt.ID+"/complete", domain.LearningActionInput{RequestID: id()}, 200, nil)
	raw = call(pc, "GET", reportPath, nil, 200, &report)
	if report.Summary.Graded != 1 || report.Summary.Correct != 1 || report.Results.Total != 1 || report.Topics.Total != 1 || len(report.Timeline) != 1 || report.Events.Total != 6 || strings.Contains(string(raw), "PRIVATE_NATIVE_SOLUTION") || strings.Contains(string(raw), "\"answer\"") {
		t.Fatal("native projection", string(raw))
	}
	call(pc, "GET", reportPath+"?result_offset=1&lesson_offset=1&assignment_offset=1&event_offset=6", nil, 200, &report)
	if len(report.Results.Items) != 0 || report.Results.Total != 1 || len(report.Lessons.Items) != 0 || len(report.Assignments.Items) != 0 || len(report.Events.Items) != 0 {
		t.Fatal("pages", report)
	}
	number := 7
	var bank domain.BankExercise
	call(tc, "POST", "/api/v1/exercise-bank", domain.BankExerciseInput{LessonID: material.ID, ExerciseID: ex.ID, BankMetadata: domain.BankMetadata{Grade: 9, Direction: "oge", Topic: "Family topic", ExamNumber: &number, Source: "Own fixture", Attribution: tutor.DisplayName, RightsStatus: "own", License: "Own permission"}}, 201, &bank)
	var collection domain.BankCollection
	call(tc, "POST", "/api/v1/bank-collections", domain.BankCollectionInput{RequestID: id(), CourseID: course.ID, TopicID: topic.ID, Title: "Family exam source", BankIDs: []string{bank.ID}}, 201, &collection)
	var specification domain.ExamSpecification
	call(tc, "POST", "/api/v1/exam-specifications", domain.ExamSpecificationInput{RequestID: id(), CollectionID: collection.ID, Title: "Own family mock exam", Direction: "oge", Year: 2026, DurationSeconds: 600, Slots: []domain.ExamSlot{{Number: 7, Count: 1, Points: 2}}}, 201, &specification)
	call(tc, "POST", "/api/v1/exam-specifications/"+specification.ID+"/publish", map[string]any{}, 200, nil)
	var exam domain.StudySession
	call(sc, "POST", "/api/v1/study-sessions", domain.StudyStartInput{RequestID: id(), SpecificationID: specification.ID, Mode: "exam", Seed: "family-fixture"}, 201, &exam)
	wrong := "0"
	examPath := "/api/v1/catalog/lessons/" + exam.State.Attempt.LessonID
	call(sc, "POST", examPath+"/exercises/"+exam.State.Version.Exercises[0].ID+"/attempts", domain.ExerciseAttemptInput{RequestID: id(), VersionID: exam.State.Version.ID, LearningAttemptID: exam.State.Attempt.ID, Answer: domain.ExerciseAnswer{Value: &wrong}}, 201, nil)
	call(pc, "GET", reportPath, nil, 200, &report)
	if report.Summary.Graded != 1 || report.Summary.Incorrect != 0 || report.Exams.Total != 0 || report.Results.Total != 1 {
		t.Fatal("parent saw active exam", report)
	}
	call(sc, "POST", examPath+"/learning-attempts/"+exam.State.Attempt.ID+"/complete", domain.LearningActionInput{RequestID: id()}, 200, nil)
	call(pc, "GET", reportPath, nil, 200, &report)
	if report.Summary.Graded != 2 || report.Summary.Incorrect != 1 || report.Exams.Total != 1 || report.Exams.Items[0].Points == nil || *report.Exams.Items[0].Points != 0 || report.Exams.Items[0].MaxPoints != 2 {
		t.Fatal("parent exam history", report)
	}
	var ownNotes domain.CatalogPage[domain.GuardianNote]
	call(tc, "GET", "/api/v1/family/students/"+student.ID+"/notes", nil, 200, &ownNotes)
	if ownNotes.Total != 0 {
		t.Fatal("withdrawn tutor note was listed")
	}
	call(foreignTutor, "GET", "/api/v1/family/students/"+student.ID+"/notes", nil, 404, nil)
	call(pc, "GET", "/api/v1/family/students/"+student.ID+"/notes", nil, 403, nil)
	// An accepted guardian cannot use a grant from a different organization.
	org := id()
	if _, e = pool.Exec(t.Context(), `insert into organizations(id,name)values($1,'Family isolation fixture')`, org); e != nil {
		t.Fatal(e)
	}
	outsider := parent
	outsider.OrganizationID = org
	if _, e = store.New(pool).FamilyReport(t.Context(), outsider, student.ID, domain.FamilyFilter{AnalyticsFilter: domain.AnalyticsFilter{From: time.Now().Add(-time.Hour), To: time.Now().Add(time.Hour)}}); e == nil {
		t.Fatal("cross-organization grant")
	}
	if _, e = pool.Exec(t.Context(), `update guardian_links set student_id=$2 where id=$1`, invite.Link.ID, tutor.ID); e == nil {
		t.Fatal("guardian student changed")
	}
	call(foreign, "POST", "/api/v1/family/links/"+invite.Link.ID+"/revoke", map[string]any{}, 404, nil)
	for range 2 {
		call(sc, "POST", "/api/v1/family/links/"+invite.Link.ID+"/revoke", map[string]any{}, 200, nil)
	}
	call(pc, "GET", reportPath, nil, 404, nil)
	call(pc, "POST", reportPath+"/read", map[string]any{}, 404, nil)
	call(pc, "GET", "/api/v1/family/links", nil, 200, &links)
	if links.Total != 0 {
		t.Fatal("revoked guardian still sees child's changing profile", links)
	}
	call(pc, "POST", "/api/v1/family/invitations/accept", map[string]string{"token": token}, 422, nil)
	// Expiry is checked against server time, never from the request body.
	token = id()
	if _, e = pool.Exec(t.Context(), `insert into guardian_links(organization_id,student_id,guardian_email,token_hash,created_at,expires_at)values($1,$2,$3,$4,clock_timestamp()-interval '8 days',clock_timestamp()-interval '1 day')`, student.OrganizationID, student.ID, parent.Email, platform.TokenHash(token)); e != nil {
		t.Fatal(e)
	}
	call(pc, "POST", "/api/v1/family/invitations/accept", map[string]string{"token": token}, 422, nil)
}
