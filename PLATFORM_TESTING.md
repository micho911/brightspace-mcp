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

Run the flow once with GNOME Keyring and once with KDE Wallet if both desktop
environments are available. Also test from a headless session without Secret
Service and confirm login reports that the Secret Service is unavailable.
Try each supported default browser you have installed (Brave, Chrome, Edge).

## Windows

Set Chrome, Edge, and Brave as the default browser in turn and run the flow
for each installed browser. A cookie encrypted with App-Bound Encryption
(`v20`) should fail with an explicit unsupported-encryption message; record
the browser and version, without including cookie data. Confirm successful
login/logout with a browser profile whose cookies can be read. If Credential
Manager rejects the session as too large, report the error and the browser
version; do not capture or share the secret value.

## macOS

Run the flow with an Apple Development-signed build and confirm Keychain
access works after rebuilding. Also verify that the SQLite reader works while
the browser is open.
