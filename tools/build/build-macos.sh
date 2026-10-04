#!/usr/bin/env bash
# Compile GoLogo pour macOS. Sortie : tools/build/bin/macos/gologo
# Prerequis : Go 1.27+ et les outils en ligne de commande Xcode (cc).
# L'icone et le dossier d'exemples sont assembles dans le bundle .app par
# dist/macos/make-app.sh.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
src="$here/../../src"
bin="$here/bin/macos"
mkdir -p "$bin"
cd "$src"
# Version minimale de macOS visee, pour le code C comme pour le code Go (Go 1.27
# exige macOS 13 Ventura) : la meme que LSMinimumSystemVersion dans Info.plist.in.
export MACOSX_DEPLOYMENT_TARGET=13.0
# Cible fixee : macOS sur Apple Silicon (seule plateforme Mac distribuee), quelles
# que soient les variables GOOS/GOARCH heritees de l'environnement.
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o "$bin/gologo" ./cmd/gologo
echo "OK -> $bin/gologo"
