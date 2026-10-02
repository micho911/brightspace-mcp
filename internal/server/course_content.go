package server

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

const (
	defaultContentDepth = 2
	maxContentDepth     = 6
	maxContentNodes     = 400
)

// GetCourseContentInput selects part of a course's content tree.
type GetCourseContentInput struct {
	CourseID int64 `json:"courseId" jsonschema:"course ID from list_courses"`
	ModuleID int64 `json:"moduleId,omitempty" jsonschema:"show only this module and what is inside it (a module ID from an earlier result)"`
	Depth    int   `json:"depth,omitempty" jsonschema:"how many levels to show (default 2, at most 6); deeper levels are counted, not listed"`
}

// ContentNode is one module or topic in the content tree.
type ContentNode struct {
	Kind     string `json:"kind" jsonschema:"module, file, link, assignment, quiz, discussion, or another kind of item"`
	ID       int64  `json:"id" jsonschema:"module ID or topic ID"`
	ParentID int64  `json:"parentId,omitempty" jsonschema:"the module this is in; none at the top"`
	Depth    int    `json:"depth" jsonschema:"0 at the top"`
	Title    string `json:"title"`
	Due      string `json:"due,omitempty" jsonschema:"RFC 3339"`
	Locked   bool   `json:"locked,omitempty" jsonschema:"not open to the user yet (dates or conditions)"`
	Broken   bool   `json:"broken,omitempty"`
	URL      string `json:"url,omitempty" jsonschema:"the address, for link items"`
	LinkedID int64  `json:"linkedId,omitempty" jsonschema:"for assignments: the ID for get_assignment; for quizzes and discussions the ID in their own tool"`
	Hidden   int    `json:"itemsNotShown,omitempty" jsonschema:"for a module: items inside that are not listed because of depth; ask again with this moduleId"`
}

// CourseContent is the result of get_course_content.
type CourseContent struct {
	Items     []ContentNode `json:"items" jsonschema:"in course order, each module followed by what is inside it"`
	Truncated bool          `json:"truncated,omitempty" jsonschema:"the tree is larger than the limit; ask for a module with moduleId"`
	URL       string        `json:"url" jsonschema:"the course's content page in Brightspace"`
}

func addGetCourseContent(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_course_content",
		Description: "Show the content tree of one Brightspace course: modules (folders) and the items in them, such as files, links, assignments, quizzes and discussions, in course order with due dates. " +
			"Use this when the user asks what a course contains, where a lecture or reading is, or what is in week N; get the course ID from list_courses. " +
			"Large courses are shown two levels deep: ask again with moduleId to open a module. " +
			"It lists titles only: use get_content_topic for one item's details and read_course_file for a file's text. Hidden items are not shown.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in GetCourseContentInput) (*mcp.CallToolResult, CourseContent, error) {
		if in.CourseID <= 0 {
			return nil, CourseContent{}, errCourseRequired
		}
		depth := clampLimit(in.Depth, defaultContentDepth, maxContentDepth)

		c, err := connect()
		if err != nil {
			return nil, CourseContent{}, sessionError(err, "")
		}
		toc, err := c.ContentTOC(ctx, in.CourseID)
		if err != nil {
			return nil, CourseContent{}, courseError(err, c.BaseURL(), in.CourseID, "content")
		}

		roots := toc
		if in.ModuleID > 0 {
			m, ok := findModule(toc, in.ModuleID)
			if !ok {
				return nil, CourseContent{}, fmt.Errorf("module %d not found in course %d: check the ID in an earlier get_course_content result", in.ModuleID, in.CourseID)
			}
			roots = []brightspace.TOCModule{m}
		}

		b := &contentBuilder{maxDepth: depth, baseURL: c.BaseURL()}
		for _, m := range roots {
			b.addModule(m, 0, 0)
		}
		return nil, CourseContent{
			Items:     append([]ContentNode{}, b.nodes...),
			Truncated: b.truncated,
			URL:       fmt.Sprintf("%s/d2l/le/content/%d/Home", c.BaseURL(), in.CourseID),
		}, nil
	})
}

func findModule(modules []brightspace.TOCModule, id int64) (brightspace.TOCModule, bool) {
	for _, m := range modules {
		if m.ID == id {
			return m, true
		}
		if sub, ok := findModule(m.Modules, id); ok {
			return sub, true
		}
	}
	return brightspace.TOCModule{}, false
}

type contentBuilder struct {
	maxDepth  int
	baseURL   string
	nodes     []ContentNode
	truncated bool
}

// full reports whether the node budget is used up, and notes it if so.
func (b *contentBuilder) full() bool {
	if len(b.nodes) >= maxContentNodes {
		b.truncated = true
	}
	return b.truncated
}

func countItems(m brightspace.TOCModule) int {
	n := 0
	for _, t := range m.Topics {
		if !t.IsHidden {
			n++
		}
	}
	for _, sub := range m.Modules {
		if !sub.IsHidden {
			n += 1 + countItems(sub)
		}
	}
	return n
}

func (b *contentBuilder) addModule(m brightspace.TOCModule, depth int, parent int64) {
	if m.IsHidden || b.full() {
		return
	}
	node := ContentNode{Kind: "module", ID: m.ID, ParentID: parent, Depth: depth, Title: m.Title, Locked: m.IsLocked}
	if depth+1 >= b.maxDepth {
		node.Hidden = countItems(m)
		b.nodes = append(b.nodes, node)
		return
	}
	b.nodes = append(b.nodes, node)
	for _, t := range m.Topics {
		if t.IsHidden || b.full() {
			continue
		}
		b.nodes = append(b.nodes, topicNode(b.baseURL, t, depth+1, m.ID))
	}
	for _, sub := range m.Modules {
		b.addModule(sub, depth+1, m.ID)
	}
}

func topicNode(baseURL string, t brightspace.TOCTopic, depth int, parent int64) ContentNode {
	kind := activityKind(string(t.ActivityType))
	n := ContentNode{Kind: kind, ID: t.ID, ParentID: parent, Depth: depth, Title: t.Title, Due: formatTime(t.DueDate), Locked: t.IsLocked, Broken: t.IsBroken}
	if kind == "link" && t.URL != "" {
		n.URL = resolveURL(baseURL, t.URL)
	}
	if id, err := strconv.ParseInt(strings.TrimSpace(string(t.ToolItemID)), 10, 64); err == nil && id > 0 {
		switch kind {
		case "assignment", "quiz", "discussion":
			n.LinkedID = id
		}
	}
	return n
}

// activityKind names a content item's ActivityType.
func activityKind(t string) string {
	switch t {
	case "1":
		return "file"
	case "2":
		return "link"
	case "3":
		return "assignment"
	case "4":
		return "quiz"
	case "5", "6":
		return "discussion"
	case "7", "27":
		return "external tool"
	case "8":
		return "chat"
	case "9":
		return "schedule"
	case "10":
		return "checklist"
	case "11":
		return "self assessment"
	case "12":
		return "survey"
	case "13":
		return "online room"
	case "14":
		return "course link"
	case "20", "21", "22", "23", "24", "25", "26":
		return "SCORM or learning object"
	case "28":
		return "org unit"
	}
	return "item"
}
