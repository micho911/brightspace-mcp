// Package version reports the version of the running binary.
package version

import "runtime/debug"

// version is set at release time with
// -ldflags "-X github.com/micho911/brightspace-mcp/internal/version.version=v1.2.3".
var version = ""

// String returns the release version if set, otherwise the version the Go
// toolchain stamped into the binary (the tag for `go install ...@vX.Y.Z`, a
// git pseudo-version for local builds), or "dev" when none is available.
func String() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
