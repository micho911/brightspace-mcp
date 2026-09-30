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
	"time"
)

// ErrSessionExpired means Brightspace did not accept the session; the user
// has to log in again.
var ErrSessionExpired = errors.New("brightspace session expired or invalid")

// lpVersion is the Learning Platform API version used for requests.
const lpVersion = "1.45"

const maxResponseBytes = 10 << 20

// Client calls the Brightspace API on behalf of a logged-in user.
type Client struct {
	baseURL string
	cookies []*http.Cookie
	http    *http.Client
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

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return ErrSessionExpired
	}
	// Without a valid session Brightspace serves an HTML page (a 403, or a
	// 200 login stub) instead of JSON.
	if mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); mediaType != "application/json" {
		return ErrSessionExpired
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %s", path, resp.Status)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); err != nil {
		return fmt.Errorf("GET %s: decode response: %w", path, err)
	}
	return nil
}
