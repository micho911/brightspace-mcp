package brightspace

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	testInstance = "https://brightspace.test"
	testFeedHost = "prd.activityfeed.eu-west-1.brightspace.com"
	testCourse   = int64(42)
)

// fakeD2L answers for the instance and for the Activity Feed host in-process,
// so the client's real host checks run. It also checks that credentials go
// only where they belong: cookies to the instance, the bearer to the feed.
type fakeD2L struct {
	t       *testing.T
	course  string                                       // body of the course page
	feed    func(w http.ResponseWriter, r *http.Request) // the feed host
	expired bool                                         // the instance answers like a dead session
	log     []string                                     // "METHOD host/path"
	mints   int
}

func newFakeD2L(t *testing.T) (*fakeD2L, *Client) {
	f := &fakeD2L{t: t, course: courseWithFeed("https://" + testFeedHost + "/"), feed: defaultFeed}
	cookies := []*http.Cookie{{Name: "d2lSessionVal", Value: "session"}}
	return f, NewClient(testInstance, cookies).WithTransport(f)
}

func (f *fakeD2L) RoundTrip(r *http.Request) (*http.Response, error) {
	f.log = append(f.log, r.Method+" "+r.URL.Host+r.URL.Path)
	rec := httptest.NewRecorder()
	switch r.URL.Host {
	case "brightspace.test":
		if c, err := r.Cookie("d2lSessionVal"); err != nil || c.Value != "session" {
			f.t.Errorf("instance request without the session cookie: %s", r.URL.Path)
		}
		f.instance(rec, r)
	case testFeedHost:
		if r.Header.Get("Cookie") != "" {
			f.t.Errorf("cookies sent to the feed host")
		}
		f.feed(rec, r)
	default:
		f.t.Errorf("request to unexpected host %s", r.URL.Host)
		rec.WriteHeader(http.StatusBadGateway)
	}
	return rec.Result(), nil
}

