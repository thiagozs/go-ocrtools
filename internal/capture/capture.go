// Package capture grabs a screenshot of the whole desktop (all monitors).
package capture

import (
	"errors"
	"image"

	"github.com/kbinani/screenshot"
)

// Desktop returns a screenshot covering every monitor. The image is in physical
// pixels; callers map monitor rectangles onto it proportionally.
func Desktop() (image.Image, error) {
	return desktop()
}

// grabNative captures the union of all displays (Windows, macOS, Linux/X11).
func grabNative() (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, errors.New("no active display found")
	}
	var union image.Rectangle
	for i := 0; i < n; i++ {
		union = union.Union(screenshot.GetDisplayBounds(i))
	}
	return screenshot.CaptureRect(union)
}
