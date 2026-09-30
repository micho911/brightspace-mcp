# CLAUDE.md

Guidance for AI coding assistants (and humans) working on this repository.
The work queue lives in [BACKLOG.md](BACKLOG.md).

## What this is

`brightspace-mcp`: an open-source MCP server for D2L Brightspace, written in Go.
First target: Aarhus University (`https://brightspace.au.dk`). It must stay
generic enough for any Brightspace instance.

## Key decisions

These are settled. Change them only deliberately, and record the change here.

1. **Go + official SDK.** `github.com/modelcontextprotocol/go-sdk`. One static
   binary: no Node.js, no bundled browser, no headless Chrome.
2. **We never see the user's password.** The user logs in to Brightspace in
   their own browser. `brightspace-mcp login <url>` reads the session cookies
   from the user's default browser's cookie store. We never automate SSO/MFA
   or type credentials. This is a deliberate contrast with
   RohanMuppa/brightspace-mcp-server (inspiration only; copy no code).
3. **Cookies only, in the OS keychain.** Only cookies set by the Brightspace
   host itself are stored (parent-domain cookies such as `.au.dk` SSO cookies
   are dropped). No files on disk and no config with secrets.
   - **macOS:** the native Security API via `keybase/go-keychain` (cgo). Never
     `/usr/bin/security`: items it creates, and "Always Allow" answers given to
     it, trust `security` for every program. The keychain item's access list
     names our binary. macOS builds need `CGO_ENABLED=1`; without it the build
     fails on purpose (`secrets_darwin_nocgo.go`).
   - **Linux/Windows:** `zalando/go-keyring` (Secret Service / Credential
     Manager, called directly). **Untested, and `login` is macOS-only for
     now.** Neither platform has per-app access control, so any program
     running as the user can read the session. Windows caps a secret at
     2560 bytes. Don't claim support before testing on real machines
     (BACKLOG).
   - All access goes through `secretStore` in `internal/auth/secrets.go`, and
     tests swap in an in-memory store.
   - Only an item's creator may delete it or change its access list, so
     `Set` updates in place (a new build gets an access prompt) and never
     deletes and re-adds.
   - A Keychain item trusts its creator twice: by designated requirement
     (the ACL) and by **Team ID** (the partition list, `teamid:…`). Builds
     without a Team ID (ad-hoc or self-signed) are recorded as `cdhash:…`,
     so every rebuild asks for the keychain password. A self-signed
     certificate fixes the first check but not the second (#9 tried it).
     Local builds are therefore signed with a free **Apple Development**
     certificate (`make build`); releases get Developer ID with GoReleaser.
   - During the beta, `login` recommends **Allow** over **Always Allow**.
     Keep that wording a recommendation, not an order.
4. **Session goes to one origin only.** `ParseBaseURL` reduces input to
   `https://host`. The HTTP client never follows redirects: a redirect, 401,
   or non-JSON response means `ErrSessionExpired`, and the user runs `login`
   again.
5. **Read-only first.** Student read tools come first. Tools that write, and
   teacher tools, come later and are opt-in behind flags. Every tool sets
   `ReadOnlyHint` correctly.
6. **Privacy by default (GDPR).** Return only what the task needs. Roster or
   other students' personal data is off unless explicitly enabled.
7. **Browser support, macOS first.** Chromium browsers (Brave, Chrome, Edge)
   via the `Safe Storage` keychain key and the `sqlite3` CLI (read-only,
   `immutable=1`). Other OSes and Firefox/Safari are backlog items; the
   `browser` interface in `internal/auth` is the extension point.
   macOS first is a starting point, not the goal: the project must become
   OS-agnostic (BACKLOG), so OS-specific code stays behind interfaces and
   build tags, and nothing outside them may depend on Apple tooling.
8. **Distribution (later):** GoReleaser to GitHub Releases, a Homebrew tap, an
   npm wrapper with prebuilt binaries, and MCPB.

## Workflow

- **Trunk-based.** `main` is always shippable. Every change is a small PR,
  one concern per branch (`feat/…`, `fix/…`, `docs/…`, `chore/…`), merged
  within 12–24 h.
- **After every PR** update BACKLOG.md (move the item to Done with its PR
  number) and this file if a decision changed, so a fresh session can pick up
  from these two files alone.
- Conventional-commit PR titles (`feat: …`, `fix: …`).
- CI must pass. On Ubuntu: `gofmt`, `go mod tidy -diff`, `go vet` (also
  `GOOS=windows`), `go test -race`, `go build`. On macOS (the cgo Keychain
  code): `go vet`, `go test -race`, `go build`.
- The real-Keychain test is opt-in:
  `BRIGHTSPACE_MCP_KEYCHAIN_TEST=1 go test ./internal/auth/`.
- Keep the repo ready to open-source at any moment: no secrets, no personal
  data (real cookies, student IDs, course contents) in code, tests or fixtures.

## Layout

```
cmd/brightspace-mcp/   CLI entry point: serve (default), login, logout, version
internal/auth/         browser cookie extraction, session type, keychain store
internal/brightspace/  Valence REST client (base URL validation, requests)
internal/server/       MCP server and tool registrations
internal/version/      build version
```

## Conventions

- In `serve` mode stdout belongs to MCP. Diagnostics go to stderr only.
- Tool descriptions say when to use the tool and what it does *not* do.
- API version constants live in `internal/brightspace` (`lpVersion`, …).
- Tests use `httptest` for Brightspace and in-memory transports for MCP.
  Never call a real Brightspace instance in tests.

## Local dev

```bash
make test
make build      # go build + codesign with your Apple Development certificate (macOS)
./brightspace-mcp login https://brightspace.au.dk
```

One-time setup on macOS:

1. Xcode → Settings → Accounts: add your Apple ID (free is enough), then
   Manage Certificates → **+** → Apple Development.
2. If `security find-identity -v -p codesigning` does not list it as valid,
   import Apple's WWDR G3 intermediate
   (https://www.apple.com/certificateauthority/AppleWWDRCAG3.cer).
3. `make build` (the first signing may ask to use the key: Always Allow is
   fine, it only lets `codesign` use it), then `logout` and `login` once so
   the item is created with your Team ID. If `logout` refuses, delete the
   `brightspace-mcp` item in Keychain Access.

Claude Code runs the local binary as the `brightspace-dev` MCP server, so
rebuild and then reconnect (`/mcp`) to pick up changes.

Checking against the real Brightspace (never in CI; the output is personal
data, so keep it out of commits and PRs):

```bash
scripts/smoke.py whoami   # MCP handshake + one tool call over stdio
# Which apps the saved session trusts (prints the ACL, not the secret):
security dump-keychain -a ~/Library/Keychains/login.keychain-db \
  | awk '/"svce"<blob>="brightspace-mcp"/{f=1} f&&/access:/{p=1} p&&/^keychain:/{exit} p' \
  | grep -E 'requirement:|teamid:|cdhash:'
```

Expected: a requirement for `brightspace-mcp` (never `security`) and a
`teamid:` partition. A `cdhash:` partition means every rebuild will ask for
the keychain password: see the one-time setup above. A plain `go build` is
ad-hoc signed and prompts after every rebuild; `make build` does not.
An old session item created by `/usr/bin/security` can only be removed by
it: `security delete-generic-password -s brightspace-mcp -a session`.
