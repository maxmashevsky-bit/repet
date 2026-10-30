package httpapi

import (
	"net/http"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) teachingPreferences(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		v, e := s.store.GetTeachingPreferences(r.Context(), u)
		catalogRespond(w, http.StatusOK, v, e)
		return
	}
	var in domain.TeachingPreferences
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	v, err := s.store.SaveTeachingPreferences(r.Context(), u, in)
	catalogRespond(w, http.StatusOK, v, err)
}
