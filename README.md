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

VS Code's Go extension (`gopls`) uses the Go build target when it checks
packages and reports diagnostics. If it is set to another OS or architecture,
dependencies may show build-constraint errors even though the project builds
for your machine. For a Linux workspace, set the target in `.vscode/settings.json`:

```json
{
  "go.toolsEnvVars": {
    "GOOS": "linux",
    "GOARCH": "amd64"
  }
}
```

Use `go env GOARCH` to check your architecture (`arm64` is common on ARM
systems). Then run **Go: Restart Language Server** from VS Code's Command
Palette. `go.toolsEnvVars` is passed to `gopls` by the Go extension.

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
