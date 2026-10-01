package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func fakeTopics(t *testing.T) Connect {
	topics := map[string]string{
		"11": `{"Id":11,"Title":"Slides","TopicType":1,"ActivityType":1,"Url":"/content/enforced/1-abc/Week%201%20slides.pdf?x=1",
			"Description":{"Text":"","Html":"<p>Week 1 <b>intro</b></p>"},"IsLocked":false,"LastModifiedDate":"2026-09-02T08:00:00.000Z"}`,
		"12": `{"Id":12,"Title":"Reading","TopicType":3,"ActivityType":2,"Url":"https://reading.example/x","Description":null}`,
		"13": `{"Id":13,"Title":"Essay","TopicType":3,"ActivityType":3,"ToolItemId":77,"DueDate":"2026-10-05T10:00:00.000Z","IsLocked":true,"Url":null}`,
	}
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		body, ok := topics[id]
		if !ok || !strings.Contains(r.URL.Path, "/1001/content/topics/") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
}

func callGetContentTopic(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "get_content_topic", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func contentTopic(t *testing.T, res *mcp.CallToolResult) ContentTopic {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var ct ContentTopic
	if err := json.Unmarshal(raw, &ct); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return ct
}

func TestGetContentTopic(t *testing.T) {
	bs := fakeTopics(t)

	file := contentTopic(t, callGetContentTopic(t, bs, map[string]any{"courseId": 1001, "topicId": 11}))
	if file.Kind != "file" || file.FileName != "Week 1 slides.pdf" || file.Description != "Week 1 intro" ||
		file.Modified != "2026-09-02T08:00:00Z" || !strings.HasSuffix(file.URL, "/d2l/le/content/1001/viewContent/11/View") {
		t.Errorf("file = %+v", file)
	}

	link := contentTopic(t, callGetContentTopic(t, bs, map[string]any{"courseId": 1001, "topicId": 12}))
	if link.Kind != "link" || link.LinkURL != "https://reading.example/x" || link.FileName != "" {
		t.Errorf("link = %+v", link)
	}

	essay := contentTopic(t, callGetContentTopic(t, bs, map[string]any{"courseId": 1001, "topicId": 13}))
	if essay.Kind != "assignment" || essay.LinkedID != 77 || !essay.Locked || essay.Due != "2026-10-05T10:00:00Z" {
		t.Errorf("essay = %+v", essay)
	}
}

func TestGetContentTopicErrors(t *testing.T) {
	bs := fakeTopics(t)
	if msg := errorText(t, callGetContentTopic(t, bs, map[string]any{"courseId": 0, "topicId": 11})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callGetContentTopic(t, bs, map[string]any{"courseId": 1001, "topicId": 0})); !strings.Contains(msg, "topicId is required") {
		t.Errorf("missing topicId: error %q", msg)
	}
	if msg := errorText(t, callGetContentTopic(t, bs, map[string]any{"courseId": 1001, "topicId": 99})); !strings.Contains(msg, "item 99 not found") {
		t.Errorf("unknown item: error %q", msg)
	}
}

func TestFileNameOf(t *testing.T) {
	for in, want := range map[string]string{
		"/content/enforced/1/a%20b.pdf?x=1": "a b.pdf",
		"https://h.example/f/x.txt":         "x.txt",
		"":                                  "",
	} {
		if got := fileNameOf(in); got != want {
			t.Errorf("fileNameOf(%q) = %q, want %q", in, got, want)
		}
	}
}
