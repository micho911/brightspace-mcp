package auth

import (
	"encoding/hex"
	"errors"
	"testing"
)

// Vectors generated independently with Python's hashlib and OpenSSL from the
// made-up password "test-safe-storage-password".
const (
	testKey = "8d980c26091f4ec99749af7dd4090b2e"
	// SHA-256("brightspace.example.edu") + "session-value-123"
	testEncrypted = "763130840ec3fc4ebf150e331f15840d508540c649cdb429d2390357017a6aa5b7abdf11be93e919b46e0b4125388cbad3da101a0b7827ff35cfd283648ff8d93f5f75"
	// "legacy-value", without the host digest used by older browsers
	testEncryptedNoDigest = "763130daa1f51916f3f9ea320ed72860a72bcf"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestChromiumKey(t *testing.T) {
	key, err := chromiumKey("test-safe-storage-password")
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(key); got != testKey {
		t.Errorf("chromiumKey = %s, want %s", got, testKey)
	}
}

func TestDecryptChromiumCookie(t *testing.T) {
	key := mustHex(t, testKey)

	got, err := decryptChromiumCookie(key, "brightspace.example.edu", mustHex(t, testEncrypted))
	if err != nil || got != "session-value-123" {
		t.Errorf("with host digest: got %q, %v; want %q", got, err, "session-value-123")
	}

	got, err = decryptChromiumCookie(key, "brightspace.example.edu", mustHex(t, testEncryptedNoDigest))
	if err != nil || got != "legacy-value" {
		t.Errorf("without host digest: got %q, %v; want %q", got, err, "legacy-value")
	}
}

func TestDecryptChromiumCookieRejects(t *testing.T) {
	key := mustHex(t, testKey)
	wrongKey := mustHex(t, "00000000000000000000000000000000")

	if _, err := decryptChromiumCookie(wrongKey, "brightspace.example.edu", mustHex(t, testEncrypted)); !errors.Is(err, errWrongKey) {
		t.Errorf("wrong key: err = %v, want errWrongKey", err)
	}
	for name, enc := range map[string][]byte{
		"unknown version": append([]byte("v11"), mustHex(t, testEncrypted)[3:]...),
		"truncated":       mustHex(t, testEncrypted)[:20],
		"empty":           []byte("v10"),
	} {
		if _, err := decryptChromiumCookie(key, "brightspace.example.edu", enc); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
