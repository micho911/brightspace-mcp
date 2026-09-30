# brightspace-mcp

An [MCP](https://modelcontextprotocol.io) server for [D2L Brightspace](https://www.d2l.com/brightspace/), written in Go.

Ask your AI assistant (Claude Code, Codex, opencode, Cursor, Claude Desktop, …) about your courses, deadlines and announcements.

> **Status:** early development. Not ready for general use yet.

## Principles

- **Your password never touches this tool.** You sign in to Brightspace yourself in your own browser; only the resulting session is stored, in your operating system's keychain.
- **Read-only by default.** Tools that change anything in Brightspace will be opt-in.
- **One small binary.** No Node.js, no bundled browser.

## Development

Requires Go 1.25 or newer.

```bash
go test ./...
go build ./cmd/brightspace-mcp
```

## License

[MIT](LICENSE)
