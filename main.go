// Package main starts the Canvas Assignment Tracker web application.
package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"

	"canvas-assignment-tracker/views"
)

const defaultLookAheadDays = 14

// render writes the HTTP status code and renders a Templ component into the
// current Echo response.
func render(c echo.Context, statusCode int, component templ.Component) error {
	c.Response().WriteHeader(statusCode)

	return component.Render(
		c.Request().Context(),
		c.Response().Writer,
	)
}

// dashboardHandler loads courses first, then optionally loads assignments for
// the course selected by the form. Keeping those calls separate makes the
// Canvas endpoints visible in the app's normal request flow.
func dashboardHandler(c echo.Context) error {
	loadDotEnv(".env")

	data := views.PageData{
		CourseID: c.QueryParam("course_id"),
		Days:     parseDays(c.QueryParam("days")),
	}
	if data.CourseID == "" {
		data.CourseID = "all"
	}

	client, err := clientFromEnvironment()
	if err != nil {
		data.Error = err.Error()
		return render(c, http.StatusOK, views.Home(data))
	}

	courses, err := client.ListCourses(c.Request().Context())
	if err != nil {
		data.Error = friendlyError(err)
		return render(c, http.StatusOK, views.Home(data))
	}
	data.Courses = courseOptions(courses)

	if c.QueryParam("load") != "1" {
		data.Message = "Choose a course and time window, then select Load assignments."
		return render(c, http.StatusOK, views.Home(data))
	}

	selectedCourses, err := chooseCourses(courses, data.CourseID)
	if err != nil {
		data.Error = err.Error()
		return render(c, http.StatusOK, views.Home(data))
	}

	data.Assignments, data.Warnings = loadAssignments(c, client, selectedCourses, data.Days)
	if len(data.Assignments) == 0 && len(data.Warnings) == 0 {
		data.Message = fmt.Sprintf("No assignments due in the next %d days.", data.Days)
	}

	return render(c, http.StatusOK, views.Home(data))
}

func clientFromEnvironment() (*CanvasClient, error) {
	token := strings.TrimSpace(os.Getenv("CANVAS_API_TOKEN"))
	if token == "" || token == "your_canvas_token_here" {
		return nil, fmt.Errorf("Canvas API token is missing. Copy .env.example to .env and add your token")
	}

	baseURL := strings.TrimSpace(os.Getenv("CANVAS_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://boisestatecanvas.instructure.com"
	}

	return NewCanvasClient(baseURL, token)
}

func parseDays(value string) int {
	days, err := strconv.Atoi(value)
	if err != nil || days < 1 || days > 365 {
		return defaultLookAheadDays
	}
	return days
}

func courseOptions(courses []Course) []views.CourseOption {
	options := make([]views.CourseOption, 0, len(courses)+1)
	options = append(options, views.CourseOption{ID: "all", Name: "All active courses"})
	for _, course := range courses {
		name := course.Name
		if strings.TrimSpace(name) == "" {
			name = "Untitled course"
		}
		options = append(options, views.CourseOption{
			ID:   strconv.Itoa(course.ID),
			Name: name,
		})
	}
	return options
}

func chooseCourses(courses []Course, selectedID string) ([]Course, error) {
	if selectedID == "all" {
		return courses, nil
	}

	id, err := strconv.Atoi(selectedID)
	if err != nil {
		return nil, fmt.Errorf("course selection %q is not valid", selectedID)
	}
	for _, course := range courses {
		if course.ID == id {
			return []Course{course}, nil
		}
	}
	return nil, fmt.Errorf("course %q was not found in your active courses", selectedID)
}

func loadAssignments(c echo.Context, client *CanvasClient, courses []Course, days int) ([]views.AssignmentRow, []string) {
	type rowWithDue struct {
		due time.Time
		row views.AssignmentRow
	}
	rowsWithDue := make([]rowWithDue, 0)
	warnings := make([]string, 0)
	seen := make(map[int]struct{})
	now := time.Now()
	cutoff := now.AddDate(0, 0, days)

	for _, course := range courses {
		for _, bucket := range []string{"upcoming", "future"} {
			assignments, err := client.ListAssignments(c.Request().Context(), course.ID, bucket)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s (%s): %s", course.Name, bucket, friendlyError(err)))
				continue
			}

			for _, assignment := range assignments {
				due, ok := parseDueDate(assignment.DueAt)
				if !ok || due.Before(now) || due.After(cutoff) {
					continue
				}
				if _, alreadySeen := seen[assignment.ID]; alreadySeen {
					continue
				}
				seen[assignment.ID] = struct{}{}

				rowsWithDue = append(rowsWithDue, rowWithDue{
					due: due,
					row: assignmentRow(course, assignment, due),
				})
			}
		}
	}

	sort.SliceStable(rowsWithDue, func(i, j int) bool {
		return rowsWithDue[i].due.Before(rowsWithDue[j].due)
	})
	rows := make([]views.AssignmentRow, 0, len(rowsWithDue))
	for _, item := range rowsWithDue {
		rows = append(rows, item.row)
	}
	return rows, warnings
}

func parseDueDate(value *string) (time.Time, bool) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return time.Time{}, false
	}
	due, err := time.Parse(time.RFC3339, *value)
	return due, err == nil
}

func assignmentRow(course Course, assignment Assignment, due time.Time) views.AssignmentRow {
	urgency := "normal"
	urgencyLabel := "On track"
	daysUntil := int(time.Until(due).Hours() / 24)
	if daysUntil < 0 {
		urgency = "overdue"
		urgencyLabel = "Overdue"
	} else if daysUntil <= 2 {
		urgency = "soon"
		urgencyLabel = "Due soon"
	}

	return views.AssignmentRow{
		CourseName:   course.Name,
		Name:         assignment.Name,
		DueLabel:     due.Local().Format("Mon, Jan 2 · 3:04 PM"),
		PointsLabel:  formatPoints(assignment.PointsPossible),
		Urgency:      urgency,
		UrgencyLabel: urgencyLabel,
		URL:          assignment.HTMLURL,
	}
}

func formatPoints(points *float64) string {
	if points == nil {
		return "Points not set"
	}
	if *points == float64(int(*points)) {
		return fmt.Sprintf("%d pts", int(*points))
	}
	return fmt.Sprintf("%.1f pts", *points)
}

func friendlyError(err error) string {
	if apiErr, ok := err.(*CanvasAPIError); ok {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return "Canvas rejected the token. Check that it is valid and has not expired."
		case http.StatusTooManyRequests:
			return "Canvas rate-limited the request. Wait a moment and try again."
		}
		return fmt.Sprintf("Canvas returned %s.", apiErr.Status)
	}
	return fmt.Sprintf("Could not reach Canvas: %v", err)
}

func main() {
	loadDotEnv(".env")
	e := echo.New()
	e.Static("/static", "static")
	e.GET("/", dashboardHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
