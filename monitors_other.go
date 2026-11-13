//go:build !(linux && cgo && !gtk3 && !android && !server)

package main

import (
	"image"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// listMonitors uses Wails' screen list (physical pixels, matches the capture on X11/Windows/macOS).
func listMonitors(app *application.App) []monitor {
	var out []monitor
	for _, s := range app.Screen.GetAll() {
		b := s.PhysicalBounds
		out = append(out, monitor{
			Rect:   image.Rect(b.X, b.Y, b.X+b.Width, b.Y+b.Height),
			Screen: s,
		})
	}
	return out
}

// overlayOptions places the overlay on monitor m, fullscreen.
func overlayOptions(o application.WebviewWindowOptions, m monitor) application.WebviewWindowOptions {
	if s, ok := m.Screen.(*application.Screen); ok {
		o.Screen = s
	}
	o.StartState = application.WindowStateFullscreen
	return o
}

// afterCreate is a no-op here: Wails already honoured Options.Screen.
func afterCreate(*application.WebviewWindow, int) {}
