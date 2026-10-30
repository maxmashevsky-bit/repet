package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/store"
)

func (s *Server) catalogRoutes(r chi.Router) {
	s.lessonSharingRoutes(r)
	s.exerciseRoutes(r)
	r.Get("/api/v1/catalog/courses", s.catalogCourses)
	r.Post("/api/v1/catalog/courses", s.catalogCreateCourse)
	r.Get("/api/v1/catalog/courses/{courseID}", s.catalogGetCourse)
	r.Put("/api/v1/catalog/courses/{courseID}", s.catalogUpdateCourse)
	r.Post("/api/v1/catalog/courses/{courseID}/publish", s.catalogCourseStatus("published"))
	r.Post("/api/v1/catalog/courses/{courseID}/archive", s.catalogCourseStatus("archived"))
	r.Get("/api/v1/catalog/courses/{courseID}/topics", s.catalogTopics)
	r.Post("/api/v1/catalog/courses/{courseID}/topics", s.catalogCreateTopic)
	r.Get("/api/v1/catalog/topics/{topicID}", s.catalogGetTopic)
	r.Put("/api/v1/catalog/topics/{topicID}", s.catalogUpdateTopic)
	r.Post("/api/v1/catalog/topics/{topicID}/publish", s.catalogTopicStatus("published"))
	r.Post("/api/v1/catalog/topics/{topicID}/archive", s.catalogTopicStatus("archived"))
	r.Post("/api/v1/catalog/courses/{courseID}/lessons", s.catalogCreateLesson)
	r.Get("/api/v1/catalog/lessons", s.catalogLessons)
	r.Get("/api/v1/catalog/lessons/{learningLessonID}", s.catalogGetLesson)
	r.Put("/api/v1/catalog/lessons/{learningLessonID}", s.catalogUpdateLesson)
	r.Post("/api/v1/catalog/lessons/{learningLessonID}/publish", s.catalogPublishLesson)
	r.Post("/api/v1/catalog/lessons/{learningLessonID}/archive", s.catalogArchiveLesson)
	r.Get("/api/v1/catalog/lessons/{learningLessonID}/versions", s.catalogVersions)
	r.Get("/api/v1/catalog/lessons/{learningLessonID}/versions/{number}", s.catalogGetVersion)
	r.Post("/api/v1/catalog/lessons/{learningLessonID}/exercises", s.catalogCreateExercise)
	r.Get("/api/v1/catalog/lessons/{learningLessonID}/exercises", s.catalogExercises)
	r.Get("/api/v1/catalog/lessons/{learningLessonID}/exercises/{exerciseID}", s.catalogGetExercise)
	r.Put("/api/v1/catalog/lessons/{learningLessonID}/exercises/{exerciseID}", s.catalogUpdateExercise)
	r.Get("/api/v1/catalog/lessons/{learningLessonID}/answer-keys", s.catalogAnswerKeys)
}

func catalogRespond(w http.ResponseWriter, status int, result any, err error) {
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, status, result)
}

func catalogFilter(r *http.Request, scope string) (domain.CatalogFilter, error) {
	q := r.URL.Query()
	allowed := map[string]bool{"limit": true, "offset": true}
	switch scope {
	case "courses":
		for _, key := range []string{"q", "track", "grade", "difficulty", "author_id", "status"} {
			allowed[key] = true
		}
	case "topics":
		allowed["q"] = true
		allowed["status"] = true
	case "lessons":
		for _, key := range []string{"q", "track", "grade", "difficulty", "author_id", "course_id", "topic_id", "status"} {
			allowed[key] = true
		}
	case "exercises":
		allowed["q"] = true
		allowed["difficulty"] = true
	}
	for key, values := range q {
		if !allowed[key] || len(values) != 1 || key != "q" && values[0] == "" {
			return domain.CatalogFilter{}, ErrBadRequest
		}
	}
	f := domain.CatalogFilter{Query: q.Get("q"), Track: q.Get("track"), AuthorID: q.Get("author_id"), CourseID: q.Get("course_id"), TopicID: q.Get("topic_id"), Status: q.Get("status"), Limit: 20}
	for key, target := range map[string]*int{"limit": &f.Limit, "offset": &f.Offset, "grade": &f.Grade, "difficulty": &f.Difficulty} {
		if q.Has(key) {
			n, err := strconv.Atoi(q.Get(key))
			if err != nil {
				return f, ErrBadRequest
			}
			if (key == "grade" || key == "difficulty") && n == 0 {
				return f, ErrBadRequest
			}
			*target = n
		}
	}
	if f.Limit == 0 {
		return f, ErrBadRequest
	}
	return f, store.ValidateCatalogFilter(&f)
}

