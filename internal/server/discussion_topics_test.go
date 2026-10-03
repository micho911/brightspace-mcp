package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func fakeDiscussions(t *testing.T) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/1001/discussions/forums/"):
			_, _ = w.Write([]byte(`[
				{"ForumId":1,"Name":"General","IsHidden":false,"IsLocked":false},
				{"ForumId":2,"Name":"Staff only","IsHidden":true},
				{"ForumId":3,"Name":"Closed forum","IsHidden":false,"IsLocked":true},
				{"ForumId":4,"Name":"Broken","IsHidden":false}]`))
		case strings.HasSuffix(r.URL.Path, "/forums/1/topics/"):
			_, _ = w.Write([]byte(`[
				{"ForumId":1,"TopicId":7,"Name":"Introduce yourself","Description":{"Text":"","Html":"<p>Say <b>hi</b></p>"},"IsLocked":false,
				 "MustPostToParticipate":true,"DueDate":"2026-10-05T10:00:00.000Z"},
				{"ForumId":1,"TopicId":8,"Name":"Secret","IsHidden":true},
				{"ForumId":1,"TopicId":9,"Name":"Plain","Description":"just text","IsLocked":true}]`))
		case strings.HasSuffix(r.URL.Path, "/forums/3/topics/"):
			_, _ = w.Write([]byte(`[{"ForumId":3,"TopicId":30,"Name":"Old topic"}]`))
		case strings.HasSuffix(r.URL.Path, "/forums/4/topics/"):
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusForbidden)
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callListDiscussionTopics(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_discussion_topics", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func TestListDiscussionTopics(t *testing.T) {
	res := callListDiscussionTopics(t, fakeDiscussions(t), map[string]any{"courseId": 1001})
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var list DiscussionTopicList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var names []string
	for _, tp := range list.Topics {
		names = append(names, tp.Forum+"/"+tp.Name)
	}
	if got, want := strings.Join(names, ","), "General/Introduce yourself,General/Plain,Closed forum/Old topic"; got != want {
		t.Fatalf("topics = %s, want %s (hidden forum and topic dropped, unreadable forum skipped)", got, want)
	}
	intro, plain, old := list.Topics[0], list.Topics[1], list.Topics[2]
	if intro.ForumID != 1 || intro.TopicID != 7 || intro.Description != "Say hi" || !intro.MustPost || intro.Due != "2026-10-05T10:00:00Z" || intro.Locked ||
		!strings.HasSuffix(intro.URL, "/d2l/le/1001/discussions/topics/7/View") {
		t.Errorf("intro = %+v", intro)
	}
	if !plain.Locked || plain.Description != "just text" {
		t.Errorf("plain = %+v", plain)
	}
	if !old.Locked {
		t.Errorf("old = %+v, want locked because its forum is", old)
	}
}

func TestListDiscussionTopicsErrors(t *testing.T) {
	bs := fakeDiscussions(t)
	if msg := errorText(t, callListDiscussionTopics(t, bs, map[string]any{"courseId": 0})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callListDiscussionTopics(t, bs, map[string]any{"courseId": 42})); !strings.Contains(msg, "course 42 not found") {
		t.Errorf("unknown course: error %q", msg)
	}
}
