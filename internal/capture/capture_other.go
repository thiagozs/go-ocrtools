//go:build !linux

package capture

import "image"

func desktop() (image.Image, error) { return grabNative() }
