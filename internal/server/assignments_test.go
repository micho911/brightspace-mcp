package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const foldersJSON = `[
	{"Id":10,"Name":"Essay","DueDate":"2026-10-05T10:00:00.000Z","IsHidden":false,"GroupTypeId":null,
	 "Availability":{"StartDate":"2026-09-01T00:00:00.000Z","EndDate":null},"Assessment":{"ScoreDenominator":100}},
	{"Id":11,"Name":"No date","DueDate":null,"IsHidden":false,"Availability":{"StartDate":null,"EndDate":null},"Assessment":{"ScoreDenominator":null}},
	{"Id":12,"Name":"Hidden","DueDate":"2026-10-01T10:00:00.000Z","IsHidden":true,"Availability":{}},
	{"Id":13,"Name":"Done","DueDate":"2026-09-20T10:00:00.000Z","IsHidden":false,"GroupTypeId":4,"Availability":{},"Assessment":{"ScoreDenominator":50}},
	{"Id":14,"Name":"Broken","DueDate":"2026-10-20T10:00:00.000Z","IsHidden":false,"Availability":{}}]`

func fakeDropbox(t *testing.T) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/1001/dropbox/folders/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(foldersJSON))
		case strings.HasSuffix(path, "/folders/10/submissions/mysubmissions/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"Entity":{"EntityId":1},"Status":2,"Submissions":[]}]`))
		case strings.HasSuffix(path, "/folders/13/submissions/mysubmissions/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"Entity":{"EntityId":9},"Status":3,"Feedback":{"Score":41.5,"IsGraded":true},
				"Submissions":[{"Id":1,"SubmissionDate":"2026-09-18T08:00:00.000Z"},{"Id":2,"SubmissionDate":"2026-09-19T08:00:00.000Z"}]}]`))
		case strings.HasSuffix(path, "/folders/14/submissions/mysubmissions/"):
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusForbidden)
		case strings.Contains(path, "/mysubmissions/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callListAssignments(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_assignments", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func assignments(t *testing.T, res *mcp.CallToolResult) AssignmentList {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var list AssignmentList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return list
}

func TestListAssignments(t *testing.T) {
	list := assignments(t, callListAssignments(t, fakeDropbox(t), map[string]any{"courseId": 1001}))

	var names []string
	for _, a := range list.Assignments {
		names = append(names, a.Name)
	}
	if got, want := strings.Join(names, ","), "Done,Essay,Broken,No date"; got != want {
		t.Fatalf("assignments = %s, want %s (hidden dropped, soonest due first, undated last)", got, want)
	}
	done, essay, broken, undated := list.Assignments[0], list.Assignments[1], list.Assignments[2], list.Assignments[3]
	if done.Status != "feedback published" || done.Score == nil || *done.Score != 41.5 || done.Submissions != 2 ||
		done.SubmittedAt != "2026-09-19T08:00:00Z" || !done.Group || *done.MaxPoints != 50 {
		t.Errorf("done = %+v", done)
	}
	if essay.Status != "draft" || essay.AvailableFrom != "2026-09-01T00:00:00Z" || !strings.Contains(essay.URL, "db=10") || !strings.Contains(essay.URL, "ou=1001") {
		t.Errorf("essay = %+v", essay)
	}
	if broken.Status != "unknown" {
		t.Errorf("broken status = %q, want unknown when the status call is refused", broken.Status)
	}
	if undated.Status != "not submitted" {
		t.Errorf("undated status = %q", undated.Status)
	}
}

func TestListAssignmentsOnlyPending(t *testing.T) {
	list := assignments(t, callListAssignments(t, fakeDropbox(t), map[string]any{"courseId": 1001, "onlyPending": true}))

	var names []string
	for _, a := range list.Assignments {
		names = append(names, a.Name)
	}
	if got, want := strings.Join(names, ","), "Essay,No date"; got != want {
		t.Errorf("pending = %s, want %s (draft and not submitted)", got, want)
	}
}

func TestListAssignmentsErrors(t *testing.T) {
	if msg := errorText(t, callListAssignments(t, fakeDropbox(t), map[string]any{"courseId": 0})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callListAssignments(t, fakeDropbox(t), map[string]any{"courseId": 42})); !strings.Contains(msg, "course 42 not found") {
		t.Errorf("unknown course: error %q", msg)
	}
}
