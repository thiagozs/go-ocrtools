# STEPS

History of what has been done, in order. Newest last. Update when work is added.

## 1. Feasibility and scope
- Confirmed a tray OCR app in Go + Wails v3 is feasible.
- Targets: Linux, Windows and macOS.

## 2. Project bootstrap
- Generated the project with `wails3 init`; fixed the nested directory and the
  bad module name (`false`) in `go.mod`.
- Linux build deps installed: GTK4, WebKitGTK 6.0, appindicator, Tesseract
  (+ Portuguese data), `xdg-desktop-portal`.

## 3. Capture and OCR core
- `internal/capture`: whole-desktop screenshot (X11/native, Wayland via portal
  over D-Bus).
- `internal/ocr`: Tesseract CLI engine with preprocessing (grayscale, upscale,
  invert dark images).
- Tray menu, overlay window for region selection, result window, settings page.

## 4. Multi-monitor and resolution fix
- Problem: low-resolution screenshot of a single screen on a 3-monitor Wayland
  setup.
- Fix: one overlay per monitor using GDK monitor geometry (cgo), selection
  mapped as fractions of the overlay to monitor pixels, cropping the original
  image. Tests for offsets and HiDPI.
- Overlay sizing and fullscreen-on-monitor re-applied after 400 ms.
- User confirmed it works.

## 5. App died after selection
- Cause: Wails quit when the last window closed.
- Fix: `DisableQuitOnLastWindowClosed` (Linux/Windows). User confirmed.

## 6. Global hotkey
- Wails `GlobalShortcut` backend, plus an own GlobalShortcuts portal client for
  Wayland (`internal/hotkey`).
- First attempt failed ("An app id is required"); fixed by registering the host
  app (`Registry.Register`) and creating a `.desktop` file. User confirmed.

## 7. Tesseract detection and language download
- Detect Tesseract (PATH and well-known locations), guided install, download of
  `tessdata_fast` languages into `~/.config/ocrtools/tessdata`.
- Setup card in Settings and a check at startup.

## 8. Tray icon
- New icon (`build/appicon.svg`, `render_icon.py` with Pillow); regenerated
  PNG, `.ico` and `.icns`, plus a tray icon.

## 9. Start at login
- Settings toggle using Wails `Autostart`. Since it is keyed by the running
  executable, state is handed to the relaunched process via an env var.

## 10. In-app install
- Install/uninstall for user scope and all-users scope following each OS's
  standard paths (see README). Linux uses `pkexec` with a fixed script;
  Windows an elevated PowerShell; macOS `ditto` + `osascript`.
- Verified only by compile (`go vet GOOS=windows/darwin`) and Linux unit tests.

## 11. Dock/taskbar entry (latest)
- Problem: after the screenshot the dock showed a gear icon and the raw name
  `org.wails.ocr_tools`.
- Cause: Wayland app id did not match the launcher
  `io.github.thiagozs.ocrtools.desktop`.
- Fix: `Linux.ApplicationID = launcher.AppID`; `launcher.Ensure` creates the
  user `.desktop` on every start (keeps an existing one whose target exists,
  repairs a dangling one); `StartupWMClass` added for X11.
- Verified visually: dock shows the new icon. Tooltip text not hover-checked.

## 12. Documentation
- Rewrote `README.md` for this project and created this `STEPS.md`.

## Pending
- Selection spanning multiple monitors (idea: keep receiving pointer events in
  the originating overlay, offset by monitor position, mirror on other overlays).
- Release packaging (.deb/AppImage, Windows installer, .app/DMG); untested on
  Windows and macOS.
- Real-world test of system-wide install, uninstall and "Install Tesseract".
