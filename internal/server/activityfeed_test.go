package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

const feedTestHost = "prd.activityfeed.eu-west-1.brightspace.com"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// fakeFeed answers for a Brightspace instance (https://brightspace.test) and
// its Activity Feed host in-process, so the client's real host checks run.
// With expired set, the instance answers like a dead session.
func fakeFeed(expired bool, feed http.HandlerFunc) Connect {
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		switch {
		case r.URL.Host == feedTestHost:
			feed(rec, r)
		case expired:
			rec.Header().Set("Content-Type", "text/html")
			rec.WriteHeader(http.StatusForbidden)
		case r.URL.Path == "/d2l/lp/auth/xsrf-tokens":
			rec.Header().Set("Content-Type", "application/json")
			fmt.Fprint(rec, `{"referrerToken":"csrf"}`)
		case r.URL.Path == "/d2l/lp/auth/oauth2/token":
			rec.Header().Set("Content-Type", "application/json")
			fmt.Fprint(rec, `{"access_token":"token"}`)
		case r.URL.Path == "/d2l/home/1001":
			rec.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(rec, `<d2l-widget api-endpoint="https://%s/"></d2l-widget>`, feedTestHost)
		default:
			rec.Header().Set("Content-Type", "text/html")
			rec.WriteHeader(http.StatusNotFound)
		}
		return rec.Result(), nil
	})
	return func() (*brightspace.Client, error) {
		return brightspace.NewClient("https://brightspace.test", nil).WithTransport(rt), nil
	}
}

func feedPost(uuid, published, content string, comments int, attachment string) string {
	att := "[]"
	if attachment != "" {
		att = fmt.Sprintf(`[{"name":%q}]`, attachment)
	}
	return fmt.Sprintf(`{"id":"https://%s/api/v1/d2l:orgUnit:1001/article/%s","published":%q,`+
		`"object":{"type":"Article","content":%q,"attachment":%s,"replies":{"totalItems":%d}}}`,
		feedTestHost, uuid, published, content, att, comments)
}

func servePosts(posts ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"orderedItems":[%s]}`, strings.Join(posts, ","))
	}
}

func callListActivityFeed(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_activity_feed", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func activityFeed(t *testing.T, res *mcp.CallToolResult) ActivityFeedList {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var list ActivityFeedList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	return list
}

func TestListActivityFeed(t *testing.T) {
	bs := fakeFeed(false, servePosts(
		feedPost("old", "2026-09-01T08:00:00.000Z", "<p>Old</p>", 0, ""),
		feedPost("new", "2026-09-20T08:00:00.000Z", `<p>Exam <a href="https://example.org/exam">here</a></p>`, 3, "plan.pdf"),
		feedPost("mid", "2026-09-10T08:00:00.000Z", "<p>Mid</p>", 0, ""),
	))
	list := activityFeed(t, callListActivityFeed(t, bs, map[string]any{"courseId": 1001}))

	var ids []string
	for _, p := range list.Posts {
		ids = append(ids, p.ID)
	}
	if got, want := strings.Join(ids, ","), "new,mid,old"; got != want {
		t.Fatalf("posts = %s, want %s (newest first)", got, want)
	}
	p := list.Posts[0]
	if p.Text != "Exam here (https://example.org/exam)" || p.Date != "2026-09-20T08:00:00Z" ||
		p.Comments != 3 || len(p.Attachments) != 1 || p.Attachments[0] != "plan.pdf" || p.Type != "Article" {
		t.Errorf("newest post = %+v", p)
	}
	if list.More || list.URL != "https://brightspace.test/d2l/home/1001" {
		t.Errorf("more = %v, url = %q", list.More, list.URL)
	}
}

func TestListActivityFeedLimitAndTruncation(t *testing.T) {
	bs := fakeFeed(false, servePosts(
		feedPost("a", "2026-09-20T08:00:00.000Z", "<p>"+strings.Repeat("x", maxFeedText+100)+"</p>", 0, ""),
		feedPost("b", "2026-09-10T08:00:00.000Z", "<p>b</p>", 0, ""),
	))
	list := activityFeed(t, callListActivityFeed(t, bs, map[string]any{"courseId": 1001, "limit": 1}))
	if len(list.Posts) != 1 || !list.More {
		t.Fatalf("got %d posts, more = %v; want 1 post and more = true", len(list.Posts), list.More)
	}
	if !list.Posts[0].Truncated || len([]rune(list.Posts[0].Text)) > maxFeedText+1 {
		t.Errorf("long post was not cut: truncated = %v, %d runes", list.Posts[0].Truncated, len([]rune(list.Posts[0].Text)))
	}
}

func TestListActivityFeedNeedsACourse(t *testing.T) {
	res := callListActivityFeed(t, fakeFeed(false, servePosts()), map[string]any{})
	if got := errorText(t, res); !strings.Contains(got, "courseId") {
		t.Errorf("error = %q, want it to ask for courseId", got)
	}
}

func TestListActivityFeedWithoutAccess(t *testing.T) {
	bs := fakeFeed(false, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"Permission Denied"}`)
	})
	got := errorText(t, callListActivityFeed(t, bs, map[string]any{"courseId": 1001}))
	if !strings.Contains(got, "no Activity Feed") || strings.Contains(got, "expired") {
		t.Errorf("error = %q, want a no-feed message and not a session message", got)
	}
}

func TestListActivityFeedUnknownCourse(t *testing.T) {
	got := errorText(t, callListActivityFeed(t, fakeFeed(false, servePosts()), map[string]any{"courseId": 999}))
	if !strings.Contains(got, "not found") {
		t.Errorf("error = %q, want a not-found message", got)
	}
}

func TestListActivityFeedExpiredSession(t *testing.T) {
	got := errorText(t, callListActivityFeed(t, fakeFeed(true, servePosts()), map[string]any{"courseId": 1001}))
	if !strings.Contains(got, "expired") || !strings.Contains(got, "brightspace-mcp login https://brightspace.test") {
		t.Errorf("error = %q, want the login instruction", got)
	}
}

func TestAllToolsAreReadOnly(t *testing.T) {
	res, err := connect(t, fakeFeed(false, servePosts())).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) < 5 {
		t.Fatalf("only %d tools listed", len(res.Tools))
	}
	for _, tool := range res.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %s does not set ReadOnlyHint (CLAUDE.md decision 5)", tool.Name)
		}
	}
}
