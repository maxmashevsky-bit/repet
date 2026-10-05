package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func TestProductFlowPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	st := New(pool)
	unique, err := platform.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	register := func(name string, role domain.Role) (domain.User, string) {
		t.Helper()
		user, session, err := st.RegisterUser(ctx, name+unique+"@example.test", name, "password123", role, "test", "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		return user, session
	}
	tutor, session := register("tutor", domain.RoleTutor)
	student, _ := register("student", domain.RoleStudent)
	other, _ := register("other", domain.RoleStudent)
	otherTutor, _ := register("other-tutor", domain.RoleTutor)
	if got, err := st.UserBySessionToken(ctx, session); err != nil || got.ID != tutor.ID {
		t.Fatalf("session lookup: %v", err)
	}
	inv, token, err := st.CreateInvitation(ctx, tutor, student.Email, student.DisplayName)
	if err != nil || inv.Status != "pending" {
		t.Fatalf("create invitation: %v", err)
	}
	if _, err := st.AcceptInvitation(ctx, other, token); !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("different email accepted: %v", err)
	}
	relation, err := st.AcceptInvitation(ctx, student, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AcceptInvitation(ctx, student, token); !errors.Is(err, apperr.ErrInvalidState) {
		t.Fatalf("duplicate invitation accepted: %v", err)
	}
	_, concurrentToken, err := st.CreateInvitation(ctx, tutor, student.Email, student.DisplayName)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	acceptErrors := make([]error, 2)
	for i := range acceptErrors {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, acceptErrors[index] = st.AcceptInvitation(ctx, student, concurrentToken)
		}(i)
	}
	wg.Wait()
	if !(acceptErrors[0] == nil && errors.Is(acceptErrors[1], apperr.ErrInvalidState) || acceptErrors[1] == nil && errors.Is(acceptErrors[0], apperr.ErrInvalidState)) {
		t.Fatalf("concurrent invitation results: %v", acceptErrors)
	}
	if _, err := st.GetRelationForActor(ctx, otherTutor, relation.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("unrelated tutor sees relation: %v", err)
	}
	conversations, err := st.ListConversations(ctx, student)
	if err != nil || len(conversations) != 1 {
		t.Fatalf("conversation after invitation: %v, %d", err, len(conversations))
	}
	starts := time.Now().UTC().Add(24 * time.Hour)
	lesson, err := st.CreateLesson(ctx, tutor, relation.ID, "Algebra", starts, starts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetLessonForActor(ctx, student, lesson.ID); err != nil {
		t.Fatalf("student lesson: %v", err)
	}
	if _, err := st.GetLessonForActor(ctx, other, lesson.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("other student sees lesson: %v", err)
	}
	if _, err := st.CreateLesson(ctx, student, relation.ID, "Wrong", starts, starts.Add(time.Hour)); !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("student created lesson: %v", err)
	}
	assignment, err := st.CreateAssignment(ctx, tutor, student.ID, "Homework", "Math", "Algebra", "Solve", starts.Add(time.Hour), 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.getAssignmentForActor(ctx, other, assignment.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("other student sees assignment: %v", err)
	}
	if _, err := st.SubmitAssignment(ctx, other, assignment.ID, "stolen"); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("other student submitted: %v", err)
	}
	if _, err := st.SubmitAssignment(ctx, student, assignment.ID, "answer"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SubmitAssignment(ctx, student, assignment.ID, "twice"); !errors.Is(err, apperr.ErrInvalidState) {
		t.Fatalf("duplicate submission: %v", err)
	}
	second, err := st.CreateAssignment(ctx, tutor, student.ID, "Concurrent", "Math", "Algebra", "Solve", starts.Add(time.Hour), 100)
	if err != nil {
		t.Fatal(err)
	}
	submitErrors := make([]error, 2)
	for i := range submitErrors {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, submitErrors[index] = st.SubmitAssignment(ctx, student, second.ID, "answer")
		}(i)
	}
	wg.Wait()
	if !(submitErrors[0] == nil && errors.Is(submitErrors[1], apperr.ErrInvalidState) || submitErrors[1] == nil && errors.Is(submitErrors[0], apperr.ErrInvalidState)) {
		t.Fatalf("concurrent submission results: %v", submitErrors)
	}
	graded, err := st.GradeAssignment(ctx, tutor, assignment.ID, 85, "Good work", false)
	if err != nil || graded.Status != "done" || graded.Answer != "answer" {
		t.Fatalf("grade: %v, %+v", err, graded)
	}
	visible, err := st.getAssignmentForActor(ctx, student, assignment.ID)
	if err != nil || visible.Feedback != "Good work" || visible.Score == nil || *visible.Score != 85 {
		t.Fatalf("student feedback: %v, %+v", err, visible)
	}
	if _, err := st.GradeAssignment(ctx, otherTutor, assignment.ID, 0, "No", false); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("other tutor graded: %v", err)
	}
	conversationID := conversations[0].ID
	msg, err := st.SendMessage(ctx, tutor, conversationID, "Hello", "test-key")
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := st.SendMessage(ctx, tutor, conversationID, "Hello", "test-key")
	if err != nil || repeated.ID != msg.ID {
		t.Fatalf("message idempotency: %v", err)
	}
	history, err := st.ListMessages(ctx, student, conversationID, 0)
	if err != nil || len(history) != 1 || history[0].AuthorID != tutor.ID {
		t.Fatalf("message history: %v, %+v", err, history)
	}
	if _, err := st.ListMessages(ctx, other, conversationID, 0); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("other student sees chat: %v", err)
	}
	notifications, err := st.ListNotifications(ctx, student)
	if err != nil || len(notifications) < 3 {
		t.Fatalf("student notifications: %v, %d", err, len(notifications))
	}
	if err := st.MarkNotificationRead(ctx, other, notifications[0].ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("other student marked notification: %v", err)
	}
	if err := st.MarkNotificationRead(ctx, student, notifications[0].ID); err != nil {
		t.Fatal(err)
	}
	resetToken, _, err := st.IssuePasswordReset(ctx, student.Email)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ResetPassword(ctx, resetToken, "newpassword123"); err != nil {
		t.Fatal(err)
	}
	if err := st.ResetPassword(ctx, resetToken, "anotherpassword123"); !errors.Is(err, apperr.ErrInvalidState) {
		t.Fatalf("reset token reused: %v", err)
	}
	if _, _, err := st.LoginUser(ctx, student.Email, "newpassword123", "test", "127.0.0.1"); err != nil {
		t.Fatalf("new password login: %v", err)
	}
	if err := st.RevokeSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UserBySessionToken(ctx, session); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("revoked session valid: %v", err)
	}
}
