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

const enrollmentsJSON = `{"PagingInfo":{"Bookmark":null,"HasMoreItems":false},"Items":[
	{"OrgUnit":{"Id":1,"Name":"Old","Code":"OLD","HomeUrl":"/d2l/home/1"},
	 "Access":{"IsActive":true,"CanAccess":true,"StartDate":"2025-02-01T00:00:00.000Z","EndDate":"2025-06-30T00:00:00.000Z","ClasslistRoleName":"Student"}},
	{"OrgUnit":{"Id":2,"Name":"Undated","Code":"UND","HomeUrl":"/d2l/home/2"},
	 "Access":{"IsActive":true,"CanAccess":true,"StartDate":null,"EndDate":null,"ClasslistRoleName":"Student"}},
	{"OrgUnit":{"Id":3,"Name":"New","Code":"NEW","HomeUrl":"/d2l/home/3"},
	 "Access":{"IsActive":true,"CanAccess":true,"StartDate":"2026-09-01T00:00:00.000Z","EndDate":null,"ClasslistRoleName":"Student"}},
	{"OrgUnit":{"Id":4,"Name":"Closed","Code":"CLO","HomeUrl":"/d2l/home/4"},
	 "Access":{"IsActive":false,"CanAccess":false,"StartDate":"2024-09-01T00:00:00.000Z","EndDate":null,"ClasslistRoleName":"Student"}}]}`

func callListCourses(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_courses", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func courseIDs(t *testing.T, res *mcp.CallToolResult) ([]int64, CourseList) {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var list CourseList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	var ids []int64
	for _, c := range list.Courses {
		ids = append(ids, c.ID)
	}
	return ids, list
}

func enrollments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(enrollmentsJSON))
}

func TestListCourses(t *testing.T) {
	ids, list := courseIDs(t, callListCourses(t, fakeBrightspace(t, enrollments), nil))

	if want := []int64{3, 1, 2}; !slices.Equal(ids, want) {
		t.Fatalf("course IDs = %v, want %v (active only, newest first, undated last)", ids, want)
	}
	c := list.Courses[0]
	if c.Name != "New" || c.Code != "NEW" || !c.Active || c.Role != "Student" ||
		c.StartDate != "2026-09-01T00:00:00Z" || c.EndDate != "" ||
		!strings.HasPrefix(c.URL, "http://127.0.0.1") || !strings.HasSuffix(c.URL, "/d2l/home/3") {
		t.Errorf("course = %+v", c)
	}
}

func TestListCoursesIncludeInactive(t *testing.T) {
	ids, list := courseIDs(t, callListCourses(t, fakeBrightspace(t, enrollments), map[string]any{"includeInactive": true}))

	if want := []int64{3, 1, 4, 2}; !slices.Equal(ids, want) {
		t.Fatalf("course IDs = %v, want %v", ids, want)
	}
	if list.Courses[2].Active {
		t.Errorf("closed course reported as active: %+v", list.Courses[2])
	}
}

func TestListCoursesSessionExpired(t *testing.T) {
	bs := fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/d2l/login", http.StatusFound)
	})

	if msg := errorText(t, callListCourses(t, bs, nil)); !strings.Contains(msg, "brightspace-mcp login http://127.0.0.1") {
		t.Errorf("error %q does not tell the user to log in again", msg)
	}
}
