package server

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

// ListMyGroupsInput selects a course.
type ListMyGroupsInput struct {
	CourseID int64 `json:"courseId" jsonschema:"course ID from list_courses"`
}

// GroupInfo is a group the user belongs to.
type GroupInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category,omitempty" jsonschema:"the group category, e.g. Project groups"`
	Description string `json:"description,omitempty" jsonschema:"plain text"`
	Members     int    `json:"members" jsonschema:"how many people are in the group; who they are is not shared"`
	Full        bool   `json:"full,omitempty"`
}

// SectionInfo is a section the user is in.
type SectionInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty" jsonschema:"plain text"`
}

// MyGroups is the result of list_my_groups.
type MyGroups struct {
	Groups   []GroupInfo   `json:"groups"`
	Sections []SectionInfo `json:"sections"`
}

func addListMyGroups(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_my_groups",
		Description: "List the groups and sections the user belongs to in one Brightspace course, with each group's category and member count. " +
			"Use this when the user asks which group or section they are in; get the course ID from list_courses. " +
			"To protect other students' privacy it never says who the other members are. It does not join or leave a group.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListMyGroupsInput) (*mcp.CallToolResult, MyGroups, error) {
		if in.CourseID <= 0 {
			return nil, MyGroups{}, errCourseRequired
		}
		c, err := connect()
		if err != nil {
			return nil, MyGroups{}, sessionError(err, "")
		}
		groups, err := c.MyGroups(ctx, in.CourseID)
		if err != nil {
			return nil, MyGroups{}, courseError(err, c.BaseURL(), in.CourseID, "groups")
		}
		out := MyGroups{Groups: []GroupInfo{}, Sections: []SectionInfo{}}
		for _, g := range groups {
			info := GroupInfo{ID: g.ID, Name: g.Name, Category: g.CategoryName, Members: g.MemberCount, Full: g.IsFull}
			info.Description, _ = truncate(htmlToText(string(g.Description)), maxTopicBlurb)
			out.Groups = append(out.Groups, info)
		}

		// Sections are optional: a course without them, or one that hides
		// them, still has useful groups. Only a lost session is an error.
		sections, err := c.MySections(ctx, in.CourseID)
		if errors.Is(err, brightspace.ErrSessionExpired) {
			return nil, MyGroups{}, sessionError(err, c.BaseURL())
		}
		if err == nil {
			for _, sec := range sections {
				info := SectionInfo{ID: sec.ID, Name: sec.Name, Code: sec.Code}
				info.Description, _ = truncate(htmlToText(string(sec.Description)), maxTopicBlurb)
				out.Sections = append(out.Sections, info)
			}
		}
		return nil, out, nil
	})
}
