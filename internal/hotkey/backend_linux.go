//go:build linux

package hotkey

import (
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// NewBackend picks the portal client on Wayland (the only sanctioned way to get
// global shortcuts there) and Wails' X11 grab otherwise. icon is the app icon
// PNG used for the launcher entry the portal requires.
func NewBackend(app *application.App, icon []byte) Backend {
	if isWayland() {
		return newPortal(icon)
	}
	return &wailsBackend{app: app}
}

func isWayland() bool {
	switch strings.ToLower(os.Getenv("XDG_SESSION_TYPE")) {
	case "wayland":
		return true
	case "x11":
		return false
	}
	return os.Getenv("WAYLAND_DISPLAY") != ""
}
