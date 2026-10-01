package server

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

const maxGradeComment = 1000

// GetGradesInput selects a course.
type GetGradesInput struct {
	CourseID int64 `json:"courseId" jsonschema:"course ID from list_courses"`
}

// GradeEntry is the user's grade on one item.
type GradeEntry struct {
	Name    string   `json:"name"`
	Grade   string   `json:"grade,omitempty" jsonschema:"the grade as Brightspace displays it, e.g. 82 % or Pass or B"`
	Points  *float64 `json:"points,omitempty"`
	OutOf   *float64 `json:"outOf,omitempty"`
	Percent *float64 `json:"percent,omitempty" jsonschema:"points as a percentage of outOf, rounded to one decimal"`
	Comment string   `json:"comment,omitempty" jsonschema:"the teacher's comment on this grade, plain text"`
	Updated string   `json:"updated,omitempty" jsonschema:"when the grade last changed, RFC 3339"`
}

// Grades is the result of get_grades.
type Grades struct {
	Final *GradeEntry  `json:"final,omitempty" jsonschema:"the final grade, when the course shows one"`
	Items []GradeEntry `json:"items" jsonschema:"graded items, in the order Brightspace lists them; ungraded items are left out"`
	URL   string       `json:"url" jsonschema:"the course's grades page in Brightspace"`
}

func addGetGrades(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_grades",
		Description: "Get the user's own grades in one Brightspace course: each graded item with grade, points, percentage and the teacher's comment, plus the final grade when the course shows one. " +
			"Use this when the user asks how they are doing in a course, what they got on something, or what their final grade is; get the course ID from list_courses. " +
			"It shows only the user's own grades, never other students'. Items the teacher has not graded or released are not listed, and weights and grading schemes are not included.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetGradesInput) (*mcp.CallToolResult, Grades, error) {
		if in.CourseID <= 0 {
			return nil, Grades{}, errCourseRequired
		}
		c, err := connect()
		if err != nil {
			return nil, Grades{}, sessionError(err, "")
		}
		values, err := c.MyGradeValues(ctx, in.CourseID)
		if err != nil {
			return nil, Grades{}, courseError(err, c.BaseURL(), in.CourseID, "grades")
		}
		out := Grades{
			Items: []GradeEntry{},
			URL:   fmt.Sprintf("%s/d2l/lms/grades/my_grades/main.d2l?ou=%d", c.BaseURL(), in.CourseID),
		}
		for _, v := range values {
			if e, ok := gradeEntry(v); ok {
				out.Items = append(out.Items, e)
			}
		}

		// No final grade is normal: the course may not use one, or may not have
		// released it yet. Only a lost session is an error here.
		final, err := c.MyFinalGrade(ctx, in.CourseID)
		switch {
		case errors.Is(err, brightspace.ErrSessionExpired):
			return nil, Grades{}, sessionError(err, c.BaseURL())
		case err == nil:
			if e, ok := gradeEntry(final); ok {
				if e.Name == "" {
					e.Name = "Final grade"
				}
				out.Final = &e
			}
		}
		return nil, out, nil
	})
}

// gradeEntry converts a grade value; ok is false for an ungraded item.
func gradeEntry(v brightspace.GradeValue) (GradeEntry, bool) {
	e := GradeEntry{
		Name:    v.Name,
		Grade:   string(v.DisplayedGrade),
		Points:  v.PointsNumerator,
		OutOf:   v.PointsDenominator,
		Updated: formatTime(v.LastModified),
	}
	if e.Grade == "" && e.Points == nil {
		return GradeEntry{}, false
	}
	if e.Points != nil && e.OutOf != nil && *e.OutOf > 0 {
		pct := math.Round(*e.Points / *e.OutOf * 1000) / 10
		e.Percent = &pct
	}
	e.Comment, _ = truncate(htmlToText(string(v.Comments)), maxGradeComment)
	return e, true
}
