//go:build darwin && cgo

package auth

import (
	"errors"
	"os"
	"testing"
)

// TestKeychainRoundTrip uses the real login Keychain, so it only runs when
// asked for: BRIGHTSPACE_MCP_KEYCHAIN_TEST=1 go test ./internal/auth/
func TestKeychainRoundTrip(t *testing.T) {
	if os.Getenv("BRIGHTSPACE_MCP_KEYCHAIN_TEST") != "1" {
		t.Skip("set BRIGHTSPACE_MCP_KEYCHAIN_TEST=1 to use the real Keychain")
	}
	const service, account = "brightspace-mcp-test", "roundtrip"
	s := osSecrets{}
	t.Cleanup(func() { _ = s.Delete(service, account) })

	if _, err := s.Get(service, account); !errors.Is(err, errSecretNotFound) {
		t.Fatalf("Get before Set: err = %v, want errSecretNotFound", err)
	}
	for _, want := range []string{"first", "second"} {
		if err := s.Set(service, account, want); err != nil {
			t.Fatalf("Set(%q): %v", want, err)
		}
		if got, err := s.Get(service, account); err != nil || got != want {
			t.Fatalf("Get = %q, %v; want %q", got, err, want)
		}
	}
	if err := s.Delete(service, account); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s.Delete(service, account); !errors.Is(err, errSecretNotFound) {
		t.Errorf("Delete twice: err = %v, want errSecretNotFound", err)
	}
}
