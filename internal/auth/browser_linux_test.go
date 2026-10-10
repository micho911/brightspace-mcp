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
	oldLookup := linuxSecretServicePassword
	linuxSecretServicePassword = func(string) (string, error) { return "", errSecretNotFound }
	t.Cleanup(func() { linuxSecretServicePassword = oldLookup })
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
	oldLookup := linuxSecretServicePassword
	linuxSecretServicePassword = func(string) (string, error) { return "", errSecretNotFound }
	t.Cleanup(func() { linuxSecretServicePassword = oldLookup })
	b := linuxBrowsers[0]
	if err := b.Unlock(); err != nil {
		t.Fatal(err)
	}
	if len(b.v11Key) != 0 {
		t.Fatalf("v11 key should be absent, got %d bytes", len(b.v11Key))
	}
	want, err := chromiumLinuxKey("peanuts")
	if err != nil {
		t.Fatal(err)
	}
	if string(b.v10Key) != string(want) {
		t.Fatal("did not derive the v10 fallback key")
	}
}

func TestLinuxUnlockReadsBrowserKeyringItem(t *testing.T) {
	store := memSecrets{"Chrome Keys/Chrome Safe Storage": "linux-test-key"}
	old := secrets
	secrets = store
	t.Cleanup(func() { secrets = old })
	oldLookup := linuxSecretServicePassword
	linuxSecretServicePassword = func(string) (string, error) { return "", errSecretNotFound }
	t.Cleanup(func() { linuxSecretServicePassword = oldLookup })
	b := linuxBrowsers[0]
	if err := b.Unlock(); err != nil {
		t.Fatal(err)
	}
	want, err := chromiumLinuxKey("linux-test-key")
	if err != nil {
		t.Fatal(err)
	}
	if string(b.v11Key) != string(want) {
		t.Fatal("did not use browser keyring item")
	}
}

func TestLinuxUnlockReadsChromiumLibsecretItem(t *testing.T) {
	old := secrets
	secrets = memSecrets{}
	t.Cleanup(func() { secrets = old })
	oldLookup := linuxSecretServicePassword
	linuxSecretServicePassword = func(application string) (string, error) {
		if application != "chrome" {
			t.Fatalf("application = %q, want chrome", application)
		}
		return "linux-v11-test-key", nil
	}
	t.Cleanup(func() { linuxSecretServicePassword = oldLookup })

	b := linuxBrowsers[0]
	if err := b.Unlock(); err != nil {
		t.Fatal(err)
	}
	want, err := chromiumLinuxKey("linux-v11-test-key")
	if err != nil {
		t.Fatal(err)
	}
	if string(b.v11Key) != string(want) {
		t.Fatal("did not derive v11 key from the libsecret item")
	}
}

func TestLinuxUnlockTriesBraveSecretServiceNames(t *testing.T) {
	old := secrets
	secrets = memSecrets{}
	t.Cleanup(func() { secrets = old })
	oldLookup := linuxSecretServicePassword
	var tried []string
	linuxSecretServicePassword = func(application string) (string, error) {
		tried = append(tried, application)
		if application == "chromium" {
			return "brave-v11-test-key", nil
		}
		return "", errSecretNotFound
	}
	t.Cleanup(func() { linuxSecretServicePassword = oldLookup })

	b := linuxBrowsers[1]
	if err := b.Unlock(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(tried, ",") != "brave,chromium" {
		t.Fatalf("looked up applications %v, want [brave chromium]", tried)
	}
	want, err := chromiumLinuxKey("brave-v11-test-key")
	if err != nil {
		t.Fatal(err)
	}
	if string(b.v11Key) != string(want) {
		t.Fatal("did not derive v11 key from the Brave libsecret item")
	}
}
