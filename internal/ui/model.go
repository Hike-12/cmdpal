package ui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Hike-12/cmdpal/internal/config"
	projectcontext "github.com/Hike-12/cmdpal/internal/context"
	"github.com/Hike-12/cmdpal/internal/terminal"
)

const (
	listFocus   = -2
	searchFocus = -1
)

type row struct {
	label string
	index int
}

type Model struct {
	project                      projectcontext.Project
	store                        config.Store
	shell                        terminal.Shell
	history                      []string
	recipes                      []config.Recipe
	rows                         []row
	selected, listTop, detailTop int
	width, height                int
	focus                        int
	search                       textinput.Model
	params                       []string
	inputs                       []textinput.Model
	suggestions                  map[string][]string
	form                         []textinput.Model
	formFocus                    int
	confirm                      bool
	status                       string
	toastID                      int
	Command                      string
}

func New(project projectcontext.Project, store config.Store, shell terminal.Shell, custom []config.Recipe, history []string, warning string) *Model {
	m := &Model{
		project: project, store: store, shell: shell, history: history,
		recipes: append(config.Catalog(), custom...), selected: -1,
		width: 100, height: 30, focus: listFocus, status: warning,
		search:      input("Filter commands", ""),
		suggestions: map[string][]string{"branch": project.Branches(), "file": project.Files()},
	}
	m.refresh(true)
	return m
}

func input(placeholder, value string) textinput.Model {
	field := textinput.New()
	field.Prompt = ""
	field.Placeholder = placeholder
	field.PlaceholderStyle = subtle
	field.TextStyle = strong
	field.Cursor.Style = strong
	field.SetValue(value)
	return field
}

type containersMsg []string
type copiedMsg struct{ err error }
type clearToast int

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		output, err := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}").Output()
		if err != nil {
			return containersMsg(nil)
		}
		return containersMsg(strings.Fields(string(output)))
	}
}

func (m *Model) refresh(keepSelection bool) {
	previous := -1
	if m.selected >= 0 && m.selected < len(m.rows) {
		previous = m.rows[m.selected].index
	}
	query := m.search.Value()
	m.rows = nil
	addGroup := func(label string, indices []int) {
		start := len(m.rows)
		for _, index := range indices {
			if len(config.Filter(m.recipes[index:index+1], query)) != 0 {
				m.rows = append(m.rows, row{index: index})
			}
		}
		if len(m.rows) > start {
			if query != "" {
				slices.SortStableFunc(m.rows[start:], func(a, b row) int {
					return config.MatchScore(m.recipes[b.index], query) - config.MatchScore(m.recipes[a.index], query)
				})
			}
			m.rows = slices.Insert(m.rows, start, row{label: label, index: -1})
		}
	}
	addGroup("★ CONTEXT", config.Pinned(m.recipes, m.project.Badges, m.history))
	for _, category := range []string{config.Git, config.Docker, config.Network, config.Media, config.System, "CUSTOM"} {
		var indices []int
		for i, recipe := range m.recipes {
			if recipe.Category == category {
				indices = append(indices, i)
			}
		}
		addGroup(category, indices)
	}
	m.listTop = 0
	m.selected = -1
	for i, item := range m.rows {
		if item.index >= 0 && (m.selected < 0 || (keepSelection && item.index == previous)) {
			m.selected = i
		}
		if keepSelection && item.index == previous && previous >= 0 {
			break
		}
	}
	if r, ok := m.recipe(); !ok || m.rows[m.selected].index != previous || (len(m.inputs) == 0 && len(r.Params(m.shell.Name)) > 0) {
		m.loadInputs()
	}
	m.revealSelection()
}

func (m *Model) recipe() (config.Recipe, bool) {
	if m.selected < 0 || m.selected >= len(m.rows) || m.rows[m.selected].index < 0 {
		return config.Recipe{}, false
	}
	return m.recipes[m.rows[m.selected].index], true
}

