package terminal

import (
	"os/exec"
	"syscall"
)

func shellCommand(shell Shell, args []string) *exec.Cmd {
	if shell.Name != "cmd" {
		return exec.Command(shell.Path, args...)
	}
	// cmd uses its own quoting rules for the command after /s /c.
	process := exec.Command(shell.Path)
	process.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: `"` + shell.Path + `" /d /v:off /s /c "` + args[len(args)-1] + `"`,
	}
	return process
}
