package brightspace

// The course Activity Feed lives on a separate D2L host and takes a bearer
// token instead of cookies. CLAUDE.md decision 4 allows exactly this: a
// short-lived token, minted from the session, kept in memory, sent only to
// the https *.brightspace.com host that the course page names, never over a
// redirect.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrNoActivityFeed means the course has no Activity Feed, or the user may
// not see it.
var ErrNoActivityFeed = errors.New("the course has no Activity Feed, or the user cannot see it")

const (
	// tokenScope is what Brightspace's own course page asks for; nothing
	// shows that a narrower scope is accepted.
	tokenScope = "*:*:*"
	// defaultTokenLife applies when the token answer carries no expiry.
	defaultTokenLife = 50 * time.Minute
	// tokenMargin renews a token slightly before it expires.
	tokenMargin = time.Minute
	// maxFeedPages bounds how many result pages a feed read follows.
	maxFeedPages = 10
)

// WithTransport sets how the client sends requests. Tests use it to answer
// for Brightspace's hosts in-process; the host checks still apply.
func (c *Client) WithTransport(rt http.RoundTripper) *Client {
	c.http.Transport = rt
	return c
}

// FeedPost is one post in a course's Activity Feed.
type FeedPost struct {
	// ID is the post's identifier in the feed.
	ID string
	// Type is the kind of post, e.g. "Article".
	Type string
	// Published is when the post was made; zero if unknown.
	Published time.Time
	// HTML is the post body as HTML written by the author.
	HTML string
	// Attachments are the names of attached files, where the feed gives them.
	Attachments []string
	// Comments is the number of replies.
	Comments int
}

type feedPage struct {
	OrderedItems []struct {
		ID        string `json:"id"`
		Published string `json:"published"`
		Object    struct {
			Type       string           `json:"type"`
			Content    string           `json:"content"`
			Attachment []map[string]any `json:"attachment"`
			Replies    struct {
				TotalItems int `json:"totalItems"`
			} `json:"replies"`
		} `json:"object"`
	} `json:"orderedItems"`
	Next *struct {
		Href string `json:"href"`
	} `json:"next"`
}

// CourseFeed returns up to limit of the newest posts in a course's Activity
// Feed, and whether older posts exist. It returns ErrNoActivityFeed when the
// course has no feed or the user cannot see it.
func (c *Client) CourseFeed(ctx context.Context, orgUnitID int64, limit int) ([]FeedPost, bool, error) {
	// The token comes first: it is also the cheapest check that the session
	// is still valid, so a dead session is reported as such and not as a
	// course without a feed.
	token, err := c.bearer(ctx, false)
	if err != nil {
		return nil, false, err
	}
	origin, err := c.feedOrigin(ctx, orgUnitID)
	if err != nil {
		return nil, false, err
	}

	id := strconv.FormatInt(orgUnitID, 10)
	prefix := "/api/v1/d2l:orgUnit:" + id + "/article"
	next := origin + prefix + "/"
	seen := map[string]bool{}
	var posts []FeedPost
	for range maxFeedPages {
		if next == "" || seen[next] {
			break
		}
		seen[next] = true

		var page feedPage
		if err := c.getFeed(ctx, &token, next, &page); err != nil {
			return nil, false, err
		}
		for _, it := range page.OrderedItems {
			p := FeedPost{
				ID:       it.ID[strings.LastIndex(it.ID, "/")+1:],
				Type:     it.Object.Type,
				HTML:     it.Object.Content,
				Comments: it.Object.Replies.TotalItems,
			}
			if t, err := time.Parse(time.RFC3339, it.Published); err == nil {
				p.Published = t
			}
			for _, a := range it.Object.Attachment {
				for _, key := range []string{"name", "fileName", "filename", "title"} {
					if name, ok := a[key].(string); ok && name != "" {
						p.Attachments = append(p.Attachments, name)
						break
					}
				}
			}
			posts = append(posts, p)
		}
		if len(posts) > limit {
			return posts[:limit], true, nil
		}
		next = ""
		if page.Next != nil {
			next = sameFeed(page.Next.Href, origin, prefix)
		}
	}
	return posts, next != "", nil
}

// sameFeed returns href if it points into the same course feed on the same
// host, and "" otherwise. The feed's own links are followed, but never to
// another host or another part of the API.
func sameFeed(href, origin, prefix string) string {
	u, err := url.Parse(href)
	if err != nil || !u.IsAbs() || u.User != nil {
		return ""
	}
	o, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, o.Host) {
		return ""
	}
	if u.Path != prefix && !strings.HasPrefix(u.Path, prefix+"/") {
		return ""
	}
	return u.String()
}

