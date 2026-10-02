package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

const maxAssignments = 100

// ListAssignmentsInput selects a course's assignments.
type ListAssignmentsInput struct {
	CourseID    int64 `json:"courseId" jsonschema:"course ID from list_courses"`
	OnlyPending bool  `json:"onlyPending,omitempty" jsonschema:"only assignments the user has not submitted yet"`
}

// Assignment is one assignment folder with the user's own status.
type Assignment struct {
	ID             int64    `json:"id" jsonschema:"folder ID, used by get_assignment"`
	Name           string   `json:"name"`
	Due            string   `json:"due,omitempty" jsonschema:"due date, RFC 3339"`
	AvailableFrom  string   `json:"availableFrom,omitempty" jsonschema:"RFC 3339"`
	AvailableUntil string   `json:"availableUntil,omitempty" jsonschema:"RFC 3339"`
	MaxPoints      *float64 `json:"maxPoints,omitempty"`
	Group          bool     `json:"group,omitempty" jsonschema:"a group assignment"`
	Status         string   `json:"status" jsonschema:"not submitted, draft, submitted, feedback published, or unknown when Brightspace would not say"`
	SubmittedAt    string   `json:"submittedAt,omitempty" jsonschema:"latest submission, RFC 3339"`
	Submissions    int      `json:"submissions,omitempty"`
	Score          *float64 `json:"score,omitempty" jsonschema:"the user's score once feedback is published"`
	URL            string   `json:"url" jsonschema:"the assignment in Brightspace"`
}

// AssignmentList is the result of list_assignments.
type AssignmentList struct {
	Assignments []Assignment `json:"assignments"`
	More        bool         `json:"more,omitempty" jsonschema:"the course has more assignments than listed"`
}

func addListAssignments(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_assignments",
		Description: "List the assignments (dropbox folders) in one Brightspace course with due dates, availability, points and the user's own submission status, soonest due first. " +
			"Use this when the user asks what assignments a course has, what is still to hand in, or whether something was submitted; get the course ID from list_courses. " +
			"It covers one course per call and only the user's own submissions. It does not show instructions or feedback text (use get_assignment) and does not submit anything.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListAssignmentsInput) (*mcp.CallToolResult, AssignmentList, error) {
		if in.CourseID <= 0 {
			return nil, AssignmentList{}, errCourseRequired
		}
		c, err := connect()
		if err != nil {
			return nil, AssignmentList{}, sessionError(err, "")
		}
		folders, err := c.DropboxFolders(ctx, in.CourseID)
		if err != nil {
			return nil, AssignmentList{}, courseError(err, c.BaseURL(), in.CourseID, "assignments")
		}
		folders = slices.DeleteFunc(folders, func(f brightspace.DropboxFolder) bool { return f.IsHidden })

		list := AssignmentList{Assignments: []Assignment{}}
		if len(folders) > maxAssignments {
			folders, list.More = folders[:maxAssignments], true
		}

		entities := make([][]brightspace.DropboxEntity, len(folders))
		errs := make([]error, len(folders))
		forEachLimit(len(folders), func(i int) {
			entities[i], errs[i] = c.MySubmissions(ctx, in.CourseID, folders[i].ID)
		})
		for _, err := range errs {
			if errors.Is(err, brightspace.ErrSessionExpired) {
				return nil, AssignmentList{}, sessionError(err, c.BaseURL())
			}
		}

		for i, f := range folders {
			a := assignmentOf(c.BaseURL(), in.CourseID, f)
			if errs[i] == nil {
				a.applyEntities(entities[i])
			}
			if in.OnlyPending && a.Status != "not submitted" && a.Status != "draft" {
				continue
			}
			list.Assignments = append(list.Assignments, a)
		}
		// Soonest due first; no due date goes last. RFC 3339 in UTC sorts as text.
		slices.SortStableFunc(list.Assignments, func(a, b Assignment) int {
			if (a.Due == "") != (b.Due == "") {
				if a.Due == "" {
					return 1
				}
				return -1
			}
			return cmp.Compare(a.Due, b.Due)
		})
		return nil, list, nil
	})
}

func assignmentOf(baseURL string, courseID int64, f brightspace.DropboxFolder) Assignment {
	return Assignment{
		ID:             f.ID,
		Name:           f.Name,
		Due:            formatTime(f.DueDate),
		AvailableFrom:  formatTime(f.Availability.StartDate),
		AvailableUntil: formatTime(f.Availability.EndDate),
		MaxPoints:      f.Assessment.ScoreDenominator,
		Group:          f.GroupTypeID != nil,
		Status:         "unknown",
		URL:            fmt.Sprintf("%s/d2l/lms/dropbox/user/folder_submit_files.d2l?db=%d&grpid=0&isprv=0&bp=0&ou=%d", baseURL, f.ID, courseID),
	}
}

// applyEntities fills in the user's submission status from the dropbox entities.
func (a *Assignment) applyEntities(entities []brightspace.DropboxEntity) {
	a.Status = "not submitted"
	var latest *time.Time
	for _, e := range entities {
		switch statusName(string(e.Status)) {
		case "feedback published":
			a.Status = "feedback published"
			if e.Feedback != nil && e.Feedback.Score != nil {
				a.Score = e.Feedback.Score
			}
		case "submitted":
			if a.Status != "feedback published" {
				a.Status = "submitted"
			}
		case "draft":
			if a.Status == "not submitted" {
				a.Status = "draft"
			}
		}
		for _, s := range e.Submissions {
			a.Submissions++
			if s.Date != nil && (latest == nil || s.Date.After(*latest)) {
				latest = s.Date
			}
		}
	}
	// Submissions without a status still count as submitted.
	if a.Status == "not submitted" && a.Submissions > 0 {
		a.Status = "submitted"
	}
	a.SubmittedAt = formatTime(latest)
}

// statusName maps an entity status, sent as a number or a name, to words.
func statusName(s string) string {
	switch strings.ToLower(s) {
	case "0", "unsubmitted":
		return "not submitted"
	case "1", "submitted":
		return "submitted"
	case "2", "draft":
		return "draft"
	case "3", "published":
		return "feedback published"
	}
	return "unknown"
}
