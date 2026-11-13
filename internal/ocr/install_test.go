package ocr

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakePath(have ...string) func(string) (string, error) {
	return func(b string) (string, error) {
		for _, h := range have {
			if h == b {
				return "/bin/" + b, nil
			}
		}
		return "", errors.New("not found")
	}
}

func TestPlanInstall(t *testing.T) {
	defer func(f func(string) (string, error)) { lookPath = f }(lookPath)

	lookPath = fakePath("apt-get", "pkexec")
	p := planFor("linux")
	if !p.CanRun() || p.Argv[0] != "pkexec" || !strings.Contains(p.Display, "apt install") {
		t.Fatalf("apt plan = %+v", p)
	}
	lookPath = fakePath("apt-get")
	if p := planFor("linux"); p.CanRun() || p.Note == "" {
		t.Fatalf("without pkexec must not be runnable: %+v", p)
	}
	lookPath = fakePath("brew")
	if p := planFor("darwin"); !p.CanRun() || p.Argv[0] != "brew" {
		t.Fatalf("brew plan = %+v", p)
	}
	lookPath = fakePath()
	if p := planFor("windows"); p.CanRun() || p.Display == "" || p.Note == "" {
		t.Fatalf("windows without winget = %+v", p)
	}
}

func TestDownloaderEnsure(t *testing.T) {
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		if r.URL.Path == "/xx.traineddata" {
			http.NotFound(w, r)
			return
		}
		w.Write(make([]byte, 200<<10))
	}))
	defer srv.Close()
	dir := filepath.Join(t.TempDir(), "tessdata")
	d := Downloader{BaseURL: srv.URL, Dir: dir}

	if err := d.Ensure(context.Background(), "por"); err != nil {
		t.Fatal(err)
	}
	for _, l := range []string{"por", "eng"} {
		if _, err := os.Stat(filepath.Join(dir, l+".traineddata")); err != nil {
			t.Errorf("%s missing: %v", l, err)
		}
	}
	if err := d.Ensure(context.Background(), "por"); err != nil || hits["/por.traineddata"] != 1 {
		t.Fatalf("second call must not re-download (hits=%d, err=%v)", hits["/por.traineddata"], err)
	}
	if err := d.Ensure(context.Background(), "xx"); err == nil {
		t.Fatal("404 must fail")
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "*.part")); len(left) != 0 {
		t.Fatalf("partial files left: %v", left)
	}
	for _, bad := range []string{"../etc/passwd", "a/b", ""} {
		if err := d.Ensure(context.Background(), bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestDataDirArgsOnlyWhenComplete(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "por.traineddata"), []byte("x"), 0o644)
	tt := Tesseract{DataDir: dir}
	if got := tt.dataDirArgs("por+eng"); got != nil {
		t.Fatalf("eng missing, expected system dir, got %v", got)
	}
	os.WriteFile(filepath.Join(dir, "eng.traineddata"), []byte("x"), 0o644)
	if got := tt.dataDirArgs("por+eng"); len(got) != 2 || got[1] != dir {
		t.Fatalf("got %v", got)
	}
}
