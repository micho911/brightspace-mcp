# brightspace-mcp

An [MCP](https://modelcontextprotocol.io) server for [D2L Brightspace](https://www.d2l.com/brightspace/), written in Go.

Ask your AI assistant (Claude Code, Codex, opencode, Cursor, Claude Desktop, …) about your courses, deadlines and announcements.

> **Status:** early development. Not ready for general use yet.

## Principles

- **Your password never touches this tool.** You sign in to Brightspace yourself in your own browser; only the resulting session is stored, in your operating system's keychain.
- **Read-only by default.** Tools that change anything in Brightspace will be opt-in.
- **One small binary.** No Node.js, no bundled browser.

## Tools

All tools are read-only and show only your own data. Other people's names
are never shared (discussion posts say "Participant A, B, …"; group members
are only counted).

| Tool | What it answers |
|------|-----------------|
| `whoami` | Who am I logged in as? |
| `list_courses` | Which courses am I in? |
| `get_course_info` | What is this course (description, semester, department)? |
| `list_upcoming` | What is due or on in the next days, across my courses? |
| `get_unread_counts` | What have I missed (unread posts, feedback, quizzes to try)? |
| `list_announcements` | What did teachers announce (News)? |
| `list_activity_feed` | What did teachers post in the course Activity Feed? |
| `get_course_content` | What is in the course (modules, files, links)? |
| `get_content_topic` | Details of one content item |
| `read_course_file` | The text of a file (text, HTML, Word, PowerPoint, PDF with text) |
| `list_assignments` | Which assignments, when due, did I submit? |
| `get_assignment` | Instructions, my submission, score and feedback |
| `get_grades` | My grades and final grade in a course |
| `list_quizzes` | Quizzes, dates, my attempts (scores once published) |
| `list_discussion_topics` | Which discussions does the course have? |
| `read_discussion_posts` | What is being said in a discussion |
| `list_my_groups` | Which group and section am I in? |
| `list_awards` | My badges and certificates |
| `get_server_info` | Version of this server |

## Development

Requires Go 1.26 or newer.

```bash
go test ./...
go build ./cmd/brightspace-mcp
```

On macOS, build with `make build`. It signs the binary with your Apple
Development certificate, so the Keychain does not ask for your password again
after every rebuild (setup in [CLAUDE.md](CLAUDE.md#local-dev)).

## License

[MIT](LICENSE)
