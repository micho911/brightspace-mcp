package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const eventsJSON = `{"Next":null,"Objects":[
	{"CalendarEventId":2,"OrgUnitId":3,"OrgUnitName":"New","Title":"Essay due","Description":{"Text":"","Html":"<p>Hand in <a href=\"https://x.example/a\">the essay</a></p>"},
	 "StartDateTime":"2026-10-05T10:00:00.000Z","EndDateTime":"2026-10-05T10:00:00.000Z","IsAllDayEvent":false,
	 "AssociatedEntity":{"AssociatedEntityType":"D2L.LE.Dropbox.Dropbox","AssociatedEntityId":77},"CalendarEventViewUrl":"/d2l/le/calendar/3/event/2/detail"},
	{"CalendarEventId":1,"OrgUnitId":3,"Title":"Lecture","Description":"plain text","StartDateTime":"2026-10-02T08:00:00.000Z",
	 "EndDateTime":"2026-10-02T10:00:00.000Z","IsAllDayEvent":false,"LocationName":"Room 1","AssociatedEntity":null}]}`

func fakeCalendar(t *testing.T, gotQuery *string) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/enrollments/myenrollments/"):
			_, _ = w.Write([]byte(enrollmentsJSON))
		case strings.HasSuffix(r.URL.Path, "/calendar/events/myEvents/"):
			*gotQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(eventsJSON))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callListUpcoming(t *testing.T, bs Connect, args map[string]any) UpcomingList {
	t.Helper()
	now = func() time.Time { return time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = time.Now })
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_upcoming", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var list UpcomingList
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return list
}

func TestListUpcoming(t *testing.T) {
	var query string
	list := callListUpcoming(t, fakeCalendar(t, &query), nil)

	// Courses 2 and 3 are active; 1 has ended and 4 is closed.
	for _, want := range []string{"orgUnitIdsCSV=2%2C3", "startDateTime=2026-10-01T12%3A00%3A00.000Z", "endDateTime=2026-10-15T12%3A00%3A00.000Z"} {
		if !strings.Contains(query, want) {
			t.Errorf("query %q lacks %q", query, want)
		}
	}
	if len(list.Events) != 2 || list.Events[0].Title != "Lecture" || list.Events[1].Title != "Essay due" {
		t.Fatalf("events = %+v, want Lecture then Essay due", list.Events)
	}
	lec, essay := list.Events[0], list.Events[1]
	if lec.Location != "Room 1" || lec.Description != "plain text" || lec.Course != "New" {
		t.Errorf("lecture = %+v", lec)
	}
	if essay.Description != "Hand in the essay (https://x.example/a)" || essay.Kind != "D2L.LE.Dropbox.Dropbox" ||
		!strings.HasSuffix(essay.URL, "/d2l/le/calendar/3/event/2/detail") || !strings.HasPrefix(essay.URL, "http") {
		t.Errorf("essay = %+v", essay)
	}
}

func TestListUpcomingOneCourseAndLimit(t *testing.T) {
	var query string
	list := callListUpcoming(t, fakeCalendar(t, &query), map[string]any{"courseId": 3, "days": 500, "limit": 1})

	if !strings.Contains(query, "orgUnitIdsCSV=3&") || !strings.Contains(query, "endDateTime=2026-12-30") {
		t.Errorf("query = %q, want course 3 only and days capped at 90", query)
	}
	if len(list.Events) != 1 || !list.More {
		t.Errorf("list = %+v, want one event and more=true", list)
	}
}
