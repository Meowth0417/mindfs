//go:build !windows

package usecase

import "os/exec"

func configureEnvironmentCommandProcess(cmd *exec.Cmd) {}
