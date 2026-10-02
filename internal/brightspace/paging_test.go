package brightspace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func serve(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, nil)
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func TestGetAllFollowsNextLinks(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			writeJSON(w, `{"Next":null,"Objects":[3]}`)
			return
		}
		if r.URL.Query().Get("kind") != "x" {
			t.Errorf("query = %q, want kind=x kept", r.URL.RawQuery)
		}
		writeJSON(w, `{"Next":"`+"http://"+r.Host+r.URL.Path+`?page=2","Objects":[1,2]}`)
	})
	got, err := getAll[int](context.Background(), c, "/list/", map[string][]string{"kind": {"x"}})
	if err != nil {
		t.Fatalf("getAll: %v", err)
	}
	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("items = %v, want %v", got, want)
	}
}

func TestGetAllFollowsBookmarks(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("bookmark") {
		case "":
			writeJSON(w, `{"PagingInfo":{"Bookmark":"b1","HasMoreItems":true},"Items":[1]}`)
		case "b1":
			writeJSON(w, `{"PagingInfo":{"Bookmark":null,"HasMoreItems":false},"Items":[2]}`)
		}
	})
	got, err := getAll[int](context.Background(), c, "/list/", nil)
	if err != nil || !slices.Equal(got, []int{1, 2}) {
		t.Errorf("getAll = %v, %v, want [1 2]", got, err)
	}
}

func TestGetAllRefusesForeignNextLink(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"Next":"https://evil.example/steal","Objects":[1]}`)
	})
	if _, err := getAll[int](context.Background(), c, "/list/", nil); err == nil {
		t.Fatal("getAll followed a link to another host")
	}
}

func TestForbiddenIsNotExpiredSession(t *testing.T) {
	cases := map[string]string{"json": "application/json", "plain": "text/plain", "none": ""}
	for name, contentType := range cases {
		t.Run(name, func(t *testing.T) {
			c := serve(t, func(w http.ResponseWriter, r *http.Request) {
				if contentType != "" {
					w.Header().Set("Content-Type", contentType)
				}
				w.WriteHeader(http.StatusForbidden)
			})
			if _, err := c.CourseNews(context.Background(), 1); !errors.Is(err, ErrForbidden) {
				t.Errorf("err = %v, want ErrForbidden", err)
			}
		})
	}
}
