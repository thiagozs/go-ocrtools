//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func relaunch(newExe string, autostart bool) error {
	cmd := exec.Command(newExe)
	cmd.Env = append(os.Environ(), relaunchEnv+"=1", autostartEnv+"="+boolEnv(autostart))
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x00000008} // NEW_PROCESS_GROUP | DETACHED_PROCESS
	return cmd.Start()
}
