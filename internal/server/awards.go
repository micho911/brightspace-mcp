package server

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

const maxAwards = 100

// ListAwardsInput selects which awards to list.
type ListAwardsInput struct {
	CourseID       int64 `json:"courseId,omitempty" jsonschema:"only awards from this course (ID from list_courses)"`
	IncludeExpired bool  `json:"includeExpired,omitempty" jsonschema:"also list awards that have expired (default false)"`
}

// AwardInfo is one badge or certificate.
type AwardInfo struct {
	Title       string `json:"title"`
	Kind        string `json:"kind,omitempty" jsonschema:"badge or certificate"`
	Description string `json:"description,omitempty" jsonschema:"plain text, shortened"`
	Criteria    string `json:"criteria,omitempty" jsonschema:"what it was earned for, plain text, shortened"`
	Issuer      string `json:"issuer,omitempty"`
	CourseID    int64  `json:"courseId,omitempty"`
	Issued      string `json:"issued,omitempty" jsonschema:"RFC 3339"`
	Expires     string `json:"expires,omitempty" jsonschema:"RFC 3339"`
}

// AwardList is the result of list_awards.
type AwardList struct {
	Awards []AwardInfo `json:"awards" jsonschema:"newest first"`
	More   bool        `json:"more,omitempty" jsonschema:"there are more awards than listed"`
}

func addListAwards(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "list_awards",
		Description: "List the badges and certificates the user has earned in Brightspace, newest first, with title, what it was earned for, issuer and dates. " +
			"Use this when the user asks what badges or certificates they have or whether they completed something that awards one. " +
			"It shows only the user's own awards, does not download certificate files and does not share or issue anything. " +
			"Some institutions do not let students see awards: that is reported as such.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ListAwardsInput) (*mcp.CallToolResult, AwardList, error) {
		c, err := connect()
		if err != nil {
			return nil, AwardList{}, sessionError(err, "")
		}
		me, err := c.UserID(ctx)
		if err != nil {
			return nil, AwardList{}, sessionError(err, c.BaseURL())
		}
		awards, err := c.IssuedAwards(ctx, me, in.IncludeExpired)
		switch {
		case errors.Is(err, brightspace.ErrForbidden):
			return nil, AwardList{}, errors.New("Brightspace does not let the user see badges or certificates here: the institution may have turned awards off for students")
		case errors.Is(err, brightspace.ErrNotFound):
			return nil, AwardList{Awards: []AwardInfo{}}, nil
		case err != nil:
			return nil, AwardList{}, sessionError(err, c.BaseURL())
		}

		out := AwardList{Awards: []AwardInfo{}}
		for _, a := range awards {
			if in.CourseID > 0 && a.OrgUnitID != in.CourseID {
				continue
			}
			info := AwardInfo{
				Title:    a.Award.Title,
				Kind:     awardKind(string(a.Award.AwardType)),
				Issuer:   a.Award.IssuerName,
				CourseID: a.OrgUnitID,
				Issued:   formatTime(a.IssuedDate),
				Expires:  formatTime(a.ExpiryDate),
			}
			info.Description, _ = truncate(htmlToText(string(a.Award.Description)), maxTopicBlurb)
			info.Criteria, _ = truncate(htmlToText(string(a.Criteria)), maxTopicBlurb)
			out.Awards = append(out.Awards, info)
		}
		// Newest first; RFC 3339 in UTC sorts correctly as text.
		slices.SortStableFunc(out.Awards, func(a, b AwardInfo) int { return cmp.Compare(b.Issued, a.Issued) })
		if len(out.Awards) > maxAwards {
			out.Awards, out.More = out.Awards[:maxAwards], true
		}
		return nil, out, nil
	})
}

func awardKind(t string) string {
	switch t {
	case "1", "Badge":
		return "badge"
	case "2", "Certificate":
		return "certificate"
	}
	return ""
}
