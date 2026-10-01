package brightspace

import (
	"context"
	"strconv"
	"time"
)

// TOCModule is a module (a folder of course content) in the table of contents.
type TOCModule struct {
	ID            int64       `json:"ModuleId"`
	Title         string      `json:"Title"`
	StartDateTime *time.Time  `json:"StartDateTime"`
	EndDateTime   *time.Time  `json:"EndDateTime"`
	IsHidden      bool        `json:"IsHidden"`
	IsLocked      bool        `json:"IsLocked"`
	Modules       []TOCModule `json:"Modules"`
	Topics        []TOCTopic  `json:"Topics"`
}

// TOCTopic is one item (a file, link, assignment, quiz, …) in a module.
type TOCTopic struct {
	ID       int64      `json:"TopicId"`
	Title    string     `json:"Title"`
	URL      string     `json:"Url"`
	DueDate  *time.Time `json:"DueDate"`
	IsHidden bool       `json:"IsHidden"`
	IsLocked bool       `json:"IsLocked"`
	IsBroken bool       `json:"IsBroken"`
	// ActivityType says what the item is: 1 file, 2 link, 3 assignment,
	// 4 quiz, 5 discussion forum, 6 discussion topic, and so on.
	ActivityType FlexText `json:"ActivityType"`
	// ToolItemID is the ID of the assignment, quiz or discussion topic the
	// item points to, in that tool.
	ToolItemID FlexText `json:"ToolItemId"`
}

// Topic is one content item in full.
type Topic struct {
	ID             int64      `json:"Id"`
	Title          string     `json:"Title"`
	Description    RichText   `json:"Description"`
	URL            string     `json:"Url"`
	ParentModuleID int64      `json:"ParentModuleId"`
	StartDate      *time.Time `json:"StartDate"`
	EndDate        *time.Time `json:"EndDate"`
	DueDate        *time.Time `json:"DueDate"`
	IsHidden       bool       `json:"IsHidden"`
	IsLocked       bool       `json:"IsLocked"`
	IsBroken       bool       `json:"IsBroken"`
	// TopicType is 1 for a file, 3 for a link, 5 to 8 for SCORM.
	TopicType        FlexText   `json:"TopicType"`
	ActivityType     FlexText   `json:"ActivityType"`
	ToolItemID       FlexText   `json:"ToolItemId"`
	LastModifiedDate *time.Time `json:"LastModifiedDate"`
}

// ContentTopic returns one content item.
func (c *Client) ContentTopic(ctx context.Context, orgUnitID, topicID int64) (Topic, error) {
	var t Topic
	err := c.get(ctx, "/d2l/api/le/"+leVersion+"/"+strconv.FormatInt(orgUnitID, 10)+"/content/topics/"+strconv.FormatInt(topicID, 10), &t)
	return t, err
}

// ContentTOC returns the course's table of contents: modules, sub-modules
// and topics, in course order.
func (c *Client) ContentTOC(ctx context.Context, orgUnitID int64) ([]TOCModule, error) {
	var toc struct {
		Modules []TOCModule `json:"Modules"`
	}
	err := c.get(ctx, "/d2l/api/le/"+leVersion+"/"+strconv.FormatInt(orgUnitID, 10)+"/content/toc", &toc)
	return toc.Modules, err
}