func (m *Model) loadInputs() {
	m.detailTop = 0
	m.params, m.inputs = nil, nil
	r, ok := m.recipe()
	if !ok {
		return
	}
	for _, name := range r.Params(m.shell.Name) {
		m.params = append(m.params, name)
		m.inputs = append(m.inputs, input(name, m.defaultValue(r, name)))
	}
}

func (m *Model) defaultValue(r config.Recipe, name string) string {
	if value, ok := r.Defaults[name]; ok {
		return value
	}
	switch name {
	case "port":
		return "8080"
	case "branch":
		branches := m.suggestions[name]
		if r.Name == "Switch branch" {
			for _, branch := range branches {
				if branch != m.project.Branch {
					return branch
				}
			}
		}
		if m.project.Branch != "" {
			return m.project.Branch
		}
	case "count":
		return "3"
	case "crf":
		return "23"
	case "url":
		return "https://example.com"
	case "message":
		return filepath.Base(m.project.Dir)
	case "output":
		switch r.Name {
		case "Compress video":
			return "compressed.mp4"
		case "Extract MP3 audio":
			return "audio.mp3"
		case "Create tar.gz archive":
			return "archive.tar.gz"
		}
	case "file", "container":
		values := m.suggestions[name]
		if name == "file" {
			for _, file := range values {
				if r.Category != config.Media ||
					(r.Name == "Extract tar.gz archive" && strings.HasSuffix(file, ".tar.gz")) ||
					(r.Name != "Extract tar.gz archive" && isMedia(file)) {
					return file
				}
			}
			if r.Category == config.Media && r.Name != "Create tar.gz archive" {
				break
			}
		}
		if len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func isMedia(file string) bool {
	return slices.Contains([]string{".mp4", ".mkv", ".mov", ".avi", ".webm", ".wav", ".m4a", ".mp3"}, strings.ToLower(filepath.Ext(file)))
}

func (m *Model) values() map[string]string {
	fields := make(map[string]string, len(m.params))
	for i, name := range m.params {
		fields[name] = m.inputs[i].Value()
	}
	return fields
}

func (m *Model) setFocus(focus int) tea.Cmd {
	m.search.Blur()
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.focus = focus
	if focus == searchFocus {
		return m.search.Focus()
	}
	if focus >= 0 && focus < len(m.inputs) {
		cmd := m.inputs[focus].Focus()
		m.revealInput(focus)
		return cmd
	}
	return nil
}

func (m *Model) move(step int) {
	for i := m.selected + step; i >= 0 && i < len(m.rows); i += step {
		if m.rows[i].index >= 0 {
			m.selectRow(i)
			return
		}
	}
}

func (m *Model) selectRow(index int) {
	if index != m.selected {
		previous := -1
		if m.selected >= 0 {
			previous = m.rows[m.selected].index
		}
		m.selected = index
		if m.rows[index].index != previous {
			m.loadInputs()
		}
	}
	m.revealSelection()
}

func (m *Model) action(action string) tea.Cmd {
	switch action {
	case "new":
		m.form = []textinput.Model{input("Name", ""), input("Command", ""), input("Summary (optional)", "")}
		m.form[1].CharLimit = 4000
		m.formFocus = 0
		m.status = ""
		return m.form[0].Focus()
	case "copy", "run":
		r, ok := m.recipe()
		if !ok {
			return nil
		}
		command, err := r.Render(m.shell.Name, m.values())
		if err != nil {
			m.status = err.Error()
			return nil
		}
		if action == "copy" {
			return func() tea.Msg { return copiedMsg{terminal.Copy(command)} }
		}
		if r.Safety == config.Destructive && !m.confirm {
			m.confirm = true
			return nil
		}
		m.Command = command
		return tea.Quit
	}
	return nil
}

func (m *Model) nextInput(step int) tea.Cmd {
	next := m.focus + step
	if next >= len(m.inputs) {
		next = listFocus
	}
	if next < listFocus {
		next = len(m.inputs) - 1
	}
	return m.setFocus(next)
}

func (m *Model) cycleSuggestion() {
	if m.focus < 0 {
		return
	}
	values := m.suggestions[m.params[m.focus]]
	if len(values) != 0 {
		field := &m.inputs[m.focus]
		field.SetValue(values[(slices.Index(values, field.Value())+1)%len(values)])
	}
}

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.revealSelection()
		if m.focus >= 0 {
			m.revealInput(m.focus)
		}
		return m, nil
	case containersMsg:
		m.suggestions["container"] = msg
		if i := slices.Index(m.params, "container"); i >= 0 && m.inputs[i].Value() == "" && len(msg) > 0 {
			r, _ := m.recipe()
			m.inputs[i].SetValue(m.defaultValue(r, "container"))
		}
		return m, nil
	case copiedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		m.status = "Copied!"
		m.toastID++
		id := m.toastID
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearToast(id) })
	case clearToast:
		if int(msg) == m.toastID && m.status == "Copied!" {
			m.status = ""
		}
		return m, nil
	case tea.MouseMsg:
		return m, m.mouse(msg)
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.confirm {
			switch key {
			case "y":
				return m, m.action("run")
			case "esc", "n", "enter":
				m.confirm = false
			}
			return m, nil
		}
		if m.form != nil {
			switch key {
			case "esc":
				m.form, m.status = nil, ""
			case "tab", "down":
				return m, m.focusForm((m.formFocus + 1) % len(m.form))
			case "shift+tab", "up":
				return m, m.focusForm((m.formFocus + len(m.form) - 1) % len(m.form))
			case "enter":
				if m.formFocus == len(m.form)-1 {
					return m, m.saveForm()
				}
				return m, m.focusForm(m.formFocus + 1)
			default:
				return m, m.updateText(message)
			}
			return m, nil
		}
		switch key {
		case "esc":
			if m.focus == listFocus {
				return m, tea.Quit
			}
			return m, m.setFocus(listFocus)
		case "tab":
			return m, m.nextInput(1)
		case "shift+tab":
			return m, m.nextInput(-1)
		case "up", "down":
			step := 1
			if key == "up" {
				step = -1
			}
			if m.focus >= 0 {
				return m, m.nextInput(step)
			}
			m.move(step)
			return m, nil
		case "pgup", "pgdown":
			delta := m.dimensions().detailRows
			if key == "pgup" {
				delta = -delta
			}
			m.scrollDetail(delta)
			return m, nil
		case "enter":
			return m, m.action("run")
		case "ctrl+space":
			m.cycleSuggestion()
			return m, nil
		}
		if m.focus == listFocus {
			switch key {
			case "/":
				return m, m.setFocus(searchFocus)
			case "c":
				return m, m.action("copy")
			case "n":
				return m, m.action("new")
			case "q":
				return m, tea.Quit
			}
		}
	}
	return m, m.updateText(message)
}

