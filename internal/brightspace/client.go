// Package brightspace is a small client for the Brightspace (D2L Valence) REST API.
package brightspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// ErrSessionExpired means Brightspace did not accept the session; the user
// has to log in again.
var ErrSessionExpired = errors.New("brightspace session expired or invalid")

// ErrNotFound means the requested item does not exist, or the user cannot
// see it.
var ErrNotFound = errors.New("not found in Brightspace")

// ErrForbidden means the session is valid but the user may not use this part
// of Brightspace, e.g. the tool is turned off in the course.
var ErrForbidden = errors.New("not allowed in Brightspace")

// API versions used for requests: lp is the Learning Platform (users,
// enrollments), le the Learning Environment (course tools such as news).
const (
	lpVersion = "1.45"
	leVersion = "1.74"
)

const maxResponseBytes = 10 << 20

// Client calls the Brightspace API on behalf of a logged-in user.
type Client struct {
	baseURL string
	cookies []*http.Cookie
	http    *http.Client

	// The Activity Feed bearer token (see feed.go); it is kept in memory only.
	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// NewClient returns a client that sends the session cookies to baseURL only.
func NewClient(baseURL string, cookies []*http.Cookie) *Client {
	return &Client{
		baseURL: baseURL,
		cookies: cookies,
		http: &http.Client{
			Timeout: 30 * time.Second,
			// Brightspace answers a rejected session with a redirect to its
			// login page. Never follow redirects: report them instead.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// BaseURL returns the Brightspace address the client talks to.
func (c *Client) BaseURL() string { return c.baseURL }

// Identity is the logged-in user.
type Identity struct {
	Identifier string `json:"Identifier"`
	FirstName  string `json:"FirstName"`
	LastName   string `json:"LastName"`
	UniqueName string `json:"UniqueName"`
}

// WhoAmI returns the user the session belongs to.
func (c *Client) WhoAmI(ctx context.Context) (Identity, error) {
	var id Identity
	err := c.get(ctx, "/d2l/api/lp/"+lpVersion+"/users/whoami", &id)
	return id, err
}

// courseOfferingType is the org unit type ID of a course offering. It is
// built into Brightspace and the same on every instance.
const courseOfferingType = "3"

// maxPages bounds how many result pages a listing follows.
const maxPages = 20

// Enrollment is a course the user is enrolled in.
type Enrollment struct {
	OrgUnit struct {
		ID      int64  `json:"Id"`
		Name    string `json:"Name"`
		Code    string `json:"Code"`
		HomeURL string `json:"HomeUrl"`
	} `json:"OrgUnit"`
	Access struct {
		IsActive  bool       `json:"IsActive"`
		CanAccess bool       `json:"CanAccess"`
		StartDate *time.Time `json:"StartDate"`
		EndDate   *time.Time `json:"EndDate"`
		// RoleName is the user's own role in the course, e.g. "Student".
		RoleName string `json:"ClasslistRoleName"`
	} `json:"Access"`
}

// MyCourses returns the course offerings the user is enrolled in.
func (c *Client) MyCourses(ctx context.Context) ([]Enrollment, error) {
	q := url.Values{"orgUnitTypeId": {courseOfferingType}}
	return getAll[Enrollment](ctx, c, "/d2l/api/lp/"+lpVersion+"/enrollments/myenrollments/", q)
}

// NewsItem is an announcement in a course.
type NewsItem struct {
	ID    int64  `json:"Id"`
	Title string `json:"Title"`
	Body  struct {
		Text string `json:"Text"`
	} `json:"Body"`
	// StartDate is when the item is shown; CreatedDate is a fallback.
	StartDate   *time.Time `json:"StartDate"`
	CreatedDate *time.Time `json:"CreatedDate"`
	IsHidden    bool       `json:"IsHidden"`
	IsPublished bool       `json:"IsPublished"`
	Attachments []struct {
		FileName string `json:"FileName"`
		Size     int64  `json:"Size"`
	} `json:"Attachments"`
}

// CourseNews returns the announcements in a course.
func (c *Client) CourseNews(ctx context.Context, orgUnitID int64) ([]NewsItem, error) {
	var items []NewsItem
	err := c.get(ctx, "/d2l/api/le/"+leVersion+"/"+strconv.FormatInt(orgUnitID, 10)+"/news/", &items)
	return items, err
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, nil, out)
}

// send makes a request to the instance with the session cookies. The caller
// closes the response body.
func (c *Client) send(ctx context.Context, method, path string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range header {
		req.Header[k] = v
	}
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	return resp, nil
}

// doJSON sends a request to the instance with the session cookies and
// decodes the JSON answer into out.
func (c *Client) doJSON(ctx context.Context, method, path string, header http.Header, body io.Reader, out any) error {
	resp, err := c.send(ctx, method, path, header, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%s %s: %w", method, path, ErrNotFound)
	}
	if resp.StatusCode == http.StatusUnauthorized || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return ErrSessionExpired
	}
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	// Without a valid session Brightspace serves an HTML page (a 403, or a
	// 200 login stub) instead of JSON. A 403 that is not HTML is the API
	// refusing this user, not a lost session.
	if resp.StatusCode == http.StatusForbidden && mediaType != "text/html" {
		return fmt.Errorf("%s %s: %w", method, path, ErrForbidden)
	}
	if mediaType != "application/json" {
		return ErrSessionExpired
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s: unexpected status %s", method, path, resp.Status)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); err != nil {
		return fmt.Errorf("%s %s: decode response: %w", method, path, err)
	}
	return nil
}
