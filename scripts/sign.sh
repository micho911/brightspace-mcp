#!/bin/sh
# Sign a local macOS build so the Keychain keeps trusting it across rebuilds.
#
# A Keychain item records the Team ID of the app that created it (its
# partition list). Ad-hoc and self-signed builds have no Team ID, so macOS
# records their cdhash, which changes on every rebuild: each new build then
# asks for the keychain password. An Apple Development certificate (free with
# an Apple ID, created in Xcode) has a Team ID, so rebuilds stay trusted.
#
# Usage: scripts/sign.sh <binary>
# Env:   SIGN_IDENTITY (default: the first valid "Apple Development" identity)
set -eu

bin="$1"
id="${SIGN_IDENTITY:-$(security find-identity -v -p codesigning | awk '/"Apple Development: /{print $2; exit}')}"

if [ -z "$id" ]; then
	echo "No Apple Development certificate: $bin is ad-hoc signed, so the Keychain" >&2
	echo "asks for your password after every rebuild. See CLAUDE.md (Local dev)." >&2
	exit 0
fi

codesign --force --identifier brightspace-mcp --sign "$id" "$bin"
team="$(codesign -dv "$bin" 2>&1 | sed -n 's/^TeamIdentifier=//p')"
echo "Signed $bin (Team ID $team)."
