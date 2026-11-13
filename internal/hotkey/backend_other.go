//go:build !linux

package hotkey

import "github.com/wailsapp/wails/v3/pkg/application"

// NewBackend returns the Wails-native backend; icon is unused here.
func NewBackend(app *application.App, _ []byte) Backend { return &wailsBackend{app: app} }
