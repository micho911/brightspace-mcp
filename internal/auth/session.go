// Package auth handles signing in to Brightspace and keeping the session.
//
// The user signs in through their own browser; this package never sees a
// password. Only the Brightspace session cookies are kept, in the OS keychain.
package auth

import (
	"net/http"
	"strings"
)

// Session is a logged-in Brightspace session.
type Session struct {
	BaseURL string   `json:"baseURL"`
	Cookies []Cookie `json:"cookies"`
}

// Cookie is the part of a browser cookie needed to replay the session.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HTTPCookies returns the session cookies for use in requests.
func (s Session) HTTPCookies() []*http.Cookie {
	out := make([]*http.Cookie, len(s.Cookies))
	for i, c := range s.Cookies {
		out[i] = &http.Cookie{Name: c.Name, Value: c.Value}
	}
	return out
}

// hostCookies keeps only cookies set by host itself. Cookies of parent
// domains (e.g. a university-wide SSO cookie on ".au.dk") are dropped: they
// are not needed for Brightspace and must not be stored.
func hostCookies(host string, cookies []*http.Cookie) []Cookie {
	host = strings.ToLower(host)
	var out []Cookie
	for _, c := range cookies {
		if strings.ToLower(strings.TrimPrefix(c.Domain, ".")) == host {
			out = append(out, Cookie{Name: c.Name, Value: c.Value})
		}
	}
	return out
}
