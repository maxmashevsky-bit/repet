package httpapi

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) rewardRoutes(r chi.Router) {
	r.Get("/api/v1/learning-rewards/{studentID}", s.learningRewards)
	r.Get("/api/v1/learning-rewards/{studentID}/preference", s.rewardPreference)
	r.Put("/api/v1/learning-rewards/{studentID}/preference", s.setRewardPreference)
}
func (s *Server) learningRewards(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.LearningRewards(r.Context(), u, chi.URLParam(r, "studentID"), f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) rewardPreference(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetRewardPreference(r.Context(), u, chi.URLParam(r, "studentID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) setRewardPreference(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.RewardPreferenceInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.SetRewardPreference(r.Context(), u, chi.URLParam(r, "studentID"), in)
	catalogRespond(w, 200, out, err)
}
