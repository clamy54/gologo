#!/usr/bin/env bash
# Construit un paquet Debian/Ubuntu (.deb) de GoLogo.
# Prerequis : avoir compile l'executable avec tools/build/build-linux.sh,
# et disposer de dpkg-deb. Usage : ./build-deb.sh [version]
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)" # github/
ver="${1:-2.1}"
arch="$(dpkg --print-architecture)"
bin="$root/tools/build/bin/linux/gologo"

if [ ! -x "$bin" ]; then
	echo "Executable absent : compile d'abord avec tools/build/build-linux.sh" >&2
	exit 1
fi

pkg="$here/build/gologo_${ver}_${arch}"
rm -rf "$pkg"
mkdir -p "$pkg/DEBIAN" \
	"$pkg/usr/bin" \
	"$pkg/usr/share/doc/gologo/examples" \
	"$pkg/usr/share/applications"

install -m 0755 "$bin" "$pkg/usr/bin/gologo"
cp -r "$root/src/examples/." "$pkg/usr/share/doc/gologo/examples/"
install -m 0644 "$here/gologo.desktop" "$pkg/usr/share/applications/gologo.desktop"
install -m 0644 "$root/LICENSE" "$pkg/usr/share/doc/gologo/copyright"

# Icones dans le theme hicolor.
for s in 16 24 32 48 64 128; do
	d="$pkg/usr/share/icons/hicolor/${s}x${s}/apps"
	mkdir -p "$d"
	install -m 0644 "$root/tools/build/icons/png/gologo-${s}.png" "$d/gologo.png"
done

sed -e "s/@VERSION@/$ver/" -e "s/@ARCH@/$arch/" "$here/control.in" > "$pkg/DEBIAN/control"

dpkg-deb --build --root-owner-group "$pkg"
echo "OK -> ${pkg}.deb"
