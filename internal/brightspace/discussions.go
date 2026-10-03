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

// DiscussionPost is a post in a discussion topic. It deliberately has no
// field for the author's name: other people's names are not read at all.
type DiscussionPost struct {
	ID           int64      `json:"PostId"`
	ThreadID     int64      `json:"ThreadId"`
	ParentPostID *int64     `json:"ParentPostId"`
	AuthorID     FlexText   `json:"PostingUserId"`
	Subject      string     `json:"Subject"`
	Message      FlexText   `json:"Message"`
	DatePosted   *time.Time `json:"DatePosted"`
	IsAnonymous  bool       `json:"IsAnonymous"`
	IsDeleted    bool       `json:"IsDeleted"`
	Attachments  []FileRef  `json:"Attachments"`
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

// TopicPosts returns up to pageSize of the newest posts in a discussion
// topic, newest first. The route answers with a bare JSON array.
func (c *Client) TopicPosts(ctx context.Context, orgUnitID, forumID, topicID int64, pageSize int) ([]DiscussionPost, error) {
	path := discussionsPath(orgUnitID) + strconv.FormatInt(forumID, 10) + "/topics/" + strconv.FormatInt(topicID, 10) +
		"/posts/?pageSize=" + strconv.Itoa(pageSize) + "&pageNumber=1"
	var posts []DiscussionPost
	err := c.get(ctx, path, &posts)
	return posts, err
}

// ForumTopics returns the discussion topics in a forum.
func (c *Client) ForumTopics(ctx context.Context, orgUnitID, forumID int64) ([]DiscussionTopic, error) {
	var topics []DiscussionTopic
	err := c.get(ctx, discussionsPath(orgUnitID)+strconv.FormatInt(forumID, 10)+"/topics/", &topics)
	return topics, err
}
