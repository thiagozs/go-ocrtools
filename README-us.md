# OCR Tools

🇧🇷 [Versão em português](README.md)

Cross-platform (Linux, Windows, macOS) system tray utility built with Go and
[Wails v3](https://v3.wails.io/). Capture any region of the screen, extract the
text with [Tesseract](https://github.com/tesseract-ocr/tesseract) and copy it to
the clipboard.

> Status: Linux (GNOME/Wayland) is tested end to end. Windows and macOS code
> compiles (`go vet` with `GOOS`), but has never been run.

## Features

- **Region capture** – a full-screen overlay opens on every monitor; drag to
  select the area. Works with multiple monitors, HiDPI and Wayland (via the
  XDG Desktop Portal). The crop is taken from the original screenshot, so
  resolution is preserved.
- **OCR** – Tesseract CLI (no cgo), with preprocessing: grayscale, upscaling of
  small crops and automatic inversion of dark backgrounds.
- **Result** – text is copied automatically (optional) and shown in a window.
- **System tray** – menu with *Capture text*, *Settings* and *Quit*; clicking
  the icon starts a capture.
- **Global hotkey** – default `CmdOrCtrl+Alt+O`, configurable. On Wayland it
  uses the GlobalShortcuts portal.
- **Tesseract setup** – detects the binary, offers a guided/automatic install
  and downloads language data (`tessdata_fast`) into `~/.config/ocrtools/tessdata`.
- **Start at login** – per-OS autostart toggle in Settings.
- **Self-install** – install the app for the current user or for all users,
  following each OS's conventions (see below), and uninstall from Settings.
- **Proper desktop identity** – app id `io.github.thiagozs.ocrtools`, so the
  dock/taskbar shows the right name and icon.

## Requirements

- Go 1.25+ (see `go.mod`), Node.js + npm (frontend), `wails3` CLI.
- **Tesseract** at runtime (the app can help install it).
- Linux build: `libgtk-4-dev`, `libwebkitgtk-6.0-dev`,
  `libayatana-appindicator3-dev`. On Wayland: `xdg-desktop-portal` plus your
  desktop's portal backend.

## Build and run

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails3 dev                  # development, hot reload
wails3 build               # production binary in bin/
wails3 package             # installers/packages for the current OS
go test ./...
```

After changing exported service methods, regenerate the bindings:

```sh
wails3 generate bindings -clean=true
```

## Install locations

| OS      | User                                  | All users                     |
|---------|---------------------------------------|-------------------------------|
| Linux   | `~/.local/opt/ocrtools` (+ `~/.local/bin` link, `.desktop`) | `/opt/ocrtools` (via `pkexec`) |
| Windows | `%LOCALAPPDATA%\Programs\OCR Tools`   | `Program Files` (elevated)    |
| macOS   | `~/Applications` (`.app` bundle)      | `/Applications` (admin prompt)|

Settings are stored in the user config dir under `ocrtools/settings.json`.

## Project layout

```
main.go, service.go     app setup, tray, OCRService exposed to the frontend
monitors*.go            per-monitor geometry (GDK on Linux, Wails elsewhere)
relaunch_*.go           restart after self-install
internal/capture        screenshot (X11/native, Wayland portal)
internal/ocr            Tesseract engine, installer plan, language downloader
internal/hotkey         global shortcut backends (Wails / Wayland portal)
internal/launcher       .desktop entry and app id (Linux)
internal/install        per-OS install/uninstall
internal/config         settings store
frontend/               vite pages: settings (index), overlay, result
build/                  icons (appicon.svg, render_icon.py) and packaging
```

## Known limitations

- A selection cannot yet span more than one monitor.
- System-wide install, uninstall and the "Install Tesseract" button need
  elevated privileges and have not been exercised end to end.
- Windows and macOS have not been run.

See [STEPS.md](STEPS.md) for the development history.

## License

MIT. See the [LICENSE](LICENSE) file.

## Author

2026, Thiago Zilli Sarmento :heart:
