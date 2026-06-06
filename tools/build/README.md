# Building GoLogo

GoLogo uses [Gio](https://gioui.org), which relies on **CGO** and the native GUI
libraries of each platform. It is therefore built **on each target OS** rather
than cross-compiled.

Common requirements: **Go 1.25+** and a C compiler.

| Platform | Script | Extra requirements |
|----------|--------|--------------------|
| Windows  | [`build-windows.ps1`](build-windows.ps1) | MSYS2 / mingw-w64 (gcc) |
| Linux    | [`build-linux.sh`](build-linux.sh) | Gio dev libs (X11/Wayland, EGL/GL, xkbcommon, vulkan) |
| macOS    | [`build-macos.sh`](build-macos.sh) | Xcode command line tools |

Each script produces the binary in `bin/<platform>/`:

```
bin/windows/gologo.exe
bin/linux/gologo
bin/macos/gologo
```

The Windows build embeds the application **icon** into the executable
(`icons/gologo.ico`, via `goversioninfo`, fetched automatically). On Linux and
macOS the icon is attached when packaging (see [`../../dist`](../../dist)).

The `icons/` folder holds the icons generated from the logo: `gologo.ico`
(Windows), `gologo.icns` (macOS) and `png/gologo-<size>.png` (Linux hicolor and
the macOS iconset).

## Packaging

Once the binary is built, build a distributable package for the platform from
[`../../dist`](../../dist):

- Windows: `dist/windows/gologo.iss` (Inno Setup installer).
- Linux: `dist/linux/build-deb.sh` (Debian/Ubuntu `.deb`).
- macOS: `dist/macos/make-app.sh` then `dist/macos/make-dmg.sh` (`.app` + `.dmg`).
