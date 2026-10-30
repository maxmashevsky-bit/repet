package store

import (
	"testing"
	"time"
	"tutor-platform/services/core/internal/domain"
)

func TestAnalyticsCalculations(t *testing.T) {
	for _, tc := range []struct {
		n, d int
		want *float64
	}{{0, 0, nil}, {1, 3, ptrAnalytics(33.3)}, {20, 20, ptrAnalytics(100)}, {0, 10, ptrAnalytics(0)}} {
		got := analyticsPercent(tc.n, tc.d)
		if got == nil && tc.want != nil || got != nil && (tc.want == nil || *got != *tc.want) {
			t.Fatalf("percent %d/%d=%v", tc.n, tc.d, got)
		}
	}
	m := domain.AnalyticsMetric{Attempts: 30, Graded: 20, Correct: 15, Incorrect: 5, Pending: 2, Hinted: 11}
	analyticsMetric(&m)
	if *m.Accuracy != 75 || *m.HintRate != 50 {
		t.Fatal(m)
	}
	for _, tc := range []struct {
		n    int
		p    float64
		want string
	}{{4, 100, "insufficient"}, {5, 69.9, "practice"}, {5, 70, "developing"}, {5, 85, "secure"}} {
		if got := analyticsLevel(tc.n, &tc.p); got != tc.want {
			t.Fatal(got, tc)
		}
	}
	d := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, f := range []domain.AnalyticsFilter{{}, {From: d, To: d.AddDate(0, 0, -1)}, {From: d, To: d.AddDate(0, 0, 366)}, {From: d, To: d, TopicOffset: -1}, {From: d, To: d, ExamOffset: 100001}} {
		if ValidateAnalyticsFilter(f) == nil {
			t.Fatal("accepted invalid period", f)
		}
	}
	if ValidateAnalyticsFilter(domain.AnalyticsFilter{From: d, To: d.AddDate(0, 0, 365)}) != nil {
		t.Fatal("366 inclusive days should fit")
	}
}
func ptrAnalytics(n float64) *float64 { return &n }
