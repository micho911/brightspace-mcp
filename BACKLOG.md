# Backlog

Ordered, top first. Each item should fit in one small PR. Move finished items
to **Done** with the PR number. Decisions and conventions are in
[CLAUDE.md](CLAUDE.md).

## Next

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
- [x] `serve` uses the saved session; `whoami` tool with login hints on missing/expired session
