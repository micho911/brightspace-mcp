# Real-machine login checks

Run these checks locally, never in CI. The MCP `whoami` response contains
personal data; keep output out of commits and issue reports.

For each supported machine, build the binary, log in, check the saved session,
and log out:

```sh
make build
./brightspace-mcp login https://brightspace.au.dk
scripts/smoke.py whoami
./brightspace-mcp logout
```

On Windows, use PowerShell and `brightspace-mcp.exe`:

```powershell
make build
.\brightspace-mcp.exe login https://brightspace.au.dk
python scripts/smoke.py whoami
.\brightspace-mcp.exe logout
```

## Linux

Verified on Fedora GNOME: Brave, Chrome, and Edge each passed login, the MCP
`whoami` smoke check, and logout. Login without Secret Service also reported
the expected unavailable-service error.

Flatpak and Snap browser profiles were not tested; their profile paths remain
an open support question in [BACKLOG.md](BACKLOG.md).

Still to verify: run the flow with KDE Wallet. Try each supported default
browser available in that desktop session (Brave, Chrome, Edge).

## Windows

Windows is not supported yet. Before claiming support, choose an approach for
browser cookie encryption that does not require users to weaken browser
security, then validate login/logout with Chrome, Edge, and Brave on real
machines. Measure session size against Credential Manager's 2560-byte limit.
Never capture or share cookie values.

## macOS

Run `make build-signed` after completing the signing setup in `CLAUDE.md`.
Confirm Keychain access works after rebuilding, and verify that the SQLite
reader works while the browser is open.
