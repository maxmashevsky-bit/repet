package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/store"
)

func (s *Server) familyRoutes(r chi.Router) {
	r.Get("/api/v1/family/links", s.guardianLinks)
	r.Post("/api/v1/family/invitations", s.createGuardianInvitation)
	r.Post("/api/v1/family/invitations/preview", s.guardianInvitation)
	r.Post("/api/v1/family/invitations/accept", s.guardianInvitation)
	r.Post("/api/v1/family/links/{linkID}/revoke", s.revokeGuardianLink)
	r.Get("/api/v1/family/children/{studentID}", s.familyReport)
	r.Post("/api/v1/family/children/{studentID}/read", s.familyRead)
	r.Post("/api/v1/family/notes", s.guardianNote)
	r.Get("/api/v1/family/students/{studentID}/notes", s.guardianNotes)
	r.Post("/api/v1/family/notes/{noteID}/withdraw", s.withdrawGuardianNote)
}
func (s *Server) guardianNotes(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	f, e := catalogFilter(r, "versions")
	if e != nil {
		WriteError(w, e)
		return
	}
	out, e := s.store.GuardianNotes(r.Context(), u, chi.URLParam(r, "studentID"), f)
	catalogRespond(w, 200, out, e)
}
func (s *Server) guardianLinks(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	f, e := catalogFilter(r, "versions")
	if e != nil {
		WriteError(w, e)
		return
	}
	out, e := s.store.GuardianLinks(r.Context(), u, f)
	catalogRespond(w, 200, out, e)
}
func (s *Server) createGuardianInvitation(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	var in domain.GuardianInviteInput
	if e = decodeJSON(r, &in); e != nil {
		WriteError(w, e)
		return
	}
	link, token, e := s.store.CreateGuardianInvitation(r.Context(), u, in)
	catalogRespond(w, 201, map[string]any{"link": link, "accept_url": "/family-invite#token=" + token}, e)
}
func (s *Server) guardianInvitation(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if e = decodeJSON(r, &in); e != nil {
		WriteError(w, e)
		return
	}
	out, e := s.store.GuardianInvitation(r.Context(), u, in.Token, r.URL.Path == "/api/v1/family/invitations/accept")
	catalogRespond(w, 200, out, e)
}
func (s *Server) revokeGuardianLink(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	var in struct{}
	if e = decodeJSON(r, &in); e != nil {
		WriteError(w, e)
		return
	}
	e = s.store.RevokeGuardianLink(r.Context(), u, chi.URLParam(r, "linkID"))
	catalogRespond(w, 200, map[string]string{"status": "revoked"}, e)
}
func familyFilter(r *http.Request) (domain.FamilyFilter, error) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	f := domain.FamilyFilter{AnalyticsFilter: domain.AnalyticsFilter{From: today.AddDate(0, 0, -29), To: today.AddDate(0, 0, 30)}}
	for k, values := range r.URL.Query() {
		if len(values) != 1 {
			return f, apperr.ErrBadRequest
		}
		v := values[0]
		if k == "from" || k == "to" {
			day, e := time.Parse("2006-01-02", v)
			if e != nil {
				return f, apperr.ErrBadRequest
			}
			if k == "from" {
				f.From = day
			} else {
				f.To = day
			}
			continue
		}
		n, e := strconv.Atoi(v)
		if e != nil {
			return f, apperr.ErrBadRequest
		}
		switch k {
		case "topic_offset":
			f.TopicOffset = n
		case "exam_offset":
			f.ExamOffset = n
		case "lesson_offset":
			f.LessonOffset = n
		case "assignment_offset":
			f.AssignmentOffset = n
		case "note_offset":
			f.NoteOffset = n
		case "event_offset":
			f.EventOffset = n
		case "result_offset":
			f.ResultOffset = n
		default:
			return f, apperr.ErrBadRequest
		}
	}
	return f, store.ValidateFamilyFilter(f)
}
func (s *Server) familyReport(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	f, e := familyFilter(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	out, e := s.store.FamilyReport(r.Context(), u, chi.URLParam(r, "studentID"), f)
	catalogRespond(w, 200, out, e)
}
func (s *Server) familyRead(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	var in struct{}
	if e = decodeJSON(r, &in); e != nil {
		WriteError(w, e)
		return
	}
	e = s.store.MarkFamilyRead(r.Context(), u, chi.URLParam(r, "studentID"))
	catalogRespond(w, 200, map[string]string{"status": "read"}, e)
}
func (s *Server) guardianNote(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	var in domain.GuardianNoteInput
	if e = decodeJSON(r, &in); e != nil {
		WriteError(w, e)
		return
	}
	out, e := s.store.CreateGuardianNote(r.Context(), u, in)
	catalogRespond(w, 201, out, e)
}
func (s *Server) withdrawGuardianNote(w http.ResponseWriter, r *http.Request) {
	u, e := s.currentUser(r)
	if e != nil {
		WriteError(w, e)
		return
	}
	var in struct{}
	if e = decodeJSON(r, &in); e != nil {
		WriteError(w, e)
		return
	}
	e = s.store.WithdrawGuardianNote(r.Context(), u, chi.URLParam(r, "noteID"))
	catalogRespond(w, 200, map[string]string{"status": "withdrawn"}, e)
}
