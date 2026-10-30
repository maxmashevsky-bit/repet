package httpapi

import (
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) uploadChatFile(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in struct {
		Filename  string `json:"filename"`
		Data      []byte `json:"data"`
		RequestID string `json:"request_id"`
	}
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	file, err := s.store.UploadChatFile(r.Context(), u, chi.URLParam(r, "conversationID"), in.Filename, in.Data, in.RequestID)
	catalogRespond(w, http.StatusCreated, file, err)
}
func (s *Server) downloadChatFile(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, reader, err := s.store.OpenChatFile(r.Context(), u, chi.URLParam(r, "conversationID"), chi.URLParam(r, "fileID"))
	if err != nil {
		WriteError(w, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Filename}))
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.WriteHeader(http.StatusOK)
	if _, err = io.Copy(w, reader); err != nil {
		s.log.Warn("chat download interrupted", "file_id", f.ID)
	}
}
func (s *Server) deleteChatFile(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	err = s.store.DeleteChatFile(r.Context(), u, chi.URLParam(r, "conversationID"), chi.URLParam(r, "fileID"))
	catalogRespond(w, http.StatusOK, map[string]string{"status": "deleted"}, err)
}
func (s *Server) listChatReferences(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	offset, err := pageOffset(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	items, err := s.store.ListChatReferences(r.Context(), u, chi.URLParam(r, "conversationID"), r.URL.Query().Get("kind"), r.URL.Query().Get("q"), offset)
	catalogRespond(w, http.StatusOK, map[string]any{"items": items}, err)
}
func (s *Server) chatAssignment(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	item, err := s.store.ChatAssignment(r.Context(), u, chi.URLParam(r, "conversationID"), chi.URLParam(r, "assignmentID"))
	catalogRespond(w, http.StatusOK, item, err)
}
