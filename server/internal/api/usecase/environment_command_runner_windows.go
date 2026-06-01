//go:build windows

package usecase

import (
	"os/exec"
	"syscall"
)

func configureEnvironmentCommandProcess(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}
