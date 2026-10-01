package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

const (
	maxQuizzes          = 50
	maxQuizInstructions = 1500
)

// ListQuizzesInput selects a course's quizzes.
type ListQuizzesInput struct {
	CourseID int64 `json:"courseId" jsonschema:"course ID from list_courses"`
}

// QuizAttemptInfo is one of the user's attempts.
type QuizAttemptInfo struct {
	Number    int      `json:"number"`
	Started   string   `json:"started,omitempty" jsonschema:"RFC 3339"`
	Completed string   `json:"completed,omitempty" jsonschema:"RFC 3339; empty while the attempt is in progress"`
	Score     *float64 `json:"score,omitempty" jsonschema:"only once the teacher has published the attempt"`
	Published bool     `json:"published" jsonschema:"whether the teacher has released the result to the user"`
}

// QuizInfo is one quiz with the user's own attempts.
type QuizInfo struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	Active           bool              `json:"active" jsonschema:"whether the quiz is switched on for students"`
	Start            string            `json:"start,omitempty" jsonschema:"opens, RFC 3339"`
	End              string            `json:"end,omitempty" jsonschema:"closes, RFC 3339"`
	Due              string            `json:"due,omitempty" jsonschema:"RFC 3339"`
	AttemptsAllowed  string            `json:"attemptsAllowed,omitempty" jsonschema:"a number, or unlimited"`
	TimeLimitMinutes int               `json:"timeLimitMinutes,omitempty" jsonschema:"when the time limit is enforced"`
	Instructions     string            `json:"instructions,omitempty" jsonschema:"plain text, shown to students before they start"`
	Status           string            `json:"status" jsonschema:"not attempted, in progress, completed, or unknown when attempts could not be read"`
	Attempts         []QuizAttemptInfo `json:"attempts"`
	URL              string            `json:"url" jsonschema:"the quizzes page in Brightspace"`
}

// QuizList is the result of list_quizzes.
type QuizList struct {
	Quizzes []QuizInfo `json:"quizzes"`
	More    bool       `json:"more,omitempty" jsonschema:"the course has more quizzes than listed"`
}

func addListQuizzes(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_quizzes",
		Description: "List the quizzes in one Brightspace course with open and due dates, attempts allowed, time limit and the user's own attempts (when started, finished, and the score once the teacher has published it). " +
			"Use this when the user asks what quizzes a course has, when one closes, or whether they have taken it; get the course ID from list_courses. " +
			"It shows only the user's own attempts, never questions or answers, and it does not start a quiz.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListQuizzesInput) (*mcp.CallToolResult, QuizList, error) {
		if in.CourseID <= 0 {
			return nil, QuizList{}, errCourseRequired
		}
		c, err := connect()
		if err != nil {
			return nil, QuizList{}, sessionError(err, "")
		}
		quizzes, err := c.Quizzes(ctx, in.CourseID)
		if err != nil {
			return nil, QuizList{}, courseError(err, c.BaseURL(), in.CourseID, "quizzes")
		}
		list := QuizList{Quizzes: []QuizInfo{}}
		if len(quizzes) > maxQuizzes {
			quizzes, list.More = quizzes[:maxQuizzes], true
		}
		if len(quizzes) == 0 {
			return nil, list, nil
		}
		me, err := c.UserID(ctx)
		if err != nil {
			return nil, QuizList{}, sessionError(err, c.BaseURL())
		}

		attempts := make([][]brightspace.QuizAttempt, len(quizzes))
		errs := make([]error, len(quizzes))
		forEachLimit(len(quizzes), func(i int) {
			attempts[i], errs[i] = c.QuizAttempts(ctx, in.CourseID, quizzes[i].ID, me)
		})
		for _, err := range errs {
			if errors.Is(err, brightspace.ErrSessionExpired) {
				return nil, QuizList{}, sessionError(err, c.BaseURL())
			}
		}

		for i, q := range quizzes {
			info := quizInfo(c.BaseURL(), in.CourseID, q)
			if errs[i] == nil {
				info.applyAttempts(attempts[i])
			}
			list.Quizzes = append(list.Quizzes, info)
		}
		// Soonest due first; no due date goes last. RFC 3339 in UTC sorts as text.
		slices.SortStableFunc(list.Quizzes, func(a, b QuizInfo) int {
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

func quizInfo(baseURL string, courseID int64, q brightspace.Quiz) QuizInfo {
	info := QuizInfo{
		ID:       q.ID,
		Name:     q.Name,
		Active:   q.IsActive,
		Start:    formatTime(q.StartDate),
		End:      formatTime(q.EndDate),
		Due:      formatTime(q.DueDate),
		Status:   "unknown",
		Attempts: []QuizAttemptInfo{},
		URL:      fmt.Sprintf("%s/d2l/lms/quizzing/user/quizzes_list.d2l?ou=%d", baseURL, courseID),
	}
	switch {
	case q.AttemptsAllowed.Unlimited:
		info.AttemptsAllowed = "unlimited"
	case q.AttemptsAllowed.Count != nil:
		info.AttemptsAllowed = fmt.Sprint(*q.AttemptsAllowed.Count)
	}
	if q.SubmissionTimeLimit.IsEnforced {
		info.TimeLimitMinutes = q.SubmissionTimeLimit.TimeLimitValue
	}
	if q.Instructions.IsDisplayed {
		info.Instructions, _ = truncate(htmlToText(q.Instructions.Text.String()), maxQuizInstructions)
	}
	return info
}

func (q *QuizInfo) applyAttempts(attempts []brightspace.QuizAttempt) {
	q.Status = "not attempted"
	slices.SortStableFunc(attempts, func(a, b brightspace.QuizAttempt) int { return cmp.Compare(a.Number, b.Number) })
	for _, a := range attempts {
		ai := QuizAttemptInfo{Number: a.Number, Started: formatTime(a.Started), Completed: formatTime(a.Completed), Published: a.IsPublished}
		// A score the teacher has not published stays hidden: the API may not
		// apply the quiz's own rules about when students see results.
		if a.IsPublished {
			ai.Score = a.Score
		}
		q.Attempts = append(q.Attempts, ai)
		if a.Completed == nil {
			q.Status = "in progress"
		} else if q.Status != "in progress" {
			q.Status = "completed"
		}
	}
}
