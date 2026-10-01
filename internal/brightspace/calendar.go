package brightspace

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxCalendarOrgUnits bounds how many courses go into one request.
const maxCalendarOrgUnits = 50

// CalendarEvent is an event in the user's calendar (a due date, a lecture, …).
type CalendarEvent struct {
	ID            int64      `json:"CalendarEventId"`
	OrgUnitID     int64      `json:"OrgUnitId"`
	OrgUnitName   string     `json:"OrgUnitName"`
	Title         string     `json:"Title"`
	Description   FlexText   `json:"Description"`
	Start         *time.Time `json:"StartDateTime"`
	End           *time.Time `json:"EndDateTime"`
	IsAllDay      bool       `json:"IsAllDayEvent"`
	LocationName  string     `json:"LocationName"`
	ViewURL       string     `json:"CalendarEventViewUrl"`
	AssociatedRef *struct {
		Type FlexText `json:"AssociatedEntityType"`
		ID   FlexText `json:"AssociatedEntityId"`
	} `json:"AssociatedEntity"`
}

// MyEvents returns the user's calendar events in the given courses between
// from and to.
func (c *Client) MyEvents(ctx context.Context, orgUnitIDs []int64, from, to time.Time) ([]CalendarEvent, error) {
	var all []CalendarEvent
	for len(orgUnitIDs) > 0 {
		n := min(len(orgUnitIDs), maxCalendarOrgUnits)
		ids := make([]string, n)
		for i, id := range orgUnitIDs[:n] {
			ids[i] = strconv.FormatInt(id, 10)
		}
		orgUnitIDs = orgUnitIDs[n:]

		q := url.Values{
			"orgUnitIdsCSV": {strings.Join(ids, ",")},
			"startDateTime": {from.UTC().Format("2006-01-02T15:04:05.000Z")},
			"endDateTime":   {to.UTC().Format("2006-01-02T15:04:05.000Z")},
		}
		events, err := getAll[CalendarEvent](ctx, c, "/d2l/api/le/"+leVersion+"/calendar/events/myEvents/", q)
		if err != nil {
			return nil, err
		}
		all = append(all, events...)
	}
	return all, nil
}
