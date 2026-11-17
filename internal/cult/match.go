package cult

import "strings"

// Class states from the listing. Anything else (WAITLIST_AVAILABLE,
// WAITLIST_FULL, ...) is deliberately not bookable by this tool.
const (
	StateAvailable = "AVAILABLE"
	StateBooked    = "BOOKED"
)

// Match picks the first class on date that fits the wanted time and workout
// (case-insensitive substring) and is either bookable or already booked by
// you. centerID may be empty to accept any.
func Match(classes []Class, date, startTime, workout, centerID string) (Class, bool) {
	for _, c := range classes {
		if c.Date != date || c.StartTime != startTime {
			continue
		}
		if !strings.Contains(strings.ToLower(c.Workout), strings.ToLower(workout)) {
			continue
		}
		if centerID != "" && c.CenterID != centerID {
			continue
		}
		if c.State != StateAvailable && c.State != StateBooked {
			continue
		}
		return c, true
	}
	return Class{}, false
}
