# brightspace-mcp

An [MCP](https://modelcontextprotocol.io) server for [D2L Brightspace](https://www.d2l.com/brightspace/), written in Go.

Ask your AI assistant (Claude Code, Codex, opencode, Cursor, Claude Desktop, …) about your courses, deadlines and announcements.

> **Status:** early development. Not ready for general use yet.

## Principles

- **Your password never touches this tool.** You sign in to Brightspace yourself in your own browser; only the resulting session is stored, in your operating system's keychain.
- **Read-only by default.** Tools that change anything in Brightspace will be opt-in.
- **One small binary.** No Node.js, no bundled browser.

## Development

Requires Go 1.26 or newer.

```bash
go test ./...
go build ./cmd/brightspace-mcp
```

`make build` works on macOS, Linux, and Windows. Login reads cookies from
Brave, Chrome, or Edge on macOS and Linux. Windows has an implementation for
DPAPI and AES GCM cookies, but browser and Credential Manager behavior still
needs real-machine verification; App-Bound (`v20`) cookies cannot be read.
Linux login requires a running Secret Service (GNOME Keyring or KDE Wallet);
headless systems without one cannot log in. On macOS, `make build-signed`
signs the binary with an Apple Development certificate to keep Keychain access
trusted across rebuilds (setup in [CLAUDE.md](CLAUDE.md#local-dev)).

## License

[MIT](LICENSE)
