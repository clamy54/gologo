#!/usr/bin/env bash
# Cree un .dmg de GoLogo : une fenetre contenant GoLogo.app et un raccourci vers
# /Applications (glisser-deposer pour installer). Prerequis : avoir construit le
# bundle avec make-app.sh. Usage : ./make-dmg.sh [version] (par defaut celle de
# src/logo/version.go)
#
# Notarisation (optionnelle) : apres signature Developer ID (MACOS_SIGN_ID dans
# make-app.sh), soumettre le .dmg avec :
#   xcrun notarytool submit GoLogo-<ver>.dmg --keychain-profile <profil> --wait
#   xcrun stapler staple GoLogo-<ver>.dmg
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"
ver="${1:-$(sed -n 's/^const Version = "\(.*\)"$/\1/p' "$root/src/logo/version.go")}"
app="$here/build/GoLogo.app"

# La version entre dans des chemins (nettoyes par rm) et dans une commande sed : on
# n'accepte que la forme d'un numero de version, sans / ni .. ni caractere special.
if ! [[ "$ver" =~ ^[0-9][0-9A-Za-z.+~-]*$ ]]; then
	echo "Version invalide : '$ver' (attendu par exemple 2.2)" >&2
	exit 1
fi

if [ ! -d "$app" ]; then
	echo "Bundle absent : construis d'abord avec make-app.sh" >&2
	exit 1
fi

stage="$here/build/dmg"
rm -rf "$stage"
mkdir -p "$stage"
cp -R "$app" "$stage/"
ln -s /Applications "$stage/Applications"

dmg="$here/build/GoLogo-${ver}.dmg"
rm -f "$dmg"
hdiutil create -volname "GoLogo" -srcfolder "$stage" -ov -format UDZO "$dmg"
rm -rf "$stage"
echo "OK -> $dmg"
