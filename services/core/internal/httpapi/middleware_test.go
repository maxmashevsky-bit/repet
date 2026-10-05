package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRequestLogDoesNotExposePathToken(t *testing.T) {
	var output bytes.Buffer
	router := chi.NewRouter()
	router.Use(RequestLog(slog.New(slog.NewTextHandler(&output, nil))))
	router.Post("/invitations/{token}/accept", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/invitations/private-invitation-code/accept", nil))
	if strings.Contains(output.String(), "private-invitation-code") || !strings.Contains(output.String(), "/invitations/{token}/accept") {
		t.Fatalf("request log must contain only the route pattern: %s", output.String())
	}
}
