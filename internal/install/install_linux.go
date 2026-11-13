//go:build linux

package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/thiagozs/ocrtools/internal/launcher"
)

const (
	binName   = "ocrtools"
	sysDir    = "/opt/ocrtools"
	sysBinDir = "/usr/local/bin"
	sysAppDir = "/usr/share/applications"
)

func home() string { h, _ := os.UserHomeDir(); return h }

func dataHome() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d
	}
	return filepath.Join(home(), ".local", "share")
}

func userDir() string    { return filepath.Join(home(), ".local", "opt", "ocrtools") }
func userBinDir() string { return filepath.Join(home(), ".local", "bin") }
func userDesktop() string {
	return filepath.Join(dataHome(), "applications", launcher.AppID+".desktop")
}
func systemDir() string { return sysDir }

func canSystem() bool {
	_, err := exec.LookPath("pkexec")
	return err == nil
}

func canInstall(exe string) (bool, string) {
	if exe == "" {
		return false, "não foi possível localizar o executável em execução"
	}
	return true, ""
}

func doInstall(scope Scope, exe string, icon []byte) (string, error) {
	if scope == ScopeSystem {
		return installSystem(exe, icon)
	}
	dir := userDir()
	target := filepath.Join(dir, binName)
	if err := copyFile(exe, target, 0o755); err != nil {
		return "", fmt.Errorf("copiar executável: %w", err)
	}
	iconPath := filepath.Join(dir, "icon.png")
	if len(icon) > 0 {
		if err := os.WriteFile(iconPath, icon, 0o644); err != nil {
			return "", err
		}
	}
	link := filepath.Join(userBinDir(), binName)
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return "", err
	}
	_ = os.Remove(link)
	if err := os.Symlink(target, link); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(userDesktop()), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(userDesktop(), []byte(launcher.Entry(target, iconPath)), 0o644); err != nil {
		return "", err
	}
	return target, nil
}

// installSystem copies files into root-owned dirs with a fixed script run through pkexec.
// The script only copies; the (user-writable) app binary is never executed as root.
func installSystem(exe string, icon []byte) (string, error) {
	tmp, err := os.MkdirTemp("", "ocrtools-install-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	iconSrc := filepath.Join(tmp, "icon.png")
	if err := os.WriteFile(iconSrc, icon, 0o644); err != nil {
		return "", err
	}
	desktopSrc := filepath.Join(tmp, launcher.AppID+".desktop")
	target := filepath.Join(sysDir, binName)
	if err := os.WriteFile(desktopSrc, []byte(launcher.Entry(target, filepath.Join(sysDir, "icon.png"))), 0o644); err != nil {
		return "", err
	}
	const script = `set -e
install -Dm755 "$1" "$4/ocrtools"
install -Dm644 "$2" "$4/icon.png"
install -Dm644 "$3" "$5/$6.desktop"
mkdir -p "$7"
ln -sf "$4/ocrtools" "$7/ocrtools"`
	out, err := exec.Command("pkexec", "sh", "-c", script, "sh",
		exe, iconSrc, desktopSrc, sysDir, sysAppDir, launcher.AppID, sysBinDir).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("instalação para todos falhou: %w: %s", err, out)
	}
	return target, nil
}

func doUninstall(scope Scope, _ string) error {
	if scope == ScopeSystem {
		const script = `rm -f "$1/ocrtools" "$2/$3.desktop"
rm -rf "$4"`
		out, err := exec.Command("pkexec", "sh", "-c", script, "sh",
			sysBinDir, sysAppDir, launcher.AppID, sysDir).CombinedOutput()
		if err != nil {
			return fmt.Errorf("desinstalação falhou: %w: %s", err, out)
		}
		return nil
	}
	link := filepath.Join(userBinDir(), binName)
	if t, err := os.Readlink(link); err == nil && t == filepath.Join(userDir(), binName) {
		_ = os.Remove(link)
	}
	_ = os.Remove(userDesktop())
	return os.RemoveAll(userDir())
}
