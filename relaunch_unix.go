//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// relaunch starts newExe detached from this process, flagged so it waits for us to exit.
// autostart tells it whether login-start was enabled for this (old) binary.
func relaunch(newExe string, autostart bool) error {
	cmd := exec.Command(newExe)
	cmd.Env = append(os.Environ(), relaunchEnv+"=1", autostartEnv+"="+boolEnv(autostart))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
