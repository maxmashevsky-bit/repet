package httpapi

import (
	"net/http"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) tutorPlan(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		p, err := s.store.GetTutorPlan(r.Context(), u)
		catalogRespond(w, http.StatusOK, p, err)
		return
	}
	var in domain.TutorPlan
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	p, err := s.store.SaveTutorPlan(r.Context(), u, in)
	catalogRespond(w, http.StatusOK, p, err)
}

func (s *Server) publishPlanCalendar(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.PlanCalendarInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.PublishPlanCalendar(r.Context(), u, in)
	catalogRespond(w, http.StatusOK, result, err)
}
