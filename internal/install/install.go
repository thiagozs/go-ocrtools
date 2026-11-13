// Package install copies the running app to a standard per-OS location
// (per-user or system-wide) and removes it again.
package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Scope selects where the app is installed.
type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

// Status describes where the app is running from and what can be done.
type Status struct {
	Exe        string `json:"exe"`
	Installed  bool   `json:"installed"`
	Scope      Scope  `json:"scope"`
	UserDir    string `json:"userDir"`
	SystemDir  string `json:"systemDir"`
	CanInstall bool   `json:"canInstall"`
	CanSystem  bool   `json:"canSystem"`
	Note       string `json:"note"`
}

// ErrAlreadyInstalled is returned when the running binary already is the installed one.
var ErrAlreadyInstalled = errors.New("o app já está instalado neste local")

// GetStatus inspects the running binary.
func GetStatus() Status {
	exe := runningExe()
	st := Status{Exe: exe, UserDir: userDir(), SystemDir: systemDir(), CanSystem: canSystem()}
	st.CanInstall, st.Note = canInstall(exe)
	switch {
	case within(exe, st.UserDir):
		st.Installed, st.Scope = true, ScopeUser
	case within(exe, st.SystemDir):
		st.Installed, st.Scope = true, ScopeSystem
	}
	return st
}

// Install copies the app to scope's location and returns the path of the new executable.
func Install(scope Scope, icon []byte) (string, error) {
	if scope != ScopeUser && scope != ScopeSystem {
		return "", fmt.Errorf("escopo inválido %q", scope)
	}
	st := GetStatus()
	if !st.CanInstall {
		return "", errors.New(st.Note)
	}
	if scope == ScopeSystem && !st.CanSystem {
		return "", errors.New("instalação para todos não é suportada neste sistema")
	}
	if st.Installed && st.Scope == scope {
		return "", ErrAlreadyInstalled
	}
	return doInstall(scope, st.Exe, icon)
}

// Uninstall removes the installed copy (the running one must be installed).
func Uninstall() error {
	st := GetStatus()
	if !st.Installed {
		return errors.New("o app não está rodando de um local instalado")
	}
	return doUninstall(st.Scope, st.Exe)
}

func runningExe() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	return exe
}

// within reports whether path is inside dir (both resolved, case-insensitive on Windows/macOS via filepath cleaning only).
func within(path, dir string) bool {
	if path == "" || dir == "" {
		return false
	}
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		dir = r
	}
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// copyFile copies src to dst atomically (temp file + rename) with the given mode.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".install-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
