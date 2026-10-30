package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) exerciseRoutes(r chi.Router) {
	s.learningRoutes(r)
	r.Post("/api/v1/catalog/lessons/{learningLessonID}/exercises/{exerciseID}/attempts", s.submitExerciseAnswer)
	r.Get("/api/v1/catalog/lessons/{learningLessonID}/attempts", s.exerciseAttempts)
	r.Put("/api/v1/catalog/lessons/{learningLessonID}/attempts/{attemptID}/review", s.reviewExerciseAttempt)
}

func (s *Server) submitExerciseAnswer(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.ExerciseAttemptInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.SubmitExerciseAnswer(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "exerciseID"), in)
	catalogRespond(w, http.StatusCreated, out, err)
}

func (s *Server) exerciseAttempts(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.ListExerciseAttempts(r.Context(), u, chi.URLParam(r, "learningLessonID"), f)
	catalogRespond(w, http.StatusOK, out, err)
}

func (s *Server) reviewExerciseAttempt(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.ExerciseReviewInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ReviewExerciseAttempt(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "attemptID"), in)
	catalogRespond(w, http.StatusOK, out, err)
}
