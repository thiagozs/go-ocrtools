//go:build linux

package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCreatesAndKeepsExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	path := filepath.Join(home, ".local", "share", "applications", AppID+".desktop")

	if err := Ensure([]byte("png")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), "StartupWMClass="+AppID) {
		t.Fatalf("entry missing or without StartupWMClass: %v\n%s", err, b)
	}

	// An entry that points to an existing program must not be overwritten.
	keep := filepath.Join(home, "installed-bin")
	os.WriteFile(keep, []byte("x"), 0o755)
	os.WriteFile(path, []byte(Entry(keep, "")), 0o644)
	if err := Ensure(nil); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), "Exec="+keep) {
		t.Fatalf("existing launcher was hijacked:\n%s", b)
	}

	// A dangling entry is repaired to point at the running binary.
	os.WriteFile(path, []byte(Entry(filepath.Join(home, "gone"), "")), 0o644)
	if err := Ensure(nil); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); strings.Contains(string(b), "gone") {
		t.Fatalf("dangling launcher not repaired:\n%s", b)
	}
}

func TestExecTargetQuoted(t *testing.T) {
	if got := execTarget(Entry("/opt/my app/bin", "")); got != "/opt/my app/bin" {
		t.Fatalf("got %q", got)
	}
}
