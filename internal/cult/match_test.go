package cult

import "testing"

func TestMatch(t *testing.T) {
	classes := []Class{
		{ID: "1", Workout: "HRX Workout", Date: "2026-11-17", StartTime: "07:00", CenterID: "10", State: "WAITLIST_AVAILABLE"},
		{ID: "2", Workout: "HRX Workout", Date: "2026-11-17", StartTime: "07:00", CenterID: "11", State: "AVAILABLE"},
		{ID: "3", Workout: "Yoga", Date: "2026-11-17", StartTime: "07:00", CenterID: "10", State: "AVAILABLE"},
		{ID: "4", Workout: "ADIDAS STRENGTH+", Date: "2026-11-17", StartTime: "07:00", CenterID: "10", State: "BOOKED"},
	}
	tests := []struct {
		name, workout, center, wantID string
		ok                            bool
	}{
		{"skips waitlist-only class", "hrx", "", "2", true},
		{"center filter", "hrx", "10", "", false},
		{"other workout", "yoga", "10", "3", true},
		{"no match", "boxing", "", "", false},
		{"already booked is a match", "strength", "", "4", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Match(classes, "2026-11-17", "07:00", tt.workout, tt.center)
			if ok != tt.ok || got.ID != tt.wantID {
				t.Fatalf("got (%q, %v), want (%q, %v)", got.ID, ok, tt.wantID, tt.ok)
			}
		})
	}
}
