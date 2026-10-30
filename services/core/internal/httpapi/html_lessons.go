package httpapi

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/url"
	"strings"
	"tutor-platform/services/core/internal/config"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/store"
)

func (s *Server) htmlLessonRoutes(r chi.Router) {
	r.Get("/api/v1/html-lessons", s.htmlLessons)
	r.Post("/api/v1/html-lessons", s.importHTML)
	r.Get("/api/v1/html-lessons/{htmlID}", s.htmlLesson)
	r.Post("/api/v1/html-lessons/{htmlID}/versions", s.importHTML)
	r.Get("/api/v1/html-lessons/{htmlID}/versions", s.htmlVersions)
	r.Post("/api/v1/html-lessons/{htmlID}/publish", s.publishHTML)
	r.Post("/api/v1/html-lessons/{htmlID}/archive", s.archiveHTML)
	r.Post("/api/v1/html-lessons/{htmlID}/preview", s.previewHTML)
	r.Get("/api/v1/html-lessons/{htmlID}/source", s.htmlSource)
	r.Post("/api/v1/html-lessons/{htmlID}/assign", s.assignHTML)
	r.Post("/api/v1/html-lessons/{htmlID}/convert", s.convertHTML)
}
func (s *Server) htmlLessons(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := catalogFilter(r, "lessons")
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ListHTMLLessons(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) htmlLesson(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetHTMLLesson(r.Context(), u, chi.URLParam(r, "htmlID"), r.URL.Query().Get("version_id"), r.URL.Query().Get("assignment_id"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) importHTML(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.HTMLImportInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ImportHTMLLesson(r.Context(), u, chi.URLParam(r, "htmlID"), in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) htmlVersions(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.HTMLVersions(r.Context(), u, chi.URLParam(r, "htmlID"))
	catalogRespond(w, 200, map[string]any{"items": out}, err)
}

type htmlVersionAction struct {
	VersionID string `json:"version_id"`
}

func (s *Server) publishHTML(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in htmlVersionAction
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.PublishHTMLLesson(r.Context(), u, chi.URLParam(r, "htmlID"), in.VersionID)
	catalogRespond(w, 200, out, err)
}
func (s *Server) archiveHTML(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	err = s.store.ArchiveHTMLLesson(r.Context(), u, chi.URLParam(r, "htmlID"))
	catalogRespond(w, 200, map[string]string{"status": "archived"}, err)
}
func (s *Server) previewHTML(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if s.cfg.HTMLPreviewPublicURL == "" {
		WriteError(w, ErrInvalidState)
		return
	}
	var in struct {
		VersionID    string `json:"version_id"`
		AssignmentID string `json:"assignment_id"`
	}
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	token, err := s.store.CreateHTMLPreview(r.Context(), u, chi.URLParam(r, "htmlID"), in.VersionID, in.AssignmentID)
	catalogRespond(w, 200, map[string]any{"url": strings.TrimRight(s.cfg.HTMLPreviewPublicURL, "/") + "/view/" + token, "expires_in": 300}, err)
}
func (s *Server) htmlSource(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	source, err := s.store.HTMLOriginal(r.Context(), u, chi.URLParam(r, "htmlID"), r.URL.Query().Get("version_id"))
	catalogRespond(w, 200, map[string]string{"source": source}, err)
}
func (s *Server) assignHTML(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.HTMLAssignInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.AssignHTMLLesson(r.Context(), u, chi.URLParam(r, "htmlID"), in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) convertHTML(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in htmlVersionAction
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ConvertHTMLLesson(r.Context(), u, chi.URLParam(r, "htmlID"), in.VersionID)
	catalogRespond(w, 201, out, err)
}

// The content listener has no session middleware, application routes, or CORS grants.
func NewHTMLPreviewServer(cfg config.Config, st *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), clipboard-read=(), clipboard-write=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data:; font-src data:; media-src data:; connect-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none'; worker-src 'none'; sandbox allow-scripts; frame-ancestors "+strings.Join(cfg.CORSAllowedOrigins, " "))
		u, err := url.Parse(cfg.HTMLPreviewPublicURL)
		if err != nil || u.Host == "" || !strings.EqualFold(r.Host, u.Host) || r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/view/") {
			http.NotFound(w, r)
			return
		}
		source, err := st.ReadHTMLPreview(r.Context(), strings.TrimPrefix(r.URL.Path, "/view/"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(source))
	})
}
