package brightspace

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ParseBaseURL validates a Brightspace address typed by the user and reduces
// it to "https://host". Everything else (path, query, credentials) is dropped
// or rejected, so the session is only ever sent to that exact origin.
func ParseBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("brightspace URL is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid brightspace URL %q: %w", raw, err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("brightspace URL must use https, got %q", u.Scheme)
	}
	if u.User != nil {
		return "", errors.New("brightspace URL must not contain a username or password")
	}
	if u.Hostname() == "" {
		return "", fmt.Errorf("brightspace URL %q has no host", raw)
	}
	return "https://" + strings.ToLower(u.Host), nil
}
