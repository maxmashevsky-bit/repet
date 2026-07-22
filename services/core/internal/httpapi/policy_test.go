package httpapi

import (
	"testing"

	"tutor-platform/services/core/internal/domain"
)

func TestRoleValidation(t *testing.T) {
	if domain.Role("owner").Valid() {
		t.Fatal("unexpected custom role accepted")
	}
	if !domain.RoleTutor.Valid() || !domain.RoleStudent.Valid() || !domain.RoleAdmin.Valid() {
		t.Fatal("expected built-in roles to be valid")
	}
}
