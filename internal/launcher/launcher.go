// Package launcher builds the freedesktop launcher (.desktop) used by the app.
// It is shared by the portal hotkey code and the installer so both write the same file.
package launcher

import (
	"fmt"
	"strings"
)

// AppID identifies the app to desktop portals; the .desktop file must be named <AppID>.desktop.
const AppID = "io.github.thiagozs.ocrtools"

// Entry renders the .desktop file content for the given binary and icon path.
func Entry(exe, icon string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=OCR Tools
Comment=Captura uma área da tela e extrai o texto via OCR
Exec=%s
Icon=%s
Terminal=false
Categories=Utility;
StartupWMClass=%s
`, QuoteExec(exe), icon, AppID)
}

// QuoteExec escapes a path for the Desktop Entry Exec key.
func QuoteExec(p string) string {
	if !strings.ContainsAny(p, " \t\"'\\$`") {
		return p
	}
	r := strings.NewReplacer(`\`, `\\\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	return `"` + r.Replace(p) + `"`
}
