package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"tutor-platform/services/core/internal/auth"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
	"tutor-platform/services/core/internal/store"
)

type Server struct {
	cfg      config.Config
	log      *slog.Logger
	store    *store.Store
	provider auth.IdentityProvider
}

func NewServer(cfg config.Config, log *slog.Logger, st *store.Store, provider auth.IdentityProvider) http.Handler {
	s := &Server{cfg: cfg, log: log, store: st, provider: provider}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLog(log))
	r.Use(middleware.Recoverer)
	r.Use(SecurityHeaders)
	r.Use(CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.SetHeader("Cache-Control", "no-store"))
	r.Use(middleware.RequestSize(int64(cfg.RequestBodyLimitByte)))

	r.Get("/healthz", s.health)
	r.Get("/readyz", s.ready)
	r.Get("/metrics", s.metrics)
	r.Post("/api/v1/auth/register", s.register)
	r.Post("/api/v1/auth/login", s.login)
	r.Post("/api/v1/auth/forgot-password", s.forgotPassword)
	r.Post("/api/v1/auth/reset-password", s.resetPassword)

	r.Group(func(api chi.Router) {
		api.Use(s.authenticate)
		api.Get("/api/v1/me", s.me)
		api.Post("/api/v1/auth/logout", s.logout)
		api.Get("/api/v1/dashboard", s.dashboard)
		api.Get("/api/v1/profile", s.me)
		api.Post("/api/v1/profile", s.updateProfile)
		api.Post("/api/v1/invitations", s.createInvitation)
		api.Post("/api/v1/invitations/{token}/accept", s.acceptInvitation)
		api.Get("/api/v1/relations", s.listRelations)
		api.Post("/api/v1/lessons", s.createLesson)
		api.Get("/api/v1/lessons", s.listLessons)
		api.Get("/api/v1/lessons/{lessonID}", s.getLesson)
		api.Get("/api/v1/messages/conversations", s.listConversations)
		api.Get("/api/v1/messages/conversations/{conversationID}", s.listMessages)
		api.Post("/api/v1/messages/conversations/{conversationID}", s.sendMessage)
		api.Get("/api/v1/assignments", s.listAssignments)
		api.Post("/api/v1/assignments", s.createAssignment)
		api.Post("/api/v1/assignments/{assignmentID}/submit", s.submitAssignment)
		api.Post("/api/v1/assignments/{assignmentID}/grade", s.gradeAssignment)
		api.Get("/api/v1/notifications", s.listNotifications)
		api.Post("/api/v1/notifications/{notificationID}/read", s.markNotificationRead)
		api.Post("/api/v1/notifications/mark-all-read", s.markAllNotificationsRead)
	})
	return r
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		WriteError(w, ErrInternal)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte("# HELP repet_core_up Process liveness.\n# TYPE repet_core_up gauge\nrepet_core_up 1\n"))
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DisplayName string      `json:"display_name"`
		Email       string      `json:"email"`
		Password    string      `json:"password"`
		Role        domain.Role `json:"role"`
		AcceptTerms bool        `json:"accept_terms"`
	}
	if err := decodeJSON(r, &req); err != nil || !req.AcceptTerms {
		WriteError(w, ErrBadRequest)
		return
	}
	user, token, err := s.store.RegisterUser(r.Context(), req.Email, req.DisplayName, req.Password, req.Role, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		WriteError(w, err)
		return
	}
	s.setSessionCookie(w, token)
	WriteJSON(w, http.StatusCreated, map[string]domain.User{"user": user})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	user, token, err := s.store.LoginUser(r.Context(), req.Email, req.Password, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			WriteError(w, ErrUnauthorized)
		} else {
			WriteError(w, err)
		}
		return
	}
	s.setSessionCookie(w, token)
	WriteJSON(w, http.StatusOK, map[string]domain.User{"user": user})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(store.SessionCookieName); err == nil {
		if err := s.store.RevokeSession(r.Context(), cookie.Value); err != nil {
			WriteError(w, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     store.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	token, email, err := s.store.IssuePasswordReset(r.Context(), req.Email)
	if err != nil {
		WriteError(w, err)
		return
	}
	if token != "" {
		if err := sendResetEmail(r.Context(), s.cfg, email, token); err != nil {
			s.log.Error("password reset email failed", "error", err)
			WriteError(w, ErrInternal)
			return
		}
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "if_account_exists_email_will_be_sent"})
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	if err := s.store.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "password_updated"})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, user)
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct {
		DisplayName string `json:"display_name"`
		Timezone    string `json:"timezone"`
		Locale      string `json:"locale"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	updated, err := s.store.UpdateProfile(r.Context(), user, req.DisplayName, req.Timezone, req.Locale)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, updated)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	relations, err := s.store.ListRelations(r.Context(), user)
	if err != nil {
		WriteError(w, err)
		return
	}
	lessons, err := s.store.ListLessons(r.Context(), user)
	if err != nil {
		WriteError(w, err)
		return
	}
	assignments, err := s.store.ListAssignments(r.Context(), user)
	if err != nil {
		WriteError(w, err)
		return
	}
	conversations, err := s.store.ListConversations(r.Context(), user)
	if err != nil {
		WriteError(w, err)
		return
	}
	notifications, err := s.store.ListNotifications(r.Context(), user)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, dashboardResponse{
		User:          user,
		Relations:     relations,
		Lessons:       lessons,
		Assignments:   assignments,
		Conversations: conversations,
		Notifications: notifications,
	})
}

func (s *Server) createInvitation(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct {
		StudentEmail string `json:"student_email"`
		StudentName  string `json:"student_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	inv, token, err := s.store.CreateInvitation(r.Context(), user, req.StudentEmail, req.StudentName)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{
		"invitation": inv,
		"token":      token,
		"accept_url": "/invite/" + token,
	})
}

