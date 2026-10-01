package brightspace

import (
	"context"
	"net/url"
	"time"
)

// awardsFallbackVersion is used when the instance does not list the awards
// product.
const awardsFallbackVersion = "1.0"

// IssuedAward is a badge or certificate given to the user.
type IssuedAward struct {
	ID         int64      `json:"IssuedId"`
	OrgUnitID  int64      `json:"OrgUnitId"`
	Criteria   FlexText   `json:"Criteria"`
	IssuedDate *time.Time `json:"IssuedDate"`
	ExpiryDate *time.Time `json:"ExpiryDate"`
	Award      struct {
		Title       string   `json:"Title"`
		Description FlexText `json:"Description"`
		IssuerName  string   `json:"IssuerName"`
		// AwardType is 1 for a badge and 2 for a certificate.
		AwardType FlexText `json:"AwardType"`
	} `json:"Award"`
}

// IssuedAwards returns the badges and certificates issued to the user.
func (c *Client) IssuedAwards(ctx context.Context, userID string, includeExpired bool) ([]IssuedAward, error) {
	version, err := c.ProductVersion(ctx, "bas", awardsFallbackVersion)
	if err != nil {
		return nil, err
	}
	q := url.Values{"limit": {"100"}}
	if includeExpired {
		q.Set("includeExpired", "true")
	}
	return getAll[IssuedAward](ctx, c, "/d2l/api/bas/"+version+"/issued/users/"+url.PathEscape(userID)+"/", q)
}
