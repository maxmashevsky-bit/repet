package store

import (
	"testing"
	"tutor-platform/services/core/internal/domain"
)

func TestReviewFilter(t *testing.T) {
	for _, f := range []domain.ReviewFilter{{Mode: "secret"}, {Limit: 101}, {Limit: -1}, {Offset: -1}, {Offset: 100001}, {ActiveOffset: -1}, {ActiveOffset: 100001}} {
		if ValidateReviewFilter(&f) == nil {
			t.Fatal(f)
		}
	}
	f := domain.ReviewFilter{}
	if err := ValidateReviewFilter(&f); err != nil || f.Mode != "due" || f.Limit != 20 {
		t.Fatalf("defaults: %+v, %v", f, err)
	}
}
