package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const awardsJSON = `{"Next":null,"Objects":[
	{"IssuedId":8,"OrgUnitId":3,"Criteria":"Finish <b>all</b> modules","IssuedDate":"2026-08-01T00:00:00.000Z","ExpiryDate":null,
	 "IssuedByUserId":77,"IssuedToUserId":42,"Award":{"AwardId":1,"Title":"Course complete","Description":{"Text":"Done","Html":""},"IssuerName":"The University","AwardType":2}},
	{"IssuedId":9,"OrgUnitId":2,"Criteria":null,"IssuedDate":"2026-09-01T00:00:00.000Z","ExpiryDate":"2027-09-01T00:00:00.000Z",
	 "Award":{"AwardId":2,"Title":"Active learner","Description":null,"IssuerName":"","AwardType":1}}]}`

func fakeAwards(t *testing.T, versions, issued string, gotPath *string) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/users/whoami"):
			_, _ = w.Write([]byte(`{"Identifier":"42"}`))
		case r.URL.Path == "/d2l/api/versions/":
			if versions == "" {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(versions))
		case strings.Contains(r.URL.Path, "/issued/users/"):
			*gotPath = r.URL.RequestURI()
			if issued == "forbidden" {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(issued))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callListAwards(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "list_awards", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func awardList(t *testing.T, res *mcp.CallToolResult) AwardList {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var l AwardList
	if err := json.Unmarshal(raw, &l); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return l
}

const versionsJSON = `[{"ProductCode":"lp","LatestVersion":"1.50","SupportedVersions":["1.45","1.50"]},{"ProductCode":"bas","LatestVersion":"1.4","SupportedVersions":["1.0","1.4"]}]`

func TestListAwards(t *testing.T) {
	var path string
	l := awardList(t, callListAwards(t, fakeAwards(t, versionsJSON, awardsJSON, &path), nil))

	if !strings.HasPrefix(path, "/d2l/api/bas/1.4/issued/users/42/") {
		t.Errorf("path = %q, want the awards version the instance reports and the user's own ID", path)
	}
	if len(l.Awards) != 2 || l.Awards[0].Title != "Active learner" || l.Awards[1].Title != "Course complete" {
		t.Fatalf("awards = %+v, want newest first", l.Awards)
	}
	badge, cert := l.Awards[0], l.Awards[1]
	if badge.Kind != "badge" || badge.Expires != "2027-09-01T00:00:00Z" || badge.CourseID != 2 {
		t.Errorf("badge = %+v", badge)
	}
	if cert.Kind != "certificate" || cert.Description != "Done" || cert.Criteria != "Finish all modules" || cert.Issuer != "The University" || cert.Issued != "2026-08-01T00:00:00Z" {
		t.Errorf("certificate = %+v", cert)
	}
}

func TestListAwardsCourseFilterAndFallbackVersion(t *testing.T) {
	var path string
	l := awardList(t, callListAwards(t, fakeAwards(t, "", awardsJSON, &path), map[string]any{"courseId": 3, "includeExpired": true}))

	if !strings.HasPrefix(path, "/d2l/api/bas/1.0/issued/users/42/") || !strings.Contains(path, "includeExpired=true") {
		t.Errorf("path = %q, want the fallback version when versions are unavailable, and includeExpired", path)
	}
	if len(l.Awards) != 1 || l.Awards[0].Title != "Course complete" {
		t.Errorf("awards = %+v, want only course 3", l.Awards)
	}
}

func TestListAwardsForbidden(t *testing.T) {
	var path string
	msg := errorText(t, callListAwards(t, fakeAwards(t, versionsJSON, "forbidden", &path), nil))
	if !strings.Contains(msg, "does not let the user see badges") || strings.Contains(msg, "login") {
		t.Errorf("error %q should say awards are not available, not that the session expired", msg)
	}
}
