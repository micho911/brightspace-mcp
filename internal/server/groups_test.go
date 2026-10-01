package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func fakeGroups(t *testing.T, sections string) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/1001/groups/"):
			w.Header().Set("Content-Type", "application/json")
			// A bare array here; a paged answer works too (see the pager tests).
			_, _ = w.Write([]byte(`[{"GroupCategoryId":1,"GroupCategoryName":"Project groups","GroupId":5,"Name":"Group 3",
				"Description":{"Text":"","Html":"<p>Meets <b>Tuesdays</b></p>"},"MemberCount":4,"IsFull":true}]`))
		case strings.HasSuffix(r.URL.Path, "/1001/sections/mysections/") && sections != "":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sections))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callListMyGroups(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_my_groups", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func myGroups(t *testing.T, res *mcp.CallToolResult) MyGroups {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var g MyGroups
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return g
}

func TestListMyGroups(t *testing.T) {
	g := myGroups(t, callListMyGroups(t, fakeGroups(t, `[{"SectionId":2,"Name":"Section B","Code":"B","Description":{"Text":"Thursday","Html":""}}]`), map[string]any{"courseId": 1001}))

	if len(g.Groups) != 1 {
		t.Fatalf("groups = %+v", g.Groups)
	}
	if grp := g.Groups[0]; grp.ID != 5 || grp.Name != "Group 3" || grp.Category != "Project groups" || grp.Members != 4 || !grp.Full || grp.Description != "Meets Tuesdays" {
		t.Errorf("group = %+v", grp)
	}
	if len(g.Sections) != 1 || g.Sections[0].Name != "Section B" || g.Sections[0].Code != "B" || g.Sections[0].Description != "Thursday" {
		t.Errorf("sections = %+v", g.Sections)
	}
}

func TestListMyGroupsWithoutSections(t *testing.T) {
	g := myGroups(t, callListMyGroups(t, fakeGroups(t, ""), map[string]any{"courseId": 1001}))

	if len(g.Groups) != 1 || len(g.Sections) != 0 {
		t.Errorf("result = %+v, want the group and no sections (a missing sections route is not an error)", g)
	}
}

func TestListMyGroupsErrors(t *testing.T) {
	bs := fakeGroups(t, "")
	if msg := errorText(t, callListMyGroups(t, bs, map[string]any{"courseId": 0})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callListMyGroups(t, bs, map[string]any{"courseId": 42})); !strings.Contains(msg, "course 42 not found") {
		t.Errorf("unknown course: error %q", msg)
	}
}
