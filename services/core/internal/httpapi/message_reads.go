package httpapi

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (s *Server) markMessagesRead(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in struct {
		ThroughID string `json:"through_message_id"`
	}
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	unread, err := s.store.MarkMessagesRead(r.Context(), u, chi.URLParam(r, "conversationID"), in.ThroughID)
	catalogRespond(w, 200, map[string]int{"unread_count": unread}, err)
}
