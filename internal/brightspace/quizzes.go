package brightspace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

// AttemptsAllowed is how many attempts a quiz permits. Brightspace sends an
// object; a bare number is accepted too.
type AttemptsAllowed struct {
	Unlimited bool
	Count     *int
}

func (a *AttemptsAllowed) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '{' {
		var o struct {
			IsUnlimited bool `json:"IsUnlimited"`
			Number      *int `json:"NumberOfAttemptsAllowed"`
		}
		if err := json.Unmarshal(b, &o); err != nil {
			return err
		}
		a.Unlimited, a.Count = o.IsUnlimited, o.Number
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return nil // an unknown shape is "not stated", not a failure
	}
	a.Count = &n
	return nil
}

// Quiz is a quiz in a course.
type Quiz struct {
	ID                  int64           `json:"QuizId"`
	Name                string          `json:"Name"`
	IsActive            bool            `json:"IsActive"`
	StartDate           *time.Time      `json:"StartDate"`
	EndDate             *time.Time      `json:"EndDate"`
	DueDate             *time.Time      `json:"DueDate"`
	AttemptsAllowed     AttemptsAllowed `json:"AttemptsAllowed"`
	SubmissionTimeLimit struct {
		IsEnforced     bool `json:"IsEnforced"`
		TimeLimitValue int  `json:"TimeLimitValue"`
	} `json:"SubmissionTimeLimit"`
	Instructions struct {
		Text        RichText `json:"Text"`
		IsDisplayed bool     `json:"IsDisplayed"`
	} `json:"Instructions"`
}

// QuizAttempt is one attempt at a quiz.
type QuizAttempt struct {
	ID          int64      `json:"AttemptId"`
	UserID      FlexText   `json:"UserId"`
	Number      int        `json:"AttemptNumber"`
	Score       *float64   `json:"Score"`
	Started     *time.Time `json:"Started"`
	Completed   *time.Time `json:"Completed"`
	IsPublished bool       `json:"IsPublished"`
}

func quizzesPath(orgUnitID int64) string {
	return "/d2l/api/le/" + leVersion + "/" + strconv.FormatInt(orgUnitID, 10) + "/quizzes/"
}

// Quizzes returns the quizzes in a course that the user may see.
func (c *Client) Quizzes(ctx context.Context, orgUnitID int64) ([]Quiz, error) {
	return getAll[Quiz](ctx, c, quizzesPath(orgUnitID), nil)
}

// QuizAttempts returns the user's attempts at a quiz. It asks for the user's
// own attempts and drops any that belong to someone else.
func (c *Client) QuizAttempts(ctx context.Context, orgUnitID, quizID int64, userID string) ([]QuizAttempt, error) {
	q := url.Values{"userId": {userID}}
	all, err := getAll[QuizAttempt](ctx, c, quizzesPath(orgUnitID)+strconv.FormatInt(quizID, 10)+"/attempts/", q)
	if err != nil {
		return nil, err
	}
	own := all[:0]
	for _, a := range all {
		if string(a.UserID) == userID {
			own = append(own, a)
		}
	}
	return own, nil
}
