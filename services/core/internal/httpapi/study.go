package httpapi

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) studyRoutes(r chi.Router) {
	r.Get("/api/v1/exam-specifications", s.studySpecifications)
	r.Post("/api/v1/exam-specifications", s.studyCreateSpecification)
	r.Get("/api/v1/exam-specifications/{specID}", s.studySpecification)
	r.Post("/api/v1/exam-specifications/{specID}/publish", s.studySpecificationStatus("published"))
	r.Post("/api/v1/exam-specifications/{specID}/archive", s.studySpecificationStatus("archived"))
	r.Get("/api/v1/study-sessions", s.studyHistory)
	r.Post("/api/v1/study-sessions", s.studyStart)
	r.Get("/api/v1/study-sessions/{sessionID}", s.studySession)
	r.Get("/api/v1/study-reviews", s.studyReviews)
}
func (s *Server) studySpecifications(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := catalogFilter(r, "versions")
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ListExamSpecifications(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) studySpecification(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetExamSpecification(r.Context(), u, chi.URLParam(r, "specID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) studyCreateSpecification(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.ExamSpecificationInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.CreateExamSpecification(r.Context(), u, in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) studySpecificationStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.currentUser(r)
		if err != nil {
			WriteError(w, err)
			return
		}
		var in struct{}
		if err = decodeJSON(r, &in); err != nil {
			WriteError(w, err)
			return
		}
		out, err := s.store.SetExamSpecificationStatus(r.Context(), u, chi.URLParam(r, "specID"), status)
		catalogRespond(w, 200, out, err)
	}
}
func (s *Server) studyHistory(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := catalogFilter(r, "versions")
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ListStudySessions(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) studyStart(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.StudyStartInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.StartStudySession(r.Context(), u, in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) studySession(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetStudySession(r.Context(), u, chi.URLParam(r, "sessionID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) studyReviews(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := catalogFilter(r, "versions")
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ListStudyReviews(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
