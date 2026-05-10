package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m shellModel) cachedOverlayWorkbench(viewportWidth int, viewportHeight int) (string, bool) {
	if !m.overlayFrameCacheable() || m.overlayFrameCache == nil {
		return "", false
	}
	cached := m.overlayFrameCache
	overlayID := m.activeOverlayID()
	if cached.overlay != overlayID || cached.width != viewportWidth || cached.height != viewportHeight || cached.frame == "" {
		return "", false
	}
	overlayBody, ok := m.cacheableOverlayBodyWithHeight(viewportHeight)
	if !ok {
		return "", false
	}
	return applyModalOverlay(scrubShellFrameForOverlay(cached.frame, viewportWidth, viewportHeight), overlayBody, viewportWidth, viewportHeight), true
}

func (m shellModel) cacheableOverlayBodyWithHeight(viewportHeight int) (string, bool) {
	parts := m.overlayStaticParts()
	maxOverlayHeight := overlayHeightBudget(viewportHeight)
	remainingHeight := overlayRemainingHeight(parts, maxOverlayHeight)
	overlayWidth := normalizedOverlayWidth(m.width)
	overlay, ok := m.cacheableOverlayBody(overlayWidth, remainingHeight)
	if !ok || strings.TrimSpace(overlay) == "" {
		return "", false
	}
	parts = append(parts, overlay)
	contentHeight := lipgloss.Height(strings.Join(parts, "\n"))
	if contentHeight > maxOverlayHeight {
		contentHeight = maxOverlayHeight
	}
	content := constrainShellBlock(strings.Join(parts, "\n"), overlayWidth, contentHeight)
	return content, true
}

func (m shellModel) cacheableOverlayBody(overlayWidth int, remainingHeight int) (string, bool) {
	switch m.activeOverlayID() {
	case overlayIDQuickJump:
		return centeredOverlayBlockBounded(overlayIDQuickJump, m.renderPalette(), overlayWidth, remainingHeight), true
	case overlayIDSessions:
		return centeredOverlayBlockBounded(overlayIDSessions, m.renderSessionQuickSwitcher(), overlayWidth, remainingHeight), true
	case overlayIDLeader:
		return centeredOverlayBlockBounded(overlayIDLeader, m.renderLeaderWhichKey(), overlayWidth, remainingHeight), true
	case overlayIDQuitConfirm:
		return centeredOverlayBlockBounded(overlayIDQuitConfirm, m.renderQuitConfirmDialog(), overlayWidth, remainingHeight), true
	default:
		return "", false
	}
}

func (m shellModel) overlayBodyWithHeight(surface routeSurface, layout shellLayoutPlan, viewportHeight int) (string, int) {
	parts := m.overlayStaticParts()
	maxOverlayHeight := overlayHeightBudget(viewportHeight)
	remainingHeight := overlayRemainingHeight(parts, maxOverlayHeight)
	overlayWidth := normalizedOverlayWidth(m.width)
	overlay := m.activeOverlayBody(surface, layout, overlayWidth, remainingHeight)
	if strings.TrimSpace(overlay) != "" {
		parts = append(parts, overlay)
	}
	if len(parts) == 0 {
		return "", 0
	}
	contentHeight := lipgloss.Height(strings.Join(parts, "\n"))
	if contentHeight > maxOverlayHeight {
		contentHeight = maxOverlayHeight
	}
	content := constrainShellBlock(strings.Join(parts, "\n"), overlayWidth, contentHeight)
	return content, lipgloss.Height(content)
}

func (m shellModel) overlayStaticParts() []string {
	return nil
}

func overlayHeightBudget(viewportHeight int) int {
	maxOverlayHeight := viewportHeight - 8
	if maxOverlayHeight < 1 {
		return 1
	}
	return maxOverlayHeight
}

func overlayRemainingHeight(parts []string, maxOverlayHeight int) int {
	remainingHeight := maxOverlayHeight - lipgloss.Height(strings.Join(parts, "\n"))
	if len(parts) > 0 {
		remainingHeight--
	}
	if remainingHeight < 1 {
		return 1
	}
	return remainingHeight
}

func (m shellModel) activeOverlayBody(surface routeSurface, layout shellLayoutPlan, overlayWidth int, remainingHeight int) string {
	switch {
	case m.palette.IsOpen():
		return centeredOverlayBlockBounded(overlayIDQuickJump, m.renderPalette(), overlayWidth, remainingHeight)
	case m.sessions.IsOpen():
		return centeredOverlayBlockBounded(overlayIDSessions, m.renderSessionQuickSwitcher(), overlayWidth, remainingHeight)
	case m.leader.Active():
		return centeredOverlayBlockBounded(overlayIDLeader, m.renderLeaderWhichKey(), overlayWidth, remainingHeight)
	case m.quitConfirm.active:
		return centeredOverlayBlockBounded(overlayIDQuitConfirm, m.renderQuitConfirmDialog(), overlayWidth, remainingHeight)
	case m.narrowSidebarOn && m.breakpoint() == shellBreakpointNarrow:
		return centeredOverlayBlockBounded(overlayIDSidebar, m.renderSidebar(), overlayWidth, remainingHeight)
	case m.narrowInspectOn && m.breakpoint() == shellBreakpointNarrow:
		return m.narrowInspectorOverlayBody(surface, layout, overlayWidth, remainingHeight)
	default:
		return ""
	}
}

