package httpapi

import (
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strconv"
	"tutor-platform/services/core/internal/domain"
)

func (s *Server) pdfRoutes(r chi.Router) {
	r.Get("/api/v1/pdf-imports", s.pdfImport)
	r.Post("/api/v1/pdf-imports", s.pdfImport)
	r.Get("/api/v1/pdf-imports/{pdfID}", s.pdfImport)
	r.Get("/api/v1/pdf-imports/{pdfID}/source", s.pdfSource)
	r.Put("/api/v1/pdf-imports/{pdfID}/pages/{page}", s.pdfPage)
	r.Post("/api/v1/pdf-imports/{pdfID}/figures", s.pdfFigure)
	r.Post("/api/v1/pdf-imports/{pdfID}/lessons", s.pdfLesson)
	r.Get("/api/v1/pdf-assets/{assetID}", s.pdfAsset)
	r.Get("/api/v1/catalog/lessons/{lessonID}/pdf-source", s.pdfLessonReview)
	r.Post("/api/v1/catalog/lessons/{lessonID}/pdf-review", s.pdfLessonReview)
}

func (s *Server) pdfFigure(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in struct {
		Page   int `json:"page_number"`
		X      int `json:"x"`
		Y      int `json:"y"`
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.CropPDFFigure(r.Context(), u, chi.URLParam(r, "pdfID"), in.Page, in.X, in.Y, in.Width, in.Height)
	catalogRespond(w, 201, out, err)
}
func (s *Server) pdfImport(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if r.Method == http.MethodPost {
		var in domain.PDFImportInput
		if err = decodeJSON(r, &in); err != nil {
			WriteError(w, err)
			return
		}
		out, err := s.store.ImportPDF(r.Context(), u, in)
		catalogRespond(w, 202, out, err)
		return
	}
	if id := chi.URLParam(r, "pdfID"); id != "" {
		out, err := s.store.GetPDFImport(r.Context(), u, id)
		catalogRespond(w, 200, out, err)
		return
	}
	f, err := catalogFilter(r, "lessons")
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ListPDFImports(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) pdfSource(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	body, err := s.store.PDFOriginal(r.Context(), u, chi.URLParam(r, "pdfID"))
	if err != nil {
		WriteError(w, err)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="source.pdf"`)
	_, _ = io.Copy(w, body)
}
func (s *Server) pdfAsset(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	body, err := s.store.PDFAsset(r.Context(), u, chi.URLParam(r, "assetID"))
	if err != nil {
		WriteError(w, err)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "inline")
	_, _ = io.Copy(w, body)
}
func (s *Server) pdfPage(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	number, err := strconv.Atoi(chi.URLParam(r, "page"))
	if err != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ReviewPDFPage(r.Context(), u, chi.URLParam(r, "pdfID"), number, in.Text)
	catalogRespond(w, 200, out, err)
}
func (s *Server) pdfLesson(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in struct {
		CourseID string                     `json:"course_id"`
		Lesson   domain.LearningLessonInput `json:"lesson"`
	}
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.CreatePDFLesson(r.Context(), u, chi.URLParam(r, "pdfID"), in.CourseID, in.Lesson)
	catalogRespond(w, 201, out, err)
}
func (s *Server) pdfLessonReview(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	id := chi.URLParam(r, "lessonID")
	if r.Method == http.MethodPost {
		var in struct {
			Revision  int64 `json:"revision"`
			Confirmed bool  `json:"confirmed"`
		}
		if err = decodeJSON(r, &in); err != nil {
			WriteError(w, err)
			return
		}
		if err = s.store.ApprovePDFLesson(r.Context(), u, id, in.Revision, in.Confirmed); err != nil {
			WriteError(w, err)
			return
		}
	}
	out, err := s.store.PDFLessonSource(r.Context(), u, id)
	catalogRespond(w, 200, out, err)
}
