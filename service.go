package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/thiagozs/ocrtools/internal/capture"
	"github.com/thiagozs/ocrtools/internal/config"
	"github.com/thiagozs/ocrtools/internal/hotkey"
	"github.com/thiagozs/ocrtools/internal/install"
	"github.com/thiagozs/ocrtools/internal/ocr"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// OCRService is bound to the frontend and orchestrates capture -> select -> OCR.
type OCRService struct {
	app *application.App
	cfg *config.Store

	mu        sync.Mutex
	shot      image.Image // whole desktop, physical pixels
	mons      []monitor
	overlays  []*application.WebviewWindow
	result    *application.WebviewWindow
	settings  *application.WebviewWindow
	lastText  string
	lastError string
	busy      bool
	hk        hotkey.Backend
	icon      []byte // app icon PNG used by the installer
}

func NewOCRService(cfg *config.Store) *OCRService { return &OCRService{cfg: cfg} }

func (s *OCRService) engine() ocr.Tesseract {
	return ocr.Tesseract{Path: s.cfg.Get().TesseractPath, DataDir: config.TessdataDir()}
}

// TesseractStatus is what the settings screen shows about the OCR engine.
type TesseractStatus struct {
	Installed      bool     `json:"installed"`
	Version        string   `json:"version"`
	Languages      []string `json:"languages"`
	InstallCommand string   `json:"installCommand"`
	CanInstall     bool     `json:"canInstall"`
	Note           string   `json:"note"`
}

// GetTesseractStatus checks the engine and, when missing, how to install it.
func (s *OCRService) GetTesseractStatus() TesseractStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	st := TesseractStatus{Languages: []string{}}
	v, err := s.engine().Version(ctx)
	if err != nil {
		plan := ocr.PlanInstall()
		st.InstallCommand, st.CanInstall, st.Note = plan.Display, plan.CanRun(), plan.Note
		return st
	}
	st.Installed, st.Version = true, v
	if langs, err := s.engine().Languages(ctx); err == nil {
		st.Languages = langs
	}
	return st
}

// InstallTesseract runs the OS install command (e.g. through pkexec) after the user asked for it.
func (s *OCRService) InstallTesseract() (string, error) {
	return ocr.PlanInstall().Run(context.Background())
}

// DownloadLanguage fetches language models (e.g. "por" or "por+eng") into the app's data dir.
func (s *OCRService) DownloadLanguage(spec string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return ocr.Downloader{Dir: config.TessdataDir()}.Ensure(ctx, spec)
}

// CheckSetup opens the settings window when the OCR engine is missing.
func (s *OCRService) CheckSetup() {
	if !s.GetTesseractStatus().Installed {
		s.OpenSettings()
	}
}

// ---- Tray actions (Go only) ----

// StartCapture grabs the screen and opens the selection overlay on top of it.
func (s *OCRService) StartCapture() {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return
	}
	s.busy = true
	s.mu.Unlock()

	go func() {
		if !s.GetTesseractStatus().Installed {
			s.mu.Lock()
			s.busy = false
			s.mu.Unlock()
			s.OpenSettings() // guides the user through the install
			return
		}
		time.Sleep(250 * time.Millisecond) // let the tray menu close first
		img, err := capture.Desktop()
		if err != nil {
			s.finish("", "Falha ao capturar a tela: "+err.Error())
			return
		}
		mons := listMonitors(s.app)
		if len(mons) == 0 {
			s.finish("", "Nenhum monitor encontrado.")
			return
		}
		s.mu.Lock()
		s.shot, s.mons = img, mons
		s.mu.Unlock()

		// One fullscreen overlay per monitor, each showing only its own slice.
		var wins []*application.WebviewWindow
		for i, m := range mons {
			opts := overlayOptions(application.WebviewWindowOptions{
				Name:             fmt.Sprintf("overlay-%d", i),
				Title:            "OCR Tools — selecionar área",
				URL:              fmt.Sprintf("/overlay.html?m=%d", i),
				Frameless:        true,
				AlwaysOnTop:      true,
				DisableResize:    true,
				Width:            m.Rect.Dx(),
				Height:           m.Rect.Dy(),
				BackgroundColour: application.NewRGB(0, 0, 0),
			}, m)
			w := s.app.Window.NewWithOptions(opts)
			afterCreate(w, i)
			// GTK ignores the first request on a window that is not mapped yet; repeat once it is.
			go func(w *application.WebviewWindow, i int) { time.Sleep(400 * time.Millisecond); afterCreate(w, i) }(w, i)
			wins = append(wins, w)
		}
		s.mu.Lock()
		s.overlays = wins
		s.mu.Unlock()
	}()
}

