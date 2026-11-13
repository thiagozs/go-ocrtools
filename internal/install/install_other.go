//go:build !linux && !windows && !darwin

package install

import "errors"

func userDir() string                  { return "" }
func systemDir() string                { return "" }
func canSystem() bool                  { return false }
func canInstall(string) (bool, string) { return false, "instalação não suportada neste sistema" }
func doInstall(Scope, string, []byte) (string, error) {
	return "", errors.New("instalação não suportada neste sistema")
}
func doUninstall(Scope, string) error { return errors.New("não suportado") }
