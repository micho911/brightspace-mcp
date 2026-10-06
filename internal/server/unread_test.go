package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const updatesJSON = `{"Next":null,"Objects":[
	{"OrgUnitId":2,"UserId":42,"UnreadDiscussions":0,"UnreadAssignmentFeedback":0,"UnattemptedQuizzes":0,"UngradedQuizzes":9},
	{"OrgUnitId":3,"UserId":42,"UnreadDiscussions":5,"UnreadAssignmentFeedback":1,"UnattemptedQuizzes":null,"UnreadAssignmentSubmissions":40}]}`

func fakeUpdates(t *testing.T, gotQuery *string) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/enrollments/myenrollments/"):
			_, _ = w.Write([]byte(enrollmentsJSON))
		case strings.HasSuffix(r.URL.Path, "/updates/myUpdates/"):
			*gotQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(updatesJSON))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callGetUnreadCounts(t *testing.T, bs Connect, args map[string]any) UnreadCounts {
	t.Helper()
	now = func() time.Time { return time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = time.Now })
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "get_unread_counts", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var u UnreadCounts
	if err := json.Unmarshal(raw, &u); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return u
}

func TestGetUnreadCounts(t *testing.T) {
	var query string
	u := callGetUnreadCounts(t, fakeUpdates(t, &query), nil)

	// Courses 2 and 3 are active and not ended; 1 has ended and 4 is closed.
	if !strings.Contains(query, "orgUnitIdsCSV=2%2C3") {
		t.Errorf("query = %q, want the active courses 2 and 3", query)
	}
	if len(u.Courses) != 1 {
		t.Fatalf("courses = %+v, want only the course with something waiting", u.Courses)
	}
	c := u.Courses[0]
	if c.CourseID != 3 || c.Course != "New" || c.UnreadDiscussions != 5 || c.UnreadFeedback != 1 || c.UnattemptedQuizzes != 0 || c.Total != 6 {
		t.Errorf("course = %+v (teacher-side counts must be ignored)", c)
	}
}

func TestGetUnreadCountsOneCourse(t *testing.T) {
	var query string
	callGetUnreadCounts(t, fakeUpdates(t, &query), map[string]any{"courseId": 3})

	if !strings.HasPrefix(query, "orgUnitIdsCSV=3") || strings.Contains(query, "%2C") {
		t.Errorf("query = %q, want course 3 only", query)
	}
}
