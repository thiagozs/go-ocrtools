//go:build linux

package launcher

import (
	"os"
	"path/filepath"
	"strings"
)

// Ensure makes sure the per-user launcher exists so the desktop can match the app's
// windows to a name and icon (and portals can identify it). An existing entry whose
// target still exists is left alone, so running a throwaway build does not hijack
// the launcher of an installed copy.
func Ensure(icon []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dataHome(home), "applications", AppID+".desktop")

	if cur, err := os.ReadFile(path); err == nil {
		if target := execTarget(string(cur)); target != "" {
			if _, err := os.Stat(target); err == nil {
				return nil
			}
		}
	}

	iconPath := ""
	if cfg, err := os.UserConfigDir(); err == nil && len(icon) > 0 {
		iconPath = filepath.Join(cfg, "ocrtools", "icon.png")
		if err := os.MkdirAll(filepath.Dir(iconPath), 0o755); err == nil {
			_ = os.WriteFile(iconPath, icon, 0o644)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(Entry(exe, iconPath)), 0o644)
}

func dataHome(home string) string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return d
	}
	return filepath.Join(home, ".local", "share")
}

// execTarget extracts the program path from the Exec= line (handles our quoted form).
func execTarget(entry string) string {
	for _, l := range strings.Split(entry, "\n") {
		v, ok := strings.CutPrefix(l, "Exec=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if strings.HasPrefix(v, `"`) {
			if end := strings.Index(v[1:], `"`); end >= 0 {
				return strings.NewReplacer(`\\`, `\`, `\"`, `"`).Replace(v[1 : 1+end])
			}
		}
		return strings.Fields(v)[0]
	}
	return ""
}
