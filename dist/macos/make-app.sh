#!/usr/bin/env bash
# Assemble le bundle GoLogo.app (executable + icone + exemples) et le signe.
# Prerequis : avoir compile l'executable avec tools/build/build-macos.sh.
# Usage : ./make-app.sh [version]
# Signature : par defaut ad-hoc ("-"). Pour une vraie signature Developer ID,
# exporter MACOS_SIGN_ID (ex. "Developer ID Application: Nom (TEAMID)").
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)" # github/
ver="${1:-2.0}"
bin="$root/tools/build/bin/macos/gologo"
icns="$root/tools/build/icons/gologo.icns"

if [ ! -x "$bin" ]; then
	echo "Executable absent : compile d'abord avec tools/build/build-macos.sh" >&2
	exit 1
fi

app="$here/build/GoLogo.app"
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources/examples"

cp "$bin" "$app/Contents/MacOS/gologo"
cp "$icns" "$app/Contents/Resources/gologo.icns"
cp -R "$root/src/examples/." "$app/Contents/Resources/examples/"
sed "s/@VERSION@/$ver/g" "$here/Info.plist.in" > "$app/Contents/Info.plist"

sign_id="${MACOS_SIGN_ID:--}" # "-" = signature ad-hoc
codesign --force --deep --options runtime --sign "$sign_id" "$app"
echo "OK -> $app  (signe avec : $sign_id)"
