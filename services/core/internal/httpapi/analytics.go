package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/store"
)

func (s *Server) analyticsRoutes(r chi.Router) {
	r.Get("/api/v1/learning-analytics/students", s.analyticsStudents)
	r.Get("/api/v1/learning-analytics/{studentID}", s.analyticsReport)
	r.Post("/api/v1/learning-analytics/{studentID}/repeat", s.analyticsRepeat)
}
func (s *Server) analyticsStudents(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.AnalyticsStudents(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func analyticsFilter(r *http.Request) (domain.AnalyticsFilter, error) {
	now := time.Now().UTC()
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	f := domain.AnalyticsFilter{From: to.AddDate(0, 0, -29), To: to}
	for key, values := range r.URL.Query() {
		if len(values) != 1 {
			return f, apperr.ErrBadRequest
		}
		value := values[0]
		switch key {
		case "from", "to":
			day, err := time.Parse("2006-01-02", value)
			if err != nil {
				return f, apperr.ErrBadRequest
			}
			if key == "from" {
				f.From = day
			} else {
				f.To = day
			}
		case "topic_offset", "exam_offset":
			n, err := strconv.Atoi(value)
			if err != nil {
				return f, apperr.ErrBadRequest
			}
			if key == "topic_offset" {
				f.TopicOffset = n
			} else {
				f.ExamOffset = n
			}
		default:
			return f, apperr.ErrBadRequest
		}
	}
	return f, store.ValidateAnalyticsFilter(f)
}
func (s *Server) analyticsReport(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := analyticsFilter(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.LearningAnalytics(r.Context(), u, chi.URLParam(r, "studentID"), f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) analyticsRepeat(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.AnalyticsRepeatInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.RepeatAnalyticsTopic(r.Context(), u, chi.URLParam(r, "studentID"), in)
	catalogRespond(w, 201, out, err)
}
