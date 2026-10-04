#!/usr/bin/env bash
# Construit un paquet Debian/Ubuntu (.deb) de GoLogo.
# Prerequis : avoir compile l'executable avec tools/build/build-linux.sh,
# et disposer de dpkg-deb. Usage : ./build-deb.sh [version]
# Sans argument, la version est celle de src/logo/version.go.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)" # github/
ver="${1:-$(sed -n 's/^const Version = "\(.*\)"$/\1/p' "$root/src/logo/version.go")}"
bin="$root/tools/build/bin/linux/gologo"

# La version entre dans des chemins (nettoyes par rm) et dans une commande sed : on
# n'accepte que la forme d'un numero de version, sans / ni .. ni caractere special.
if ! [[ "$ver" =~ ^[0-9][0-9A-Za-z.+~-]*$ ]]; then
	echo "Version invalide : '$ver' (attendu par exemple 2.2)" >&2
	exit 1
fi

if [ ! -x "$bin" ]; then
	echo "Executable absent : compile d'abord avec tools/build/build-linux.sh" >&2
	exit 1
fi

# L'architecture du paquet est celle de l'executable qu'on y met, lue dans son
# en-tete ELF (champ e_machine), et non celle de la machine qui construit le paquet.
if [ "$(head -c 4 "$bin" | od -An -c | tr -d ' ')" != '177ELF' ]; then
	echo "$bin n'est pas un executable ELF (Linux)" >&2
	exit 1
fi
case "$(od -An -t u2 -j 18 -N 2 "$bin" | tr -d ' ')" in
	62) arch=amd64 ;;
	183) arch=arm64 ;;
	3) arch=i386 ;;
	40) arch=armhf ;;
	*)
		echo "Architecture de l'executable non reconnue" >&2
		exit 1
		;;
esac

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
