# Backlog

Ordered, top first. Each item should fit in one small PR. Move finished items
to **Done** with the PR number. Decisions and conventions are in
[CLAUDE.md](CLAUDE.md).

## Next

Read-only API parity for a student (stacked PRs, in this order). Routes that
need teacher or admin rights are out of scope and listed under Later.

- [ ] `docs`: README tool list and a final parity review

## Later

- [ ] `feat`: show teachers' names in discussion posts (needs a roles lookup that
      does not hand the roster to the assistant, behind the roster opt-in)
- [ ] `feat`: final grades across all courses in one call (needs a live check that
      `/le/{v}/grades/final/values/myGradeValues/` says which course each value belongs to)
- [ ] `feat`: grade weights and categories in `get_grades` (`GradeObject.Weight` needs a newer `le` API version than 1.74)
- [ ] `chore`: real-machine validation of Windows login with Chrome, Edge and
      Brave. Verify behavior with App-Bound cookies and measure session size
      against Credential Manager's 2560-byte limit.
- [ ] `chore`: real-machine validation of Linux login with KDE Wallet; Fedora
      GNOME is already verified.
- [ ] `chore`: verify macOS login/logout after changing the SQLite reader.
- [ ] `feat`: decide whether to support Flatpak/Snap browser installs; if so,
      find their sandboxed cookie databases, verify Secret Service access, and
      test each browser/package combination we claim to support. Document any
      unsupported package formats.
- [ ] `chore`: check the Activity Feed attachment shape on a post that has an
      attachment (the sample feed had none, so attachment names are best-effort)

- [ ] `feat`: show who posted in the Activity Feed (`actor` is only a URL to
      another D2L service, so it needs a lookup on a second host)
- [ ] `chore`: `login` feels slow after signing in (Chromium writes new
      cookies to disk in batches and we read the on-disk database); find out
      how long it takes and say so in the message, or read them sooner
- [ ] `feat`: `login` status command (who is logged in, which instance)
- [ ] `feat`: detect expired session mid-`serve` and tell the assistant how to recover
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

- [x] Fedora GNOME login checks with Chrome, Brave, and Edge; missing
      Secret Service fails clearly (#35)
- [x] Real-machine platform test checklist in [PLATFORM_TESTING.md](PLATFORM_TESTING.md) (#35)
- [x] OS-agnostic development path: shared SQLite reader, platform browser
      adapters, portable unsigned builds, optional macOS signing, and CI on
      Linux, macOS, and Windows (#35)

- [x] Client groundwork: shared pager for both Brightspace page shapes, `ErrForbidden` (a 403 that is not the login page is "no access", not "session expired"), shared course error helpers (#17)

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
- [x] `list_activity_feed` tool: a course's Activity Feed posts, newest first; token minted in memory and sent only to the feed host the course page names; every tool now has a test for `ReadOnlyHint` (#16)
- [x] `list_upcoming` tool: calendar events across active courses, soonest first (#18)
- [x] `list_assignments` tool: dropbox folders with due dates and my own submission status, soonest due first (#19)
- [x] `get_assignment` tool: instructions, attachment names, my submissions, and score and feedback once published (#20)
- [x] `get_course_content` tool: module/topic tree as a flat, depth-limited list (#21)
- [x] `get_content_topic` tool: one content item's description, dates, file name or link, and the linked assignment/quiz/discussion ID (#22)
- [x] `read_course_file` tool: text of a course file (text, HTML, .docx, .pptx), 25 MB limit, in memory only (#23)
- [x] `read_course_file` reads PDF text (`ledongthuc/pdf`, decision 9); scanned PDFs are reported as unreadable (#24)
- [x] `get_grades` tool: my grades, points, percent, comments and final grade in a course (#25)
- [x] `list_quizzes` tool: quizzes with dates, attempts allowed and my own attempts; scores only once published (#26)
- [x] `list_discussion_topics` tool: forums and topics with dates and lock state (#28)
- [x] `read_discussion_posts` tool: newest posts in a topic; own posts as "me", everyone else as Participant A, B, … (decision 6) (#29)
- [x] `get_course_info` tool: course description, dates, semester and department (#30)
- [x] `list_my_groups` tool: my groups (category, member count only) and sections; pager now also accepts bare-array answers (#31)
- [x] `get_unread_counts` tool: unread discussions, unread feedback and quizzes to attempt, per active course (#32)
- [x] `list_awards` tool: my badges and certificates; the awards API version is asked of the instance (`/d2l/api/versions/`) with a fallback (#33)
