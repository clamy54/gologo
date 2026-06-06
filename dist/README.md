# Packaging GoLogo

Build the binary first (see [`../tools/build`](../tools/build)), then create a
distributable package for the target platform. Each packaging bundles the
examples and the application icon, placed where GoLogo looks for them at runtime.

## Windows (`windows/`)

Inno Setup installer.

1. `tools\build\build-windows.ps1` (produces `bin\windows\gologo.exe`).
2. Compile the installer: `ISCC.exe windows\gologo.iss` (or open it in the Inno
   Setup IDE). Output: `windows\output\GoLogo-Setup-1.0.exe`.

The installer puts `gologo.exe` and the `examples\` folder in the install
directory, and creates Start Menu (and optional desktop) shortcuts.

## Linux (`linux/`)

Debian/Ubuntu `.deb`.

1. `tools/build/build-linux.sh` (produces `bin/linux/gologo`).
2. `linux/build-deb.sh [version]` (needs `dpkg-deb`). Output: `linux/build/gologo_<version>_<arch>.deb`.

Installs `gologo` to `/usr/bin`, examples to `/usr/share/doc/gologo/examples`, a
`gologo.desktop` launcher and the icon in the hicolor theme.

## macOS (`macos/`)

`.app` bundle then `.dmg`.

1. `tools/build/build-macos.sh` (produces `bin/macos/gologo`).
2. `macos/make-app.sh [version]` builds `GoLogo.app` (ad-hoc signed by default;
   set `MACOS_SIGN_ID` for a Developer ID signature).
3. `macos/make-dmg.sh [version]` builds a drag-to-Applications `.dmg`.

Notarization (optional) is documented at the top of `make-dmg.sh`.
