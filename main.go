package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Hike-12/cmdpal/internal/config"
	projectcontext "github.com/Hike-12/cmdpal/internal/context"
	"github.com/Hike-12/cmdpal/internal/terminal"
	"github.com/Hike-12/cmdpal/internal/ui"
)

func main() {
	if err := run(); err != nil {
		var exited *exec.ExitError
		if errors.As(err, &exited) && exited.ExitCode() > 0 {
			os.Exit(exited.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "cmdpal:", err)
		os.Exit(1)
	}
}

func run() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	project, err := projectcontext.Detect(dir)
	if err != nil {
		return err
	}
	store := config.Store{Path: filepath.Join(home, ".cmdpal", "commands.yaml")}
	custom, err := store.Load()
	warning := ""
	if err != nil {
		warning = fmt.Sprintf("Custom commands: %v", err)
	}
	shell := terminal.NativeShell()
	history := projectcontext.HistoryFiles(home, os.Getenv("APPDATA"))
	history["cmdpal"] = filepath.Join(home, ".cmdpal", "history")
	if path := os.Getenv("HISTFILE"); path != "" {
		history[shell.Name] = path
	}
	state := ui.New(project, store, shell, custom, projectcontext.RecentHistory(history), warning)
	program := tea.NewProgram(state, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		return err
	}
	if state.Command == "" {
		return nil
	}
	historyPath := history[shell.Name]
	if historyPath == "" {
		historyPath = history["cmdpal"]
	}
	if err := projectcontext.AppendHistory(historyPath, shell.Name, state.Command); err != nil {
		fmt.Fprintln(os.Stderr, "cmdpal: could not record command:", err)
	}
	return shell.Run(state.Command)
}
