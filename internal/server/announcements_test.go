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

const newsJSON = `[
	{"Id":1,"Title":"Old","Body":{"Text":"old text","Html":"<p>old text</p>"},"StartDate":"2026-09-01T08:00:00.000Z","CreatedDate":null,"IsHidden":false,"Attachments":[]},
	{"Id":2,"Title":"Hidden","Body":{"Text":"draft","Html":""},"StartDate":"2026-09-20T08:00:00.000Z","IsHidden":true,"Attachments":[]},
	{"Id":3,"Title":"New","Body":{"Text":"new text","Html":""},"StartDate":null,"CreatedDate":"2026-09-15T08:00:00.000Z","IsHidden":false,
	 "Attachments":[{"FileId":9,"FileName":"slides.pdf","Size":1024}]}]`

func callListAnnouncements(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_announcements", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func announcements(t *testing.T, res *mcp.CallToolResult) AnnouncementList {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var list AnnouncementList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	return list
}

func fakeNews(t *testing.T) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/1001/news/") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(newsJSON))
	})
}

func TestListAnnouncements(t *testing.T) {
	list := announcements(t, callListAnnouncements(t, fakeNews(t), map[string]any{"courseId": 1001}))

	var ids []int64
	for _, a := range list.Announcements {
		ids = append(ids, a.ID)
	}
	if want := []int64{3, 1}; !slices.Equal(ids, want) {
		t.Fatalf("announcement IDs = %v, want %v (hidden dropped, newest first)", ids, want)
	}
	a := list.Announcements[0]
	if a.Title != "New" || a.Text != "new text" || a.Date != "2026-09-15T08:00:00Z" ||
		!slices.Equal(a.Attachments, []string{"slides.pdf"}) || a.Truncated {
		t.Errorf("announcement = %+v", a)
	}
	if list.More || !strings.HasSuffix(list.URL, "/d2l/lms/news/main.d2l?ou=1001") {
		t.Errorf("list = %+v", list)
	}
}

func TestListAnnouncementsLimit(t *testing.T) {
	list := announcements(t, callListAnnouncements(t, fakeNews(t), map[string]any{"courseId": 1001, "limit": 1}))

	if len(list.Announcements) != 1 || list.Announcements[0].ID != 3 || !list.More {
		t.Errorf("list = %+v, want only the newest and more=true", list)
	}
}

func TestListAnnouncementsErrors(t *testing.T) {
	if msg := errorText(t, callListAnnouncements(t, fakeNews(t), map[string]any{"courseId": 0})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callListAnnouncements(t, fakeNews(t), map[string]any{"courseId": 42})); !strings.Contains(msg, "course 42 not found") {
		t.Errorf("unknown course: error %q", msg)
	}
}

func TestTruncate(t *testing.T) {
	if got, cut := truncate("æøå", 3); got != "æøå" || cut {
		t.Errorf("truncate at length = %q, %v", got, cut)
	}
	if got, cut := truncate("æøåx", 3); got != "æøå…" || !cut {
		t.Errorf("truncate = %q, %v", got, cut)
	}
}
