// Package prefs loads which classes you want booked.
package prefs

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Preference describes one class you'd like on a given weekday.
type Preference struct {
	Weekday string `json:"weekday"` // "Monday" ... "Sunday"
	Time    string `json:"time"`    // 24h "HH:MM", e.g. "07:00"
	Workout string `json:"workout"` // substring match, case-insensitive, e.g. "HRX"
}

// File is the top-level shape of preferences.json.
type File struct {
	CenterID  string       `json:"centerId"`
	DaysAhead int          `json:"daysAhead"` // how far ahead classes open for booking
	Classes   []Preference `json:"classes"`
}

// Load reads and validates a preferences file.
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read preferences: %w", err)
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse preferences: %w", err)
	}
	if f.DaysAhead <= 0 {
		f.DaysAhead = 1
	}
	for i, c := range f.Classes {
		if _, err := time.Parse("15:04", c.Time); err != nil {
			return nil, fmt.Errorf("classes[%d]: bad time %q (want HH:MM)", i, c.Time)
		}
		if _, ok := weekdays[strings.ToLower(c.Weekday)]; !ok {
			return nil, fmt.Errorf("classes[%d]: bad weekday %q", i, c.Weekday)
		}
	}
	return &f, nil
}

var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday,
	"friday": time.Friday, "saturday": time.Saturday,
}

// ForDay returns the preferences that apply to the given weekday.
func (f *File) ForDay(d time.Weekday) []Preference {
	var out []Preference
	for _, c := range f.Classes {
		if weekdays[strings.ToLower(c.Weekday)] == d {
			out = append(out, c)
		}
	}
	return out
}
