package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Verify reports whether Brightspace accepts the candidate session cookies.
type Verify func(ctx context.Context, cookies []*http.Cookie) error

// browser is the user's own web browser, whose Brightspace cookies we read.
type browser interface {
	Name() string
	// Unlock gets access to the browser's encrypted cookies. The operating
	// system may ask the user to allow this.
	Unlock() error
	Cookies(ctx context.Context, host string) ([]Cookie, error)
	Open(url string) error
}

const (
	loginTimeout = 5 * time.Minute
	pollInterval = 2 * time.Second
)

// Login takes the Brightspace session from the user's default browser. If
// the browser is not logged in yet, it opens Brightspace there and waits
// until the user has signed in and verify accepts the session.
func Login(ctx context.Context, baseURL string, verify Verify, out io.Writer) (Session, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return Session{}, err
	}
	b, err := defaultBrowser()
	if err != nil {
		return Session{}, err
	}

	fmt.Fprintf(out, "Reading your Brightspace session from %s.\n"+
		"Your computer may ask to allow access to the browser's saved data.\n"+
		"brightspace-mcp is in beta: until the stable release we recommend Allow, not Always Allow.\n", b.Name())
	if err := b.Unlock(); err != nil {
		return Session{}, err
	}

	check := func() (Session, bool, error) {
		cookies, err := b.Cookies(ctx, u.Hostname())
		if err != nil {
			return Session{}, false, err
		}
		s := Session{BaseURL: baseURL, Cookies: cookies}
		return s, len(cookies) > 0 && verify(ctx, s.HTTPCookies()) == nil, nil
	}

	if s, ok, err := check(); err != nil || ok {
		return s, err
	}

	fmt.Fprintf(out, "Opening %s in %s. Log in there; this finishes automatically.\n", baseURL, b.Name())
	if err := b.Open(baseURL + "/d2l/home"); err != nil {
		return Session{}, fmt.Errorf("open %s: %w", b.Name(), err)
	}

	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return Session{}, fmt.Errorf("login not completed: %w", ctx.Err())
		case <-ticker.C:
		}
		if s, ok, err := check(); err != nil || ok {
			return s, err
		}
	}
}
