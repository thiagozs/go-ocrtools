package ocr

import (
	"context"
	"image"
	"image/color"
	"os/exec"
	"strings"
	"testing"
)

func TestRecognizeRejectsBadLanguage(t *testing.T) {
	_, err := Tesseract{}.Recognize(context.Background(), image.NewGray(image.Rect(0, 0, 4, 4)), "eng; rm -rf")
	if err == nil {
		t.Fatal("expected error for invalid language")
	}
}

func TestPreprocessUpscalesSmallImages(t *testing.T) {
	out := preprocess(image.NewRGBA(image.Rect(0, 0, 100, 40)))
	if out.Bounds().Dx() != 300 || out.Bounds().Dy() != 120 {
		t.Fatalf("got %v", out.Bounds())
	}
}

func TestRecognizeBlankImage(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract not installed")
	}
	img := image.NewGray(image.Rect(0, 0, 200, 80))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	_ = color.White
	txt, err := Tesseract{}.Recognize(context.Background(), img, "eng")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(txt) != "" {
		t.Fatalf("expected empty text, got %q", txt)
	}
}

func TestPreprocessInvertsDarkImages(t *testing.T) {
	dark := image.NewGray(image.Rect(0, 0, 500, 500)) // all zero = black
	out := preprocess(dark).(*image.Gray)
	if out.GrayAt(10, 10).Y != 255 {
		t.Fatalf("dark image should be inverted to light, got %d", out.GrayAt(10, 10).Y)
	}
	light := image.NewGray(image.Rect(0, 0, 500, 500))
	for i := range light.Pix {
		light.Pix[i] = 255
	}
	if preprocess(light).(*image.Gray).GrayAt(10, 10).Y != 255 {
		t.Fatal("light image should be left as is")
	}
}
