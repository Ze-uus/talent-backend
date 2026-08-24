package jobs

import (
	"testing"
	"time"
)

func TestReminderWindow(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC) // 10 days

	cases := []struct {
		name                string
		now                 time.Time
		mid, d3, d1         bool
	}{
		{"before_mid", start.Add(2 * 24 * time.Hour), false, false, false},
		{"at_mid", start.Add(5 * 24 * time.Hour), true, false, false},
		{"ending_3d", end.Add(-48 * time.Hour), true, true, false},
		{"ending_24h", end.Add(-12 * time.Hour), true, false, true},
		{"after_end", end.Add(time.Hour), true, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mid, d3, d1 := ReminderWindow(start, end, tc.now)
			if mid != tc.mid || d3 != tc.d3 || d1 != tc.d1 {
				t.Fatalf("got mid=%v d3=%v d1=%v want mid=%v d3=%v d1=%v",
					mid, d3, d1, tc.mid, tc.d3, tc.d1)
			}
		})
	}
}

func TestReminderWindowInvalid(t *testing.T) {
	mid, d3, d1 := ReminderWindow(time.Time{}, time.Now(), time.Now())
	if mid || d3 || d1 {
		t.Fatalf("expected all false for zero start")
	}
}
