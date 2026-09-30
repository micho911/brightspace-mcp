package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// chromiumBrowser is a Chromium-based browser installed on macOS.
type chromiumBrowser struct {
	name            string
	bundleID        string // lowercase, as macOS records the default browser
	dataDir         string // under ~/Library/Application Support
	keychainService string
	keychainAccount string
	key             []byte
}

var supportedBrowsers = []chromiumBrowser{
	{name: "Brave", bundleID: "com.brave.browser", dataDir: "BraveSoftware/Brave-Browser",
		keychainService: "Brave Safe Storage", keychainAccount: "Brave"},
	{name: "Google Chrome", bundleID: "com.google.chrome", dataDir: "Google/Chrome",
		keychainService: "Chrome Safe Storage", keychainAccount: "Chrome"},
	{name: "Microsoft Edge", bundleID: "com.microsoft.edgemac", dataDir: "Microsoft Edge",
		keychainService: "Microsoft Edge Safe Storage", keychainAccount: "Microsoft Edge"},
}

// defaultBrowser returns the user's default browser if it is supported.
func defaultBrowser() (browser, error) {
	id, err := defaultBrowserID()
	if err != nil {
		return nil, err
	}
	for _, b := range supportedBrowsers {
		if b.bundleID == id {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("your default browser (%s) is not supported yet; supported: Brave, Google Chrome, Microsoft Edge", id)
}

// defaultBrowserID reads the https handler from the Launch Services settings.
// Without an entry, the default browser is Safari.
func defaultBrowserID() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	plist := filepath.Join(home, "Library/Preferences/com.apple.LaunchServices/com.apple.launchservices.secure.plist")
	out, err := exec.Command("plutil", "-convert", "json", "-o", "-", plist).Output()
	if err != nil {
		return "com.apple.safari", nil
	}
	var prefs struct {
		LSHandlers []struct {
			Scheme string `json:"LSHandlerURLScheme"`
			Role   string `json:"LSHandlerRoleAll"`
		}
	}
	if err := json.Unmarshal(out, &prefs); err != nil {
		return "", fmt.Errorf("read default browser: %w", err)
	}
	for _, h := range prefs.LSHandlers {
		if h.Scheme == "https" && h.Role != "" {
			return strings.ToLower(h.Role), nil
		}
	}
	return "com.apple.safari", nil
}

func (b *chromiumBrowser) Name() string { return b.name }

// Unlock fetches the cookie password from the Keychain; macOS asks the user
// to allow this.
func (b *chromiumBrowser) Unlock() error {
	password, err := secrets.Get(b.keychainService, b.keychainAccount)
	if err != nil {
		return fmt.Errorf("access to %q in the Keychain was not granted: %w", b.keychainService, err)
	}
	b.key, err = chromiumKey(password)
	return err
}

func (b *chromiumBrowser) Open(url string) error {
	return exec.Command("open", "-b", b.bundleID, url).Run()
}

func (b *chromiumBrowser) Cookies(ctx context.Context, host string) ([]Cookie, error) {
	if b.key == nil {
		return nil, errors.New("browser cookies are locked")
	}
	db, err := b.cookieDB()
	if err != nil {
		return nil, err
	}
	return readChromiumCookies(ctx, db, host, b.key)
}

// cookieDB returns the cookie database of the most recently used profile.
func (b *chromiumBrowser) cookieDB() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library/Application Support", b.dataDir)

	profile := "Default"
	if data, err := os.ReadFile(filepath.Join(dir, "Local State")); err == nil {
		var state struct {
			Profile struct {
				LastUsed string `json:"last_used"`
			} `json:"profile"`
		}
		if json.Unmarshal(data, &state) == nil && state.Profile.LastUsed != "" &&
			filepath.Base(state.Profile.LastUsed) == state.Profile.LastUsed {
			profile = state.Profile.LastUsed
		}
	}

	for _, p := range []string{"Cookies", "Network/Cookies"} {
		path := filepath.Join(dir, profile, p)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no %s cookie database found for profile %q", b.name, profile)
}
