package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const tocJSON = `{"Modules":[
	{"ModuleId":1,"Title":"Week 1","IsHidden":false,"IsLocked":false,
	 "Topics":[
		{"TopicId":11,"Title":"Slides","ActivityType":1,"Url":"/content/enforced/1/slides.pdf","IsHidden":false},
		{"TopicId":12,"Title":"Reading","ActivityType":2,"Url":"https://reading.example/x","IsHidden":false},
		{"TopicId":13,"Title":"Essay","ActivityType":3,"ToolItemId":77,"DueDate":"2026-10-05T10:00:00.000Z","IsHidden":false},
		{"TopicId":14,"Title":"Secret","ActivityType":1,"IsHidden":true}],
	 "Modules":[
		{"ModuleId":2,"Title":"Extras","IsHidden":false,"IsLocked":true,
		 "Topics":[{"TopicId":21,"Title":"Bonus","ActivityType":4,"ToolItemId":"5","IsHidden":false}],"Modules":[]},
		{"ModuleId":3,"Title":"Old","IsHidden":true,"Topics":[],"Modules":[]}]},
	{"ModuleId":4,"Title":"Week 2","IsHidden":false,"Topics":[],"Modules":[]}]}`

func fakeTOC(t *testing.T, body string) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/1001/content/toc") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
}

func callGetCourseContent(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "get_course_content", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func courseContent(t *testing.T, res *mcp.CallToolResult) CourseContent {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var cc CourseContent
	if err := json.Unmarshal(raw, &cc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return cc
}

func titles(cc CourseContent) string {
	var out []string
	for _, n := range cc.Items {
		out = append(out, fmt.Sprintf("%s%s", strings.Repeat(">", n.Depth), n.Title))
	}
	return strings.Join(out, "|")
}

func TestGetCourseContent(t *testing.T) {
	cc := courseContent(t, callGetCourseContent(t, fakeTOC(t, tocJSON), map[string]any{"courseId": 1001, "depth": 3}))

	if got, want := titles(cc), "Week 1|>Slides|>Reading|>Essay|>Extras|>>Bonus|Week 2"; got != want {
		t.Fatalf("tree = %s, want %s (hidden items and modules dropped)", got, want)
	}
	byTitle := map[string]ContentNode{}
	for _, n := range cc.Items {
		byTitle[n.Title] = n
	}
	if n := byTitle["Slides"]; n.Kind != "file" || n.URL != "" || n.ParentID != 1 {
		t.Errorf("slides = %+v", n)
	}
	if n := byTitle["Reading"]; n.Kind != "link" || n.URL != "https://reading.example/x" {
		t.Errorf("reading = %+v", n)
	}
	if n := byTitle["Essay"]; n.Kind != "assignment" || n.LinkedID != 77 || n.Due != "2026-10-05T10:00:00Z" {
		t.Errorf("essay = %+v", n)
	}
	if n := byTitle["Extras"]; !n.Locked || n.ParentID != 1 {
		t.Errorf("extras = %+v", n)
	}
	if n := byTitle["Bonus"]; n.Kind != "quiz" || n.LinkedID != 5 {
		t.Errorf("bonus = %+v, want ToolItemId read from a string too", n)
	}
}

func TestGetCourseContentDepth(t *testing.T) {
	cc := courseContent(t, callGetCourseContent(t, fakeTOC(t, tocJSON), map[string]any{"courseId": 1001}))

	// Depth 2: modules and what is directly in them; Extras' own items are counted.
	if got, want := titles(cc), "Week 1|>Slides|>Reading|>Essay|>Extras|Week 2"; got != want {
		t.Fatalf("tree = %s, want %s", got, want)
	}
	for _, n := range cc.Items {
		if n.Title == "Extras" && n.Hidden != 1 {
			t.Errorf("extras itemsNotShown = %d, want 1", n.Hidden)
		}
	}
}

func TestGetCourseContentOneModule(t *testing.T) {
	cc := courseContent(t, callGetCourseContent(t, fakeTOC(t, tocJSON), map[string]any{"courseId": 1001, "moduleId": 2}))

	if got, want := titles(cc), "Extras|>Bonus"; got != want {
		t.Errorf("tree = %s, want %s", got, want)
	}
}

func TestGetCourseContentLimit(t *testing.T) {
	var topics []string
	for i := range 450 {
		topics = append(topics, fmt.Sprintf(`{"TopicId":%d,"Title":"T%d","ActivityType":1}`, i+100, i))
	}
	big := `{"Modules":[{"ModuleId":1,"Title":"Big","Topics":[` + strings.Join(topics, ",") + `],"Modules":[]}]}`
	cc := courseContent(t, callGetCourseContent(t, fakeTOC(t, big), map[string]any{"courseId": 1001}))

	if len(cc.Items) != maxContentNodes || !cc.Truncated {
		t.Errorf("items = %d, truncated = %v, want %d and true", len(cc.Items), cc.Truncated, maxContentNodes)
	}
}

func TestGetCourseContentErrors(t *testing.T) {
	bs := fakeTOC(t, tocJSON)
	if msg := errorText(t, callGetCourseContent(t, bs, map[string]any{"courseId": 0})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callGetCourseContent(t, bs, map[string]any{"courseId": 42})); !strings.Contains(msg, "course 42 not found") {
		t.Errorf("unknown course: error %q", msg)
	}
	if msg := errorText(t, callGetCourseContent(t, bs, map[string]any{"courseId": 1001, "moduleId": 999})); !strings.Contains(msg, "module 999 not found") {
		t.Errorf("unknown module: error %q", msg)
	}
}
