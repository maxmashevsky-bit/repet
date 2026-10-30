package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/store"
)

func (s *Server) bankRoutes(r chi.Router) {
	r.Get("/api/v1/exercise-bank", s.bankList)
	r.Post("/api/v1/exercise-bank", s.bankCreate)
	r.Get("/api/v1/exercise-bank/facets", s.bankFacets)
	r.Get("/api/v1/exercise-bank/{bankID}", s.bankGet)
	r.Get("/api/v1/exercise-bank/{bankID}/answer-key", s.bankKey)
	r.Get("/api/v1/bank-collections", s.bankCollections)
	r.Post("/api/v1/bank-collections", s.bankCreateCollection)
	r.Get("/api/v1/bank-collections/{collectionID}", s.bankCollection)
	r.Post("/api/v1/bank-collections/{collectionID}/assign", s.bankAssign)
}
func bankFilter(r *http.Request) (domain.BankFilter, error) {
	f := domain.BankFilter{Limit: 20}
	q := r.URL.Query()
	texts := map[string]*string{"q": &f.Query, "direction": &f.Direction, "topic": &f.Topic, "subtopic": &f.Subtopic, "kind": &f.Kind}
	numbers := map[string]*int{"grade": &f.Grade, "exam_number": &f.ExamNumber, "difficulty": &f.Difficulty, "limit": &f.Limit, "offset": &f.Offset}
	for key, values := range q {
		if len(values) != 1 || values[0] == "" {
			return f, ErrBadRequest
		}
		if target, ok := texts[key]; ok {
			*target = values[0]
			continue
		}
		target, ok := numbers[key]
		if !ok {
			return f, ErrBadRequest
		}
		n, err := strconv.Atoi(values[0])
		if err != nil || n == 0 && key != "offset" {
			return f, ErrBadRequest
		}
		*target = n
	}
	return f, store.ValidateBankFilter(&f)
}
func (s *Server) bankList(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := bankFilter(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.ListBankExercises(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) bankFacets(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	f, err := bankFilter(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.BankFacets(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) bankGet(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetBankExercise(r.Context(), u, chi.URLParam(r, "bankID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) bankKey(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.BankAnswerKey(r.Context(), u, chi.URLParam(r, "bankID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) bankCreate(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.BankExerciseInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.CreateBankExercise(r.Context(), u, in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) bankCollections(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.store.ListBankCollections(r.Context(), u, f)
	catalogRespond(w, 200, out, err)
}
func (s *Server) bankCollection(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.GetBankCollection(r.Context(), u, chi.URLParam(r, "collectionID"))
	catalogRespond(w, 200, out, err)
}
func (s *Server) bankCreateCollection(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.BankCollectionInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.CreateBankCollection(r.Context(), u, in)
	catalogRespond(w, 201, out, err)
}
func (s *Server) bankAssign(w http.ResponseWriter, r *http.Request) {
	u, err := s.currentUser(r)
	if err != nil {
		WriteError(w, err)
		return
	}
	var in domain.BankAssignInput
	if err = decodeJSON(r, &in); err != nil {
		WriteError(w, err)
		return
	}
	out, err := s.store.AssignBankCollection(r.Context(), u, chi.URLParam(r, "collectionID"), in)
	catalogRespond(w, 200, out, err)
}
