package auth

import (
	"net/http"
	"testing"
)

func TestDevProviderRequiresCompleteHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Dev-User-ID", "11111111-1111-4111-8111-111111111111")
	req.Header.Set("X-Dev-Email", "tutor@example.test")
	provider := DevProvider{Enabled: true}
	if _, err := provider.Authenticate(req); err == nil {
		t.Fatal("expected missing role to fail")
	}
}
