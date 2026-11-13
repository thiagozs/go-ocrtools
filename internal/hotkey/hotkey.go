// Package hotkey registers the global capture shortcut on every platform.
package hotkey

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Backend binds one global shortcut; Set("") removes it. If the new accelerator
// is rejected synchronously, the previous binding is kept.
type Backend interface {
	Set(accel string, cb func()) error
}

// wailsBackend uses Wails' native global shortcuts (X11, Windows, macOS).
type wailsBackend struct {
	app *application.App
	mu  sync.Mutex
	cur string
}

func (b *wailsBackend) Set(accel string, cb func()) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if accel == b.cur {
		return nil
	}
	old := b.cur
	if old != "" {
		_ = b.app.GlobalShortcut.Unregister(old)
	}
	if accel != "" {
		if err := b.app.GlobalShortcut.Register(accel, cb); err != nil {
			if old != "" {
				_ = b.app.GlobalShortcut.Register(old, cb)
			}
			return err
		}
	}
	b.cur = accel
	return nil
}
