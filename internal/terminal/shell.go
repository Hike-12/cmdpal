package terminal

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"
)

type Shell struct {
	Name string
	Path string
}

func NativeShell() Shell {
	if runtime.GOOS == "windows" {
		for _, name := range []string{"pwsh", "powershell"} {
			if path, err := exec.LookPath(name); err == nil {
				return Shell{"pwsh", path}
			}
		}
		path := os.Getenv("ComSpec")
		if path == "" {
			path = "cmd.exe"
		}
		return Shell{"cmd", path}
	}
	path := os.Getenv("SHELL")
	if path == "" {
		path = "/bin/sh"
	}
	return Shell{filepath.Base(path), path}
}

func (s Shell) Run(command string) error {
	args := []string{"-c", command}
	switch s.Name {
	case "pwsh":
		args = []string{"-NoLogo", "-NoProfile", "-Command", command}
	case "cmd":
		args = []string{"/d", "/v:off", "/s", "/c", command}
	}
	child := shellCommand(s, args)
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	fmt.Fprintln(os.Stdout, command)
	return child.Run()
}

func Copy(text string) error {
	var tools [][]string
	input := []byte(text)
	switch runtime.GOOS {
	case "windows":
		tools = [][]string{{"clip.exe"}}
		input = []byte{0xff, 0xfe}
		for _, unit := range utf16.Encode([]rune(text)) {
			input = append(input, byte(unit), byte(unit>>8))
		}
	case "darwin":
		tools = [][]string{{"pbcopy"}}
	default:
		tools = [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}}
	}
	for _, tool := range tools {
		command := exec.Command(tool[0], tool[1:]...)
		command.Stdin = bytes.NewReader(input)
		if err := command.Run(); err == nil {
			return nil
		}
	}
	var names []string
	for _, tool := range tools {
		names = append(names, tool[0])
	}
	return fmt.Errorf("clipboard unavailable (%s)", strings.Join(names, ", "))
}
