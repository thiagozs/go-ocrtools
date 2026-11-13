package ocr

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"golang.org/x/image/draw"
)

var langRe = regexp.MustCompile(`^[A-Za-z0-9_]+(\+[A-Za-z0-9_]+)*$`)

// Tesseract runs the tesseract CLI (no cgo, simple cross-compilation).
type Tesseract struct {
	// Path to the binary; empty means "look in PATH, then well-known locations".
	Path string
	// DataDir holds downloaded *.traineddata files; used when it has every requested language.
	DataDir string
}

func (t Tesseract) bin() string {
	if t.Path != "" {
		return t.Path
	}
	if p, err := exec.LookPath("tesseract"); err == nil {
		return p
	}
	// GUI apps on macOS/Windows often do not inherit the shell PATH.
	for _, c := range wellKnownPaths() {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "tesseract"
}

func wellKnownPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			`C:\Program Files\Tesseract-OCR\tesseract.exe`,
			`C:\Program Files (x86)\Tesseract-OCR\tesseract.exe`,
		}
	case "darwin":
		return []string{"/opt/homebrew/bin/tesseract", "/usr/local/bin/tesseract", "/opt/local/bin/tesseract"}
	}
	return nil
}

// Version returns the first line of `tesseract --version`; an error means it is not usable.
func (t Tesseract) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, t.bin(), "--version").CombinedOutput()
	if err != nil {
		return "", err
	}
	line, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(line), nil
}

// dataDirArgs selects the downloaded tessdata dir only when it has every requested language.
func (t Tesseract) dataDirArgs(lang string) []string {
	if t.DataDir == "" {
		return nil
	}
	for _, l := range strings.Split(lang, "+") {
		if _, err := os.Stat(filepath.Join(t.DataDir, l+".traineddata")); err != nil {
			return nil
		}
	}
	return []string{"--tessdata-dir", t.DataDir}
}

// Recognize preprocesses img and returns the recognized text.
func (t Tesseract) Recognize(ctx context.Context, img image.Image, lang string) (string, error) {
	if !langRe.MatchString(lang) {
		return "", fmt.Errorf("invalid language %q", lang)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, preprocess(img)); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	args := append([]string{"stdin", "stdout"}, t.dataDirArgs(lang)...)
	args = append(args, "-l", lang, "--psm", "6")
	cmd := exec.CommandContext(ctx, t.bin(), args...)
	cmd.Stdin = &buf
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// Languages lists system-installed plus downloaded Tesseract languages.
func (t Tesseract) Languages(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, t.bin(), "--list-langs").Output()
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for i, l := range strings.Split(string(out), "\n") {
		l = strings.TrimSpace(l)
		if i == 0 || l == "" || l == "osd" {
			continue
		}
		set[l] = true
	}
	if t.DataDir != "" {
		files, _ := os.ReadDir(t.DataDir)
		for _, f := range files {
			if n, ok := strings.CutSuffix(f.Name(), ".traineddata"); ok && n != "osd" {
				set[n] = true
			}
		}
	}
	langs := make([]string, 0, len(set))
	for l := range set {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	return langs, nil
}

// preprocess converts to grayscale and upscales small crops, which helps Tesseract a lot.
func preprocess(img image.Image) image.Image {
	b := img.Bounds()
	scale := 1
	if b.Dy() < 150 {
		scale = 3
	} else if b.Dy() < 400 {
		scale = 2
	}
	gray := image.NewGray(image.Rect(0, 0, b.Dx()*scale, b.Dy()*scale))
	draw.CatmullRom.Scale(gray, gray.Bounds(), img, b, draw.Src, nil)
	if meanLuma(gray) < 128 {
		invert(gray) // Tesseract reads dark text on light background far better (dark themes)
	}
	return gray
}

func meanLuma(g *image.Gray) float64 {
	if len(g.Pix) == 0 {
		return 255
	}
	var sum uint64
	for _, v := range g.Pix {
		sum += uint64(v)
	}
	return float64(sum) / float64(len(g.Pix))
}

func invert(g *image.Gray) {
	for i, v := range g.Pix {
		g.Pix[i] = 255 - v
	}
}