func (s *Server) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	rel, err := s.store.AcceptInvitation(r.Context(), user, chi.URLParam(r, "token"))
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, rel)
}

func (s *Server) listRelations(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	relations, err := s.store.ListRelations(r.Context(), user)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"relations": relations})
}

func (s *Server) createLesson(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct {
		RelationID string    `json:"relation_id"`
		Title      string    `json:"title"`
		StartsAt   time.Time `json:"starts_at"`
		EndsAt     time.Time `json:"ends_at"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	lesson, err := s.store.CreateLesson(r.Context(), user, req.RelationID, req.Title, req.StartsAt, req.EndsAt)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, lesson)
}

func (s *Server) listLessons(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	offset, err := pageOffset(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	lessons, err := s.store.ListLessonsPage(r.Context(), user, offset)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"lessons": lessons})
}

func (s *Server) getLesson(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	lesson, err := s.store.GetLessonForActor(r.Context(), user, chi.URLParam(r, "lessonID"))
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, lesson)
}

func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	items, err := s.store.ListConversations(r.Context(), user)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string][]domain.Conversation{"conversations": items})
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	offset, err := pageOffset(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	items, err := s.store.ListMessages(r.Context(), user, chi.URLParam(r, "conversationID"), offset)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string][]domain.Message{"messages": items})
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		key, err = platform.NewUUID()
		if err != nil {
			WriteError(w, err)
			return
		}
	}
	msg, err := s.store.SendMessage(r.Context(), user, chi.URLParam(r, "conversationID"), req.Body, key)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, msg)
}

func (s *Server) listAssignments(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	offset, err := pageOffset(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	items, err := s.store.ListAssignmentsPage(r.Context(), user, offset)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string][]domain.Assignment{"assignments": items})
}

func (s *Server) createAssignment(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct {
		StudentID string    `json:"student_id"`
		Title     string    `json:"title"`
		Subject   string    `json:"subject"`
		Topic     string    `json:"topic"`
		Body      string    `json:"body"`
		DueAt     time.Time `json:"due_at"`
		MaxScore  int       `json:"max_score"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	item, err := s.store.CreateAssignment(r.Context(), user, req.StudentID, req.Title, req.Subject, req.Topic, req.Body, req.DueAt, req.MaxScore)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, item)
}

func (s *Server) submitAssignment(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct {
		Answer string `json:"answer"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	item, err := s.store.SubmitAssignment(r.Context(), user, chi.URLParam(r, "assignmentID"), req.Answer)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, item)
}

func (s *Server) gradeAssignment(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct {
		Score    *int   `json:"score"`
		Feedback string `json:"feedback"`
		Revision bool   `json:"revision"`
	}
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, err)
		return
	}
	if req.Score == nil {
		WriteError(w, ErrBadRequest)
		return
	}
	item, err := s.store.GradeAssignment(r.Context(), user, chi.URLParam(r, "assignmentID"), *req.Score, req.Feedback, req.Revision)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, item)
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	offset, err := pageOffset(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	items, err := s.store.ListNotificationsPage(r.Context(), user, offset)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string][]domain.Notification{"notifications": items})
}

func (s *Server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if err := s.store.MarkNotificationRead(r.Context(), user, chi.URLParam(r, "notificationID")); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if err := s.store.MarkAllNotificationsRead(r.Context(), user); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(store.SessionCookieName); err == nil {
			user, err := s.store.UserBySessionToken(r.Context(), cookie.Value)
			if err == nil {
				p := auth.Principal{UserID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Role: user.Role}
				next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
				return
			}
		}
		p, err := s.provider.Authenticate(r)
		if err != nil {
			s.log.Info("authentication failed", "request_id", middleware.GetReqID(r.Context()))
			WriteError(w, ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
	})
}

func (s *Server) currentUser(r *http.Request) (domain.User, error) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		return domain.User{}, ErrUnauthorized
	}
	return s.store.EnsureUser(r.Context(), principal)
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     store.SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   14 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

type dashboardResponse struct {
	User          domain.User           `json:"user"`
	Relations     []domain.Relation     `json:"relations"`
	Lessons       []domain.Lesson       `json:"lessons"`
	Assignments   []domain.Assignment   `json:"assignments"`
	Conversations []domain.Conversation `json:"conversations"`
	Notifications []domain.Notification `json:"notifications"`
}

func decodeJSON(r *http.Request, target any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return ErrBadRequest
	}
	if dec.More() {
		return ErrBadRequest
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return ErrBadRequest
	} else if err != io.EOF {
		return ErrBadRequest
	}
	return nil
}

func pageOffset(r *http.Request) (int, error) {
	value := r.URL.Query().Get("offset")
	if value == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(value)
	if err != nil || offset < 0 || offset > 100000 {
		return 0, ErrBadRequest
	}
	return offset, nil
}
