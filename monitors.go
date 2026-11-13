package main

import "image"

// monitor is one display. Rect is in the same coordinate space for all monitors
// (compositor layout), so the union of all Rects maps onto the desktop screenshot.
type monitor struct {
	Rect   image.Rectangle
	Screen any // *application.Screen on platforms where Wails positions the window
}

func unionRect(ms []monitor) image.Rectangle {
	var u image.Rectangle
	for _, m := range ms {
		u = u.Union(m.Rect)
	}
	return u
}
