//go:build windows

package auth

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type windowsBrowser struct {
	name, dataDir string
	key           []byte
}

var windowsBrowserIDs = map[string]windowsBrowser{
	"chromehtml": {name: "Google Chrome", dataDir: "Google\\Chrome\\User Data"},
	"bravehtml":  {name: "Brave", dataDir: "BraveSoftware\\Brave-Browser\\User Data"},
	"msedgehtm":  {name: "Microsoft Edge", dataDir: "Microsoft\\Edge\\User Data"},
}

func selectWindowsBrowser(progID string) (browser, error) {
	selected := strings.ToLower(progID)
	var b windowsBrowser
	var ok bool
	for id, candidate := range windowsBrowserIDs {
		if strings.HasPrefix(selected, id) {
			b, ok = candidate, true
			break
		}
	}
	if !ok {
		return nil, fmt.Errorf("default browser %q is not supported; supported: Brave, Google Chrome, Microsoft Edge", progID)
	}
	return &b, nil
}

func defaultBrowser() (browser, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\Shell\Associations\UrlAssociations\https\UserChoice`, registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("read default browser from Windows settings: %w", err)
	}
	defer k.Close()
	progID, _, err := k.GetStringValue("ProgId")
	if err != nil {
		return nil, fmt.Errorf("read default browser from Windows settings: %w", err)
	}
	return selectWindowsBrowser(progID)
}

func (b *windowsBrowser) Name() string { return b.name }
func (b *windowsBrowser) Unlock() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(home, "AppData", "Local", b.dataDir, "Local State"))
	if err != nil {
		return fmt.Errorf("read %s encryption settings: %w", b.name, err)
	}
	var state struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("read %s encryption settings: %w", b.name, err)
	}
	blob, err := base64.StdEncoding.DecodeString(state.OSCrypt.EncryptedKey)
	if err != nil {
		return fmt.Errorf("decode %s encryption key: %w", b.name, err)
	}
	if !strings.HasPrefix(string(blob), "DPAPI") {
		return errors.New("unsupported Windows browser encryption key")
	}
	plain, err := dpapiUnprotect(blob[5:])
	if err != nil {
		return fmt.Errorf("Windows could not unlock %s cookies: %w", b.name, err)
	}
	if len(plain) != 32 {
		return fmt.Errorf("unexpected %s cookie key length", b.name)
	}
	b.key = plain
	return nil
}
func (b *windowsBrowser) Open(url string) error {
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Run()
}
func (b *windowsBrowser) Cookies(ctx context.Context, host string) ([]Cookie, error) {
	if b.key == nil {
		return nil, errors.New("browser cookies are locked")
	}
	db, err := b.cookieDB()
	if err != nil {
		return nil, err
	}
	return readChromiumCookiesWithDecrypt(ctx, db, host, func(host string, encrypted []byte) (string, error) {
		return decryptChromiumCookieWindows(b.key, host, encrypted)
	})
}
func (b *windowsBrowser) cookieDB() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return chromiumCookieDB(filepath.Join(home, "AppData", "Local", b.dataDir), b.name)
}
func dpapiUnprotect(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("empty DPAPI input")
	}
	in := windows.DataBlob{Size: uint32(len(input)), Data: &input[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
func decryptChromiumCookieGCM(key []byte, host string, encrypted []byte) (string, error) {
	if bytes.HasPrefix(encrypted, []byte("v20")) {
		return "", errors.New("this cookie uses Windows App-Bound Encryption and cannot be read by brightspace-mcp")
	}
	data, ok := bytes.CutPrefix(encrypted, []byte("v10"))
	if !ok {
		return "", errors.New("unsupported browser cookie encryption")
	}
	if len(data) < 12+16 {
		return "", errors.New("malformed encrypted cookie")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, data[:12], data[12:], nil)
	if err != nil {
		return "", err
	}
	if digest := sha256.Sum256([]byte(host)); bytes.HasPrefix(plain, digest[:]) {
		plain = plain[len(digest):]
	}
	return string(plain), nil
}

func decryptChromiumCookieWindows(key []byte, host string, encrypted []byte) (string, error) {
	if bytes.HasPrefix(encrypted, []byte("v10")) {
		return decryptChromiumCookieGCM(key, host, encrypted)
	}
	if bytes.HasPrefix(encrypted, []byte("v20")) {
		return "", errors.New("this cookie uses Windows App-Bound Encryption and cannot be read by brightspace-mcp")
	}
	plain, err := dpapiUnprotect(encrypted)
	if err != nil {
		return "", fmt.Errorf("cannot decrypt legacy Windows browser cookie: %w", err)
	}
	return string(plain), nil
}
