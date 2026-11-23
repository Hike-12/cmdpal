package ui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Hike-12/cmdpal/internal/config"
)

var (
	canvas        = lipgloss.Color("#09090B")
	surface       = lipgloss.Color("#111114")
	border        = lipgloss.Color("#27272A")
	ink           = lipgloss.Color("#FAFAFA")
	muted         = lipgloss.Color("#71717A")
	strong        = lipgloss.NewStyle().Foreground(ink)
	subtle        = lipgloss.NewStyle().Foreground(muted)
	title         = strong.Bold(true)
	outline       = lipgloss.NewStyle().Foreground(border).Background(canvas)
	selectedStyle = strong.Background(border).Bold(true)
	button        = strong.Background(border).Bold(true).Padding(0, 1)
	primary       = lipgloss.NewStyle().Foreground(canvas).Background(ink).Bold(true).Padding(0, 1)
)

func safetyTag(level string) string {
	color := muted
	switch level {
	case config.Safe:
		color = "#10B981"
	case config.Caution:
		color = "#F59E0B"
	case config.Destructive:
		color = "#EF4444"
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render("[" + level + "]")
}

func clean(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
}

func fit(text string, width int, bg lipgloss.Color) string {
	text = ansi.Truncate(text, width, "…")
	return text + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", max(0, width-lipgloss.Width(text))))
}

func panelLine(line string, width int) string {
	pad := lipgloss.NewStyle().Background(surface).Render(" ")
	return outline.Render("│") + pad + fit(line, width-4, surface) + pad + outline.Render("│")
}

func panelEdge(width int, caption string, top bool) string {
	if !top {
		return outline.Render("╰" + strings.Repeat("─", width-2) + "╯")
	}
	if caption != "" {
		caption = " " + caption + " "
	}
	return outline.Render("╭─") + subtle.Render(caption) +
		outline.Render(strings.Repeat("─", max(0, width-3-lipgloss.Width(caption)))+"╮")
}
