package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type dimensions struct {
	width, left, right, listRows, detailRows int
	stack                                    bool
}

func (m *Model) dimensions() dimensions {
	w := max(32, m.width-4)
	d := dimensions{width: w, stack: w < 78}
	if d.stack {
		d.left, d.right = w, w
		d.listRows = max(2, (m.height-13)/3)
		d.detailRows = max(2, m.height-12-d.listRows)
	} else {
		d.left = (w - 1) * 42 / 100
		d.right = w - 1 - d.left
		d.listRows = max(2, m.height-10)
		d.detailRows = d.listRows - 1
	}
	return d
}

type hit struct {
	x, y, width int
	action      string
	index       int
}

type frame struct {
	lines []string
	hits  []hit
}

func (f *frame) line(text string, width int) {
	f.lines = append(f.lines, fit("  "+text, width+4, canvas))
}

type detail struct {
	text  string
	input int
}

func (m *Model) details(width int) []detail {
	r, ok := m.recipe()
	if !ok {
		return []detail{{subtle.Render("No matching commands"), -1}}
	}
	var lines []detail
	line := func(text string) { lines = append(lines, detail{text, -1}) }
	line(title.Render(clean(r.Name)))
	line(safetyTag(r.Safety))
	if r.Summary != "" {
		for _, part := range wrap(r.Summary, width) {
			line(subtle.Render(part))
		}
	}
	if len(m.inputs) > 0 {
		line("")
		line(subtle.Render("ARGUMENTS"))
		for i, field := range m.inputs {
			label := ansi.Truncate(m.params[i], max(6, width/3), "…")
			prefix := lipgloss.NewStyle().Width(max(9, width/3) + 1).Render(label)
			pointer := "· "
			if m.focus == i {
				pointer = "› "
			}
			field.Width = max(2, width-2-lipgloss.Width(prefix)-1)
			lines = append(lines, detail{subtle.Render(pointer+prefix) + field.View(), i})
		}
	}
	line("")
	line(subtle.Render("COMMAND"))
	for _, part := range wrap(r.Preview(m.shell.Name, m.values()), width) {
		line(strong.Render(part))
	}
	if len(r.Flags) > 0 {
		line("")
		line(subtle.Render("FLAGS"))
		for _, flag := range r.Flags {
			for _, part := range wrap(flag.Name+"  "+flag.Description, width) {
				line(subtle.Render(part))
			}
		}
	}
	return lines
}

func wrap(text string, width int) []string {
	return strings.Split(ansi.Hardwrap(ansi.Wrap(clean(text), width, ""), width, true), "\n")
}

func (m *Model) revealSelection() {
	count := m.dimensions().listRows
	if m.selected < m.listTop {
		m.listTop = m.selected
	} else if m.selected >= m.listTop+count {
		m.listTop = m.selected - count + 1
	}
	m.listTop = max(0, min(m.listTop, len(m.rows)-count))
}

func (m *Model) scrollDetail(delta int) {
	d := m.dimensions()
	m.detailTop = min(max(0, m.detailTop+delta), max(0, len(m.details(d.right-4))-d.detailRows))
}

func (m *Model) revealInput(index int) {
	d := m.dimensions()
	for i, row := range m.details(d.right - 4) {
		if row.input == index {
			if i < m.detailTop {
				m.detailTop = i
			} else if i >= m.detailTop+d.detailRows {
				m.detailTop = i - d.detailRows + 1
			}
			break
		}
	}
}

func (m *Model) View() string {
	return strings.Join(m.frame().lines, "\n")
}

