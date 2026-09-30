package brightspace

import "testing"

func TestParseBaseURL(t *testing.T) {
	valid := map[string]string{
		"https://brightspace.au.dk":                  "https://brightspace.au.dk",
		"brightspace.au.dk":                          "https://brightspace.au.dk",
		"  https://Brightspace.AU.dk/  ":             "https://brightspace.au.dk",
		"https://brightspace.au.dk/d2l/home?ou=6606": "https://brightspace.au.dk",
		"https://school.brightspace.com:8443/":       "https://school.brightspace.com:8443",
	}
	for in, want := range valid {
		got, err := ParseBaseURL(in)
		if err != nil {
			t.Errorf("ParseBaseURL(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseBaseURL(%q) = %q, want %q", in, got, want)
		}
	}

	invalid := []string{
		"",
		"http://brightspace.au.dk",
		"ftp://brightspace.au.dk",
		"https://user:pass@brightspace.au.dk",
		"https://",
		"https://%zz",
	}
	for _, in := range invalid {
		if got, err := ParseBaseURL(in); err == nil {
			t.Errorf("ParseBaseURL(%q) = %q, want error", in, got)
		}
	}
}
