package httpapi

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) lessonSharingRoutes(r chi.Router) {
	r.Get("/api/v1/catalog/shared-lessons", s.sharedLessonList)
	r.Get("/api/v1/catalog/shared-lessons/{shareID}", s.sharedLesson)
	r.Post("/api/v1/catalog/shared-lessons/{shareID}/copy", s.sharedLessonCopy)
	r.Get("/api/v1/catalog/lessons/{learningLessonID}/shares", s.sharedLessonList)
	r.Post("/api/v1/catalog/lessons/{learningLessonID}/shares", s.lessonShareCreate)
	r.Post("/api/v1/catalog/lessons/{learningLessonID}/shares/{shareID}/revoke", s.lessonShareRevoke)
}
func (s *Server) sharedLessonList(w http.ResponseWriter, r *http.Request) {
	actor, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := catalogFilter(r, "lessons")
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ListLessonShares(r.Context(), actor, chi.URLParam(r, "learningLessonID"), f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) sharedLesson(w http.ResponseWriter, r *http.Request) {
	actor, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetSharedLesson(r.Context(), actor, chi.URLParam(r, "shareID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) lessonShareCreate(w http.ResponseWriter, r *http.Request) {
	actor, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.LessonShareInput
	if decodeJSON(r, &in) != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	out, err := s.store.CreateLessonShare(r.Context(), actor, chi.URLParam(r, "learningLessonID"), in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) sharedLessonCopy(w http.ResponseWriter, r *http.Request) {
	actor, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.LessonCopyInput
	if decodeJSON(r, &in) != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	out, err := s.store.CopySharedLesson(r.Context(), actor, chi.URLParam(r, "shareID"), in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) lessonShareRevoke(w http.ResponseWriter, r *http.Request) {
	actor, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if !catalogEmptyAction(w, r) {
		return
	}
	if err = s.store.RevokeLessonShare(r.Context(), actor, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "shareID")); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, 200, map[string]string{"status": "revoked"})
}
