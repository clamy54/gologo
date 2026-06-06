#!/usr/bin/env bash
# Compile GoLogo pour macOS. Sortie : tools/build/bin/macos/gologo
# Prerequis : Go 1.25+ et les outils en ligne de commande Xcode (cc).
# L'icone et le dossier d'exemples sont assembles dans le bundle .app par
# dist/macos/make-app.sh.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
src="$here/../../src"
bin="$here/bin/macos"
mkdir -p "$bin"
cd "$src"
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o "$bin/gologo" ./cmd/gologo
echo "OK -> $bin/gologo"
