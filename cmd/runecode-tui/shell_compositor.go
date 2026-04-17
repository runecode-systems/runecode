package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type shellPaneSpec struct {
	Title   string
	Body    string
	Width   int
	Height  int
	Focused bool
}

func renderShellPane(spec shellPaneSpec) string {
	width := nonNegativeDimension(spec.Width)
	height := nonNegativeDimension(spec.Height)
	if width < 6 {
		width = 6
	}
	if height < 4 {
		height = 4
	}

	title := strings.TrimSpace(spec.Title)
	if title == "" {
		title = "Pane"
	}
	header := tableHeader(title)
	if spec.Focused {
		header = appTheme.FocusLine.Render(title) + " " + infoBadge("FOCUS")
	}
	body := strings.TrimSpace(spec.Body)
	if body == "" {
		body = "(empty pane)"
	}

	innerWidth := width - 2
	innerHeight := height - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		MaxWidth(innerWidth).
		MaxHeight(innerHeight).
		Render(compactLines(header, body))

	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("242")).
		Width(width).
		Height(height)
	if spec.Focused {
		border = border.BorderForeground(lipgloss.Color("183"))
	}
	return border.Render(content)
}

func joinPanesHorizontal(panes ...string) string {
	nonEmpty := make([]string, 0, len(panes))
	for _, pane := range panes {
		if strings.TrimSpace(pane) == "" {
			continue
		}
		nonEmpty = append(nonEmpty, pane)
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, nonEmpty...)
}

func joinPanesVertical(parts ...string) string {
	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		nonEmpty = append(nonEmpty, part)
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, nonEmpty...)
}