func (m shellModel) narrowInspectorOverlayBody(surface routeSurface, layout shellLayoutPlan, overlayWidth int, remainingHeight int) string {
	inspector := strings.TrimSpace(surface.Regions.Inspector.Body)
	title := strings.TrimSpace(surface.Regions.Inspector.Title)
	if title == "" {
		title = "Inspector"
	}
	if inspector != "" && routeInspectorAvailable(surface) && m.inspectorOn {
		return centeredOverlayBlockBounded(overlayIDInspector, compactLines(title, inspector), overlayWidth, remainingHeight)
	}
	if !layout.InspectorVisible && !routeInspectorAvailable(surface) {
		return centeredOverlayBlockBounded(overlayIDInspector, "Inspector unavailable for current route.", overlayWidth, remainingHeight)
	}
	return centeredOverlayBlockBounded(overlayIDInspector, compactLines(title, "No inspector item selected."), overlayWidth, remainingHeight)
}

func (m shellModel) activeOverlayHeight(viewportHeight int) int {
	surface := m.activeShellSurfaceWithoutOverlayHeight()
	layout := m.planShellLayout(surface)
	_, height := m.overlayBodyWithHeight(surface, layout, viewportHeight)
	return height
}

func (m shellModel) activeShellSurfaceWithoutOverlayHeight() routeSurface {
	active := m.routeModels[m.currentRouteID()]
	if active == nil {
		return routeSurface{}
	}
	baseCtx := routeShellContext{Width: m.width, Height: m.height, Focus: m.focus, Focused: m.focusedRouteRegion(), Breakpoint: m.breakpoint(), Render: routeShellRenderPreferences{PreferredPresentation: normalizePresentationMode(m.preferredMode), ThemePreset: normalizeThemePreset(m.themePreset)}}
	surface := active.ShellSurface(baseCtx)
	layout := m.planShellLayout(surface)
	ctx := baseCtx
	ctx.Regions = layout.Regions
	ctx.Breakpoint = layout.Breakpoint
	return m.withLocationChrome(active.ShellSurface(ctx))
}

func normalizedOverlayWidth(width int) int {
	if width <= 0 {
		return 1
	}
	return width
}

func scrubShellFrameForOverlay(frame string, width int, height int) string {
	if strings.TrimSpace(frame) == "" {
		return frame
	}
	lines := strings.Split(padShellFrame(frame, width, height), "\n")
	blank := lipgloss.NewStyle().Width(width).MaxWidth(width).Render("")
	blankFrom := len(lines) - (shellStatusHeight + shellBottomStripHeight)
	if blankFrom < 0 {
		blankFrom = 0
	}
	for i := range lines {
		if i >= blankFrom {
			lines[i] = blank
			continue
		}
		stripped := strings.TrimSpace(ansiStripForOverlay(lines[i]))
		if stripped == "" {
			lines[i] = blank
			continue
		}
		if containsBoxDrawing(stripped) {
			lines[i] = blank
			continue
		}
		lines[i] = appTheme.Muted.Width(width).MaxWidth(width).Render(ansiStripForOverlay(lines[i]))
	}
	return strings.Join(lines, "\n")
}

func containsBoxDrawing(line string) bool {
	for _, r := range line {
		switch r {
		case '┌', '┐', '└', '┘', '│', '─', '├', '┤', '┬', '┴', '┼':
			return true
		}
	}
	return false
}

func ansiStripForOverlay(line string) string {
	return sanitizeUIText(ansi.Strip(line))
}

func applyModalOverlay(frame string, overlay string, width int, height int) string {
	if strings.TrimSpace(overlay) == "" {
		return frame
	}
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		return frame
	}
	frameLines := strings.Split(padShellBlock(frame, width, height), "\n")
	overlayLines := strings.Split(strings.TrimRight(overlay, "\n"), "\n")
	if len(overlayLines) > height {
		overlayLines = overlayLines[:height]
	}
	start := (height - len(overlayLines)) / 2
	if start < 0 {
		start = 0
	}
	for i, line := range overlayLines {
		row := start + i
		if row < 0 || row >= len(frameLines) {
			continue
		}
		frameLines[row] = lipgloss.NewStyle().Width(width).MaxWidth(width).Align(lipgloss.Center).Render(line)
	}
	return strings.Join(frameLines, "\n")
}
