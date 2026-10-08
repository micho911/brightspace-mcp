# Local development builds. Releases will be built by GoReleaser.

BINARY := brightspace-mcp
ifeq ($(OS),Windows_NT)
BINARY := brightspace-mcp.exe
endif

.PHONY: build build-signed test

build:
	go build -o $(BINARY) ./cmd/brightspace-mcp

# Optional macOS signing keeps the Keychain trusting this local binary.
build-signed: build
	@test "$$(uname -s)" = Darwin || (echo "build-signed is macOS-only" >&2; exit 1)
	scripts/sign.sh $(BINARY)

test:
	go test ./...

.PHONY: test-race
test-race:
	go test -race ./...
