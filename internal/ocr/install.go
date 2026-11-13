package ocr

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// InstallPlan describes how to install Tesseract on this machine.
type InstallPlan struct {
	// Display is the human-readable command shown to the user.
	Display string
	// Argv is the command to run; empty when it cannot be run from the app.
	Argv []string
	// Note explains why Argv is empty or adds caveats.
	Note string
}

// CanRun reports whether the app can run the install itself.
func (p InstallPlan) CanRun() bool { return len(p.Argv) > 0 }

var lookPath = exec.LookPath

// PlanInstall picks the install command for the current OS and package manager.
// Commands are fixed templates: nothing user-controlled is interpolated.
func PlanInstall() InstallPlan { return planFor(runtime.GOOS) }

func planFor(goos string) InstallPlan {
	has := func(bin string) bool { _, err := lookPath(bin); return err == nil }
	switch goos {
	case "linux":
		type pm struct {
			bin, display string
			args         []string
		}
		for _, c := range []pm{
			{"apt-get", "sudo apt install tesseract-ocr", []string{"apt-get", "install", "-y", "tesseract-ocr"}},
			{"dnf", "sudo dnf install tesseract", []string{"dnf", "install", "-y", "tesseract"}},
			{"pacman", "sudo pacman -S tesseract tesseract-data-eng", []string{"pacman", "-S", "--noconfirm", "tesseract", "tesseract-data-eng"}},
			{"zypper", "sudo zypper install tesseract-ocr", []string{"zypper", "--non-interactive", "install", "tesseract-ocr"}},
		} {
			if !has(c.bin) {
				continue
			}
			plan := InstallPlan{Display: c.display}
			if has("pkexec") {
				plan.Argv = append([]string{"pkexec"}, c.args...) // graphical admin prompt
			} else {
				plan.Note = "pkexec não encontrado: rode o comando acima no terminal."
			}
			return plan
		}
		return InstallPlan{Display: "Instale o pacote \"tesseract-ocr\" com o gerenciador da sua distribuição.", Note: "Gerenciador de pacotes não reconhecido."}
	case "darwin":
		if has("brew") {
			return InstallPlan{Display: "brew install tesseract", Argv: []string{"brew", "install", "tesseract"}}
		}
		return InstallPlan{Display: "brew install tesseract", Note: "Instale o Homebrew (https://brew.sh) e rode o comando acima."}
	case "windows":
		const id = "UB-Mannheim.TesseractOCR"
		if has("winget") {
			return InstallPlan{
				Display: "winget install --id " + id + " -e",
				Argv:    []string{"winget", "install", "--id", id, "-e", "--accept-source-agreements", "--accept-package-agreements"},
			}
		}
		return InstallPlan{Display: "winget install --id " + id + " -e", Note: "winget não encontrado: baixe o instalador em https://github.com/UB-Mannheim/tesseract/wiki."}
	}
	return InstallPlan{Display: "Instale o Tesseract OCR para o seu sistema."}
}

// Run executes the plan and returns the tail of its output.
func (p InstallPlan) Run(ctx context.Context) (string, error) {
	if !p.CanRun() {
		return "", errNotRunnable
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, p.Argv[0], p.Argv[1:]...).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if len(s) > 2000 {
		s = s[len(s)-2000:]
	}
	return s, err
}
