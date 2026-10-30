package httpapi

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) learningRoutes(r chi.Router) {
	base := "/api/v1/catalog/lessons/{learningLessonID}/learning-attempts"
	r.Post(base, s.startLearning)
	r.Get(base, s.listLearning)
	r.Get(base+"/{learningID}", s.getLearning)
	r.Get(base+"/{learningID}/answers", s.learningAnswers)
	r.Post(base+"/{learningID}/complete", s.completeLearning)
	r.Post(base+"/{learningID}/exercises/{exerciseID}/hints", s.revealLearningHint)
	r.Post(base+"/{learningID}/navigate", s.navigateLearning)
	r.Get(base+"/{learningID}/exercises/{exerciseID}/solution", s.learningSolution)
}
func (s *Server) navigateLearning(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.LearningNavigateInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.NavigateLearning(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "learningID"), in)
	catalogRespond(w, http.StatusOK, out, err)
}
func (s *Server) learningSolution(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.LearningSolution(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "learningID"), chi.URLParam(r, "exerciseID"))
	catalogRespond(w, http.StatusOK, out, err)
}
func (s *Server) learningAnswers(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.ListLearningAnswers(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "learningID"), f)
	catalogRespond(w, http.StatusOK, out, err)
}
func (s *Server) startLearning(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.LearningStartInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.StartLearning(r.Context(), u, chi.URLParam(r, "learningLessonID"), in)
	catalogRespond(w, http.StatusCreated, out, err)
}
func (s *Server) getLearning(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetLearningState(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "learningID"))
	catalogRespond(w, http.StatusOK, out, err)
}
func (s *Server) listLearning(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.ListLearningAttempts(r.Context(), u, chi.URLParam(r, "learningLessonID"), f)
	catalogRespond(w, http.StatusOK, out, err)
}
func (s *Server) completeLearning(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.LearningActionInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.CompleteLearning(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "learningID"), in)
	catalogRespond(w, http.StatusOK, out, err)
}
func (s *Server) revealLearningHint(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.LearningActionInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.RevealLearningHint(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "learningID"), chi.URLParam(r, "exerciseID"), in)
	catalogRespond(w, http.StatusOK, out, err)
}
