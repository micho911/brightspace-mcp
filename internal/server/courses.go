package server

import (
	"cmp"
	"context"
	"net/url"
	"slices"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListCoursesInput selects which courses to list.
type ListCoursesInput struct {
	IncludeInactive bool `json:"includeInactive,omitempty" jsonschema:"also list inactive courses and courses the user can no longer open (default false)"`
}

// Course is one course the user is enrolled in.
type Course struct {
	ID        int64  `json:"id" jsonschema:"course (org unit) ID, used by other tools"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Active    bool   `json:"active" jsonschema:"whether the course is active and the user can open it"`
	StartDate string `json:"startDate,omitempty" jsonschema:"course start, RFC 3339"`
	EndDate   string `json:"endDate,omitempty" jsonschema:"course end, RFC 3339"`
	Role      string `json:"role,omitempty" jsonschema:"the user's own role in the course, e.g. Student"`
	URL       string `json:"url,omitempty" jsonschema:"course home page in Brightspace"`
}

// CourseList is the result of list_courses.
type CourseList struct {
	Courses []Course `json:"courses"`
}

func addListCourses(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_courses",
		Description: "List the Brightspace courses the user is enrolled in, newest first, with each course's ID, name, code, dates and the user's role. " +
			"Use this to find a course ID for other tools or when the user asks about their courses. " +
			"By default only active courses are listed; set includeInactive for closed ones. " +
			"Brightspace often keeps past courses active, so use startDate and endDate to tell which courses are current. " +
			"It does not list other participants.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListCoursesInput) (*mcp.CallToolResult, CourseList, error) {
		c, err := connect()
		if err != nil {
			return nil, CourseList{}, sessionError(err, "")
		}
		enrollments, err := c.MyCourses(ctx)
		if err != nil {
			return nil, CourseList{}, sessionError(err, c.BaseURL())
		}

		list := CourseList{Courses: []Course{}}
		for _, e := range enrollments {
			active := e.Access.IsActive && e.Access.CanAccess
			if !active && !in.IncludeInactive {
				continue
			}
			course := Course{
				ID:        e.OrgUnit.ID,
				Name:      e.OrgUnit.Name,
				Code:      e.OrgUnit.Code,
				Active:    active,
				StartDate: formatTime(e.Access.StartDate),
				EndDate:   formatTime(e.Access.EndDate),
				Role:      e.Access.RoleName,
			}
			if e.OrgUnit.HomeURL != "" {
				course.URL = resolveURL(c.BaseURL(), e.OrgUnit.HomeURL)
			}
			list.Courses = append(list.Courses, course)
		}
		// Newest first; courses without a start date go last. RFC 3339 in
		// UTC sorts correctly as text.
		slices.SortStableFunc(list.Courses, func(a, b Course) int {
			if (a.StartDate == "") != (b.StartDate == "") {
				if a.StartDate == "" {
					return 1
				}
				return -1
			}
			return cmp.Compare(b.StartDate, a.StartDate)
		})
		return nil, list, nil
	})
}

// resolveURL resolves ref against base. Brightspace returns some links as
// full URLs and others as paths on the instance.
func resolveURL(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ""
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return b.ResolveReference(r).String()
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
