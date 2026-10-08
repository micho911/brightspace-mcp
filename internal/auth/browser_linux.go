//go:build linux

package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type linuxBrowser struct {
	name, id, dataDir string
	keyItems          [][2]string
	key               []byte
}

func selectLinuxBrowser(id string) (*linuxBrowser, error) {
	selected := strings.ToLower(strings.TrimSpace(id))
	for i := range linuxBrowsers {
		if selected != "" && strings.Contains(selected, linuxBrowsers[i].id) {
			return &linuxBrowsers[i], nil
		}
	}
	return nil, fmt.Errorf("default browser %q is not supported; supported: Brave, Google Chrome, Microsoft Edge", selected)
}

var linuxBrowsers = []linuxBrowser{
	{name: "Google Chrome", id: "google-chrome", dataDir: "google-chrome", keyItems: [][2]string{{"Chrome Keys", "Chrome Safe Storage"}, {"chrome", "Chrome Safe Storage"}}},
	{name: "Brave", id: "brave-browser", dataDir: "BraveSoftware/Brave-Browser", keyItems: [][2]string{{"Brave Keys", "Brave Safe Storage"}, {"brave", "Brave Safe Storage"}, {"Chrome Keys", "Chrome Safe Storage"}}},
	{name: "Microsoft Edge", id: "microsoft-edge", dataDir: "microsoft-edge", keyItems: [][2]string{{"Microsoft Edge Keys", "Microsoft Edge Safe Storage"}, {"Chrome Keys", "Chrome Safe Storage"}}},
}

func defaultBrowser() (browser, error) {
	id, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read the default browser (xdg-settings): %w", err)
	}
	return selectLinuxBrowser(string(id))
}

func (b *linuxBrowser) Name() string { return b.name }
func (b *linuxBrowser) Unlock() error {
	password := ""
	for _, item := range b.keyItems {
		var err error
		password, err = secrets.Get(item[0], item[1])
		if err == nil {
			break
		}
		if !errors.Is(err, errSecretNotFound) {
			return fmt.Errorf("cannot access the desktop Secret Service (GNOME Keyring or KDE Wallet): %w", err)
		}
	}
	if password == "" {
		password = "peanuts"
	}
	key, err := chromiumKey(password)
	b.key = key
	return err
}
func (b *linuxBrowser) Open(url string) error { return exec.Command("xdg-open", url).Run() }
func (b *linuxBrowser) Cookies(ctx context.Context, host string) ([]Cookie, error) {
	if b.key == nil {
		return nil, errors.New("browser cookies are locked")
	}
	db, err := b.cookieDB()
	if err != nil {
		return nil, err
	}
	return readChromiumCookies(ctx, db, host, b.key)
}
func (b *linuxBrowser) cookieDB() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", b.dataDir)
	return chromiumCookieDB(dir, b.name)
}