func (m *Model) frame() frame {
	var f frame
	if m.width < 36 || m.height < 16 {
		f.lines = []string{ansi.Truncate("Resize to at least 36 × 16", max(1, m.width), "…")}
		return f
	}
	d := m.dimensions()
	f.line("", d.width)
	path := clean(m.project.Dir)
	f.line(title.Render("cmdpal")+"   "+subtle.Render(path), d.width)
	var badges []string
	for _, badge := range m.project.Badges {
		badges = append(badges, "["+clean(badge)+"]")
	}
	search := m.search
	badgeLine := strings.Join(badges, " ")
	badgeWidth := min(d.width/2, lipgloss.Width(badgeLine))
	searchWidth := d.width
	if badgeWidth > 0 {
		searchWidth -= badgeWidth + 2
	}
	search.Width = searchWidth - 6
	mark := subtle.Render(" / ")
	if m.focus == searchFocus && m.form == nil && !m.confirm {
		mark = title.Render(" / ")
	}
	f.hits = append(f.hits, hit{2, len(f.lines), searchWidth, "search", 0})
	f.line(fit(mark+search.View(), searchWidth, surface)+"  "+subtle.Render(ansi.Truncate(badgeLine, badgeWidth, "…")), d.width)
	f.line("", d.width)
	if m.confirm || m.form != nil {
		m.dialog(&f, d.width)
		return f
	}

	listY, detailY := len(f.lines)+1, len(f.lines)+1
	detailX := 2 + d.left + 1
	if d.stack {
		detailX = 2
		detailY += d.listRows + 2
	}
	left := []string{panelEdge(d.left, "Commands", true)}
	for line := 0; line < d.listRows; line++ {
		index := m.listTop + line
		var text string
		if index < len(m.rows) {
			row := m.rows[index]
			if row.index < 0 {
				text = subtle.Bold(true).Render(row.label)
			} else {
				text = strong.Render("  " + clean(m.recipes[row.index].Name))
				if index == m.selected {
					text = selectedStyle.Render(fit("› "+clean(m.recipes[row.index].Name), d.left-4, border))
				}
				f.hits = append(f.hits, hit{2, listY + line, d.left, "select", index})
			}
		} else if len(m.rows) == 0 && line == 0 {
			text = subtle.Render("No matches")
		}
		left = append(left, panelLine(text, d.left))
	}
	left = append(left, panelEdge(d.left, "", false))

	details := m.details(d.right - 4)
	top := min(m.detailTop, max(0, len(details)-d.detailRows))
	right := []string{panelEdge(d.right, "Inspector", true)}
	for line := 0; line < d.detailRows; line++ {
		index := top + line
		var text string
		if index < len(details) {
			text = details[index].text
			if details[index].input >= 0 {
				f.hits = append(f.hits, hit{detailX, detailY + line, d.right, "field", details[index].input})
			}
		}
		right = append(right, panelLine(text, d.right))
	}
	if !d.stack {
		indicator := ""
		if len(details) > d.detailRows {
			indicator = fmt.Sprintf("PgUp/PgDn  %d–%d of %d", top+1, min(len(details), top+d.detailRows), len(details))
		}
		right = append(right, panelLine(subtle.Render(indicator), d.right))
	}
	right = append(right, panelEdge(d.right, "", false))
	if d.stack {
		for _, line := range append(left, right...) {
			f.line(line, d.width)
		}
	} else {
		for i := range left {
			f.line(left[i]+" "+right[i], d.width)
		}
	}
	f.line("", d.width)
	run, copyText, newText := "▶ RUN COMMAND", "▣ COPY", "+ NEW"
	if d.width < 40 {
		run, copyText = "▶ RUN", "COPY"
	}
	buttons := f.buttons(2, []string{run, copyText, newText}, []string{"run", "copy", "new"})
	f.line(buttons, d.width)
	if m.status != "" {
		f.line(strong.Render(clean(m.status)), d.width)
	} else if m.focus >= 0 && len(m.suggestions[m.params[m.focus]]) > 0 {
		f.line(subtle.Render("Ctrl+Space  next suggestion    Tab  next field"), d.width)
	} else {
		f.line(subtle.Render("↑↓ Select   / Search   Tab Edit   Enter Run   c Copy   n New"), d.width)
	}
	return f
}

func (f *frame) buttons(x int, labels, actions []string) string {
	var parts []string
	for i, label := range labels {
		style := button
		if i == 0 {
			style = primary
		}
		part := style.Render(label)
		width := lipgloss.Width(part)
		f.hits = append(f.hits, hit{x, len(f.lines), width, actions[i], 0})
		x += width + 1
		parts = append(parts, part)
	}
	return strings.Join(parts, " ")
}

func (m *Model) dialog(f *frame, outer int) {
	width := min(66, outer)
	offset := (outer - width) / 2
	prefix := strings.Repeat(" ", offset)
	inner := width - 4
	line := func(text string) { f.line(prefix+panelLine(text, width), outer) }
	var heading string
	if m.confirm {
		heading = "Confirm command"
	} else {
		heading = "New command"
	}
	f.line(prefix+panelEdge(width, heading, true), outer)
	line("")
	if m.confirm {
		r, _ := m.recipe()
		line(safetyTag(r.Safety) + "  " + strong.Render(clean(r.Name)))
		preview := wrap(r.Preview(m.shell.Name, m.values()), inner)
		for _, text := range preview[:min(m.height-11, len(preview))] {
			line(strong.Render(text))
		}
		if len(preview) > m.height-11 {
			line(subtle.Render("… full command in inspector"))
		}
		line("")
		line(f.buttons(offset+4, []string{"RUN  y", "CANCEL  esc"}, []string{"confirm", "cancel"}))
	} else {
		for i, field := range m.form {
			label := []string{"Name", "Command", "Summary"}[i]
			field.Width = inner - 11
			f.hits = append(f.hits, hit{offset + 4, len(f.lines), inner, "form", i})
			line(subtle.Width(10).Render(label) + field.View())
		}
		line("")
		line(f.buttons(offset+4, []string{"SAVE", "CANCEL  esc"}, []string{"save", "cancel"}))
	}
	if m.status != "" {
		line(strong.Render(clean(m.status)))
	}
	f.line(prefix+panelEdge(width, "", false), outer)
}

func (m *Model) mouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		for _, h := range m.frame().hits {
			if msg.Y != h.y || msg.X < h.x || msg.X >= h.x+h.width {
				continue
			}
			switch h.action {
			case "search":
				return m.setFocus(searchFocus)
			case "select":
				m.selectRow(h.index)
				return m.setFocus(listFocus)
			case "field":
				return m.setFocus(h.index)
			case "form":
				return m.focusForm(h.index)
			case "save":
				return m.saveForm()
			case "confirm":
				return m.action("run")
			case "cancel":
				m.confirm, m.form, m.status = false, nil, ""
				return nil
			default:
				return m.action(h.action)
			}
		}
	}
	if m.form == nil && !m.confirm && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
		delta, d := 1, m.dimensions()
		if msg.Button == tea.MouseButtonWheelUp {
			delta = -1
		}
		onList := msg.X >= 2 && msg.X < 2+d.left && msg.Y >= 4 && msg.Y < 6+d.listRows
		if onList {
			m.move(delta)
			return m.setFocus(listFocus)
		}
		if msg.Y >= 4 {
			m.scrollDetail(delta * 3)
		}
	}
	return nil
}
