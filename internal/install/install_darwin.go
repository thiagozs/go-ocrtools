//go:build darwin

package install

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func userDir() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Applications")
}

func systemDir() string { return "/Applications" }
func canSystem() bool   { return true }

// bundleOf returns the .app bundle that contains exe, or "".
func bundleOf(exe string) string {
	i := strings.Index(exe, ".app/")
	if i < 0 {
		return ""
	}
	return exe[:i+len(".app")]
}

func canInstall(exe string) (bool, string) {
	if bundleOf(exe) == "" {
		return false, "No macOS a instalação só funciona rodando de dentro de um pacote .app (gere com `wails3 package`)."
	}
	return true, ""
}

func doInstall(scope Scope, exe string, _ []byte) (string, error) {
	bundle := bundleOf(exe)
	name := filepath.Base(bundle)
	if bundle == "" || name == ".app" {
		return "", errors.New("pacote .app inválido")
	}
	dir := userDir()
	if scope == ScopeSystem {
		dir = systemDir()
	}
	dest := filepath.Join(dir, name)
	newExe := dest + strings.TrimPrefix(exe, bundle)

	if scope == ScopeUser {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		_ = os.RemoveAll(dest)
		if out, err := exec.Command("ditto", bundle, dest).CombinedOutput(); err != nil {
			return "", fmt.Errorf("ditto: %w: %s", err, out)
		}
		return newExe, nil
	}
	// System-wide: try plain first (admin users can usually write to /Applications).
	_ = os.RemoveAll(dest)
	if out, err := exec.Command("ditto", bundle, dest).CombinedOutput(); err == nil {
		return newExe, nil
	} else if !errors.Is(err, os.ErrPermission) && !strings.Contains(string(out), "Permission denied") {
		return "", fmt.Errorf("ditto: %w: %s", err, out)
	}
	if err := admin(`"rm -rf " & quoted form of item 2 of argv & " && ditto " & quoted form of item 1 of argv & " " & quoted form of item 2 of argv`, bundle, dest); err != nil {
		return "", err
	}
	return newExe, nil
}

func doUninstall(scope Scope, exe string) error {
	bundle := bundleOf(exe)
	if bundle == "" {
		return errors.New("pacote .app não encontrado")
	}
	if scope == ScopeUser {
		return os.RemoveAll(bundle)
	}
	return admin(`"rm -rf " & quoted form of item 1 of argv`, bundle)
}

// admin runs a fixed AppleScript shell command with an administrator prompt; paths
// go through argv + `quoted form of`, so they are never interpolated into the script.
func admin(shellExpr string, args ...string) error {
	script := []string{"-e", "on run argv", "-e", "do shell script " + shellExpr + " with administrator privileges", "-e", "end run"}
	out, err := exec.Command("osascript", append(script, args...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}
