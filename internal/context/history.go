package context

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var zshPrefix = regexp.MustCompile(`^: [0-9]+:[0-9]+;`)
var bashTimestamp = regexp.MustCompile(`^#[0-9]+$`)

func HistoryFiles(home, appdata string) map[string]string {
	paths := map[string]string{
		"bash": filepath.Join(home, ".bash_history"),
		"zsh":  filepath.Join(home, ".zsh_history"),
	}
	if appdata != "" {
		paths["pwsh"] = filepath.Join(appdata, "Microsoft", "Windows", "PowerShell", "PSReadLine", "ConsoleHost_history.txt")
	} else {
		paths["pwsh"] = filepath.Join(home, ".local", "share", "powershell", "PSReadLine", "ConsoleHost_history.txt")
	}
	return paths
}

func ParseHistory(input io.Reader, shell string) ([]string, error) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 256*1024)
	var lines []string
	skip := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if shell == "bash" && bashTimestamp.MatchString(line) {
			continue
		}
		multiline := (shell == "zsh" && strings.HasSuffix(line, "\\")) ||
			(shell == "pwsh" && strings.HasSuffix(line, "`"))
		if skip || multiline {
			skip = multiline
			continue
		}
		if shell == "zsh" {
			line = zshPrefix.ReplaceAllString(line, "")
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func RecentHistory(paths map[string]string) []string {
	var recent []string
	for _, shell := range []string{"zsh", "bash", "pwsh", "fish", "cmd", "sh"} {
		file, err := os.Open(paths[shell])
		if err != nil {
			continue
		}
		info, err := file.Stat()
		if err == nil && info.Size() > 256*1024 {
			_, _ = file.Seek(-256*1024, io.SeekEnd)
			reader := bufio.NewReader(file)
			_, _ = reader.ReadString('\n')
			lines, _ := ParseHistory(reader, shell)
			recent = append(recent, tail(lines)...)
		} else {
			lines, _ := ParseHistory(file, shell)
			recent = append(recent, tail(lines)...)
		}
		_ = file.Close()
	}
	return recent
}

func tail(lines []string) []string {
	if len(lines) > 100 {
		return lines[len(lines)-100:]
	}
	return lines
}

func AppendHistory(path, shell, command string) error {
	if path == "" {
		return nil
	}
	if strings.ContainsAny(command, "\r\n") {
		return fmt.Errorf("cannot record a multiline command")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if shell == "zsh" {
		command = fmt.Sprintf(": %d:0;%s", time.Now().Unix(), command)
	}
	_, err = fmt.Fprintln(file, command)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
