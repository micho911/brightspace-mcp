# Local development builds. Releases will be built by GoReleaser.

BINARY        := brightspace-mcp
SIGN_IDENTITY ?= brightspace-mcp dev

.PHONY: build test dev-cert

# On macOS, sign with a stable identity so the Keychain keeps trusting the
# binary across rebuilds (see scripts/dev-cert.sh). Without the identity the
# build stays ad-hoc signed and the Keychain asks again after every rebuild.
build:
	go build -o $(BINARY) ./cmd/brightspace-mcp
ifeq ($(shell uname -s),Darwin)
	@if security find-identity -p codesigning | grep -q '"$(SIGN_IDENTITY)"'; then \
		codesign --force --identifier $(BINARY) --sign "$(SIGN_IDENTITY)" $(BINARY) && \
		echo "Signed $(BINARY) with \"$(SIGN_IDENTITY)\"."; \
	else \
		echo "No code-signing identity \"$(SIGN_IDENTITY)\"; $(BINARY) is ad-hoc signed." >&2; \
		echo "Run 'make dev-cert' once so the Keychain stops asking after every rebuild." >&2; \
	fi
endif

test:
	go test -race ./...

dev-cert:
	scripts/dev-cert.sh
