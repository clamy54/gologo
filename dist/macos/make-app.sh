#!/usr/bin/env bash
# Assemble le bundle GoLogo.app (executable + icone + exemples) et le signe.
# Prerequis : avoir compile l'executable avec tools/build/build-macos.sh.
# Usage : ./make-app.sh [version] (par defaut celle de src/logo/version.go)
# Signature : par defaut ad-hoc ("-"). Pour une vraie signature Developer ID,
# exporter MACOS_SIGN_ID (ex. "Developer ID Application: Nom (TEAMID)").
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)" # github/
ver="${1:-$(sed -n 's/^const Version = "\(.*\)"$/\1/p' "$root/src/logo/version.go")}"
bin="$root/tools/build/bin/macos/gologo"
icns="$root/tools/build/icons/gologo.icns"

# La version entre dans des chemins (nettoyes par rm) et dans une commande sed : on
# n'accepte que la forme d'un numero de version, sans / ni .. ni caractere special.
if ! [[ "$ver" =~ ^[0-9][0-9A-Za-z.+~-]*$ ]]; then
	echo "Version invalide : '$ver' (attendu par exemple 2.2)" >&2
	exit 1
fi

if [ ! -x "$bin" ]; then
	echo "Executable absent : compile d'abord avec tools/build/build-macos.sh" >&2
	exit 1
fi

# GoLogo est distribue pour Apple Silicon uniquement : on verifie que l'executable
# est bien un binaire arm64 avant de l'emballer.
if ! lipo -archs "$bin" | grep -qw arm64; then
	echo "L'executable n'est pas un binaire arm64 : $(lipo -archs "$bin")" >&2
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
