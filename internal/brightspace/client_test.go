package brightspace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWhoAmI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/d2l/api/lp/"+lpVersion+"/users/whoami" {
			http.NotFound(w, r)
			return
		}
		if c, err := r.Cookie("d2lSessionVal"); err != nil || c.Value != "secret" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"Identifier":"42","FirstName":"Ada","LastName":"Lovelace","UniqueName":"au123456"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, []*http.Cookie{{Name: "d2lSessionVal", Value: "secret"}})
	got, err := c.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI: %v", err)
	}
	want := Identity{Identifier: "42", FirstName: "Ada", LastName: "Lovelace", UniqueName: "au123456"}
	if got != want {
		t.Errorf("WhoAmI = %+v, want %+v", got, want)
	}
}

func TestWhoAmIRejectedSession(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"401": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
		"403 html (not logged in)": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
		},
		"200 html login stub": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<script>location.replace("/d2l/login?sessionExpired=1")</script>`))
		},
		"redirect to login": func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/d2l/login", http.StatusFound)
		},
	}
	for name, handler := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(handler)
			defer srv.Close()

			_, err := NewClient(srv.URL, nil).WhoAmI(context.Background())
			if !errors.Is(err, ErrSessionExpired) {
				t.Errorf("err = %v, want ErrSessionExpired", err)
			}
		})
	}
}

func TestWhoAmIServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, nil).WhoAmI(context.Background())
	if err == nil || errors.Is(err, ErrSessionExpired) {
		t.Errorf("err = %v, want a non-session error", err)
	}
}
