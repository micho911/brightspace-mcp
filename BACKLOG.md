# Backlog

Ordered, top first. Each item should fit in one small PR. Move finished items
to **Done** with the PR number. Decisions and conventions are in
[CLAUDE.md](CLAUDE.md).

## Next

- [ ] `feat`: `list_upcoming` tool (calendar/due dates across courses)
- [ ] `feat`: `list_assignments` tool (dropbox folders with due dates, my submission status)
- [ ] `feat`: `get_course_content` tool (content modules/topics tree)

## Later

- [ ] `chore`: **become OS-agnostic.** Today login, the session store and dev
      signing are macOS-specific (Keychain partitions/Team ID, Apple
      Development certificate, Chromium `Safe Storage` key). Before claiming
      Windows/Linux support (items below):
      - keep every OS detail behind the existing interfaces (`secretStore`,
        `browser`) and build tags; no Apple specifics in shared code or tools;
      - the dev loop (`make build`, `make test`, smoke checks) must work on
        every OS without an Apple account; signing stays an optional macOS step;
      - CI runs the tests on Linux, macOS and Windows;
      - release signing per OS (Developer ID/notarization, Windows
        Authenticode) moves to GoReleaser.
- [ ] `chore`: check the Activity Feed attachment shape on a post that has an
      attachment (the sample feed had none, so attachment names are best-effort)
- [ ] `feat`: show who posted in the Activity Feed (`actor` is only a URL to
      another D2L service, so it needs a lookup on a second host)
- [ ] `chore`: `login` feels slow after signing in (Chromium writes new
      cookies to disk in batches and we read the on-disk database); find out
      how long it takes and say so in the message, or read them sooner
- [ ] `feat`: grades tool (my grades only)
- [ ] `feat`: download/read course files (with size limit)
- [ ] `feat`: `login` status command (who is logged in, which instance)
- [ ] `feat`: detect expired session mid-`serve` and tell the assistant how to recover
- [ ] `feat`: Windows support. **Must be tested on a real Windows machine
      before we claim support.** Open questions:
      - Chrome 127+ uses App-Bound Encryption for cookies (only Chrome can
        decrypt them). Check Chrome, Edge and Brave; reading cookies from the
        browser may be impossible, which would need a different login method
        (decide in CLAUDE.md first; it must still never see the password).
      - Credential Manager limits a secret to 2560 bytes (`ErrSetDataTooBig`).
        Measure the real session size; if needed, store fewer cookies or
        split the item.
      - Credential Manager has no per-app access control: any program running
        as the user can read the session. Document it; no library fixes it.
- [ ] `feat`: Linux support. Test on a real desktop (GNOME and KDE):
      - Secret Service has no per-app access control (same caveat as Windows).
      - It is often missing (WSL, headless, minimal desktops); `login` must
        fail with a clear message.
      - Chromium's cookie key lives in Secret Service, or is the fixed
        fallback `peanuts` when there is none.
- [ ] `feat`: Firefox and Safari support
- [ ] `feat`: choose browser/profile explicitly (`login --browser brave --profile …`)
- [ ] `feat`: opt-in write tools (e.g. submit to dropbox) behind a flag
- [ ] `feat`: opt-in teacher tools; roster data off by default (GDPR)
- [ ] `chore`: GoReleaser + GitHub Releases
- [ ] `chore`: Homebrew tap
- [ ] `chore`: npm wrapper with prebuilt binaries
- [ ] `chore`: MCPB bundle
- [ ] `docs`: README install + setup for Claude Code, Codex, Cursor, Claude Desktop
- [ ] `chore`: CONTRIBUTING.md and issue templates before announcing publicly

## Done

- [x] Bootstrap: README, MIT license, .gitignore
- [x] MCP server skeleton with `get_server_info` (#1)
- [x] `login` / `logout`: session from the default browser, stored in the keychain (#2)
- [x] CLAUDE.md and BACKLOG.md (#3)
- [x] `serve` uses the saved session; `whoami` tool with login hints on missing/expired session (#4)
- [x] `login` notes the beta and recommends Allow over Always Allow (#5)
- [x] Native macOS keychain (`keybase/go-keychain`); items owned by our binary, not `/usr/bin/security`; macOS CI job (#6)
- [x] Windows/Linux keychain risks and test plan (#7)
- [x] `scripts/smoke.py` and live verification steps for local dev (#8)
- [x] Stable local code signing: `make dev-cert` + `make build` (#9)
- [x] `list_courses` tool: my course enrollments, active by default, newest first (#10)
- [x] Fix doubled course URLs: resolve `HomeUrl` against the instance (#11)
- [x] Sign dev builds with an Apple Development certificate (Team ID partition); drop the self-signed `make dev-cert` (#12)
- [x] `list_announcements` tool: News per course, newest first (#14)
- [x] Decision 4 allows a short-lived in-memory token for the Activity Feed host the instance names; Activity Feed verified on AU (#15)
- [x] `list_activity_feed` tool: a course's Activity Feed posts, newest first; token minted in memory and sent only to the feed host the course page names; every tool now has a test for `ReadOnlyHint` (PR pending)