// OpenSettings shows (or focuses) the settings window.
func (s *OCRService) OpenSettings() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settings != nil {
		s.settings.Show().Focus()
		return
	}
	w := s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "settings",
		Title:  "OCR Tools — Configurações",
		URL:    "/index.html",
		Width:  520,
		Height: 700,
	})
	w.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		s.mu.Lock()
		s.settings = nil
		s.mu.Unlock()
	})
	s.settings = w
}

// ---- Methods exposed to the frontend ----

// GetSettings returns the saved settings with Autostart reflecting the real OS registration.
func (s *OCRService) GetSettings() config.Settings {
	st := s.cfg.Get()
	if on, err := s.app.Autostart.IsEnabled(); err == nil {
		st.Autostart = on
	}
	return st
}

func (s *OCRService) SaveSettings(n config.Settings) error {
	if err := s.ApplyHotkey(n.Hotkey); err != nil {
		return fmt.Errorf("atalho inválido ou já em uso: %w", err)
	}
	if err := s.applyAutostart(n.Autostart); err != nil {
		return fmt.Errorf("iniciar com o sistema: %w", err)
	}
	return s.cfg.Save(n)
}

// applyAutostart makes the OS registration match want (no-op when it already does).
func (s *OCRService) applyAutostart(want bool) error {
	on, err := s.app.Autostart.IsEnabled()
	if err != nil {
		return err
	}
	switch {
	case want && !on:
		return s.app.Autostart.Enable()
	case !want && on:
		return s.app.Autostart.Disable()
	}
	return nil
}

// ApplyHotkey (re)registers the global capture shortcut; "" disables it. If the
// new accelerator is rejected, the previous one stays active.
func (s *OCRService) ApplyHotkey(accel string) error {
	return s.hk.Set(accel, s.StartCapture)
}

// Languages lists the Tesseract languages installed on this machine.
func (s *OCRService) Languages() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	langs, err := s.engine().Languages(ctx)
	if err != nil {
		log.Println("list languages:", err)
		return []string{}
	}
	return langs
}

// monitorPixels maps monitor i onto the desktop screenshot. The screenshot may
// be scaled relative to the compositor layout (HiDPI), so scale proportionally.
func monitorPixels(img image.Image, mons []monitor, i int) (image.Rectangle, bool) {
	if i < 0 || i >= len(mons) {
		return image.Rectangle{}, false
	}
	u := unionRect(mons)
	if u.Empty() {
		return image.Rectangle{}, false
	}
	b := img.Bounds()
	kx := float64(b.Dx()) / float64(u.Dx())
	ky := float64(b.Dy()) / float64(u.Dy())
	m := mons[i].Rect
	r := image.Rect(
		b.Min.X+int(math.Round(float64(m.Min.X-u.Min.X)*kx)),
		b.Min.Y+int(math.Round(float64(m.Min.Y-u.Min.Y)*ky)),
		b.Min.X+int(math.Round(float64(m.Max.X-u.Min.X)*kx)),
		b.Min.Y+int(math.Round(float64(m.Max.Y-u.Min.Y)*ky)),
	).Intersect(b)
	return r, !r.Empty()
}

func subImage(img image.Image, r image.Rectangle) (image.Image, bool) {
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return nil, false
	}
	return sub.SubImage(r), true
}

