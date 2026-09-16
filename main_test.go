package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestListCoursesFollowsCanvasPagination(t *testing.T) {
	client, err := NewCanvasClient("https://canvas.test", "test-token")
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization header = %q", got)
		}
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		if req.URL.Query().Get("page") == "2" {
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: header, Body: io.NopCloser(strings.NewReader(`[{"id":2,"name":"Second course"}]`)), Request: req}, nil
		}
		header.Set("Link", fmt.Sprintf("<%s/api/v1/courses?page=2>; rel=\"next\"", "https://canvas.test"))
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: header, Body: io.NopCloser(strings.NewReader(`[{"id":1,"name":"First course"}]`)), Request: req}, nil
	})}

	courses, err := client.ListCourses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(courses) != 2 || courses[0].ID != 1 || courses[1].ID != 2 {
		t.Fatalf("courses = %#v", courses)
	}
}

func TestListAssignmentsUsesRequestedBucket(t *testing.T) {
	client, err := NewCanvasClient("https://canvas.test", "test-token")
	if err != nil {
		t.Fatal(err)
	}
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.Query().Get("bucket"); got != "future" {
			t.Fatalf("bucket query parameter = %q, want future", got)
		}
		if got := req.URL.Query().Get("order_by"); got != "due_at" {
			t.Fatalf("order_by query parameter = %q", got)
		}
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		body := `[{"id":1661420,"name":"05.01 Project Specification","due_at":"2026-09-26T05:59:59Z"}]`
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}

	assignments, err := client.ListAssignments(context.Background(), 48195, "future")
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 1 || assignments[0].Name != "05.01 Project Specification" {
		t.Fatalf("assignments = %#v", assignments)
	}
}

func TestLoadAssignmentsMergesBucketsAndExcludesOverdue(t *testing.T) {
	client, err := NewCanvasClient("https://canvas.test", "test-token")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	format := func(offset time.Duration) string {
		return now.Add(offset).Format(time.RFC3339)
	}
	client.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Query().Get("bucket") {
		case "upcoming":
			body = fmt.Sprintf(`[{"id":1,"name":"Upcoming","due_at":%q},{"id":2,"name":"Shared","due_at":%q},{"id":5,"name":"Overdue","due_at":%q}]`, format(24*time.Hour), format(5*24*time.Hour), format(-24*time.Hour))
		case "future":
			body = fmt.Sprintf(`[{"id":2,"name":"Shared","due_at":%q},{"id":3,"name":"Future in range","due_at":%q},{"id":4,"name":"Future out of range","due_at":%q}]`, format(5*24*time.Hour), format(10*24*time.Hour), format(15*24*time.Hour))
		default:
			t.Fatalf("unexpected bucket %q", req.URL.Query().Get("bucket"))
		}
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	rows, warnings := loadAssignments(c, client, []Course{{ID: 48195, Name: "CS 408"}}, 14)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if len(rows) != 3 || rows[0].Name != "Upcoming" || rows[1].Name != "Shared" || rows[2].Name != "Future in range" {
		t.Fatalf("rows = %#v", rows)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDashboardShowsMissingTokenMessage(t *testing.T) {
	t.Setenv("CANVAS_API_TOKEN", "")
	t.Setenv("CANVAS_BASE_URL", "https://boisestatecanvas.instructure.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := dashboardHandler(c)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Canvas API token is missing") {
		t.Fatal("missing-token message not found")
	}
}
