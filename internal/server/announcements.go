package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

const (
	defaultAnnouncements = 10
	maxAnnouncements     = 50
	// maxAnnouncementText caps each announcement's text so one long post
	// cannot fill the assistant's context.
	maxAnnouncementText = 4000
)

// ListAnnouncementsInput selects a course's announcements.
type ListAnnouncementsInput struct {
	CourseID int64 `json:"courseId" jsonschema:"course ID from list_courses"`
	Limit    int   `json:"limit,omitempty" jsonschema:"maximum number of announcements, newest first (default 10, at most 50)"`
}

// Announcement is one news item in a course.
type Announcement struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Date        string   `json:"date,omitempty" jsonschema:"when it was posted, RFC 3339"`
	Text        string   `json:"text" jsonschema:"plain-text body"`
	Truncated   bool     `json:"truncated,omitempty" jsonschema:"the text was cut short; the full post is in Brightspace"`
	Attachments []string `json:"attachments,omitempty" jsonschema:"attached file names"`
}

// AnnouncementList is the result of list_announcements.
type AnnouncementList struct {
	Announcements []Announcement `json:"announcements"`
	More          bool           `json:"more,omitempty" jsonschema:"the course has older announcements than those listed"`
	URL           string         `json:"url" jsonschema:"the course's announcements page in Brightspace"`
}

func addListAnnouncements(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_announcements",
		Description: "List the announcements (news) in one Brightspace course, newest first, with title, date, plain-text body and attachment names. " +
			"Use this when the user asks what is new in a course or what a teacher announced; get the course ID from list_courses. " +
			"It covers one course per call and does not download attachments. " +
			"Many courses post in the course Activity Feed instead, which this tool does not read; if it finds nothing, check the Activity Feed too.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListAnnouncementsInput) (*mcp.CallToolResult, AnnouncementList, error) {
		if in.CourseID <= 0 {
			return nil, AnnouncementList{}, errors.New("courseId is required: get it from list_courses")
		}
		limit := in.Limit
		if limit <= 0 {
			limit = defaultAnnouncements
		}
		limit = min(limit, maxAnnouncements)

		c, err := connect()
		if err != nil {
			return nil, AnnouncementList{}, sessionError(err, "")
		}
		items, err := c.CourseNews(ctx, in.CourseID)
		if errors.Is(err, brightspace.ErrNotFound) {
			return nil, AnnouncementList{}, fmt.Errorf("course %d not found, or the user cannot see its announcements: check the ID with list_courses", in.CourseID)
		}
		if err != nil {
			return nil, AnnouncementList{}, sessionError(err, c.BaseURL())
		}

		list := AnnouncementList{
			Announcements: []Announcement{},
			URL:           fmt.Sprintf("%s/d2l/lms/news/main.d2l?ou=%d", c.BaseURL(), in.CourseID),
		}
		for _, item := range items {
			if item.IsHidden {
				continue
			}
			a := Announcement{ID: item.ID, Title: item.Title, Date: formatTime(cmp.Or(item.StartDate, item.CreatedDate))}
			a.Text, a.Truncated = truncate(item.Body.Text, maxAnnouncementText)
			for _, f := range item.Attachments {
				a.Attachments = append(a.Attachments, f.FileName)
			}
			list.Announcements = append(list.Announcements, a)
		}
		// Newest first; RFC 3339 in UTC sorts correctly as text.
		slices.SortStableFunc(list.Announcements, func(a, b Announcement) int {
			return cmp.Compare(b.Date, a.Date)
		})
		if len(list.Announcements) > limit {
			list.Announcements, list.More = list.Announcements[:limit], true
		}
		return nil, list, nil
	})
}

// truncate cuts s to at most n runes.
func truncate(s string, n int) (string, bool) {
	r := []rune(s)
	if len(r) <= n {
		return s, false
	}
	return string(r[:n]) + "…", true
}
