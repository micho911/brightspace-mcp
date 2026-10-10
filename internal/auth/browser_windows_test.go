//go:build windows

package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"strings"
	"testing"
)

func TestSelectWindowsBrowser(t *testing.T) {
	for id, want := range map[string]string{"ChromeHTML": "Google Chrome", "BraveHTML": "Brave", "MSEdgeHTM": "Microsoft Edge"} {
		b, err := selectWindowsBrowser(id)
		if err != nil {
			t.Fatal(err)
		}
		if b.Name() != want {
			t.Errorf("%s selected %s", id, b.Name())
		}
	}
	if _, err := selectWindowsBrowser("FirefoxURL"); err == nil {
		t.Fatal("expected unsupported browser")
	}
}

func TestDecryptChromiumCookieGCM(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	host := "brightspace.example.edu"
	nonce := []byte("123456789012")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(host))
	encrypted := append([]byte("v10"), nonce...)
	encrypted = append(encrypted, gcm.Seal(nil, nonce, append(digest[:], []byte("windows-cookie")...), nil)...)
	got, err := decryptChromiumCookieGCM(key, host, encrypted)
	if err != nil || got != "windows-cookie" {
		t.Fatalf("decrypted %q, %v", got, err)
	}
	if _, err := decryptChromiumCookieGCM(key, host, []byte("v20...")); err == nil || !strings.Contains(err.Error(), "App-Bound") {
		t.Fatalf("v20 error = %v", err)
	}
}
