package httpapi

import (
	"net/http"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) verificationAllowed(w http.ResponseWriter, r *http.Request, user domain.User) bool {
	if !s.cfg.RequireEmailVerification || user.EmailVerified {
		return true
	}
	if r.Method == http.MethodGet && r.URL.Path == "/api/v1/me" {
		return true
	}
	if r.Method == http.MethodPost {
		switch r.URL.Path {
		case "/api/v1/auth/logout", "/api/v1/auth/verification-email", "/api/v1/auth/verify-email":
			return true
		}
	}
	WriteJSON(w, http.StatusForbidden, map[string]any{"error": map[string]string{"code": "email_unverified", "message": "Подтвердите электронную почту, чтобы открыть кабинет"}})
	return false
}

func (s *Server) deliverVerification(r *http.Request, user domain.User) error {
	token, email, err := s.store.IssueEmailVerification(r.Context(), user)
	if err != nil {
		return err
	}
	if token == "" {
		return nil
	}
	if err = sendVerificationEmail(r.Context(), s.cfg, email, token); err != nil {
		s.log.Error("verification email delivery failed", "user_id", user.ID)
		return err
	}
	return nil
}

func (s *Server) verificationEmail(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct{}
	if decodeJSON(r, &req) != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	if err = s.deliverVerification(r, user); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) verifyEmail(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if decodeJSON(r, &req) != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	if err = s.store.VerifyEmail(r.Context(), user, req.Token); err != nil {
		WriteError(w, err)
		return
	}
	user.EmailVerified = true
	WriteJSON(w, http.StatusOK, map[string]domain.User{"user": user})
}
