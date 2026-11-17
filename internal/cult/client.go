// Package cult talks to cult.fit's (unofficial, undocumented) web API.
//
// Everything that may break when cult changes their API lives in this file:
// the URLs, the auth headers and the JSON shapes. Run the app with -debug to
// see raw responses and adjust.
package cult

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://next.cult.fit"

// Class is one bookable class slot.
type Class struct {
	ID             string
	Workout        string
	Date           string // YYYY-MM-DD
	StartTime      string // HH:MM
	CenterID       string
	AvailableSeats int
	State          string
}

// Client is a minimal cult.fit API client.
type Client struct {
	http    *http.Client
	baseURL string
	cookie  string // full Cookie header copied from a logged-in browser
	apiKey  string // optional "apikey" header seen in browser requests
	debug   bool
}

// New builds a client. cookie is required; apiKey and baseURL are optional.
func New(cookie, apiKey, baseURL string, debug bool) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		http:    &http.Client{Timeout: 20 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
		cookie:  cookie,
		apiKey:  apiKey,
		debug:   debug,
	}
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, body any) ([]byte, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120 Safari/537.36")
	req.Header.Set("x-source-app", "janus-next") // sent by next.cult.fit's own frontend
	if c.apiKey != "" {
		req.Header.Set("apikey", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if c.debug {
		fmt.Printf("DEBUG %s %s -> %d\n%s\n", method, u, resp.StatusCode, data)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%s %s: HTTP %d — session expired? refresh CULT_COOKIE", method, path, resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		return nil, &statusError{Method: method, Path: path, Code: resp.StatusCode, Body: truncate(string(data), 300)}
	}
	return data, nil
}

// classesResponse mirrors the listing endpoint (verified against a real
// response). Classes are nested: date -> time slot -> centre -> classes.
type classesResponse struct {
	Data struct {
		ClassByDateMap map[string]struct {
			ClassByTimeList []struct {
				CenterWiseClasses []struct {
					Classes []struct {
						ID             string      `json:"id"`
						Date           string      `json:"date"`
						StartTime      string      `json:"startTime"` // "HH:MM:SS"
						WorkoutName    string      `json:"workoutName"`
						AvailableSeats int         `json:"availableSeats"`
						State          string      `json:"state"`
						CenterID       json.Number `json:"centerID"`
					} `json:"classes"`
				} `json:"centerWiseClasses"`
			} `json:"classByTimeList"`
		} `json:"classByDateMap"`
	} `json:"data"`
}

// ListClasses returns every class listed for the centre.
func (c *Client) ListClasses(ctx context.Context, centerID string) ([]Class, error) {
	q := url.Values{"centerId": {centerID}}
	data, err := c.getWithRetry(ctx, "/fitness/classes", q)
	if err != nil {
		return nil, err
	}
	var r classesResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("decode classes (API shape changed? run with -debug): %w", err)
	}
	var out []Class
	for _, day := range r.Data.ClassByDateMap {
		for _, slot := range day.ClassByTimeList {
			for _, cw := range slot.CenterWiseClasses {
				for _, cl := range cw.Classes {
					start := cl.StartTime
					if len(start) >= 5 {
						start = start[:5]
					}
					out = append(out, Class{
						ID:             cl.ID,
						Workout:        cl.WorkoutName,
						Date:           cl.Date,
						StartTime:      start,
						CenterID:       cl.CenterID.String(),
						AvailableSeats: cl.AvailableSeats,
						State:          cl.State,
					})
				}
			}
		}
	}
	return out, nil
}

// Book books the class with the given ID.
func (c *Client) Book(ctx context.Context, classID string) error {
	// Payload copied from the browser's own request. operatingSystemVersion is
	// whatever the web app sends; it looks like a fixed value.
	body := map[string]string{
		"classId":                classID,
		"operatingSystemVersion": "14",
		"productType":            "FITNESS",
	}
	_, err := c.do(ctx, http.MethodPost, "/fitness/class-book", nil, body)
	return err
}

// statusError is returned for non-2xx responses so callers can tell 5xx
// (worth retrying) from 4xx (not).
type statusError struct {
	Method, Path string
	Code         int
	Body         string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Path, e.Code, e.Body)
}

// getWithRetry retries GETs that fail with a 5xx (cult/Cloudflare blips).
// It must not be used for the booking POST: a retry could double-book.
func (c *Client) getWithRetry(ctx context.Context, path string, q url.Values) ([]byte, error) {
	const attempts = 4
	delay := 2 * time.Second
	var err error
	for i := 1; i <= attempts; i++ {
		var data []byte
		data, err = c.do(ctx, http.MethodGet, path, q, nil)
		var se *statusError
		if err == nil || !errors.As(err, &se) || se.Code < 500 {
			return data, err
		}
		if i == attempts {
			break
		}
		log.Printf("attempt %d/%d: %v — retrying in %s", i, attempts, err, delay)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		delay *= 2
	}
	return nil, err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
