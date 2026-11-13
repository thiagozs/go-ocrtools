//go:build windows

package install

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"

	"golang.org/x/sys/windows/registry"
)

const (
	appName = "OCR Tools"
	exeName = "ocrtools.exe"
	regPath = `Software\Microsoft\Windows\CurrentVersion\Uninstall\OCRTools`
)

func userDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", appName)
}

func systemDir() string {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		pf = `C:\Program Files`
	}
	return filepath.Join(pf, appName)
}

func canSystem() bool { return true }

func canInstall(exe string) (bool, string) {
	if exe == "" {
		return false, "não foi possível localizar o executável em execução"
	}
	return true, ""
}

func startMenuDir(scope Scope) string {
	if scope == ScopeSystem {
		return filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs`)
	}
	return filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`)
}

func doInstall(scope Scope, exe string, _ []byte) (string, error) {
	dir := userDir()
	if scope == ScopeSystem {
		dir = systemDir()
	}
	target := filepath.Join(dir, exeName)
	lnk := filepath.Join(startMenuDir(scope), appName+".lnk")

	if scope == ScopeSystem {
		script := strings.Join([]string{
			`$ErrorActionPreference='Stop'`,
			`New-Item -ItemType Directory -Force -Path ` + psq(dir) + ` | Out-Null`,
			`Copy-Item -LiteralPath ` + psq(exe) + ` -Destination ` + psq(target) + ` -Force`,
			shortcutPS(lnk, target, dir),
			`$k='HKLM:\` + regPath + `'`,
			`New-Item -Path $k -Force | Out-Null`,
			setProp("DisplayName", appName, "String"),
			setProp("InstallLocation", dir, "String"),
			setProp("DisplayIcon", target, "String"),
			setProp("Publisher", appName, "String"),
			setProp("UninstallString", `"`+target+`" --uninstall`, "String"),
			setProp("NoModify", "1", "DWord"),
			setProp("NoRepair", "1", "DWord"),
		}, "\n")
		return target, runElevated(script, true)
	}

	if err := copyFile(exe, target, 0o755); err != nil {
		return "", fmt.Errorf("copiar executável: %w", err)
	}
	if err := runPS(shortcutPS(lnk, target, dir)); err != nil {
		return "", fmt.Errorf("criar atalho: %w", err)
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	for name, v := range map[string]string{
		"DisplayName": appName, "InstallLocation": dir, "DisplayIcon": target,
		"Publisher": appName, "UninstallString": `"` + target + `" --uninstall`,
	} {
		if err := k.SetStringValue(name, v); err != nil {
			return "", err
		}
	}
	_ = k.SetDWordValue("NoModify", 1)
	_ = k.SetDWordValue("NoRepair", 1)
	return target, nil
}

func doUninstall(scope Scope, exe string) error {
	dir := filepath.Dir(exe)
	lnk := filepath.Join(startMenuDir(scope), appName+".lnk")
	if scope == ScopeSystem {
		script := strings.Join([]string{
			`$ErrorActionPreference='SilentlyContinue'`,
			`Remove-Item -LiteralPath ` + psq(lnk) + ` -Force`,
			`Remove-Item -Path 'HKLM:\` + regPath + `' -Recurse -Force`,
			// The running exe is locked until the app quits; delete after a short wait.
			`Start-Sleep -Seconds 3`,
			`Remove-Item -LiteralPath ` + psq(dir) + ` -Recurse -Force`,
		}, "\n")
		return runElevated(script, false)
	}
	_ = os.Remove(lnk)
	_ = registry.DeleteKey(registry.CURRENT_USER, regPath)
	cmd := exec.Command("cmd", "/c", "ping -n 3 127.0.0.1 >nul & rmdir /s /q \""+dir+"\"")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

// psq quotes s as a PowerShell single-quoted literal.
func psq(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func setProp(name, value, typ string) string {
	return `New-ItemProperty -Path $k -Name ` + psq(name) + ` -Value ` + psq(value) + ` -PropertyType ` + typ + ` -Force | Out-Null`
}

func shortcutPS(lnk, target, dir string) string {
	return `$s=(New-Object -ComObject WScript.Shell).CreateShortcut(` + psq(lnk) + `); ` +
		`$s.TargetPath=` + psq(target) + `; $s.WorkingDirectory=` + psq(dir) + `; $s.Save()`
}

func encode(script string) string {
	u := utf16.Encode([]rune(script))
	b := make([]byte, 0, len(u)*2)
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	return base64.StdEncoding.EncodeToString(b)
}

func runPS(script string) error {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", encode(script)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

// runElevated runs script in an elevated PowerShell (UAC prompt). The script travels
// in the command line (-EncodedCommand), never through a file another process could swap.
func runElevated(script string, wait bool) error {
	w := ""
	if wait {
		w = " -Wait"
	}
	outer := `$p=Start-Process powershell -Verb RunAs -PassThru` + w +
		` -ArgumentList '-NoProfile','-EncodedCommand',` + psq(encode(script)) + `; ` +
		`if (` + boolPS(wait) + `) { exit $p.ExitCode }`
	return runPS(outer)
}

func boolPS(b bool) string {
	if b {
		return "$true"
	}
	return "$false"
}
