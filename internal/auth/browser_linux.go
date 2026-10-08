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

	secretservice "github.com/zalando/go-keyring/secret_service"
)

type linuxBrowser struct {
	name, id, dataDir string
	keyringApps       []string
	keyItems          [][2]string
	v11Key            []byte
	v10Key            []byte
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
	{name: "Google Chrome", id: "google-chrome", dataDir: "google-chrome", keyringApps: []string{"chrome"}, keyItems: [][2]string{{"Chrome Keys", "Chrome Safe Storage"}, {"chrome", "Chrome Safe Storage"}}},
	{name: "Brave", id: "brave-browser", dataDir: "BraveSoftware/Brave-Browser", keyringApps: []string{"brave", "chromium"}, keyItems: [][2]string{{"Brave Keys", "Brave Safe Storage"}, {"brave", "Brave Safe Storage"}, {"Chrome Keys", "Chrome Safe Storage"}}},
	{name: "Microsoft Edge", id: "microsoft-edge", dataDir: "microsoft-edge", keyringApps: []string{"microsoft-edge", "edge", "chromium"}, keyItems: [][2]string{{"Microsoft Edge Keys", "Microsoft Edge Safe Storage"}, {"Chrome Keys", "Chrome Safe Storage"}}},
}

var linuxSecretServicePassword = chromiumSecretServicePassword

// chromiumSecretServicePassword reads Chromium's current libsecret entry.
// Newer Chromium stores it with an application attribute rather than the
// service/account attributes used by go-keyring.Get.
func chromiumSecretServicePassword(application string) (string, error) {
	svc, err := secretservice.NewSecretService()
	if err != nil {
		return "", err
	}
	defer svc.Conn.Close()

	collection := svc.GetLoginCollection()
	if err := svc.Unlock(collection.Path()); err != nil {
		return "", err
	}
	items, err := svc.SearchItems(collection, map[string]string{"application": application})
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", errSecretNotFound
	}

	session, err := svc.OpenSession()
	if err != nil {
		return "", err
	}
	defer svc.Close(session)

	if err := svc.Unlock(items[0]); err != nil {
		return "", err
	}
	secret, err := svc.GetSecret(items[0], session.Path())
	if err != nil {
		return "", err
	}
	return string(secret.Value), nil
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
	for _, application := range b.keyringApps {
		var err error
		password, err = linuxSecretServicePassword(application)
		if err == nil && password != "" {
			break
		}
		if err != nil && !errors.Is(err, errSecretNotFound) {
			return fmt.Errorf("cannot access the desktop Secret Service (GNOME Keyring or KDE Wallet): %w", err)
		}
	}
	for _, item := range b.keyItems {
		if password != "" {
			break
		}
		var err error
		password, err = secrets.Get(item[0], item[1])
		if err == nil {
			break
		}
		if !errors.Is(err, errSecretNotFound) {
			return fmt.Errorf("cannot access the desktop Secret Service (GNOME Keyring or KDE Wallet): %w", err)
		}
	}
	var v11Key []byte
	if password != "" {
		key, err := chromiumLinuxKey(password)
		if err != nil {
			return err
		}
		v11Key = key
	}
	v10Key, err := chromiumLinuxKey("peanuts")
	if err != nil {
		return err
	}
	b.v11Key = v11Key
	b.v10Key = v10Key
	return nil
}

func (b *linuxBrowser) Open(url string) error { return exec.Command("xdg-open", url).Run() }

func (b *linuxBrowser) Cookies(ctx context.Context, host string) ([]Cookie, error) {
	if b.v10Key == nil {
		return nil, errors.New("browser cookies are locked")
	}
	db, err := b.cookieDB()
	if err != nil {
		return nil, err
	}
	return readChromiumCookiesWithDecrypt(ctx, db, host, func(host string, encrypted []byte) (string, error) {
		return decryptChromiumCookieLinux(b.v11Key, b.v10Key, host, encrypted)
	})
}

func (b *linuxBrowser) cookieDB() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", b.dataDir)
	return chromiumCookieDB(dir, b.name)
}
