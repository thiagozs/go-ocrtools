// Package config persists user settings as JSON in the OS config directory.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Settings are the user-editable options exposed to the frontend.
type Settings struct {
	// Language is a Tesseract language spec, e.g. "eng", "por" or "por+eng".
	Language string `json:"language"`
	// AutoCopy copies the recognized text to the clipboard.
	AutoCopy bool `json:"autoCopy"`
	// ShowResult opens the result window after each capture.
	ShowResult bool `json:"showResult"`
	// Hotkey is the global capture shortcut in Wails accelerator syntax
	// (e.g. "CmdOrCtrl+Alt+O"); empty disables it.
	Hotkey string `json:"hotkey"`
	// Autostart launches the app at login. The OS registration is the source of
	// truth; this field only carries the value between the UI and the service.
	Autostart bool `json:"autostart"`
	// TesseractPath overrides the tesseract binary; empty means "look in PATH".
	TesseractPath string `json:"tesseractPath"`
}

// Defaults returns the initial settings.
func Defaults() Settings {
	return Settings{Language: "eng", AutoCopy: true, ShowResult: true, Hotkey: "CmdOrCtrl+Alt+O"}
}

// Store loads and saves Settings, safe for concurrent use.
type Store struct {
	mu   sync.RWMutex
	path string
	cur  Settings
}

// Dir returns the app's config directory (created on demand by writers).
func Dir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "ocrtools")
}

// TessdataDir is where downloaded Tesseract language files are kept.
func TessdataDir() string {
	if d := Dir(); d != "" {
		return filepath.Join(d, "tessdata")
	}
	return ""
}

// NewStore loads settings from the default location, falling back to defaults.
func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewStoreAt(filepath.Join(dir, "ocrtools", "settings.json"))
}

// NewStoreAt is like NewStore with an explicit file path.
func NewStoreAt(path string) (*Store, error) {
	s := &Store{path: path, cur: Defaults()}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.cur); err != nil {
		return nil, err
	}
	if s.cur.Language == "" {
		s.cur.Language = Defaults().Language
	}
	return s, nil
}

// Get returns a copy of the current settings.
func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cur
}

// Save validates, stores and persists new settings.
func (s *Store) Save(n Settings) error {
	if n.Language == "" {
		n.Language = Defaults().Language
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.path, b, 0o644); err != nil {
		return err
	}
	s.cur = n
	return nil
}
