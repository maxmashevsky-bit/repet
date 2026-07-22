package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"tutor-platform/services/core/internal/domain"
)

type contextKey string

const principalKey contextKey = "principal"

type Principal struct {
	UserID      string
	Email       string
	DisplayName string
	Role        domain.Role
}

type IdentityProvider interface {
	Authenticate(*http.Request) (Principal, error)
}

var ErrUnauthenticated = errors.New("unauthenticated")

type DevProvider struct {
	Enabled bool
}

func (p DevProvider) Authenticate(r *http.Request) (Principal, error) {
	if !p.Enabled {
		return Principal{}, ErrUnauthenticated
	}
	id := strings.TrimSpace(r.Header.Get("X-Dev-User-ID"))
	email := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Dev-Email")))
	role := domain.Role(strings.TrimSpace(r.Header.Get("X-Dev-Role")))
	name := strings.TrimSpace(r.Header.Get("X-Dev-Display-Name"))
	if id == "" || email == "" || !role.Valid() {
		return Principal{}, ErrUnauthenticated
	}
	if name == "" {
		name = email
	}
	return Principal{UserID: id, Email: email, DisplayName: name, Role: role}, nil
}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}
