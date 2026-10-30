package httpapi

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) materialAssignmentRoutes(r chi.Router) {
	r.Get("/api/v1/assignments/{assignmentID}", s.getAssignment)
	r.Post("/api/v1/assignments/material", s.createMaterialAssignment)
	r.Post("/api/v1/assignments/{assignmentID}/repeat", s.repeatMaterialAssignment)
	r.Post("/api/v1/assignments/{assignmentID}/learning-attempt", s.startMaterialAssignment)
	r.Get("/api/v1/assignments/{assignmentID}/learning-results", s.materialResults)
	r.Get("/api/v1/exercise-sets", s.listExerciseSets)
	r.Post("/api/v1/exercise-sets", s.saveExerciseSet)
	r.Put("/api/v1/exercise-sets/{setID}", s.saveExerciseSet)
	r.Delete("/api/v1/exercise-sets/{setID}", s.deleteExerciseSet)
}
func (s *Server) getAssignment(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetAssignment(r.Context(), u, chi.URLParam(r, "assignmentID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) createMaterialAssignment(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.MaterialAssignmentInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.CreateMaterialAssignment(r.Context(), u, in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) repeatMaterialAssignment(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.MaterialRepeatInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.RepeatMaterialAssignment(r.Context(), u, chi.URLParam(r, "assignmentID"), in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) startMaterialAssignment(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.StartMaterialAssignment(r.Context(), u, chi.URLParam(r, "assignmentID"), in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) materialResults(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.MaterialResults(r.Context(), u, chi.URLParam(r, "assignmentID"), f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) listExerciseSets(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.ListExerciseSets(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) saveExerciseSet(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.ExerciseSetInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.SaveExerciseSet(r.Context(), u, chi.URLParam(r, "setID"), in)
	status := 200
	if r.Method == http.MethodPost {
		status = 201
	}
	catalogRespond(w, status, out, err)
}
func (s *Server) deleteExerciseSet(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if err = s.store.DeleteExerciseSet(r.Context(), u, chi.URLParam(r, "setID")); err != nil {
		WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
