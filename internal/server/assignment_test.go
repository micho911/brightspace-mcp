package server

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const folderJSON = `{"Id":13,"Name":"Essay","DueDate":"2026-09-20T10:00:00.000Z","IsHidden":false,"SubmissionType":1,
	"CustomInstructions":{"Text":"","Html":"<p>Write 1000 words.</p><p>See <a href=\"https://x.example/guide\">the guide</a>.</p>"},
	"Attachments":[{"FileId":1,"FileName":"brief.pdf","Size":2048}],
	"Availability":{"StartDate":null,"EndDate":null},"Assessment":{"ScoreDenominator":50}}`

const publishedJSON = `[{"Entity":{"EntityId":9,"DisplayName":"Me"},"Status":3,
	"Feedback":{"Score":41.5,"IsGraded":true,"Feedback":{"Text":"","Html":"<p>Good work.</p>"},"Files":[{"FileId":3,"FileName":"marks.pdf","Size":10}]},
	"Submissions":[{"Id":1,"SubmissionDate":"2026-09-19T08:00:00.000Z","Comment":{"Text":"my note","Html":""},"Files":[{"FileId":2,"FileName":"essay.docx","Size":99}]}]}]`

func fakeAssignment(t *testing.T, submissions string) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/1001/dropbox/folders/13"):
			_, _ = w.Write([]byte(folderJSON))
		case strings.HasSuffix(r.URL.Path, "/1001/dropbox/folders/13/submissions/mysubmissions/"):
			if submissions == "forbidden" {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(submissions))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callGetAssignment(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "get_assignment", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func assignmentDetail(t *testing.T, res *mcp.CallToolResult) AssignmentDetail {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var d AssignmentDetail
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return d
}

func TestGetAssignment(t *testing.T) {
	d := assignmentDetail(t, callGetAssignment(t, fakeAssignment(t, publishedJSON), map[string]any{"courseId": 1001, "assignmentId": 13}))

	if d.Instructions != "Write 1000 words.\nSee the guide (https://x.example/guide)." || d.SubmissionType != "text" ||
		!slices.Equal(d.Attachments, []string{"brief.pdf"}) {
		t.Errorf("folder part = %+v", d)
	}
	if d.Assignment.Status != "feedback published" || d.Assignment.SubmittedAt != "2026-09-19T08:00:00Z" {
		t.Errorf("assignment = %+v", d.Assignment)
	}
	if len(d.MySubmissions) != 1 || d.MySubmissions[0].Comment != "my note" || !slices.Equal(d.MySubmissions[0].Files, []string{"essay.docx"}) {
		t.Errorf("submissions = %+v", d.MySubmissions)
	}
	fb := d.Feedback
	if fb == nil || fb.Score == nil || *fb.Score != 41.5 || *fb.OutOf != 50 || fb.Text != "Good work." || !slices.Equal(fb.Files, []string{"marks.pdf"}) {
		t.Errorf("feedback = %+v", fb)
	}
}

func TestGetAssignmentUnpublishedFeedbackIsHidden(t *testing.T) {
	// A draft grade the teacher has not published must not show up.
	draft := `[{"Entity":{"EntityId":9},"Status":1,"Feedback":{"Score":12,"IsGraded":true,"Feedback":{"Text":"draft"}},"Submissions":[{"Id":1}]}]`
	d := assignmentDetail(t, callGetAssignment(t, fakeAssignment(t, draft), map[string]any{"courseId": 1001, "assignmentId": 13}))

	if d.Feedback != nil || d.Assignment.Status != "submitted" || d.Assignment.Score != nil {
		t.Errorf("detail = %+v, want submitted with no feedback", d)
	}
}

func TestGetAssignmentStatusRefused(t *testing.T) {
	d := assignmentDetail(t, callGetAssignment(t, fakeAssignment(t, "forbidden"), map[string]any{"courseId": 1001, "assignmentId": 13}))

	if d.Assignment.Status != "unknown" || d.Instructions == "" {
		t.Errorf("detail = %+v, want the folder with unknown status", d)
	}
}

func TestGetAssignmentErrors(t *testing.T) {
	bs := fakeAssignment(t, publishedJSON)
	if msg := errorText(t, callGetAssignment(t, bs, map[string]any{"courseId": 0, "assignmentId": 13})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callGetAssignment(t, bs, map[string]any{"courseId": 1001, "assignmentId": 0})); !strings.Contains(msg, "assignmentId is required") {
		t.Errorf("missing assignmentId: error %q", msg)
	}
	if msg := errorText(t, callGetAssignment(t, bs, map[string]any{"courseId": 1001, "assignmentId": 99})); !strings.Contains(msg, "assignment 99 not found") {
		t.Errorf("unknown assignment: error %q", msg)
	}
}
