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

Set Chrome, Edge, and Brave as the default browser in turn and run the flow
for each installed browser. A cookie encrypted with App-Bound Encryption
(`v20`) should fail with an explicit unsupported-encryption message; record
the browser and version, without including cookie data. Confirm successful
login/logout with a browser profile whose cookies can be read. If Credential
Manager rejects the session as too large, report the error and the browser
version; do not capture or share the secret value.

## macOS

Run `make build-signed` after completing the signing setup in `CLAUDE.md`.
Confirm Keychain access works after rebuilding, and verify that the SQLite
reader works while the browser is open.
