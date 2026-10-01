package brightspace

import (
	"context"
	"errors"
)

// ProductVersion returns the latest API version this instance offers for a
// product code (for example "bas" for awards). It asks once per client. If
// the instance will not say, it returns fallback; only a lost session is an
// error.
func (c *Client) ProductVersion(ctx context.Context, code, fallback string) (string, error) {
	c.mu.Lock()
	known := c.versions
	c.mu.Unlock()

	if known == nil {
		var products []struct {
			ProductCode   string `json:"ProductCode"`
			LatestVersion string `json:"LatestVersion"`
		}
		err := c.get(ctx, "/d2l/api/versions/", &products)
		if errors.Is(err, ErrSessionExpired) {
			return "", err
		}
		known = map[string]string{}
		for _, p := range products {
			known[p.ProductCode] = p.LatestVersion
		}
		if err == nil {
			c.mu.Lock()
			c.versions = known
			c.mu.Unlock()
		}
	}
	if v := known[code]; v != "" {
		return v, nil
	}
	return fallback, nil
}
