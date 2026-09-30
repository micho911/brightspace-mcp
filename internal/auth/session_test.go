package auth

import (
	"net/http"
	"reflect"
	"testing"
)

func TestHostCookies(t *testing.T) {
	browser := []*http.Cookie{
		{Name: "d2lSessionVal", Value: "a", Domain: "brightspace.au.dk"},
		{Name: "d2lSecureSessionVal", Value: "b", Domain: ".Brightspace.AU.dk"},
		{Name: "sso", Value: "c", Domain: ".au.dk"},
		{Name: "entra", Value: "d", Domain: "login.microsoftonline.com"},
		{Name: "lookalike", Value: "e", Domain: "evilbrightspace.au.dk"},
		{Name: "sub", Value: "f", Domain: "cdn.brightspace.au.dk"},
	}

	got := hostCookies("brightspace.au.dk", browser)
	want := []Cookie{
		{Name: "d2lSessionVal", Value: "a"},
		{Name: "d2lSecureSessionVal", Value: "b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("hostCookies = %+v, want %+v", got, want)
	}
}