// getFeed reads one feed page with the bearer token. A rejected token is
// replaced once; a second rejection means the session is no good.
func (c *Client) getFeed(ctx context.Context, token *string, rawURL string, out any) error {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+*token)

		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("read Activity Feed: %w", err)
		}
		err = func() error {
			defer resp.Body.Close()
			switch {
			case resp.StatusCode == http.StatusUnauthorized:
				return ErrSessionExpired
			case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound:
				return ErrNoActivityFeed
			case resp.StatusCode >= 300 && resp.StatusCode < 400:
				return errors.New("the Activity Feed answered with a redirect, which is not followed")
			case resp.StatusCode != http.StatusOK:
				return fmt.Errorf("read Activity Feed: unexpected status %s", resp.Status)
			}
			if mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); mt != "application/json" {
				return fmt.Errorf("read Activity Feed: unexpected content type %q", mt)
			}
			if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); err != nil {
				return fmt.Errorf("read Activity Feed: decode response: %w", err)
			}
			return nil
		}()
		if errors.Is(err, ErrSessionExpired) && attempt == 0 {
			if *token, err = c.bearer(ctx, true); err != nil {
				return err
			}
			continue
		}
		return err
	}
}

// bearer returns a bearer token minted from the session cookies. The token
// lives in memory only. fresh forces a new one.
func (c *Client) bearer(ctx context.Context, fresh bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !fresh && c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	c.token = ""

	var x struct {
		ReferrerToken string `json:"referrerToken"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/d2l/lp/auth/xsrf-tokens", nil, nil, &x); err != nil {
		return "", err
	}
	if x.ReferrerToken == "" {
		return "", ErrSessionExpired
	}

	header := http.Header{
		"Content-Type": {"application/x-www-form-urlencoded"},
		"X-Csrf-Token": {x.ReferrerToken},
	}
	form := strings.NewReader(url.Values{"scope": {tokenScope}}.Encode())
	var tok struct {
		AccessToken string  `json:"access_token"`
		ExpiresAt   float64 `json:"expires_at"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/d2l/lp/auth/oauth2/token", header, form, &tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", errors.New("Brightspace issued no access token")
	}

	exp := time.Now().Add(defaultTokenLife)
	if tok.ExpiresAt > 0 {
		secs := tok.ExpiresAt
		if secs > 1e12 { // milliseconds
			secs /= 1000
		}
		exp = time.Unix(int64(secs), 0).Add(-tokenMargin)
	}
	c.token, c.tokenExp = tok.AccessToken, exp
	return c.token, nil
}

// feedEndpointRE finds the API endpoints that the course page names.
var feedEndpointRE = regexp.MustCompile(`api-endpoint="(https://[^"]+)"`)

// feedOrigin finds the Activity Feed host that the course page names.
func (c *Client) feedOrigin(ctx context.Context, orgUnitID int64) (string, error) {
	page, err := c.getHTML(ctx, "/d2l/home/"+strconv.FormatInt(orgUnitID, 10))
	if err != nil {
		return "", err
	}
	for _, m := range feedEndpointRE.FindAllStringSubmatch(page, -1) {
		if origin, ok := feedHost(html.UnescapeString(m[1])); ok {
			return origin, nil
		}
	}
	return "", ErrNoActivityFeed
}

// feedHost accepts an endpoint only if it is the Activity Feed on a D2L
// host: https, no credentials, no unusual port, a *.brightspace.com name
// with an "activityfeed" label. It returns the origin (https://host).
func feedHost(endpoint string) (string, bool) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return "", false
	}
	if p := u.Port(); p != "" && p != "443" {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, ".brightspace.com") || !strings.Contains(host, ".activityfeed.") {
		return "", false
	}
	return "https://" + host, true
}

// getHTML reads a page of the instance with the session cookies.
func (c *Client) getHTML(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html")
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden:
		// The session was already checked, so this is the course itself.
		return "", fmt.Errorf("GET %s: %w", path, ErrNotFound)
	case resp.StatusCode == http.StatusUnauthorized || (resp.StatusCode >= 300 && resp.StatusCode < 400):
		return "", ErrSessionExpired
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("GET %s: unexpected status %s", path, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", path, err)
	}
	return string(b), nil
}
