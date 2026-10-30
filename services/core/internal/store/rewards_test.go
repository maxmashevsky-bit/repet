package store

import "testing"

func TestRewardLevels(t *testing.T) {
	for _, tc := range []struct{ xp, level, progress int }{{0, 1, 0}, {10, 1, 10}, {90, 1, 90}, {100, 2, 0}, {210, 3, 10}} {
		l, p := rewardLevel(tc.xp)
		if l != tc.level || p != tc.progress {
			t.Fatal(tc, l, p)
		}
	}
}
