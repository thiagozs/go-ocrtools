package main

import (
	"embed"
	"log"
	"os"
	"time"

	"github.com/thiagozs/ocrtools/internal/config"
	"github.com/thiagozs/ocrtools/internal/hotkey"
	"github.com/thiagozs/ocrtools/internal/install"
	"github.com/thiagozs/ocrtools/internal/launcher"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/trayicon.png
var trayIcon []byte

//go:embed build/appicon.png
var appIcon []byte

func main() {
	// Windows "Installed apps" runs `ocrtools.exe --uninstall`.
	if len(os.Args) > 1 && os.Args[1] == "--uninstall" {
		if err := install.Uninstall(); err != nil {
			log.Fatal(err)
		}
		return
	}
	// Right after an install the old instance is still quitting; let it release the tray and shortcuts.
	relaunched := os.Getenv(relaunchEnv) != ""
	if relaunched {
		time.Sleep(1500 * time.Millisecond)
	}

	if err := launcher.Ensure(appIcon); err != nil {
		log.Println("launcher:", err)
	}

	cfg, err := config.NewStore()
	if err != nil {
		log.Fatal(err)
	}
	svc := NewOCRService(cfg)

	app := application.New(application.Options{
		Name:        "OCR Tools",
		Description: "Captura uma área da tela e extrai o texto via OCR",
		Services:    []application.Service{application.NewService(svc)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		// Tray app: keep running when the last window (overlay/result/settings) closes.
		Linux: application.LinuxOptions{
			// Must match the .desktop file name: GTK uses it as the Wayland app_id, which is
			// how the dock finds the app's name and icon.
			ApplicationID:                 launcher.AppID,
			DisableQuitOnLastWindowClosed: true,
		},
		Windows: application.WindowsOptions{DisableQuitOnLastWindowClosed: true},
		Mac: application.MacOptions{
			// Tray-only app: no Dock icon, keep running with no windows open.
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	svc.app = app
	svc.icon = appIcon
	svc.hk = hotkey.NewBackend(app, trayIcon)
	if err := svc.ApplyHotkey(cfg.Get().Hotkey); err != nil {
		log.Println("atalho global:", err)
	}

	menu := application.NewMenu()
	menu.Add("Capturar texto").OnClick(func(*application.Context) { svc.StartCapture() })
	menu.Add("Configurações").OnClick(func(*application.Context) { svc.OpenSettings() })
	menu.AddSeparator()
	menu.Add("Sair").OnClick(func(*application.Context) { app.Quit() })

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("OCR Tools")
	tray.SetMenu(menu)
	tray.OnClick(func() { svc.StartCapture() })

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		if relaunched {
			svc.refreshAutostart()
		}
		go svc.CheckSetup()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
