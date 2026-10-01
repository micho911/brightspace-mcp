package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func fakeCourse(t *testing.T) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/courses/1001") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Identifier":"1001","Name":"Algorithms","Code":"CS101","IsActive":true,"Path":"/content/enforced/1001/",
			"StartDate":"2026-09-01T00:00:00.000Z","EndDate":null,"Description":{"Text":"","Html":"<p>Intro to <b>algorithms</b>.</p>"},
			"Semester":{"Identifier":"5","Name":"Autumn 2026","Code":"A26"},"Department":{"Name":"Computer Science"},"CourseTemplate":null}`))
	})
}

func callGetCourseInfo(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "get_course_info", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func TestGetCourseInfo(t *testing.T) {
	res := callGetCourseInfo(t, fakeCourse(t), map[string]any{"courseId": 1001})
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var info CourseInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info.ID != 1001 || info.Name != "Algorithms" || info.Code != "CS101" || !info.Active || info.StartDate != "2026-09-01T00:00:00Z" || info.EndDate != "" ||
		info.Semester != "Autumn 2026" || info.Department != "Computer Science" || info.Template != "" || info.Description != "Intro to algorithms." ||
		!strings.HasSuffix(info.URL, "/d2l/home/1001") {
		t.Errorf("info = %+v", info)
	}
}

func TestGetCourseInfoErrors(t *testing.T) {
	bs := fakeCourse(t)
	if msg := errorText(t, callGetCourseInfo(t, bs, map[string]any{"courseId": 0})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callGetCourseInfo(t, bs, map[string]any{"courseId": 42})); !strings.Contains(msg, "course 42 not found") {
		t.Errorf("unknown course: error %q", msg)
	}
}