// GetScreenshot returns monitor idx's slice of the frozen desktop as a JPEG data URL.
func (s *OCRService) GetScreenshot(idx int) (string, error) {
	s.mu.Lock()
	img, mons := s.shot, s.mons
	s.mu.Unlock()
	if img == nil {
		return "", errors.New("no screenshot available")
	}
	r, ok := monitorPixels(img, mons, idx)
	if !ok {
		return "", errors.New("invalid monitor")
	}
	crop, ok := subImage(img, r)
	if !ok {
		return "", errors.New("unsupported image type")
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, crop, &jpeg.Options{Quality: 92}); err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// SubmitSelection receives the selected rectangle on monitor idx as fractions
// (0..1) of that overlay's size, independent of HiDPI scaling.
func (s *OCRService) SubmitSelection(idx int, x, y, w, h float64) {
	s.closeOverlays()
	s.mu.Lock()
	img, mons := s.shot, s.mons
	s.shot = nil
	s.mu.Unlock()
	if img == nil || w <= 0 || h <= 0 {
		s.finish("", "")
		return
	}
	mr, ok := monitorPixels(img, mons, idx)
	if !ok {
		s.finish("", "Monitor inválido.")
		return
	}
	r := image.Rect(
		mr.Min.X+int(math.Round(x*float64(mr.Dx()))),
		mr.Min.Y+int(math.Round(y*float64(mr.Dy()))),
		mr.Min.X+int(math.Round((x+w)*float64(mr.Dx()))),
		mr.Min.Y+int(math.Round((y+h)*float64(mr.Dy()))),
	).Intersect(mr)
	crop, ok := subImage(img, r)
	if r.Empty() || !ok {
		s.finish("", "Seleção inválida.")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
		defer cancel()
		text, err := s.engine().Recognize(ctx, crop, s.cfg.Get().Language)
		if err != nil {
			msg := "Falha no OCR: " + err.Error()
			if strings.Contains(err.Error(), "Failed loading language") || strings.Contains(err.Error(), "Error opening data file") {
				msg = "O idioma \"" + s.cfg.Get().Language + "\" não está instalado. Baixe-o em Configurações."
			}
			s.finish("", msg)
			return
		}
		s.finish(text, "")
	}()
}

// CancelSelection closes the overlay without running OCR.
func (s *OCRService) CancelSelection() {
	s.closeOverlays()
	s.mu.Lock()
	s.shot = nil
	s.mu.Unlock()
	s.finish("", "")
}

// LastResult returns the last recognized text and error for the result window.
func (s *OCRService) LastResult() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]string{"text": s.lastText, "error": s.lastError}
}

// CopyText puts text on the system clipboard.
func (s *OCRService) CopyText(t string) bool { return s.app.Clipboard.SetText(t) }

// ---- internals ----

func (s *OCRService) closeOverlays() {
	s.mu.Lock()
	ws := s.overlays
	s.overlays = nil
	s.mu.Unlock()
	for _, w := range ws {
		w.Close()
	}
}

// finish records the outcome, applies settings (clipboard / result window) and frees the busy flag.
func (s *OCRService) finish(text, errMsg string) {
	s.mu.Lock()
	s.busy = false
	if text == "" && errMsg == "" {
		s.mu.Unlock()
		return // cancelled
	}
	s.lastText, s.lastError = text, errMsg
	s.mu.Unlock()

	cfg := s.cfg.Get()
	if errMsg == "" && cfg.AutoCopy {
		s.app.Clipboard.SetText(text)
	}
	if errMsg != "" || cfg.ShowResult {
		s.showResult()
	}
}

func (s *OCRService) showResult() {
	s.mu.Lock()
	old := s.result
	s.result = nil
	s.mu.Unlock()
	if old != nil {
		old.Close()
	}
	w := s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:        "result",
		Title:       "OCR Tools — Resultado",
		URL:         "/result.html",
		Width:       560,
		Height:      420,
		AlwaysOnTop: true,
	})
	s.mu.Lock()
	s.result = w
	s.mu.Unlock()
}

// ---- Installation ----

const (
	relaunchEnv  = "OCRTOOLS_RELAUNCH"
	autostartEnv = "OCRTOOLS_AUTOSTART" // "1" when the previous binary had login-start enabled
)

func boolEnv(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// GetInstallStatus reports where the app runs from and which install options exist.
func (s *OCRService) GetInstallStatus() install.Status { return install.GetStatus() }

// InstallApp copies the app to the standard location for scope ("user" or "system"),
// then reopens it from there and quits this instance.
func (s *OCRService) InstallApp(scope string) error {
	newExe, err := install.Install(install.Scope(scope), s.icon)
	if err != nil {
		return err
	}
	// Autostart lookups are keyed by the running binary's path, so only this (old) process can see it.
	wasAutostart, _ := s.app.Autostart.IsEnabled()
	if err := relaunch(newExe, wasAutostart); err != nil {
		return fmt.Errorf("instalado em %s, mas não foi possível reabrir: %w", newExe, err)
	}
	go func() { time.Sleep(300 * time.Millisecond); s.app.Quit() }()
	return nil
}

// UninstallApp removes the installed copy and quits. Settings and downloaded languages are kept.
func (s *OCRService) UninstallApp() error {
	if err := install.Uninstall(); err != nil {
		return err
	}
	_ = s.app.Autostart.Disable()
	go func() { time.Sleep(300 * time.Millisecond); s.app.Quit() }()
	return nil
}

// refreshAutostart re-registers login-start for the current binary after an install moved the app.
func (s *OCRService) refreshAutostart() {
	if os.Getenv(autostartEnv) != "1" {
		return
	}
	if err := s.app.Autostart.Enable(); err != nil {
		log.Println("autostart:", err)
	}
}
