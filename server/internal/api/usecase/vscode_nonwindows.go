//go:build !windows

package usecase

import "errors"

func startDetachedWindowsProcess(executable, workingDir string, args []string) error {
	return errors.New("windows process launch unavailable")
}
