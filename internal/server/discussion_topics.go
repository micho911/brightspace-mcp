package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

const (
	maxForums           = 50
	maxDiscussionTopics = 200
	maxTopicBlurb       = 400
)

// ListDiscussionTopicsInput selects a course's discussions.
type ListDiscussionTopicsInput struct {
	CourseID int64 `json:"courseId" jsonschema:"course ID from list_courses"`
}

// DiscussionTopicInfo is one discussion topic.
type DiscussionTopicInfo struct {
	ForumID     int64  `json:"forumId" jsonschema:"used with topicId by read_discussion_posts"`
	Forum       string `json:"forum" jsonschema:"forum name"`
	TopicID     int64  `json:"topicId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty" jsonschema:"plain text, shortened"`
	Start       string `json:"start,omitempty" jsonschema:"opens, RFC 3339"`
	End         string `json:"end,omitempty" jsonschema:"closes, RFC 3339"`
	Due         string `json:"due,omitempty" jsonschema:"RFC 3339"`
	Locked      bool   `json:"locked,omitempty" jsonschema:"closed for new posts"`
	MustPost    bool   `json:"mustPostToRead,omitempty" jsonschema:"students must post before they can read others' posts"`
	URL         string `json:"url" jsonschema:"the topic in Brightspace"`
}

// DiscussionTopicList is the result of list_discussion_topics.
type DiscussionTopicList struct {
	Topics []DiscussionTopicInfo `json:"topics"`
	More   bool                  `json:"more,omitempty" jsonschema:"the course has more forums or topics than listed"`
}

func addListDiscussionTopics(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_discussion_topics",
		Description: "List the discussion topics in one Brightspace course, grouped by forum, with dates and whether they are locked. " +
			"Use this when the user asks what discussions a course has or where to find one; get the course ID from list_courses. " +
			"It does not show the posts (use read_discussion_posts) and does not post anything. Hidden forums and topics are left out.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListDiscussionTopicsInput) (*mcp.CallToolResult, DiscussionTopicList, error) {
		if in.CourseID <= 0 {
			return nil, DiscussionTopicList{}, errCourseRequired
		}
		c, err := connect()
		if err != nil {
			return nil, DiscussionTopicList{}, sessionError(err, "")
		}
		forums, err := c.Forums(ctx, in.CourseID)
		if err != nil {
			return nil, DiscussionTopicList{}, courseError(err, c.BaseURL(), in.CourseID, "discussions")
		}
		list := DiscussionTopicList{Topics: []DiscussionTopicInfo{}}
		var shown []brightspace.Forum
		for _, f := range forums {
			if !f.IsHidden {
				shown = append(shown, f)
			}
		}
		if len(shown) > maxForums {
			shown, list.More = shown[:maxForums], true
		}

		topics := make([][]brightspace.DiscussionTopic, len(shown))
		errs := make([]error, len(shown))
		forEachLimit(len(shown), func(i int) {
			topics[i], errs[i] = c.ForumTopics(ctx, in.CourseID, shown[i].ID)
		})
		for _, err := range errs {
			if errors.Is(err, brightspace.ErrSessionExpired) {
				return nil, DiscussionTopicList{}, sessionError(err, c.BaseURL())
			}
		}

		for i, f := range shown {
			if errs[i] != nil {
				continue // a forum whose topics cannot be read is skipped, not fatal
			}
			for _, t := range topics[i] {
				if t.IsHidden {
					continue
				}
				if len(list.Topics) >= maxDiscussionTopics {
					list.More = true
					break
				}
				info := DiscussionTopicInfo{
					ForumID:  f.ID,
					Forum:    f.Name,
					TopicID:  t.ID,
					Name:     t.Name,
					Start:    formatTime(t.StartDate),
					End:      formatTime(t.EndDate),
					Due:      formatTime(t.DueDate),
					Locked:   t.IsLocked || f.IsLocked,
					MustPost: t.MustPostToParticipate,
					URL:      fmt.Sprintf("%s/d2l/le/%d/discussions/topics/%d/View", c.BaseURL(), in.CourseID, t.ID),
				}
				info.Description, _ = truncate(htmlToText(string(t.Description)), maxTopicBlurb)
				list.Topics = append(list.Topics, info)
			}
		}
		return nil, list, nil
	})
}
