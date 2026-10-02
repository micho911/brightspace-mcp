package server

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultUpcomingDays   = 14
	maxUpcomingDays       = 90
	defaultUpcomingEvents = 30
	maxUpcomingEvents     = 100
	maxEventDescription   = 500
)

// now is replaced in tests.
var now = time.Now

// ListUpcomingInput selects the calendar window.
type ListUpcomingInput struct {
	Days     int   `json:"days,omitempty" jsonschema:"how many days ahead to look (default 14, at most 90)"`
	CourseID int64 `json:"courseId,omitempty" jsonschema:"only this course (ID from list_courses); default is all active courses"`
	Limit    int   `json:"limit,omitempty" jsonschema:"maximum number of events, soonest first (default 30, at most 100)"`
}

// UpcomingEvent is one calendar event.
type UpcomingEvent struct {
	Title       string `json:"title"`
	CourseID    int64  `json:"courseId,omitempty"`
	Course      string `json:"course,omitempty" jsonschema:"course name"`
	Start       string `json:"start,omitempty" jsonschema:"RFC 3339"`
	End         string `json:"end,omitempty" jsonschema:"RFC 3339"`
	AllDay      bool   `json:"allDay,omitempty"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description,omitempty" jsonschema:"plain text, shortened"`
	Kind        string `json:"kind,omitempty" jsonschema:"what the event belongs to, e.g. a dropbox or quiz, when Brightspace says"`
	URL         string `json:"url,omitempty" jsonschema:"the event in Brightspace"`
}

// UpcomingList is the result of list_upcoming.
type UpcomingList struct {
	From   string          `json:"from" jsonschema:"start of the window, RFC 3339"`
	To     string          `json:"to" jsonschema:"end of the window, RFC 3339"`
	Events []UpcomingEvent `json:"events"`
	More   bool            `json:"more,omitempty" jsonschema:"there are more events in the window than listed"`
}

func addListUpcoming(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_upcoming",
		Description: "List the user's upcoming Brightspace calendar events (due dates, lectures, deadlines) across all active courses, soonest first. " +
			"Use this when the user asks what is due, what is coming up, or what is on this week. " +
			"Only events in the course calendars are included: it does not read assignments or quizzes themselves, so use list_assignments or list_quizzes for submission status.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListUpcomingInput) (*mcp.CallToolResult, UpcomingList, error) {
		days := clampLimit(in.Days, defaultUpcomingDays, maxUpcomingDays)
		limit := clampLimit(in.Limit, defaultUpcomingEvents, maxUpcomingEvents)

		c, err := connect()
		if err != nil {
			return nil, UpcomingList{}, sessionError(err, "")
		}
		from := now()
		to := from.AddDate(0, 0, days)

		names := map[int64]string{}
		var ids []int64
		if in.CourseID > 0 {
			ids = []int64{in.CourseID}
		} else {
			enrollments, err := c.MyCourses(ctx)
			if err != nil {
				return nil, UpcomingList{}, sessionError(err, c.BaseURL())
			}
			for _, e := range enrollments {
				ended := e.Access.EndDate != nil && e.Access.EndDate.Before(from)
				if e.Access.IsActive && e.Access.CanAccess && !ended {
					ids = append(ids, e.OrgUnit.ID)
					names[e.OrgUnit.ID] = e.OrgUnit.Name
				}
			}
		}

		list := UpcomingList{From: from.UTC().Format(time.RFC3339), To: to.UTC().Format(time.RFC3339), Events: []UpcomingEvent{}}
		if len(ids) == 0 {
			return nil, list, nil
		}
		events, err := c.MyEvents(ctx, ids, from, to)
		if err != nil {
			return nil, UpcomingList{}, courseError(err, c.BaseURL(), in.CourseID, "calendar")
		}
		for _, e := range events {
			ev := UpcomingEvent{
				Title:    e.Title,
				CourseID: e.OrgUnitID,
				Course:   cmp.Or(e.OrgUnitName, names[e.OrgUnitID]),
				Start:    formatTime(e.Start),
				End:      formatTime(e.End),
				AllDay:   e.IsAllDay,
				Location: e.LocationName,
			}
			ev.Description, _ = truncate(htmlToText(string(e.Description)), maxEventDescription)
			if e.AssociatedRef != nil {
				ev.Kind = string(e.AssociatedRef.Type)
			}
			if e.ViewURL != "" {
				ev.URL = resolveURL(c.BaseURL(), e.ViewURL)
			}
			list.Events = append(list.Events, ev)
		}
		slices.SortStableFunc(list.Events, func(a, b UpcomingEvent) int { return cmp.Compare(a.Start, b.Start) })
		if len(list.Events) > limit {
			list.Events, list.More = list.Events[:limit], true
		}
		return nil, list, nil
	})
}
