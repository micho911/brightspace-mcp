package server

import (
	"cmp"
	"context"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetUnreadCountsInput selects the courses to check.
type GetUnreadCountsInput struct {
	CourseID int64 `json:"courseId,omitempty" jsonschema:"only this course (ID from list_courses); default is all active courses"`
}

// UnreadCourse is what is waiting for the user in one course.
type UnreadCourse struct {
	CourseID           int64  `json:"courseId"`
	Course             string `json:"course,omitempty" jsonschema:"course name"`
	UnreadDiscussions  int    `json:"unreadDiscussions" jsonschema:"discussion posts the user has not read"`
	UnreadFeedback     int    `json:"unreadAssignmentFeedback" jsonschema:"assignment feedback the user has not read"`
	UnattemptedQuizzes int    `json:"unattemptedQuizzes" jsonschema:"quizzes the user has not tried"`
	Total              int    `json:"total"`
}

// UnreadCounts is the result of get_unread_counts.
type UnreadCounts struct {
	Courses []UnreadCourse `json:"courses" jsonschema:"only courses with something waiting, most first"`
}

func addGetUnreadCounts(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_unread_counts",
		Description: "Count what is waiting for the user in Brightspace, per course: unread discussion posts, unread assignment feedback and quizzes not yet attempted. " +
			"Use this when the user asks what they have missed, what needs their attention or whether anything is new; it checks all active courses in one call. " +
			"Only courses with something waiting are listed. It gives counts only: use read_discussion_posts, get_assignment or list_quizzes to see the items.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetUnreadCountsInput) (*mcp.CallToolResult, UnreadCounts, error) {
		c, err := connect()
		if err != nil {
			return nil, UnreadCounts{}, sessionError(err, "")
		}
		from := now()

		names := map[int64]string{}
		var ids []int64
		if in.CourseID > 0 {
			ids = []int64{in.CourseID}
		} else {
			enrollments, err := c.MyCourses(ctx)
			if err != nil {
				return nil, UnreadCounts{}, sessionError(err, c.BaseURL())
			}
			for _, e := range enrollments {
				ended := e.Access.EndDate != nil && e.Access.EndDate.Before(from)
				if e.Access.IsActive && e.Access.CanAccess && !ended {
					ids = append(ids, e.OrgUnit.ID)
					names[e.OrgUnit.ID] = e.OrgUnit.Name
				}
			}
		}

		out := UnreadCounts{Courses: []UnreadCourse{}}
		if len(ids) == 0 {
			return nil, out, nil
		}
		updates, err := c.MyUpdates(ctx, ids)
		if err != nil {
			return nil, UnreadCounts{}, courseError(err, c.BaseURL(), in.CourseID, "updates")
		}
		for _, u := range updates {
			total := u.UnreadDiscussions + u.UnreadAssignmentFeedback + u.UnattemptedQuizzes
			if total == 0 {
				continue
			}
			out.Courses = append(out.Courses, UnreadCourse{
				CourseID:           u.OrgUnitID,
				Course:             names[u.OrgUnitID],
				UnreadDiscussions:  u.UnreadDiscussions,
				UnreadFeedback:     u.UnreadAssignmentFeedback,
				UnattemptedQuizzes: u.UnattemptedQuizzes,
				Total:              total,
			})
		}
		slices.SortStableFunc(out.Courses, func(a, b UnreadCourse) int { return cmp.Compare(b.Total, a.Total) })
		return nil, out, nil
	})
}
