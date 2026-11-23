//go:build !windows

package terminal

import "os/exec"

func shellCommand(shell Shell, args []string) *exec.Cmd {
	return exec.Command(shell.Path, args...)
}
