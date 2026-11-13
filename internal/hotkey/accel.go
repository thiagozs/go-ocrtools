package hotkey

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// portalTrigger converts a Wails accelerator ("CmdOrCtrl+Alt+O") into the XDG
// shortcuts syntax used by the GlobalShortcuts portal ("CTRL+ALT+o").
func portalTrigger(accel string) (string, error) {
	parts := strings.Split(accel, "+")
	if len(parts) < 2 {
		return "", fmt.Errorf("atalho %q precisa de ao menos um modificador", accel)
	}
	var mods []string
	seen := map[string]bool{}
	for _, p := range parts[:len(parts)-1] {
		var m string
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "cmdorctrl", "ctrl", "control":
			m = "CTRL"
		case "alt", "option":
			m = "ALT"
		case "shift":
			m = "SHIFT"
		case "super", "cmd", "meta", "win":
			m = "LOGO"
		default:
			return "", fmt.Errorf("modificador desconhecido %q", p)
		}
		if !seen[m] {
			seen[m] = true
			mods = append(mods, m)
		}
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	if key == "" {
		return "", fmt.Errorf("atalho %q sem tecla", accel)
	}
	if utf8.RuneCountInString(key) == 1 {
		key = strings.ToLower(key) // keysym names for letters are lowercase
	} else if strings.EqualFold(key, "space") {
		key = "space"
	}
	return strings.Join(append(mods, key), "+"), nil
}
