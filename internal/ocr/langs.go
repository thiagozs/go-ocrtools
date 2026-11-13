package ocr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var errNotRunnable = errors.New("este comando não pode ser executado pelo app")

// DefaultTessdataURL hosts the official "fast" language models.
const DefaultTessdataURL = "https://raw.githubusercontent.com/tesseract-ocr/tessdata_fast/main"

const (
	minModelBytes = 100 << 10 // anything smaller is an error page, not a model
	maxModelBytes = 200 << 20
)

var langNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z_]{1,19}$`)

// Downloader fetches *.traineddata files into Dir.
type Downloader struct {
	BaseURL string // defaults to DefaultTessdataURL
	Dir     string
	Client  *http.Client
}

// Ensure downloads every language in spec ("por+eng") that is missing from Dir.
// "eng" is always included so the downloaded dir is self-sufficient: Tesseract
// uses a single tessdata directory per run.
func (d Downloader) Ensure(ctx context.Context, spec string) error {
	langs := splitLangs(spec)
	if len(langs) == 0 {
		return fmt.Errorf("idioma inválido %q", spec)
	}
	langs = append(langs, "eng")
	for _, l := range langs {
		if !langNameRe.MatchString(l) {
			return fmt.Errorf("idioma inválido %q", l)
		}
		if _, err := os.Stat(filepath.Join(d.Dir, l+".traineddata")); err == nil {
			continue
		}
		if err := d.fetch(ctx, l); err != nil {
			return fmt.Errorf("baixar %s: %w", l, err)
		}
	}
	return nil
}

func splitLangs(spec string) []string {
	var out []string
	cur := ""
	for _, r := range spec + "+" {
		if r == '+' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	return out
}

func (d Downloader) fetch(ctx context.Context, lang string) error {
	base := d.BaseURL
	if base == "" {
		base = DefaultTessdataURL
	}
	cl := d.Client
	if cl == nil {
		cl = &http.Client{Timeout: 5 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/"+lang+".traineddata", nil)
	if err != nil {
		return err
	}
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d (o idioma existe em tessdata_fast?)", resp.StatusCode)
	}
	if err := os.MkdirAll(d.Dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(d.Dir, lang+".*.part")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	n, err := io.Copy(tmp, io.LimitReader(resp.Body, maxModelBytes+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n < minModelBytes || n > maxModelBytes {
		return fmt.Errorf("arquivo com tamanho inesperado (%d bytes)", n)
	}
	return os.Rename(tmp.Name(), filepath.Join(d.Dir, lang+".traineddata"))
}
