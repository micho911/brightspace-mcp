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
   **Known gap (fix in progress, see BACKLOG):** `zalando/go-keyring` runs
   `/usr/bin/security`, so the keychain trusts `security`, not our binary.
   Any local program can read the session through it without a prompt. We
   are moving macOS to the native Security API (`keybase/go-keychain`).
   Users must click **Allow**, never **Always Allow**, for the browser's
   `Safe Storage` key: "Always Allow" trusts `security` for every program.
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
- CI must pass: `gofmt`, `go mod tidy -diff`, `go vet` (also `GOOS=darwin`),
  `go test -race`, `go build`.
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
go test -race ./...
go build ./cmd/brightspace-mcp
./brightspace-mcp login https://brightspace.au.dk
```

Claude Code runs the local binary as the `brightspace-dev` MCP server, so
rebuild and then reconnect (`/mcp`) to pick up changes.
