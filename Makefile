# Local development builds. Releases will be built by GoReleaser.

BINARY := brightspace-mcp

.PHONY: build test

# On macOS, sign with an Apple Development certificate so the Keychain keeps
# trusting the binary across rebuilds (see scripts/sign.sh).
build:
	go build -o $(BINARY) ./cmd/brightspace-mcp
ifeq ($(shell uname -s),Darwin)
	@scripts/sign.sh $(BINARY)
endif

test:
	go test -race ./...
