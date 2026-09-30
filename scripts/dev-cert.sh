#!/bin/sh
# Create a self-signed code-signing identity for local macOS builds (run once).
#
# Ad-hoc builds are identified by their cdhash, which changes on every
# rebuild, so the Keychain asks for access again each time. A build signed
# with this identity keeps the same designated requirement across rebuilds.
#
# The identity only works on this Mac. It is not trusted by Gatekeeper and is
# not used for releases (those get a Developer ID with GoReleaser).
#
# Usage: scripts/dev-cert.sh
# Env:   SIGN_IDENTITY (default "brightspace-mcp dev"),
#        KEYCHAIN (default: the login keychain)
set -eu

name="${SIGN_IDENTITY:-brightspace-mcp dev}"
keychain="${KEYCHAIN:-$HOME/Library/Keychains/login.keychain-db}"

if security find-certificate -c "$name" "$keychain" >/dev/null 2>&1; then
	echo "A certificate named \"$name\" already exists in $keychain."
	exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
# The PKCS#12 file lives only in $tmp; the password just satisfies the format.
pass="$(/usr/bin/openssl rand -hex 16)"

/usr/bin/openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
	-subj "/CN=$name" \
	-addext "basicConstraints=critical,CA:false" \
	-addext "keyUsage=critical,digitalSignature" \
	-addext "extendedKeyUsage=critical,codeSigning" \
	-keyout "$tmp/key.pem" -out "$tmp/cert.pem" 2>/dev/null
/usr/bin/openssl pkcs12 -export -inkey "$tmp/key.pem" -in "$tmp/cert.pem" \
	-name "$name" -passout "pass:$pass" -out "$tmp/id.p12"

# -T lets codesign use the private key without a prompt; nothing else may.
security import "$tmp/id.p12" -k "$keychain" -P "$pass" -T /usr/bin/codesign >/dev/null

echo "Created the code-signing identity \"$name\" in $keychain."
echo "Build with: make build"
