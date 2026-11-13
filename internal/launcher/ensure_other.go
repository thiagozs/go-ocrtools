//go:build !linux

package launcher

// Ensure is a no-op outside Linux: Windows and macOS identify apps by their executable/bundle.
func Ensure([]byte) error { return nil }
