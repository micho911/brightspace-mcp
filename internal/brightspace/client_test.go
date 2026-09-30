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

func TestMyCoursesFollowsPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/d2l/api/lp/"+lpVersion+"/enrollments/myenrollments/" ||
			r.URL.Query().Get("orgUnitTypeId") != courseOfferingType {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("bookmark") == "" {
			_, _ = w.Write([]byte(`{"PagingInfo":{"Bookmark":"1001","HasMoreItems":true},"Items":[
				{"OrgUnit":{"Id":1001,"Name":"Course A","Code":"A","HomeUrl":"/d2l/home/1001"},
				 "Access":{"IsActive":true,"CanAccess":true,"StartDate":"2026-09-01T00:00:00.000Z","EndDate":null,"ClasslistRoleName":"Student"}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"PagingInfo":{"Bookmark":null,"HasMoreItems":false},"Items":[
			{"OrgUnit":{"Id":1002,"Name":"Course B","Code":"B","HomeUrl":null},
			 "Access":{"IsActive":false,"CanAccess":false,"StartDate":null,"EndDate":null,"ClasslistRoleName":null}}]}`))
	}))
	defer srv.Close()

	got, err := NewClient(srv.URL, nil).MyCourses(context.Background())
	if err != nil {
		t.Fatalf("MyCourses: %v", err)
	}
	if len(got) != 2 || got[0].OrgUnit.ID != 1001 || got[1].OrgUnit.ID != 1002 {
		t.Fatalf("MyCourses = %+v, want courses 1001 and 1002", got)
	}
	a := got[0].Access
	if !a.IsActive || a.RoleName != "Student" || a.StartDate == nil || a.StartDate.Month() != 9 || a.EndDate != nil {
		t.Errorf("course A access = %+v", a)
	}
}
