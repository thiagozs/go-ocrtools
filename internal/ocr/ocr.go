// Package ocr defines the text-recognition engine interface and a Tesseract implementation.
package ocr

import (
	"context"
	"image"
)

// Engine turns an image into text. Other engines (e.g. a vision API) can implement it.
type Engine interface {
	Recognize(ctx context.Context, img image.Image, lang string) (string, error)
}
