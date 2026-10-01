package brightspace

import (
	"context"
	"strconv"
	"time"
)

// GradeValue is the user's grade on one grade item (or the final grade).
type GradeValue struct {
	Name              string     `json:"GradeObjectName"`
	DisplayedGrade    FlexText   `json:"DisplayedGrade"`
	PointsNumerator   *float64   `json:"PointsNumerator"`
	PointsDenominator *float64   `json:"PointsDenominator"`
	Comments          FlexText   `json:"Comments"`
	LastModified      *time.Time `json:"LastModified"`
}

func gradesPath(orgUnitID int64) string {
	return "/d2l/api/le/" + leVersion + "/" + strconv.FormatInt(orgUnitID, 10) + "/grades/"
}

// MyGradeValues returns the user's grades in a course. Brightspace leaves
// ungraded items out or sends null for them; nulls are dropped here.
func (c *Client) MyGradeValues(ctx context.Context, orgUnitID int64) ([]GradeValue, error) {
	var raw []*GradeValue
	if err := c.get(ctx, gradesPath(orgUnitID)+"values/myGradeValues/", &raw); err != nil {
		return nil, err
	}
	values := make([]GradeValue, 0, len(raw))
	for _, v := range raw {
		if v != nil {
			values = append(values, *v)
		}
	}
	return values, nil
}

// MyFinalGrade returns the user's calculated final grade in a course. It
// gives ErrNotFound when the course has none or has not released it.
func (c *Client) MyFinalGrade(ctx context.Context, orgUnitID int64) (GradeValue, error) {
	var v *GradeValue
	if err := c.get(ctx, gradesPath(orgUnitID)+"final/values/myGradeValue", &v); err != nil {
		return GradeValue{}, err
	}
	if v == nil {
		return GradeValue{}, ErrNotFound
	}
	return *v, nil
}
