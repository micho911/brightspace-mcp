package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const quizzesJSON = `{"Next":null,"Objects":[
	{"QuizId":5,"Name":"Quiz 1","IsActive":true,"StartDate":"2026-09-01T00:00:00.000Z","EndDate":null,"DueDate":"2026-10-05T10:00:00.000Z",
	 "AttemptsAllowed":{"IsUnlimited":false,"NumberOfAttemptsAllowed":2},"SubmissionTimeLimit":{"IsEnforced":true,"ShowClock":true,"TimeLimitValue":30},
	 "Instructions":{"Text":{"Text":"","Html":"<p>Read <b>carefully</b></p>"},"IsDisplayed":true}},
	{"QuizId":6,"Name":"Exam","IsActive":false,"DueDate":null,"AttemptsAllowed":{"IsUnlimited":true,"NumberOfAttemptsAllowed":null},
	 "SubmissionTimeLimit":{"IsEnforced":false,"TimeLimitValue":90},"Instructions":{"Text":{"Text":"hidden text"},"IsDisplayed":false}},
	{"QuizId":7,"Name":"Practice","IsActive":true,"DueDate":"2026-09-30T10:00:00.000Z","AttemptsAllowed":3}]}`

func fakeQuizzes(t *testing.T) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(path, "/users/whoami"):
			_, _ = w.Write([]byte(`{"Identifier":"42","FirstName":"Ada","LastName":"L","UniqueName":"au1"}`))
		case strings.HasSuffix(path, "/1001/quizzes/"):
			_, _ = w.Write([]byte(quizzesJSON))
		case strings.HasSuffix(path, "/quizzes/5/attempts/"):
			if got := r.URL.Query().Get("userId"); got != "42" {
				t.Errorf("attempts requested with userId=%q, want the user's own ID", got)
			}
			// Attempt 13 belongs to someone else and must be dropped even if the API sent it.
			_, _ = w.Write([]byte(`{"Next":null,"Objects":[
				{"AttemptId":12,"UserId":42,"AttemptNumber":2,"Score":null,"Started":"2026-10-02T09:00:00.000Z","Completed":null,"IsPublished":false},
				{"AttemptId":11,"UserId":42,"AttemptNumber":1,"Score":8,"Started":"2026-10-01T09:00:00.000Z","Completed":"2026-10-01T09:20:00.000Z","IsPublished":true},
				{"AttemptId":13,"UserId":99,"AttemptNumber":1,"Score":10,"Started":"2026-10-01T09:00:00.000Z","Completed":"2026-10-01T09:10:00.000Z","IsPublished":true}]}`))
		case strings.HasSuffix(path, "/quizzes/6/attempts/"):
			_, _ = w.Write([]byte(`{"Next":null,"Objects":[]}`))
		case strings.HasSuffix(path, "/quizzes/7/attempts/"):
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusForbidden)
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callListQuizzes(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_quizzes", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func quizList(t *testing.T, res *mcp.CallToolResult) QuizList {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var l QuizList
	if err := json.Unmarshal(raw, &l); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return l
}

func TestListQuizzes(t *testing.T) {
	l := quizList(t, callListQuizzes(t, fakeQuizzes(t), map[string]any{"courseId": 1001}))

	var names []string
	for _, q := range l.Quizzes {
		names = append(names, q.Name)
	}
	if got, want := strings.Join(names, ","), "Practice,Quiz 1,Exam"; got != want {
		t.Fatalf("quizzes = %s, want %s (soonest due first, undated last)", got, want)
	}
	practice, quiz, exam := l.Quizzes[0], l.Quizzes[1], l.Quizzes[2]

	if quiz.AttemptsAllowed != "2" || quiz.TimeLimitMinutes != 30 || quiz.Instructions != "Read carefully" || quiz.Status != "in progress" ||
		quiz.Due != "2026-10-05T10:00:00Z" || !quiz.Active {
		t.Errorf("quiz = %+v", quiz)
	}
	if len(quiz.Attempts) != 2 || quiz.Attempts[0].Number != 1 || quiz.Attempts[1].Number != 2 {
		t.Fatalf("attempts = %+v, want only the user's own, in order", quiz.Attempts)
	}
	if first := quiz.Attempts[0]; first.Score == nil || *first.Score != 8 || !first.Published || first.Completed == "" {
		t.Errorf("first attempt = %+v", first)
	}
	if second := quiz.Attempts[1]; second.Completed != "" || second.Score != nil {
		t.Errorf("second attempt = %+v", second)
	}
	if exam.AttemptsAllowed != "unlimited" || exam.TimeLimitMinutes != 0 || exam.Instructions != "" || exam.Status != "not attempted" || exam.Active {
		t.Errorf("exam = %+v", exam)
	}
	if practice.AttemptsAllowed != "3" || practice.Status != "unknown" {
		t.Errorf("practice = %+v, want a bare number accepted and unknown status when attempts are refused", practice)
	}
}

func TestListQuizzesHidesUnpublishedScore(t *testing.T) {
	bs := fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/whoami"):
			_, _ = w.Write([]byte(`{"Identifier":"42"}`))
		case strings.HasSuffix(r.URL.Path, "/quizzes/"):
			_, _ = w.Write([]byte(`{"Objects":[{"QuizId":5,"Name":"Exam"}]}`))
		default:
			_, _ = w.Write([]byte(`{"Objects":[{"AttemptId":1,"UserId":"42","AttemptNumber":1,"Score":9,"Completed":"2026-10-01T09:20:00.000Z","IsPublished":false}]}`))
		}
	})
	l := quizList(t, callListQuizzes(t, bs, map[string]any{"courseId": 1001}))

	if a := l.Quizzes[0].Attempts[0]; a.Score != nil || a.Published || l.Quizzes[0].Status != "completed" {
		t.Errorf("attempt = %+v, status %q: an unpublished score must not be shown", a, l.Quizzes[0].Status)
	}
}

func TestListQuizzesErrors(t *testing.T) {
	bs := fakeQuizzes(t)
	if msg := errorText(t, callListQuizzes(t, bs, map[string]any{"courseId": 0})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callListQuizzes(t, bs, map[string]any{"courseId": 42})); !strings.Contains(msg, "course 42 not found") {
		t.Errorf("unknown course: error %q", msg)
	}
}