func (s *Server) catalogCourses(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := catalogFilter(r, "courses")
	if err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.ListCourses(r.Context(), u, f)
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogGetCourse(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.GetCourse(r.Context(), u, chi.URLParam(r, "courseID"))
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogCreateCourse(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.CourseInput
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.CreateCourse(r.Context(), u, in)
	catalogRespond(w, http.StatusCreated, result, err)
}
func (s *Server) catalogUpdateCourse(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.CourseInput
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.UpdateCourse(r.Context(), u, chi.URLParam(r, "courseID"), in)
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogCourseStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.currentUser(r)
		if err != nil {
			WriteError(w, err)
			return
		}
		if !catalogEmptyAction(w, r) {
			return
		}
		result, err := s.store.SetCourseStatus(r.Context(), u, chi.URLParam(r, "courseID"), status)
		catalogRespond(w, http.StatusOK, result, err)
	}
}
func (s *Server) catalogTopics(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := catalogFilter(r, "topics")
	if err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.ListTopics(r.Context(), u, chi.URLParam(r, "courseID"), f)
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogGetTopic(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.GetTopic(r.Context(), u, chi.URLParam(r, "topicID"))
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogCreateTopic(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.TopicInput
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.CreateTopic(r.Context(), u, chi.URLParam(r, "courseID"), in)
	catalogRespond(w, http.StatusCreated, result, err)
}
func (s *Server) catalogUpdateTopic(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.TopicInput
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.UpdateTopic(r.Context(), u, chi.URLParam(r, "topicID"), in)
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogTopicStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.currentUser(r)
		if err != nil {
			WriteError(w, err)
			return
		}
		if !catalogEmptyAction(w, r) {
			return
		}
		result, err := s.store.SetTopicStatus(r.Context(), u, chi.URLParam(r, "topicID"), status)
		catalogRespond(w, http.StatusOK, result, err)
	}
}
func (s *Server) catalogLessons(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.store.ListLearningLessons(r.Context(), u, f)
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogGetLesson(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.GetLearningLesson(r.Context(), u, chi.URLParam(r, "learningLessonID"))
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogCreateLesson(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.LearningLessonInput
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.CreateLearningLesson(r.Context(), u, chi.URLParam(r, "courseID"), in)
	catalogRespond(w, http.StatusCreated, result, err)
}
func (s *Server) catalogUpdateLesson(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.LearningLessonInput
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.UpdateLearningLesson(r.Context(), u, chi.URLParam(r, "learningLessonID"), in)
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogPublishLesson(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if !catalogEmptyAction(w, r) {
		return
	}
	result, err := s.store.PublishLearningLesson(r.Context(), u, chi.URLParam(r, "learningLessonID"))
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogArchiveLesson(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	if !catalogEmptyAction(w, r) {
		return
	}
	result, err := s.store.ArchiveLearningLesson(r.Context(), u, chi.URLParam(r, "learningLessonID"))
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogVersions(w http.ResponseWriter, r *http.Request) {
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
	result, err := s.store.ListLearningVersions(r.Context(), u, chi.URLParam(r, "learningLessonID"), f)
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogGetVersion(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	number, err := catalogVersionNumber(chi.URLParam(r, "number"))
	if err != nil {
		WriteError(w, ErrBadRequest)
		return
	}
	result, err := s.store.GetLearningVersion(r.Context(), u, chi.URLParam(r, "learningLessonID"), number)
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogCreateExercise(w http.ResponseWriter, r *http.Request) {
	s.catalogSaveExercise(w, r, true)
}

func (s *Server) catalogExercises(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := catalogFilter(r, "exercises")
	if err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.ListLearningExercises(r.Context(), u, chi.URLParam(r, "learningLessonID"), f)
	catalogRespond(w, http.StatusOK, result, err)
}

func (s *Server) catalogGetExercise(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	result, err := s.store.GetLearningExercise(r.Context(), u, chi.URLParam(r, "learningLessonID"), chi.URLParam(r, "exerciseID"))
	catalogRespond(w, http.StatusOK, result, err)
}
func (s *Server) catalogUpdateExercise(w http.ResponseWriter, r *http.Request) {
	s.catalogSaveExercise(w, r, false)
}
func (s *Server) catalogSaveExercise(w http.ResponseWriter, r *http.Request, create bool) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.ExerciseInput
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	id := chi.URLParam(r, "exerciseID")
	status := http.StatusOK
	if create {
		status = http.StatusCreated
		id = ""
	}
	result, err := s.store.SaveExercise(r.Context(), u, chi.URLParam(r, "learningLessonID"), id, in)
	catalogRespond(w, status, result, err)
}
func (s *Server) catalogAnswerKeys(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	number := 0
	for key, values := range r.URL.Query() {
		if key != "version" || len(values) != 1 {
			WriteError(w, ErrBadRequest)
			return
		}
	}
	if r.URL.Query().Has("version") {
		number, err = catalogVersionNumber(r.URL.Query().Get("version"))
		if err != nil {
			WriteError(w, ErrBadRequest)
			return
		}
	}
	result, err := s.store.GetExerciseAnswerKeys(r.Context(), u, chi.URLParam(r, "learningLessonID"), number)
	catalogRespond(w, http.StatusOK, map[string]any{"items": result}, err)
}
func catalogEmptyAction(w http.ResponseWriter, r *http.Request) bool {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 {
		return true
	}
	var in struct{}
	if err := decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return false
	}
	return true
}

func catalogVersionNumber(raw string) (int, error) {
	number, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || number < 0 {
		return 0, ErrBadRequest
	}
	return int(number), nil
}
