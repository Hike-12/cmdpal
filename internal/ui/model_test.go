package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Hike-12/cmdpal/internal/config"
	projectcontext "github.com/Hike-12/cmdpal/internal/context"
	"github.com/Hike-12/cmdpal/internal/terminal"
)

func fixture(t *testing.T) *Model {
	t.Helper()
	dir := t.TempDir()
	return New(projectcontext.Project{Dir: dir, Badges: []string{"git:main", "go"}},
		config.Store{Path: filepath.Join(dir, "commands.yaml")}, terminal.Shell{Name: "sh"}, nil, nil, "")
}

func keys(m *Model, keys ...string) {
	for _, key := range keys {
		if key == "enter" {
			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		} else if key == "esc" {
			m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		} else {
			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		}
	}
}

func click(t *testing.T, m *Model, action string, index int) {
	t.Helper()
	for _, h := range m.frame().hits {
		if h.action == action && h.index == index {
			m.Update(tea.MouseMsg{X: h.x, Y: h.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			return
		}
	}
	t.Fatalf("missing %s target in:\n%s", action, ansi.Strip(m.View()))
}

func TestMouseSearchEditAndConfirm(t *testing.T) {
	m := fixture(t)
	m.Update(tea.WindowSizeMsg{Width: 108, Height: 26})
	click(t, m, "search", 0)
	keys(m, "stop listener")
	if !strings.Contains(ansi.Strip(m.View()), "Stop listener on a port") {
		t.Fatal(m.View())
	}
	click(t, m, "field", 0)
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	keys(m, "5678")
	if !strings.Contains(ansi.Strip(m.View()), "-iTCP:5678") {
		t.Fatal(m.View())
	}
	click(t, m, "run", 0)
	if m.Command != "" || !strings.Contains(ansi.Strip(m.View()), "Confirm command") {
		t.Fatalf("destructive command must ask for confirmation, command %q:\n%s", m.Command, ansi.Strip(m.View()))
	}
	click(t, m, "confirm", 0)
	if !strings.Contains(m.Command, "-iTCP:5678") {
		t.Fatalf("command = %q", m.Command)
	}
}

func TestSaveCustomCommand(t *testing.T) {
	m := fixture(t)
	keys(m, "n", "Hello", "enter", "echo {{message}}", "enter", "Print text", "enter")
	recipes, err := m.store.Load()
	if err != nil || len(recipes) != 1 || recipes[0].Command != "echo {{message}}" {
		t.Fatalf("custom recipes = %v, %v", recipes, err)
	}
	if !strings.Contains(ansi.Strip(m.View()), "Saved Hello") {
		t.Fatal(m.View())
	}
}

func TestLayoutFits(t *testing.T) {
	m := fixture(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 36, Height: 16}, {Width: 64, Height: 22}, {Width: 84, Height: 24}, {Width: 130, Height: 34}} {
		m.Update(size)
		lines := strings.Split(m.View(), "\n")
		if len(lines) > size.Height {
			t.Fatalf("%dx%d: got %d rows", size.Width, size.Height, len(lines))
		}
		for _, line := range lines {
			if lipgloss.Width(line) > size.Width {
				t.Fatalf("%dx%d: row too wide: %q", size.Width, size.Height, ansi.Strip(line))
			}
		}
		for _, action := range []string{"run", "copy", "new"} {
			found := false
			for _, hit := range m.frame().hits {
				if hit.action == action {
					found = hit.x+hit.width <= size.Width && hit.y < size.Height
				}
			}
			if !found {
				t.Fatalf("%dx%d: missing %s", size.Width, size.Height, action)
			}
		}
	}
}
