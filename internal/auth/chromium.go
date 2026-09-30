package auth

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
)

// Chromium-based browsers (Chrome, Brave, Edge, …) encrypt cookie values
// with AES-128-CBC. On macOS the key is derived from a random password the
// browser keeps in the Keychain.
const (
	chromiumSalt       = "saltysalt"
	chromiumIterations = 1003
	chromiumKeyLength  = 16
)

var errWrongKey = errors.New("cannot decrypt browser cookies (wrong key)")

// chromiumKey derives the cookie encryption key from the browser's password.
func chromiumKey(password string) ([]byte, error) {
	return pbkdf2.Key(sha1.New, password, []byte(chromiumSalt), chromiumIterations, chromiumKeyLength)
}

// decryptChromiumCookie decrypts one "v10" encrypted cookie value.
func decryptChromiumCookie(key []byte, hostKey string, encrypted []byte) (string, error) {
	ciphertext, ok := bytes.CutPrefix(encrypted, []byte("v10"))
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

var validHost = regexp.MustCompile(`^[a-z0-9.-]+$`)

// readChromiumCookies returns the cookies a Chromium cookie database holds
// for host, decrypted with key. The database is opened read-only, so this
// works while the browser is running.
func readChromiumCookies(ctx context.Context, dbPath, host string, key []byte) ([]Cookie, error) {
	// host ends up in the SQL query; allow hostname characters only.
	if !validHost.MatchString(host) {
		return nil, fmt.Errorf("invalid host %q", host)
	}
	query := "select host_key, name, value, hex(encrypted_value) as encrypted from cookies " +
		"where host_key like '%" + host + "'"
	db := url.URL{Scheme: "file", Path: dbPath, RawQuery: "immutable=1"}

	out, err := exec.CommandContext(ctx, "sqlite3", "-readonly", "-json", db.String(), query).Output()
	if err != nil {
		return nil, fmt.Errorf("read browser cookies: %w", err)
	}

	var rows []struct {
		HostKey   string `json:"host_key"`
		Name      string `json:"name"`
		Value     string `json:"value"`
		Encrypted string `json:"encrypted"`
	}
	if len(bytes.TrimSpace(out)) > 0 {
		if err := json.Unmarshal(out, &rows); err != nil {
			return nil, fmt.Errorf("read browser cookies: %w", err)
		}
	}

	cookies := make([]*http.Cookie, 0, len(rows))
	for _, r := range rows {
		value := r.Value
		if r.Encrypted != "" {
			encrypted, err := hex.DecodeString(r.Encrypted)
			if err != nil {
				return nil, fmt.Errorf("read browser cookies: %w", err)
			}
			if value, err = decryptChromiumCookie(key, r.HostKey, encrypted); err != nil {
				return nil, err
			}
		}
		cookies = append(cookies, &http.Cookie{Name: r.Name, Value: value, Domain: r.HostKey})
	}
	return hostCookies(host, cookies), nil
}
