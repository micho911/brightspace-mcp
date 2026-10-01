package brightspace

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// maxUpdateOrgUnits bounds how many courses go into one request (the API
// allows at most 100).
const maxUpdateOrgUnits = 50

// Update holds the counts of things waiting for the user in one course. Only
// the counts that matter to a student are read.
type Update struct {
	OrgUnitID                int64 `json:"OrgUnitId"`
	UnreadDiscussions        int   `json:"UnreadDiscussions"`
	UnreadAssignmentFeedback int   `json:"UnreadAssignmentFeedback"`
	UnattemptedQuizzes       int   `json:"UnattemptedQuizzes"`
}

// MyUpdates returns the counts of unread and unfinished things in the given
// courses.
func (c *Client) MyUpdates(ctx context.Context, orgUnitIDs []int64) ([]Update, error) {
	var all []Update
	for len(orgUnitIDs) > 0 {
		n := min(len(orgUnitIDs), maxUpdateOrgUnits)
		ids := make([]string, n)
		for i, id := range orgUnitIDs[:n] {
			ids[i] = strconv.FormatInt(id, 10)
		}
		orgUnitIDs = orgUnitIDs[n:]

		q := url.Values{"orgUnitIdsCSV": {strings.Join(ids, ",")}}
		updates, err := getAll[Update](ctx, c, "/d2l/api/le/"+leVersion+"/updates/myUpdates/", q)
		if err != nil {
			return nil, err
		}
		all = append(all, updates...)
	}
	return all, nil
}
