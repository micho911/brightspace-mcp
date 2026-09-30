# Backlog

Ordered, top first. Each item should fit in one small PR. Move finished items
to **Done** with the PR number. Decisions and conventions are in
[CLAUDE.md](CLAUDE.md).

## Next

- [ ] `feat`: native macOS keychain (`keybase/go-keychain`, cgo) instead of
      `go-keyring`, which shells out to `/usr/bin/security`. Today any local
      program can read the saved session through `security` without a prompt,
      and "Always Allow" on the browser key trusts `security` for everyone.
      With the native API, access is bound to the `brightspace-mcp` binary and
      the prompt names it. `Save` deletes and re-adds the item so old
      `security`-owned items get a fresh ACL. Needs a `macos-latest` CI job;
      Linux/Windows keep `go-keyring`.
- [ ] `chore`: stable local code signing (self-signed cert, `make build`) so
      the keychain does not re-prompt after every dev rebuild
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
- [ ] `feat`: Linux and Windows browser cookie support
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
- [x] `login` tells users to click Allow, not Always Allow
