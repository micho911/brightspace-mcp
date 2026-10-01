package brightspace

import (
	"context"
	"strconv"
	"time"
)

// Forum is a discussion forum: a group of discussion topics.
type Forum struct {
	ID          int64      `json:"ForumId"`
	Name        string     `json:"Name"`
	Description FlexText   `json:"Description"`
	IsHidden    bool       `json:"IsHidden"`
	IsLocked    bool       `json:"IsLocked"`
	StartDate   *time.Time `json:"StartDate"`
	EndDate     *time.Time `json:"EndDate"`
}

// DiscussionTopic is one discussion topic in a forum.
type DiscussionTopic struct {
	ForumID               int64      `json:"ForumId"`
	ID                    int64      `json:"TopicId"`
	Name                  string     `json:"Name"`
	Description           FlexText   `json:"Description"`
	IsHidden              bool       `json:"IsHidden"`
	IsLocked              bool       `json:"IsLocked"`
	MustPostToParticipate bool       `json:"MustPostToParticipate"`
	StartDate             *time.Time `json:"StartDate"`
	EndDate               *time.Time `json:"EndDate"`
	DueDate               *time.Time `json:"DueDate"`
}

func discussionsPath(orgUnitID int64) string {
	return "/d2l/api/le/" + leVersion + "/" + strconv.FormatInt(orgUnitID, 10) + "/discussions/forums/"
}

// Forums returns the discussion forums in a course.
func (c *Client) Forums(ctx context.Context, orgUnitID int64) ([]Forum, error) {
	var forums []Forum
	err := c.get(ctx, discussionsPath(orgUnitID), &forums)
	return forums, err
}

// ForumTopics returns the discussion topics in a forum.
func (c *Client) ForumTopics(ctx context.Context, orgUnitID, forumID int64) ([]DiscussionTopic, error) {
	var topics []DiscussionTopic
	err := c.get(ctx, discussionsPath(orgUnitID)+strconv.FormatInt(forumID, 10)+"/topics/", &topics)
	return topics, err
}
