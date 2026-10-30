package httpapi

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) reviewRoutes(r chi.Router) {
	r.Get("/api/v1/learning-reviews/{studentID}", s.learningReviewPlan)
	r.Post("/api/v1/learning-review-sessions", s.startLearningReview)
	r.Get("/api/v1/learning-review-sessions/{reviewID}", s.getLearningReview)
	r.Get("/api/v1/learning-review-checks", s.learningReviewChecks)
}
func (s *Server) learningReviewPlan(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f := domain.ReviewFilter{}
	for key, values := range r.URL.Query() {
		if len(values) != 1 {
			WriteError(w, apperr.ErrBadRequest)
			return
		}
		if key == "mode" {
			f.Mode = values[0]
			continue
		}
		n, e := strconv.Atoi(values[0])
		if e != nil {
			WriteError(w, apperr.ErrBadRequest)
			return
		}
		switch key {
		case "limit":
			f.Limit = n
		case "offset":
			f.Offset = n
		case "active_offset":
			f.ActiveOffset = n
		default:
			WriteError(w, apperr.ErrBadRequest)
			return
		}
	}
	out, err := s.store.LearningReviewPlan(r.Context(), u, chi.URLParam(r, "studentID"), f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) startLearningReview(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.ReviewStartInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.StartLearningReview(r.Context(), u, in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) getLearningReview(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetLearningReview(r.Context(), u, chi.URLParam(r, "reviewID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) learningReviewChecks(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.LearningReviewChecks(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
