package brightspace

import (
	"context"
	"strconv"
)

// MyGroup is a group the user belongs to. Brightspace gives a student the
// member count only, never who the members are, and nothing here asks for more.
type MyGroup struct {
	CategoryName string   `json:"GroupCategoryName"`
	ID           int64    `json:"GroupId"`
	Name         string   `json:"Name"`
	Description  FlexText `json:"Description"`
	MemberCount  int      `json:"MemberCount"`
	IsFull       bool     `json:"IsFull"`
}

// MySection is a section of a course the user is enrolled in.
type MySection struct {
	ID          int64    `json:"SectionId"`
	Name        string   `json:"Name"`
	Code        string   `json:"Code"`
	Description FlexText `json:"Description"`
}

// MyGroups returns the groups the user belongs to in a course.
func (c *Client) MyGroups(ctx context.Context, orgUnitID int64) ([]MyGroup, error) {
	return getAll[MyGroup](ctx, c, "/d2l/api/lp/"+lpVersion+"/"+strconv.FormatInt(orgUnitID, 10)+"/groups/", nil)
}

// MySections returns the sections of a course the user is enrolled in.
func (c *Client) MySections(ctx context.Context, orgUnitID int64) ([]MySection, error) {
	return getAll[MySection](ctx, c, "/d2l/api/lp/"+lpVersion+"/"+strconv.FormatInt(orgUnitID, 10)+"/sections/mysections/", nil)
}
