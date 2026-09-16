package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// CanvasClient contains the shared HTTP settings for Canvas requests.
type CanvasClient struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
}

// Course is the subset of a Canvas course needed by the assignment tracker.
type Course struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Assignment is the subset of a Canvas assignment needed by the assignment tracker.
type Assignment struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	DueAt          *string  `json:"due_at"`
	PointsPossible *float64 `json:"points_possible"`
	HTMLURL        string   `json:"html_url"`
}

// CanvasAPIError preserves the status returned by Canvas for friendly UI errors.
type CanvasAPIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *CanvasAPIError) Error() string {
	if e.Body == "" {
		return e.Status
	}
	return fmt.Sprintf("%s: %s", e.Status, e.Body)
}

// NewCanvasClient validates the instance URL and prepares an authenticated client.
func NewCanvasClient(baseURL, token string) (*CanvasClient, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("Canvas base URL must be a valid HTTPS URL")
	}
	return &CanvasClient{
		baseURL:    parsed,
		token:      token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// ListCourses calls GET /api/v1/courses and follows Canvas pagination links.
func (c *CanvasClient) ListCourses(ctx context.Context) ([]Course, error) {
	query := url.Values{
		"enrollment_state": {"active"},
		"per_page":         {"100"},
	}
	return getAll[Course](ctx, c, "/api/v1/courses", query)
}

// ListAssignments calls GET /api/v1/courses/:id/assignments for the requested
// Canvas bucket and follows every page returned in the Link header. The caller
// applies the look-ahead window locally after combining bucket results.
func (c *CanvasClient) ListAssignments(ctx context.Context, courseID int, bucket string) ([]Assignment, error) {
	query := url.Values{
		"bucket":   {bucket},
		"order_by": {"due_at"},
		"per_page": {"100"},
	}
	path := "/api/v1/courses/" + strconv.Itoa(courseID) + "/assignments"
	return getAll[Assignment](ctx, c, path, query)
}

func getAll[T any](ctx context.Context, client *CanvasClient, path string, query url.Values) ([]T, error) {
	nextURL := client.baseURL.ResolveReference(&url.URL{Path: path, RawQuery: query.Encode()})
	allItems := make([]T, 0)

	for nextURL != nil {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+client.token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "CS408 Canvas Assignment Tracker")

		resp, err := client.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, &CanvasAPIError{
				StatusCode: resp.StatusCode,
				Status:     resp.Status,
				Body:       strings.TrimSpace(string(body)),
			}
		}

		var page []T
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("Canvas returned invalid JSON: %w", err)
		}
		allItems = append(allItems, page...)

		nextURL, err = nextPageURL(resp.Header.Get("Link"), client.baseURL)
		if err != nil {
			return nil, err
		}
	}

	return allItems, nil
}

func nextPageURL(linkHeader string, baseURL *url.URL) (*url.URL, error) {
	for _, link := range strings.Split(linkHeader, ",") {
		parts := strings.Split(link, ";")
		if len(parts) < 2 || !hasLinkRelation(parts[1:], "next") {
			continue
		}
		value := strings.TrimSpace(parts[0])
		if len(value) < 2 || value[0] != '<' || value[len(value)-1] != '>' {
			return nil, fmt.Errorf("Canvas returned a malformed pagination link")
		}
		next, err := url.Parse(value[1 : len(value)-1])
		if err != nil {
			return nil, fmt.Errorf("Canvas returned an invalid pagination link: %w", err)
		}
		if !next.IsAbs() {
			next = baseURL.ResolveReference(next)
		}
		if next.Scheme != baseURL.Scheme || next.Host != baseURL.Host {
			return nil, fmt.Errorf("Canvas returned a pagination link to an unexpected host")
		}
		return next, nil
	}
	return nil, nil
}

func hasLinkRelation(parameters []string, wanted string) bool {
	for _, parameter := range parameters {
		key, value, found := strings.Cut(strings.TrimSpace(parameter), "=")
		if !found || strings.TrimSpace(key) != "rel" {
			continue
		}
		for _, relation := range strings.Fields(strings.Trim(strings.TrimSpace(value), `"`)) {
			if relation == wanted {
				return true
			}
		}
	}
	return false
}

// loadDotEnv loads simple KEY=value entries without overriding environment
// variables already supplied by the shell or a hosting platform.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
	}
}