func (f *fakeD2L) instance(w http.ResponseWriter, r *http.Request) {
	if f.expired {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		return
	}
	switch r.URL.Path {
	case "/d2l/lp/auth/xsrf-tokens":
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"referrerToken":"csrf"}`)
	case "/d2l/lp/auth/oauth2/token":
		if r.Method != http.MethodPost || r.Header.Get("X-Csrf-Token") != "csrf" {
			f.t.Errorf("token request without POST and CSRF header")
		}
		if err := r.ParseForm(); err != nil || r.PostForm.Get("scope") != "*:*:*" {
			f.t.Errorf("token request with scope %q", r.PostForm.Get("scope"))
		}
		f.mints++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"tok-%d","expires_at":%d}`, f.mints, time.Now().Add(time.Hour).Unix())
	case fmt.Sprintf("/d2l/home/%d", testCourse):
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, f.course)
	default:
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeD2L) hostsCalled() []string {
	var hosts []string
	for _, l := range f.log {
		host := strings.SplitN(strings.SplitN(l, " ", 2)[1], "/", 2)[0]
		if !slices.Contains(hosts, host) {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

func courseWithFeed(endpoints ...string) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	for _, e := range endpoints {
		fmt.Fprintf(&b, `<d2l-widget class="d2l-token-receiver" data-token-receiver-scope="*:*:*" api-endpoint=%q telemetry-endpoint="https://telemetry.example/"></d2l-widget>`, e)
	}
	return b.String() + "</body></html>"
}

func feedItem(uuid, published, content string, comments int, attachment string) string {
	att := "[]"
	if attachment != "" {
		att = fmt.Sprintf(`[{"type":"Document","name":%q}]`, attachment)
	}
	return fmt.Sprintf(`{"type":"Create","id":"https://%s/api/v1/d2l:orgUnit:42/article/%s","published":%q,`+
		`"object":{"type":"Article","content":%q,"attachment":%s,"replies":{"totalItems":%d}}}`,
		testFeedHost, uuid, published, content, att, comments)
}

// defaultFeed serves two pages: three posts, newest first.
func defaultFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Query().Get("from") {
	case "":
		fmt.Fprintf(w, `{"type":"OrderedCollectionPage","orderedItems":[%s,%s],"next":{"href":"https://%s/api/v1/d2l:orgUnit:42/article?from=p2&pageSize=2"}}`,
			feedItem("uuid-3", "2026-09-30T12:00:00.000Z", "<p>Third</p>", 2, "slides.pdf"),
			feedItem("uuid-2", "2026-09-29T12:00:00.000Z", "<p>Second</p>", 0, ""),
			testFeedHost)
	case "p2":
		fmt.Fprintf(w, `{"type":"OrderedCollectionPage","orderedItems":[%s]}`,
			feedItem("uuid-1", "2026-09-28T12:00:00.000Z", "<p>First</p>", 1, ""))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestCourseFeed(t *testing.T) {
	f, c := newFakeD2L(t)
	f.feed = func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-1" {
			t.Errorf("Authorization = %q, want the minted bearer", got)
		}
		defaultFeed(w, r)
	}

	posts, more, err := c.CourseFeed(t.Context(), testCourse, 10)
	if err != nil {
		t.Fatalf("CourseFeed: %v", err)
	}
	if more {
		t.Error("more = true, want false: the feed has three posts")
	}
	var ids []string
	for _, p := range posts {
		ids = append(ids, p.ID)
	}
	if want := []string{"uuid-3", "uuid-2", "uuid-1"}; !slices.Equal(ids, want) {
		t.Fatalf("post IDs = %v, want %v", ids, want)
	}
	first := posts[0]
	if first.Type != "Article" || first.HTML != "<p>Third</p>" || first.Comments != 2 ||
		!slices.Equal(first.Attachments, []string{"slides.pdf"}) ||
		!first.Published.Equal(time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("first post = %+v", first)
	}

	// A second read reuses the token instead of minting another.
	if _, _, err := c.CourseFeed(t.Context(), testCourse, 10); err != nil {
		t.Fatalf("second CourseFeed: %v", err)
	}
	if f.mints != 1 {
		t.Errorf("minted %d tokens, want 1", f.mints)
	}
}

func TestCourseFeedLimit(t *testing.T) {
	_, c := newFakeD2L(t)
	posts, more, err := c.CourseFeed(t.Context(), testCourse, 2)
	if err != nil {
		t.Fatalf("CourseFeed: %v", err)
	}
	if len(posts) != 2 || !more {
		t.Errorf("got %d posts, more = %v; want 2 posts and more = true", len(posts), more)
	}
}

func TestCourseFeedRefusesOtherFeedHosts(t *testing.T) {
	for _, endpoint := range []string{
		"https://evil.example.com/",
		"https://" + testFeedHost + ".evil.com/",
		"http://" + testFeedHost + "/",
		"https://user@" + testFeedHost + "/",
		"https://" + testFeedHost + ":8443/",
		"https://other.brightspace.com/", // a D2L host, but not the Activity Feed
	} {
		t.Run(endpoint, func(t *testing.T) {
			f, c := newFakeD2L(t)
			f.course = courseWithFeed(endpoint)
			_, _, err := c.CourseFeed(t.Context(), testCourse, 10)
			if !errors.Is(err, ErrNoActivityFeed) {
				t.Fatalf("err = %v, want ErrNoActivityFeed", err)
			}
			if hosts := f.hostsCalled(); !slices.Equal(hosts, []string{"brightspace.test"}) {
				t.Errorf("hosts called = %v: the token must not leave the instance", hosts)
			}
		})
	}
}

func TestCourseFeedPicksTheActivityFeedEndpoint(t *testing.T) {
	f, c := newFakeD2L(t)
	f.course = courseWithFeed("https://course-image-catalog.api.brightspace.com/", "https://"+testFeedHost+"/")
	if _, _, err := c.CourseFeed(t.Context(), testCourse, 10); err != nil {
		t.Fatalf("CourseFeed: %v", err)
	}
}

func TestCourseFeedDoesNotFollowForeignNextLinks(t *testing.T) {
	for _, next := range []string{
		"https://evil.example.com/api/v1/d2l:orgUnit:42/article?from=x",
		"http://" + testFeedHost + "/api/v1/d2l:orgUnit:42/article?from=x",
		"https://" + testFeedHost + "/api/v1/d2l:orgUnit:99/article?from=x",
		"https://" + testFeedHost + "/api/v1/profile",
		"/api/v1/d2l:orgUnit:42/article?from=x",
	} {
		t.Run(next, func(t *testing.T) {
			f, c := newFakeD2L(t)
			f.feed = func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"orderedItems":[%s],"next":{"href":%q}}`,
					feedItem("uuid-1", "2026-09-28T12:00:00.000Z", "x", 0, ""), next)
			}
			posts, more, err := c.CourseFeed(t.Context(), testCourse, 10)
			if err != nil || len(posts) != 1 || more {
				t.Fatalf("got %d posts, more = %v, err = %v; want the first page only", len(posts), more, err)
			}
			for _, l := range f.log {
				if strings.Contains(l, "evil.example.com") || strings.Contains(l, "orgUnit:99") || strings.Contains(l, "/profile") {
					t.Errorf("followed a foreign link: %s", l)
				}
			}
		})
	}
}

func TestCourseFeedNoAccess(t *testing.T) {
	f, c := newFakeD2L(t)
	f.feed = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"Permission Denied"}`)
	}
	if _, _, err := c.CourseFeed(t.Context(), testCourse, 10); !errors.Is(err, ErrNoActivityFeed) {
		t.Fatalf("err = %v, want ErrNoActivityFeed (a 403 is not an expired session)", err)
	}
}

func TestCourseFeedCourseWithoutFeedWidget(t *testing.T) {
	f, c := newFakeD2L(t)
	f.course = "<html><body>no widgets</body></html>"
	if _, _, err := c.CourseFeed(t.Context(), testCourse, 10); !errors.Is(err, ErrNoActivityFeed) {
		t.Fatalf("err = %v, want ErrNoActivityFeed", err)
	}
}

func TestCourseFeedUnknownCourse(t *testing.T) {
	_, c := newFakeD2L(t)
	if _, _, err := c.CourseFeed(t.Context(), 7, 10); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCourseFeedExpiredSession(t *testing.T) {
	f, c := newFakeD2L(t)
	f.expired = true
	if _, _, err := c.CourseFeed(t.Context(), testCourse, 10); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("err = %v, want ErrSessionExpired", err)
	}
	if hosts := f.hostsCalled(); !slices.Equal(hosts, []string{"brightspace.test"}) {
		t.Errorf("hosts called = %v, want the instance only", hosts)
	}
}

func TestCourseFeedReplacesARejectedToken(t *testing.T) {
	f, c := newFakeD2L(t)
	f.feed = func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer tok-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		defaultFeed(w, r)
	}
	if _, _, err := c.CourseFeed(t.Context(), testCourse, 10); err != nil {
		t.Fatalf("CourseFeed: %v", err)
	}
	if f.mints != 2 {
		t.Errorf("minted %d tokens, want 2 (one replacement)", f.mints)
	}
}

func TestCourseFeedKeepsRejectingTokens(t *testing.T) {
	f, c := newFakeD2L(t)
	f.feed = func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }
	if _, _, err := c.CourseFeed(t.Context(), testCourse, 10); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("err = %v, want ErrSessionExpired", err)
	}
	if f.mints != 2 {
		t.Errorf("minted %d tokens, want 2: one retry, then give up", f.mints)
	}
}

func TestFeedHost(t *testing.T) {
	for endpoint, want := range map[string]string{
		"https://prd.activityfeed.eu-west-1.brightspace.com/":     "https://prd.activityfeed.eu-west-1.brightspace.com",
		"https://PRD.activityfeed.eu-west-1.brightspace.com:443/": "https://prd.activityfeed.eu-west-1.brightspace.com",
		"https://prd.activityfeed.eu-west-1.brightspace.com.x.io": "",
		"https://brightspace.com/":                                "",
		"ftp://prd.activityfeed.eu-west-1.brightspace.com/":       "",
		"not a url": "",
	} {
		if got, _ := feedHost(endpoint); got != want {
			t.Errorf("feedHost(%q) = %q, want %q", endpoint, got, want)
		}
	}
}