func (m *Model) updateText(message tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch {
	case m.form != nil:
		m.form[m.formFocus], cmd = m.form[m.formFocus].Update(message)
	case m.focus == searchFocus:
		before := m.search.Value()
		m.search, cmd = m.search.Update(message)
		if m.search.Value() != before {
			m.refresh(false)
		}
	case m.focus >= 0 && m.focus < len(m.inputs):
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(message)
	}
	return cmd
}

func (m *Model) focusForm(index int) tea.Cmd {
	m.form[m.formFocus].Blur()
	m.formFocus = index
	return m.form[index].Focus()
}

func (m *Model) saveForm() tea.Cmd {
	r := config.Recipe{Name: m.form[0].Value(), Command: m.form[1].Value(), Summary: m.form[2].Value()}
	if err := m.store.Add(r); err != nil {
		m.status = err.Error()
		return nil
	}
	custom, err := m.store.Load()
	if err != nil {
		m.status = err.Error()
		return nil
	}
	m.recipes = append(config.Catalog(), custom...)
	m.form = nil
	m.search.SetValue("")
	m.refresh(true)
	for i, item := range m.rows {
		if item.index == len(m.recipes)-1 {
			m.selectRow(i)
			break
		}
	}
	m.status = fmt.Sprintf("Saved %s", r.Name)
	return m.setFocus(listFocus)
}
