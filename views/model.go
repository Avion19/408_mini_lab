package views

import (
	"strconv"

	"github.com/a-h/templ"
)

// CourseOption is a course choice shown in the filter form.
type CourseOption struct {
	ID   string
	Name string
}

// AssignmentRow contains display-ready assignment data.
type AssignmentRow struct {
	CourseName   string
	Name         string
	DueLabel     string
	PointsLabel  string
	Urgency      string
	UrgencyLabel string
	URL          string
}

// PageData is the complete view model for the dashboard.
type PageData struct {
	Courses     []CourseOption
	Assignments []AssignmentRow
	Warnings    []string
	CourseID    string
	Days        int
	Message     string
	Error       string
}

func assignmentCount(assignments []AssignmentRow) string {
	return strconv.Itoa(len(assignments))
}

func assignmentURL(value string) templ.SafeURL {
	// Canvas supplies this value in its API response. Sanitize it before using
	// it in an href so an unexpected URL scheme cannot become a script link.
	return templ.URL(value)
}
