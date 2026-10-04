#!/usr/bin/env bash
# Compile GoLogo pour Linux. Sortie : tools/build/bin/linux/gologo
# Prerequis : Go 1.27+, un compilateur C, et les dependances de developpement de
# Gio (X11/Wayland, EGL/GL, xkbcommon, vulkan). Voir https://gioui.org/doc/install
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
src="$here/../../src"
bin="$here/bin/linux"
mkdir -p "$bin"
cd "$src"
# GOOS fixe (un GOOS herite de l'environnement produirait autre chose qu'un
# executable Linux) ; l'architecture reste celle de la machine, ou GOARCH.
GOOS=linux CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o "$bin/gologo" ./cmd/gologo
echo "OK -> $bin/gologo"
