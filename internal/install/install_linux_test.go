//go:build linux

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUserInstallAndUninstallLinux(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	// Use a fake "running binary".
	exe := filepath.Join(t.TempDir(), "ocrtools-src")
	if err := os.WriteFile(exe, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := doInstall(ScopeUser, exe, []byte("png"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "opt", "ocrtools", "ocrtools"); target != want {
		t.Fatalf("target = %s, want %s", target, want)
	}
	if b, _ := os.ReadFile(target); string(b) != "binary" {
		t.Fatal("binary not copied")
	}
	if fi, _ := os.Stat(target); fi.Mode().Perm()&0o111 == 0 {
		t.Fatal("binary must be executable")
	}
	link, _ := os.Readlink(filepath.Join(home, ".local", "bin", "ocrtools"))
	if link != target {
		t.Fatalf("symlink -> %q", link)
	}
	d, _ := os.ReadFile(filepath.Join(home, ".local", "share", "applications", "io.github.thiagozs.ocrtools.desktop"))
	if !strings.Contains(string(d), "Exec="+target) {
		t.Fatalf("desktop entry wrong:\n%s", d)
	}

	// Reinstall over an existing copy must work (atomic replace).
	if _, err := doInstall(ScopeUser, exe, nil); err != nil {
		t.Fatalf("reinstall: %v", err)
	}

	if err := doUninstall(ScopeUser, target); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Dir(target), filepath.Join(home, ".local", "bin", "ocrtools"),
		filepath.Join(home, ".local", "share", "applications", "io.github.thiagozs.ocrtools.desktop")} {
		if _, err := os.Lstat(p); err == nil {
			t.Errorf("%s should be removed", p)
		}
	}
}

func TestWithin(t *testing.T) {
	d := t.TempDir()
	if !within(filepath.Join(d, "a", "b"), d) || within(d, d) || within(filepath.Join(d, "..", "x"), d) || within("", d) {
		t.Fatal("within() wrong")
	}
}
