package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

func TestMessageReadsPostgres(t *testing.T) {
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
	st := store.New(pool)
	id := func() string {
		v, e := platform.NewUUID()
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	register := func(role domain.Role) (domain.User, string) {
		u, token, e := st.RegisterUser(t.Context(), id()+"@example.test", "Message fixture", "Password123!", role, "test", "127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		return u, token
	}
	tutor, tutorSession := register(domain.RoleTutor)
	student, studentSession := register(domain.RoleStudent)
	peer, peerSession := register(domain.RoleStudent)
	admin, err := st.EnsureUser(t.Context(), auth.Principal{UserID: id(), Email: id() + "@example.test", DisplayName: "Admin test fixture", Role: domain.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	adminSession, err := st.CreateSession(t.Context(), admin, "test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	_, inv, err := st.CreateInvitation(t.Context(), tutor, student.Email, "")
	if err != nil {
		t.Fatal(err)
	}
	relation, err := st.AcceptInvitation(t.Context(), student, inv)
	if err != nil {
		t.Fatal(err)
	}
	conversations, err := st.ListConversations(t.Context(), student)
	if err != nil || len(conversations) != 1 {
		t.Fatal(err, conversations)
	}
	conv := conversations[0].ID
	srv := httptest.NewServer(NewServer(config.Config{RequestBodyLimitByte: 1 << 20}, slog.New(slog.NewTextHandler(io.Discard, nil)), st, auth.DevProvider{}))
	defer srv.Close()
	call := func(token, method, path string, body any, key string, want int, out any) {
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
		req.AddCookie(&http.Cookie{Name: store.SessionCookieName, Value: token})
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
	}
	path := "/api/v1/messages/conversations/" + conv
	nonce := id()
	var first, again, second domain.Message
	call(tutorSession, "POST", path, map[string]string{"body": "Hello"}, nonce, 201, &first)
	call(tutorSession, "POST", path, map[string]string{"body": "Hello"}, nonce, 201, &again)
	if again.ID != first.ID || again.Sequence != first.Sequence || first.Sequence < 1 {
		t.Fatal("message replay/order", first, again)
	}
	call(tutorSession, "POST", path, map[string]string{"body": "Changed"}, nonce, 409, nil)
	call(tutorSession, "POST", path, map[string]string{"body": "Second"}, id(), 201, &second)
	if second.Sequence <= first.Sequence {
		t.Fatal("non-monotone order")
	}
	var list struct {
		Items []domain.Conversation `json:"conversations"`
	}
	call(studentSession, "GET", "/api/v1/messages/conversations", nil, "", 200, &list)
	if len(list.Items) != 1 || list.Items[0].UnreadCount != 2 {
		t.Fatal("incoming unread", list)
	}
	call(tutorSession, "GET", "/api/v1/messages/conversations", nil, "", 200, &list)
	if list.Items[0].UnreadCount != 0 {
		t.Fatal("own messages counted unread")
	}
	var result struct {
		Unread int `json:"unread_count"`
	}
	readPath := path + "/read"
	call(studentSession, "POST", readPath, map[string]string{"through_message_id": first.ID}, "", 200, &result)
	if result.Unread != 1 {
		t.Fatal("read consumed unseen message", result)
	}
	call(studentSession, "POST", readPath, map[string]string{"through_message_id": second.ID}, "", 200, &result)
	if result.Unread != 0 {
		t.Fatal(result)
	}
	call(studentSession, "POST", readPath, map[string]string{"through_message_id": first.ID}, "", 200, &result)
	if result.Unread != 0 {
		t.Fatal("old cursor moved backwards", result)
	}
	for _, body := range []map[string]any{{"through_message_id": first.ID, "last_sequence": 999999}, {"through_message_id": first.ID, "user_id": student.ID}, {"through_message_id": first.ID, "role": "tutor"}} {
		call(studentSession, "POST", readPath, body, "", 400, nil)
	}
	call(studentSession, "POST", readPath, map[string]string{"through_message_id": id()}, "", 404, nil)
	for _, session := range []string{peerSession, adminSession} {
		call(session, "GET", path, nil, "", 404, nil)
		call(session, "POST", readPath, map[string]string{"through_message_id": first.ID}, "", 404, nil)
		call(session, "POST", path, map[string]string{"body": "Foreign"}, id(), 404, nil)
	}
	// A late commit with an earlier wall-clock timestamp must still be unread.
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(t.Context())
	lateID := id()
	var lateSequence int64
	if err = tx.QueryRow(t.Context(), `insert into messages(id,organization_id,conversation_id,author_id,body,idempotency_key)values($1,$2,$3,$4,'Late commit',$5)returning sequence`, lateID, tutor.OrganizationID, conv, tutor.ID, id()).Scan(&lateSequence); err != nil {
		t.Fatal(err)
	}
	call(studentSession, "POST", readPath, map[string]string{"through_message_id": second.ID}, "", 200, &result)
	if err = tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	call(studentSession, "GET", "/api/v1/messages/conversations", nil, "", 200, &list)
	if list.Items[0].UnreadCount != 1 || lateSequence <= second.Sequence {
		t.Fatal("late commit lost unread", list)
	}
	call(tutorSession, "POST", path, map[string]string{"body": strings.Repeat("Ж", 4000)}, id(), 201, nil)
	call(tutorSession, "POST", path, map[string]string{"body": strings.Repeat("Ж", 4001)}, id(), 400, nil)
	// A message from a different conversation cannot advance this cursor.
	_, otherInv, err := st.CreateInvitation(t.Context(), tutor, peer.Email, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.AcceptInvitation(t.Context(), peer, otherInv); err != nil {
		t.Fatal(err)
	}
	otherConvs, err := st.ListConversations(t.Context(), peer)
	if err != nil {
		t.Fatal(err)
	}
	otherMessage, err := st.SendMessage(t.Context(), tutor, otherConvs[0].ID, "Other conversation", id())
	if err != nil {
		t.Fatal(err)
	}
	call(studentSession, "POST", readPath, map[string]string{"through_message_id": otherMessage.ID}, "", 404, nil)
	if _, err = pool.Exec(t.Context(), `insert into message_read_cursors(conversation_id,user_id,last_sequence)values($1,$2,$3)`, conv, peer.ID, first.Sequence); err == nil {
		t.Fatal("DB allowed foreign read cursor")
	}
	if _, err = pool.Exec(t.Context(), `update relations set status='ended' where id=$1`, relation.ID); err != nil {
		t.Fatal(err)
	}
	call(studentSession, "GET", path, nil, "", 404, nil)
	call(studentSession, "POST", readPath, map[string]string{"through_message_id": lateID}, "", 404, nil)
	call(tutorSession, "POST", path, map[string]string{"body": "Revoked"}, id(), 404, nil)
	call(studentSession, "GET", "/api/v1/messages/conversations", nil, "", 200, &list)
	if len(list.Items) != 0 {
		t.Fatal("ended conversation still listed", list)
	}
}
