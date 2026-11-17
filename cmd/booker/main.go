// Command booker books your preferred cult.fit classes for tomorrow (or
// whatever daysAhead says). It runs once and exits; a scheduler (GitHub
// Actions) calls it every night.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/samarthsangam/auto-book-group-class/internal/cult"
	"github.com/samarthsangam/auto-book-group-class/internal/prefs"
)

func main() {
	prefsPath := flag.String("prefs", "preferences.json", "path to preferences file")
	dryRun := flag.Bool("dry-run", false, "find classes but don't book them")
	debug := flag.Bool("debug", false, "print raw API responses")
	flag.Parse()

	if err := run(*prefsPath, *dryRun, *debug); err != nil {
		log.Fatal(err)
	}
}

func run(prefsPath string, dryRun, debug bool) error {
	cookie := os.Getenv("CULT_COOKIE")
	if cookie == "" {
		return fmt.Errorf("CULT_COOKIE is not set (see README)")
	}
	p, err := prefs.Load(prefsPath)
	if err != nil {
		return err
	}

	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return fmt.Errorf("load IST timezone: %w", err)
	}
	target := time.Now().In(ist).AddDate(0, 0, p.DaysAhead)
	date := target.Format("2006-01-02")

	wanted := p.ForDay(target.Weekday())
	if len(wanted) == 0 {
		log.Printf("nothing to book for %s (%s)", date, target.Weekday())
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := cult.New(cookie, os.Getenv("CULT_API_KEY"), os.Getenv("CULT_BASE_URL"), debug)
	classes, err := client.ListClasses(ctx, p.CenterID)
	if err != nil {
		return err
	}
	log.Printf("fetched %d classes", len(classes))

	failed := 0
	for _, w := range wanted {
		c, ok := cult.Match(classes, date, w.Time, w.Workout, p.CenterID)
		if !ok {
			log.Printf("NO MATCH: %s %s %s on %s (not listed, or only waitlist available)", w.Workout, w.Time, p.CenterID, date)
			failed++
			continue
		}
		if c.State == cult.StateBooked {
			log.Printf("ALREADY BOOKED: %s %s %s", c.Workout, c.Date, c.StartTime)
			continue
		}
		if dryRun {
			log.Printf("DRY RUN: would book %s %s %s (id %s)", c.Workout, c.Date, c.StartTime, c.ID)
			continue
		}
		if err := client.Book(ctx, c.ID); err != nil {
			log.Printf("FAILED booking %s %s: %v", c.Workout, c.StartTime, err)
			failed++
			continue
		}
		log.Printf("BOOKED: %s %s %s (id %s)", c.Workout, c.Date, c.StartTime, c.ID)
	}
	if failed > 0 {
		return fmt.Errorf("%d booking(s) did not succeed", failed) // non-zero exit => GitHub emails you
	}
	return nil
}
