package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChromiumCookieDBProfileDiscovery(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "Profile 2", "Network", "Cookies")
	if err := os.MkdirAll(filepath.Dir(profile), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Local State"), []byte(`{"profile":{"last_used":"Profile 2"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := chromiumCookieDB(dir, "Chrome")
	if err != nil {
		t.Fatal(err)
	}
	if got != profile {
		t.Fatalf("cookie DB = %q, want %q", got, profile)
	}
}

func TestChromiumCookieDBRejectsProfileTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Local State"), []byte(`{"profile":{"last_used":"../outside"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := chromiumCookieDB(dir, "Chrome"); err == nil {
		t.Fatal("expected missing profile database error")
	}
}
