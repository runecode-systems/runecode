package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func centeredOverlayBlock(title shellOverlayID, body string, viewportWidth int) string {
	return centeredOverlayBlockBounded(title, body, viewportWidth, 0)
}

func centeredOverlayBlockBounded(title shellOverlayID, body string, viewportWidth int, maxHeight int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		body = "(empty overlay)"
	}
	outerWidth := overlayBlockWidth(viewportWidth)
	innerWidth := outerWidth - 2
	if innerWidth < 1 {
		innerWidth = 1
	}
	contentWidth := innerWidth - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	bodyHeight := lipgloss.Height(strings.TrimRight(body, "\n"))
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	if maxHeight > 0 {
		maxBodyHeight := maxHeight - 3 // 1 overlay header + 2 frame border rows.
		if maxBodyHeight < 1 {
			maxBodyHeight = 1
		}
		bodyHeight = maxBodyHeight
	}
	body = constrainShellBlock(body, contentWidth, bodyHeight)

	content := appTheme.SurfaceOverlay.
		Width(contentWidth).
		MaxWidth(contentWidth).
		Padding(0, 1).
		Render(body)

	frame := appTheme.SurfaceOverlay.
		Border(lipgloss.NormalBorder()).
		BorderForeground(appTheme.BorderStrong.GetForeground()).
		Width(innerWidth).
		MaxWidth(innerWidth).
		Render(content)

	rendered := compactLines(
		tableHeader("Overlay")+" "+neutralBadge(strings.ToUpper(string(title))),
		lipgloss.NewStyle().Width(viewportWidth).Align(lipgloss.Left).Render(frame),
	)
	if maxHeight <= 0 {
		return rendered
	}
	return constrainShellBlock(rendered, viewportWidth, maxHeight)
}

func overlayBlockWidth(viewportWidth int) int {
	width := viewportWidth - 8
	if width < 48 {
		width = 48
	}
	if width > viewportWidth {
		width = viewportWidth
	}
	if width < 1 {
		width = 1
	}
	return width
}

func centeredOverlayContentBounds(viewportWidth int) (int, int) {
	if viewportWidth <= 0 {
		viewportWidth = 120
	}
	outerWidth := overlayBlockWidth(viewportWidth)
	startX := 0
	contentStartX := startX + 2 // border + left padding
	contentEndX := startX + outerWidth - 3
	if contentEndX < contentStartX {
		contentEndX = contentStartX
	}
	return contentStartX, contentEndX
}
