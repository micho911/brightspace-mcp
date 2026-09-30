# Backlog

Ordered, top first. Each item should fit in one small PR. Move finished items
to **Done** with the PR number. Decisions and conventions are in
[CLAUDE.md](CLAUDE.md).

## Next

- [ ] `chore`: stable local code signing (self-signed cert, `make build`) so
      the keychain does not re-prompt after every dev rebuild (ad-hoc builds
      are trusted by cdhash). Release signing/notarization comes with GoReleaser.
- [ ] `feat`: `list_courses` tool (my enrollments: name, code, id, active/dates)
- [ ] `feat`: `list_announcements` tool (news per course, recent first)
- [ ] `feat`: `list_upcoming` tool (calendar/due dates across courses)
- [ ] `feat`: `list_assignments` tool (dropbox folders with due dates, my submission status)
- [ ] `feat`: `get_course_content` tool (content modules/topics tree)

## Later

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
