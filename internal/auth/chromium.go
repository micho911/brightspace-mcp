package auth

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	_ "modernc.org/sqlite"
)

// Chromium-based browsers (Chrome, Brave, Edge, …) encrypt cookie values
// with AES-128-CBC. On macOS the key is derived from a random password the
// browser keeps in the Keychain.
const (
	chromiumSalt            = "saltysalt"
	chromiumIterations      = 1003
	chromiumLinuxIterations = 1
	chromiumKeyLength       = 16
)

var errWrongKey = errors.New("cannot decrypt browser cookies (wrong key)")

// chromiumKey derives the cookie encryption key from the browser's password.
func chromiumKey(password string) ([]byte, error) {
	return pbkdf2.Key(sha1.New, password, []byte(chromiumSalt), chromiumIterations, chromiumKeyLength)
}

// chromiumLinuxKey derives the Linux Chromium cookie key. Linux uses one
// PBKDF2 iteration; macOS uses chromiumKey's 1003 iterations.
func chromiumLinuxKey(password string) ([]byte, error) {
	return pbkdf2.Key(sha1.New, password, []byte(chromiumSalt), chromiumLinuxIterations, chromiumKeyLength)
}

// decryptChromiumCookie decrypts one "v10" encrypted cookie value.
func decryptChromiumCookie(key []byte, hostKey string, encrypted []byte) (string, error) {
	return decryptChromiumCookieVersion(key, hostKey, encrypted, "v10")
}

func decryptChromiumCookieVersion(key []byte, hostKey string, encrypted []byte, version string) (string, error) {
	ciphertext, ok := bytes.CutPrefix(encrypted, []byte(version))
	if !ok {
		return "", errors.New("unsupported browser cookie encryption")
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return "", errors.New("malformed encrypted cookie")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plaintext := make([]byte, len(ciphertext))
	iv := bytes.Repeat([]byte{' '}, aes.BlockSize)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)

	// PKCS#7 padding; invalid padding means the key was wrong.
	pad := int(plaintext[len(plaintext)-1])
	if pad == 0 || pad > aes.BlockSize {
		return "", errWrongKey
	}
	for _, b := range plaintext[len(plaintext)-pad:] {
		if int(b) != pad {
			return "", errWrongKey
		}
	}
	plaintext = plaintext[:len(plaintext)-pad]

	// Newer Chromium versions prefix the value with SHA-256 of the cookie's
	// host, binding it to that host.
	if digest := sha256.Sum256([]byte(hostKey)); bytes.HasPrefix(plaintext, digest[:]) {
		plaintext = plaintext[len(digest):]
	}
	return string(plaintext), nil
}

// decryptChromiumCookieLinux handles both Linux cookie formats: v10 uses
// Chromium's built-in "peanuts" key, while v11 uses the key from Secret Service.
func decryptChromiumCookieLinux(v11Key, v10Key []byte, hostKey string, encrypted []byte) (string, error) {
	switch {
	case bytes.HasPrefix(encrypted, []byte("v10")):
		return decryptChromiumCookieVersion(v10Key, hostKey, encrypted, "v10")
	case bytes.HasPrefix(encrypted, []byte("v11")):
		if len(v11Key) != chromiumKeyLength {
			return "", errors.New("Chromium v11 cookie key was not found in the desktop Secret Service")
		}
		return decryptChromiumCookieVersion(v11Key, hostKey, encrypted, "v11")
	default:
		return "", errors.New("unsupported browser cookie encryption")
	}
}

var validHost = regexp.MustCompile(`^[a-z0-9.-]+$`)

func chromiumCookieDB(dir, browserName string) (string, error) {
	profile := "Default"
	if data, err := os.ReadFile(filepath.Join(dir, "Local State")); err == nil {
		var state struct {
			Profile struct {
				LastUsed string `json:"last_used"`
			} `json:"profile"`
		}
		if json.Unmarshal(data, &state) == nil && state.Profile.LastUsed != "" && filepath.Base(state.Profile.LastUsed) == state.Profile.LastUsed {
			profile = state.Profile.LastUsed
		}
	}
	for _, suffix := range []string{"Cookies", "Network/Cookies"} {
		p := filepath.Join(dir, profile, suffix)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no %s cookie database found for profile %q", browserName, profile)
}

// readChromiumCookies returns the cookies a Chromium cookie database holds
// for host, decrypted with key. The database is opened read-only, so this
// works while the browser is running.
func readChromiumCookies(ctx context.Context, dbPath, host string, key []byte) ([]Cookie, error) {
	return readChromiumCookiesWithDecrypt(ctx, dbPath, host, func(host string, encrypted []byte) (string, error) {
		return decryptChromiumCookie(key, host, encrypted)
	})
}

func readChromiumCookiesWithDecrypt(ctx context.Context, dbPath, host string, decrypt func(host string, encrypted []byte) (string, error)) ([]Cookie, error) {
	// host ends up in the SQL query; allow hostname characters only.
	if !validHost.MatchString(host) {
		return nil, fmt.Errorf("invalid host %q", host)
	}
	dbURL := url.URL{Scheme: "file", Path: filepath.ToSlash(dbPath), RawQuery: "mode=ro&immutable=1"}
	db, err := sql.Open("sqlite", dbURL.String())
	if err != nil {
		return nil, fmt.Errorf("read browser cookies: %w", err)
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, "SELECT host_key, name, value, encrypted_value FROM cookies WHERE host_key LIKE ?", "%"+host)
	if err != nil {
		return nil, fmt.Errorf("read browser cookies: %w", err)
	}
	defer rows.Close()

	var cookies []*http.Cookie
	for rows.Next() {
		var hostKey, name, value string
		var encrypted []byte
		if err := rows.Scan(&hostKey, &name, &value, &encrypted); err != nil {
			return nil, fmt.Errorf("read browser cookies: %w", err)
		}
		if len(encrypted) > 0 {
			if value, err = decrypt(hostKey, encrypted); err != nil {
				return nil, err
			}
		}
		cookies = append(cookies, &http.Cookie{Name: name, Value: value, Domain: hostKey})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read browser cookies: %w", err)
	}
	return hostCookies(host, cookies), nil
}
