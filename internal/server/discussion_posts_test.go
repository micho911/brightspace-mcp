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

// The display names are in the fake answer on purpose: they must never come out.
const postsJSON = `[
	{"PostId":5,"ThreadId":1,"ParentPostId":3,"PostingUserId":42,"PostingUserDisplayName":"Ada Lovelace","Subject":"Re: Hello",
	 "Message":{"Text":"","Html":"<p>My <b>reply</b></p>"},"DatePosted":"2026-10-03T10:00:00.000Z","IsAnonymous":false,"IsDeleted":false},
	{"PostId":4,"ThreadId":1,"ParentPostId":3,"PostingUserId":99,"PostingUserDisplayName":"Grace Hopper","Subject":"Re: Hello",
	 "Message":{"Text":"Welcome!","Html":""},"DatePosted":"2026-10-02T12:00:00.000Z","IsAnonymous":false,"IsDeleted":false,
	 "Attachments":[{"FileId":1,"FileName":"notes.pdf","Size":5}]},
	{"PostId":6,"ThreadId":2,"ParentPostId":null,"PostingUserId":100,"PostingUserDisplayName":"Alan Turing","Subject":"Question",
	 "Message":"plain string message","DatePosted":"2026-10-02T11:00:00.000Z","IsAnonymous":false,"IsDeleted":false},
	{"PostId":7,"ThreadId":2,"ParentPostId":6,"PostingUserId":99,"PostingUserDisplayName":"Grace Hopper","Subject":"Re: Question",
	 "Message":{"Text":"again me","Html":""},"DatePosted":"2026-10-02T11:30:00.000Z","IsAnonymous":false,"IsDeleted":false},
	{"PostId":8,"ThreadId":2,"ParentPostId":6,"PostingUserId":null,"PostingUserDisplayName":"Hidden","Subject":"",
	 "Message":{"Text":"anon post","Html":""},"DatePosted":"2026-10-02T11:40:00.000Z","IsAnonymous":true,"IsDeleted":false},
	{"PostId":9,"ThreadId":1,"ParentPostId":null,"PostingUserId":99,"PostingUserDisplayName":"Grace Hopper","Subject":"gone",
	 "Message":{"Text":"deleted text","Html":""},"DatePosted":"2026-10-01T11:40:00.000Z","IsAnonymous":false,"IsDeleted":true}]`

func fakePosts(t *testing.T, body string, gotQuery *string) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/whoami"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Identifier":"42","FirstName":"Ada","LastName":"Lovelace","UniqueName":"au1"}`))
		case strings.HasSuffix(r.URL.Path, "/1001/discussions/forums/1/topics/7/posts/"):
			if gotQuery != nil {
				*gotQuery = r.URL.RawQuery
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callReadDiscussionPosts(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "read_discussion_posts", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func discussionPosts(t *testing.T, res *mcp.CallToolResult) (DiscussionPosts, string) {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var p DiscussionPosts
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return p, string(raw)
}

var topicArgs = map[string]any{"courseId": 1001, "forumId": 1, "topicId": 7}

func TestReadDiscussionPosts(t *testing.T) {
	var query string
	res := callReadDiscussionPosts(t, fakePosts(t, postsJSON, &query), topicArgs)
	p, raw := discussionPosts(t, res)

	if !strings.Contains(query, "pageSize=31") {
		t.Errorf("query = %q, want pageSize=limit+1", query)
	}
	var authors []string
	for _, post := range p.Posts {
		authors = append(authors, fmt.Sprintf("%d:%s", post.ID, post.Author))
	}
	want := "5:me,4:Participant A,6:Participant B,7:Participant A,8:Anonymous participant"
	if got := strings.Join(authors, ","); got != want {
		t.Fatalf("authors = %s, want %s (deleted post dropped, labels stable per person, anonymous shared)", got, want)
	}
	for _, name := range []string{"Ada Lovelace", "Grace Hopper", "Alan Turing", "Hidden", "deleted text"} {
		if strings.Contains(raw, name) {
			t.Errorf("result contains %q: other people's names and deleted posts must not come out", name)
		}
	}
	mine, welcome, plain := p.Posts[0], p.Posts[1], p.Posts[2]
	if !mine.Mine || mine.Text != "My reply" || mine.ParentID != 3 || mine.ThreadID != 1 || mine.Subject != "Re: Hello" || mine.Date != "2026-10-03T10:00:00Z" {
		t.Errorf("mine = %+v", mine)
	}
	if welcome.Mine || welcome.Text != "Welcome!" || len(welcome.Attachments) != 1 || welcome.Attachments[0] != "notes.pdf" {
		t.Errorf("welcome = %+v", welcome)
	}
	if plain.Text != "plain string message" || plain.ParentID != 0 {
		t.Errorf("plain = %+v", plain)
	}
	if p.More || !strings.HasSuffix(p.URL, "/d2l/le/1001/discussions/topics/7/View") {
		t.Errorf("posts = %+v", p)
	}
}

func TestReadDiscussionPostsLimitAndBudget(t *testing.T) {
	p, _ := discussionPosts(t, callReadDiscussionPosts(t, fakePosts(t, postsJSON, nil), map[string]any{"courseId": 1001, "forumId": 1, "topicId": 7, "limit": 2}))
	if len(p.Posts) != 2 || !p.More {
		t.Errorf("limit 2: %d posts, more=%v, want 2 and true", len(p.Posts), p.More)
	}

	var posts []string
	long := strings.Repeat("x", 2000)
	for i := range 40 {
		posts = append(posts, fmt.Sprintf(`{"PostId":%d,"ThreadId":1,"PostingUserId":99,"Message":{"Text":%q},"DatePosted":"2026-10-02T11:00:00.000Z"}`, i+1, long))
	}
	p, _ = discussionPosts(t, callReadDiscussionPosts(t, fakePosts(t, "["+strings.Join(posts, ",")+"]", nil), map[string]any{"courseId": 1001, "forumId": 1, "topicId": 7, "limit": 40}))
	if !p.More || len(p.Posts) >= 40 || !p.Posts[0].Truncated {
		t.Errorf("budget: %d posts, more=%v, want fewer than 40, more=true, first post truncated", len(p.Posts), p.More)
	}
}

func TestReadDiscussionPostsErrors(t *testing.T) {
	bs := fakePosts(t, postsJSON, nil)
	if msg := errorText(t, callReadDiscussionPosts(t, bs, map[string]any{"courseId": 0, "forumId": 1, "topicId": 7})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callReadDiscussionPosts(t, bs, map[string]any{"courseId": 1001, "forumId": 0, "topicId": 7})); !strings.Contains(msg, "forumId and topicId are required") {
		t.Errorf("missing forumId: error %q", msg)
	}
	if msg := errorText(t, callReadDiscussionPosts(t, bs, map[string]any{"courseId": 1001, "forumId": 1, "topicId": 8})); !strings.Contains(msg, "discussion topic 8") {
		t.Errorf("unknown topic: error %q", msg)
	}
}

func TestParticipantLabel(t *testing.T) {
	labels := map[string]string{}
	for i := range 27 {
		participantLabel(labels, fmt.Sprint(i), false)
	}
	if got := participantLabel(labels, "26", false); got != "Participant A2" {
		t.Errorf("27th participant = %q, want Participant A2", got)
	}
	if got := participantLabel(labels, "0", false); got != "Participant A" {
		t.Errorf("first participant again = %q, want the same label", got)
	}
}
