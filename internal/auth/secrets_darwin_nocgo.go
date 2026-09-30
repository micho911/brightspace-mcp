//go:build darwin && !cgo

package auth

// On macOS the Keychain is used through cgo; without it the only fallback is
// /usr/bin/security, which weakens the Keychain's access control. Refuse to
// build instead.
var _ int = "brightspace-mcp must be built with CGO_ENABLED=1 on macOS"
