//go:build windows

package usecase

import (
	"errors"
	"os/exec"
	"syscall"
)

func startDetachedWindowsProcess(executable, workingDir string, args []string) error {
	wrapper := findWindowsPowerShellHost()
	if wrapper == "" {
		return errors.New("unable to find Windows PowerShell host")
	}
	cmd := exec.Command(
		wrapper,
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle",
		"Hidden",
		"-Command",
		buildWindowsStartProcessScript(executable, workingDir, args),
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd.Run()
}
