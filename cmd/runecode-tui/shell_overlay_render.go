package main

import (
	"fmt"
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
	frameInnerWidth := outerWidth - 2
	innerWidth := frameInnerWidth - 4
	if innerWidth < 1 {
		innerWidth = 1
	}
	contentWidth := innerWidth
	bodyHeight := lipgloss.Height(strings.TrimRight(body, "\n"))
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	if maxHeight > 0 {
		maxBodyHeight := maxHeight - 3 // 1 modal header + 2 frame border rows.
		if maxBodyHeight < 1 {
			maxBodyHeight = 1
		}
		bodyHeight = maxBodyHeight
	}
	body = constrainShellBlock(body, contentWidth, bodyHeight)
	titleLine := tableHeader(overlayDisplayTitle(title))
	dismissLine := muted(overlayDismissHint(title))

	content := appTheme.SurfaceOverlay.
		Width(contentWidth).
		MaxWidth(contentWidth).
		Padding(0, 2).
		Render(compactLines(titleLine, dismissLine, body))

	frame := appTheme.SurfaceOverlay.
		Border(lipgloss.NormalBorder()).
		BorderForeground(appTheme.BorderStrong.GetForeground()).
		Padding(0, 0).
		Render(content)

	rendered := lipgloss.NewStyle().Width(viewportWidth).Align(lipgloss.Center).Render(frame)
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
	startX := (viewportWidth - outerWidth) / 2
	if startX < 0 {
		startX = 0
	}
	contentStartX := startX + 2 // border + left padding
	contentEndX := startX + outerWidth - 5
	if contentEndX < contentStartX {
		contentEndX = contentStartX
	}
	return contentStartX, contentEndX
}

func overlayDisplayTitle(id shellOverlayID) string {
	switch id {
	case overlayIDQuickJump:
		return "Command Palette"
	case overlayIDSessions:
		return "Session Switcher"
	case overlayIDSidebar:
		return "Navigation"
	case overlayIDInspector:
		return "Inspector"
	case overlayIDLeader:
		return "Leader Help"
	case overlayIDQuitConfirm:
		return "Quit RuneCode"
	default:
		return strings.TrimSpace(string(id))
	}
}

func overlayDismissHint(id shellOverlayID) string {
	if id == overlayIDQuitConfirm {
		return "esc keep editing"
	}
	return fmt.Sprintf("esc close")
}
