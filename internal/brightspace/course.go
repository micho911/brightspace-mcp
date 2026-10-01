package brightspace

import (
	"context"
	"strconv"
	"time"
)

// CourseOffering is a course's own details.
type CourseOffering struct {
	ID          FlexText   `json:"Identifier"`
	Name        string     `json:"Name"`
	Code        string     `json:"Code"`
	IsActive    bool       `json:"IsActive"`
	Path        string     `json:"Path"`
	StartDate   *time.Time `json:"StartDate"`
	EndDate     *time.Time `json:"EndDate"`
	Description FlexText   `json:"Description"`
	Semester    *struct {
		Name string `json:"Name"`
	} `json:"Semester"`
	Department *struct {
		Name string `json:"Name"`
	} `json:"Department"`
	CourseTemplate *struct {
		Name string `json:"Name"`
	} `json:"CourseTemplate"`
}

// CourseInfo returns a course's details.
func (c *Client) CourseInfo(ctx context.Context, orgUnitID int64) (CourseOffering, error) {
	var course CourseOffering
	err := c.get(ctx, "/d2l/api/lp/"+lpVersion+"/courses/"+strconv.FormatInt(orgUnitID, 10), &course)
	return course, err
}
