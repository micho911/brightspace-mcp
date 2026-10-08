//go:build linux

package auth

import (
	"errors"
	"strings"
	"testing"
)

type failingSecrets struct{ err error }

func (f failingSecrets) Get(string, string) (string, error) { return "", f.err }
func (f failingSecrets) Set(string, string, string) error   { return f.err }
func (f failingSecrets) Delete(string, string) error        { return f.err }

func TestSelectLinuxBrowser(t *testing.T) {
	for id, want := range map[string]string{"google-chrome.desktop": "Google Chrome", "brave-browser.desktop": "Brave", "microsoft-edge.desktop": "Microsoft Edge"} {
		b, err := selectLinuxBrowser(id)
		if err != nil {
			t.Fatal(err)
		}
		if b.Name() != want {
			t.Errorf("%s selected %s, want %s", id, b.Name(), want)
		}
	}
	if _, err := selectLinuxBrowser("firefox.desktop"); err == nil {
		t.Fatal("expected unsupported browser error")
	}
}

func TestLinuxUnlockMissingSecretServiceCredential(t *testing.T) {
	old := secrets
	secrets = failingSecrets{err: errors.New("Secret Service unavailable")}
	t.Cleanup(func() { secrets = old })
	b := linuxBrowsers[0]
	err := b.Unlock()
	if err == nil || !strings.Contains(err.Error(), "Secret Service") {
		t.Fatalf("Unlock error = %v", err)
	}
}

func TestLinuxUnlockUsesDocumentedFallback(t *testing.T) {
	old := secrets
	secrets = memSecrets{}
	t.Cleanup(func() { secrets = old })
	b := linuxBrowsers[0]
	if err := b.Unlock(); err != nil {
		t.Fatal(err)
	}
	if len(b.key) != chromiumKeyLength {
		t.Fatalf("derived key length = %d", len(b.key))
	}
}

func TestLinuxUnlockReadsBrowserKeyringItem(t *testing.T) {
	store := memSecrets{"Chrome Keys/Chrome Safe Storage": "linux-test-key"}
	old := secrets
	secrets = store
	t.Cleanup(func() { secrets = old })
	b := linuxBrowsers[0]
	if err := b.Unlock(); err != nil {
		t.Fatal(err)
	}
	want, err := chromiumKey("linux-test-key")
	if err != nil {
		t.Fatal(err)
	}
	if string(b.key) != string(want) {
		t.Fatal("did not use browser keyring item")
	}
}
