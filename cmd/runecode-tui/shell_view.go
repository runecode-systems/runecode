package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m shellModel) View() string {
	if m.quitting {
		return "Goodbye from runecode-tui.\n"
	}

	viewportWidth, viewportHeight := normalizedShellViewport(m.width, m.height)
	workbench, ok := m.cachedOverlayWorkbench(viewportWidth, viewportHeight)
	if !ok {
		surface, layout := m.activeShellSurfacePlan()
		workbench = m.renderShellWorkbench(surface, layout, viewportWidth, viewportHeight)
	}
	root := appTheme.SurfaceBase.
		Width(viewportWidth).
		MaxWidth(viewportWidth).
		Height(viewportHeight).
		MaxHeight(viewportHeight)
	return root.Render(workbench)
}

func (m shellModel) renderShellWorkbench(surface routeSurface, layout shellLayoutPlan, viewportWidth int, viewportHeight int) string {
	overlayBody, _ := m.overlayBodyWithHeight(surface, layout, viewportHeight)
	frame := m.renderShellFrame(surface, layout, viewportWidth, viewportHeight)
	if strings.TrimSpace(overlayBody) == "" {
		if m.overlayFrameCache != nil {
			*m.overlayFrameCache = shellOverlayFrameCache{}
		}
		return lipgloss.JoinVertical(lipgloss.Left, frame)
	}
	return applyModalOverlay(scrubShellFrameForOverlay(frame, viewportWidth, viewportHeight), overlayBody, viewportWidth, viewportHeight)
}

func (m shellModel) renderShellFrame(surface routeSurface, layout shellLayoutPlan, viewportWidth int, viewportHeight int) string {
	if m.overlayFrameCacheable() && m.overlayFrameCache != nil {
		if cached := m.overlayFrameCache; cached.overlay == m.activeOverlayID() && cached.width == viewportWidth && cached.height == viewportHeight && cached.frame != "" {
			return cached.frame
		}
	}
	b := strings.Builder{}
	m.writeShellFrame(&b, surface, layout, viewportWidth)
	m.writeShellFooter(&b, viewportWidth)
	frame := padShellFrame(strings.TrimRight(b.String(), "\n"), viewportWidth, viewportHeight)
	if m.overlayFrameCacheable() && m.overlayFrameCache != nil {
		*m.overlayFrameCache = shellOverlayFrameCache{overlay: m.activeOverlayID(), width: viewportWidth, height: viewportHeight, frame: frame}
	}
	return frame
}

func (m shellModel) writeShellFrame(b *strings.Builder, surface routeSurface, layout shellLayoutPlan, viewportWidth int) {
	b.WriteString(constrainShellBlock(m.renderTopStatus(surface, layout), viewportWidth, shellTopStatusHeight))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(m.renderSyncHealth(), viewportWidth, shellSyncHealthHeight))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock("", viewportWidth, shellPaneSpacerHeight))
	b.WriteString("\n")
	b.WriteString(m.renderShellPanes(surface, layout))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(m.renderStatusSurface(surface), viewportWidth, layout.Regions.Status.Height))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(m.renderBottomStrip(surface), viewportWidth, layout.Regions.Bottom.Height))
}

func (m shellModel) writeShellFooter(b *strings.Builder, viewportWidth int) {
	if shellFooterHeight <= 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(renderHelp(m.keys, m.palette.IsOpen() || m.sessions.IsOpen(), m.actions), viewportWidth, shellFooterHeight))
}

func constrainShellBlock(block string, width int, height int) string {
	if height <= 0 {
		return ""
	}
	if width <= 0 {
		width = 1
	}
	trimmed := strings.TrimRight(block, "\n")
	if trimmed == "" {
		return strings.TrimRight(lipgloss.NewStyle().Width(width).Height(height).Render(""), "\n")
	}
	rawLines := strings.Split(trimmed, "\n")
	lines := make([]string, 0, height)
	lineStyle := lipgloss.NewStyle().Width(width).MaxWidth(width).Height(1).MaxHeight(1)
	for _, line := range rawLines {
		if len(lines) >= height {
			break
		}
		lines = append(lines, lineStyle.Render(strings.TrimRight(line, "\r")))
	}
	for len(lines) < height {
		lines = append(lines, lineStyle.Render(""))
	}
	return strings.Join(lines, "\n")
}

func (m shellModel) renderTopStatus(surface routeSurface, layout shellLayoutPlan) string {
	routeSummary := fmt.Sprintf("%s  %s", appTheme.AppTitle.Render("RuneCode Workbench"), neutralBadge("ROUTE "+strings.ToUpper(sanitizeUIText(m.routeLabel(m.currentRouteID())))))
	workbenchSummary := []string{humanFocusLabel(m.focus) + " focus"}
	if activity := strings.TrimSpace(humanShellActivitySummary(m.watch.projection.Activity)); activity != "" {
		workbenchSummary = append(workbenchSummary, activity)
	} else if strings.TrimSpace(m.activeSessionID) != "" {
		workbenchSummary = append(workbenchSummary, fmt.Sprintf("Session %s", sanitizeUIText(m.activeSessionID)))
	}
	if m.selectionMode {
		workbenchSummary = append(workbenchSummary, "Selection mode on")
	}
	if routeInspectorAvailable(surface) && layout.InspectorVisible {
		workbenchSummary = append(workbenchSummary, "Inspector ready")
	}
	return compactLines(
		appTheme.SurfaceChrome.Padding(0, 1).Render(routeSummary),
		appTheme.SurfaceChrome.Padding(0, 1).Render(strings.Join(workbenchSummary, "  •  ")),
	)
}

func renderRunningSuffix(indicator string) string {
	indicator = strings.TrimSpace(indicator)
	if indicator == "" {
		return ""
	}
	return "  •  " + indicator
}

func (m shellModel) renderSyncHealth() string {
	prefix := "Product truth: "
	if m.width > 0 && m.width < shellMediumMinWidth {
		prefix = "Truth: "
	}
	text := prefix + renderShellSyncState(m.watch.projection.Health.State)
	if strings.TrimSpace(m.watch.projection.Health.ErrorText) != "" {
		text += "  •  " + muted("Status has detail: "+sanitizeUIText(m.watch.projection.Health.ErrorText))
	}
	if actions := strings.TrimSpace(m.renderPrimaryWorkbenchActions()); actions != "" {
		text += "  •  " + actions
	}
	return text
}

func (m shellModel) renderBreadcrumbs(surface routeSurface) string {
	breadcrumbs := surface.Chrome.Breadcrumbs
	if len(breadcrumbs) == 0 {
		breadcrumbs = []string{"Home", m.routeLabel(m.currentRouteID())}
	}
	safe := make([]string, 0, len(breadcrumbs))
	for _, breadcrumb := range breadcrumbs {
		label := sanitizeUIText(breadcrumb)
		if label == "" {
			continue
		}
		safe = append(safe, label)
	}
	if len(safe) == 0 {
		safe = []string{"Home", sanitizeUIText(m.routeLabel(m.currentRouteID()))}
	}
	return muted("Path: " + strings.Join(safe, " > "))
}

func (m shellModel) renderHistory() string {
	if len(m.history) == 0 {
		return muted("History: empty")
	}
	items := make([]string, 0, len(m.history))
	for _, loc := range m.history {
		entry := sanitizeUIText(m.routeLabel(loc.Primary.RouteID))
		if id := strings.TrimSpace(loc.Primary.Object.ID); id != "" && strings.ToLower(strings.TrimSpace(loc.Primary.Object.Kind)) != "route" {
			entry += ":" + sanitizeUIText(id)
		}
		if loc.Inspector != nil {
			if inspectID := strings.TrimSpace(loc.Inspector.Object.ID); inspectID != "" {
				entry += " [inspect:" + sanitizeUIText(inspectID) + "]"
			}
		}
		items = append(items, entry)
	}
	if len(items) > 5 {
		items = items[len(items)-5:]
	}
	return muted("History: " + strings.Join(items, " <- "))
}

func (m shellModel) renderActiveWorkSummary() string {
	active := m.watch.projection.Activity.Active
	if strings.TrimSpace(active.ID) == "" {
		return ""
	}
	label := sanitizeUIText(active.Kind)
	if label == "" {
		label = "work"
	}
	return fmt.Sprintf("Active %s %s", label, sanitizeUIText(active.ID))
}

func (m shellModel) renderChromeDiagnosticHint(surface routeSurface, layout shellLayoutPlan) string {
	parts := []string{}
	if routeInspectorAvailable(surface) {
		parts = append(parts, fmt.Sprintf("inspector %s", boolOnOff(layout.InspectorVisible)))
	}
	if m.selectionMode {
		parts = append(parts, "selection text")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " • ")
}

func boolOnOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func (m shellModel) renderShellPanes(surface routeSurface, layout shellLayoutPlan) string {
	viewportWidth, viewportHeight := normalizedShellViewport(m.width, m.height)
	mainTitle := "Main pane"
	if title := strings.TrimSpace(surface.Regions.Main.Title); title != "" {
		mainTitle += " — " + title
	}
	if activity := strings.TrimSpace(m.renderPaneActivityMarker()); activity != "" {
		mainTitle += " " + activity
	}
	mainBody := strings.Trim(surface.Regions.Main.Body, "\n")
	if modes := renderModeSwitchTabs(surface.Actions.ModeTabs, surface.Actions.ActiveTab); strings.TrimSpace(modes) != "" {
		mainBody = compactLines(mainBody, modes)
	}
	mainPane := renderShellPane(shellPaneSpec{Title: mainTitle, Body: mainBody, Width: routeRegionWidth(layout.Regions.Main, viewportWidth), Height: routeRegionHeight(layout.Regions.Main, viewportHeight), Focused: m.focus == focusContent, Border: shellPaneBorder{Top: true, Bottom: true, Left: true, Right: true}})
	row := mainPane
	paneFrameHeight := layout.Regions.Main.Height
	if layout.NavigationVisible && layout.Regions.Sidebar.Width > 0 {
		sidebarTitle := "Sidebar"
		if layout.Breakpoint == shellBreakpointWide {
			sidebarTitle += fmt.Sprintf(" (%.0f%%)", clampPaneRatio(m.sidebarRatio)*100)
		}
		sidebarPane := renderShellPane(shellPaneSpec{Title: sidebarTitle, Body: m.renderSidebar(), Width: routeRegionWidth(layout.Regions.Sidebar, 0), Height: routeRegionHeight(layout.Regions.Sidebar, viewportHeight), Focused: m.focus == focusNav, Border: shellPaneBorder{Top: true, Bottom: true, Left: true, Right: false}})
		row = joinPanesHorizontal(sidebarPane, row)
	}
	if layout.InspectorVisible && layout.Regions.Inspector.Width > 0 {
		inspectorTitle := strings.TrimSpace(surface.Regions.Inspector.Title)
		if inspectorTitle == "" {
			inspectorTitle = "Inspector pane"
		}
		if layout.Breakpoint == shellBreakpointWide {
			inspectorTitle += fmt.Sprintf(" (%.0f%%)", clampPaneRatio(m.inspectorRatio)*100)
		}
		inspectorPane := renderShellPane(shellPaneSpec{Title: inspectorTitle, Body: strings.Trim(surface.Regions.Inspector.Body, "\n"), Width: routeRegionWidth(layout.Regions.Inspector, 0), Height: routeRegionHeight(layout.Regions.Inspector, viewportHeight), Focused: m.focus == focusInspector, Border: shellPaneBorder{Top: true, Bottom: true, Left: false, Right: true}})
		row = joinPanesHorizontal(row, inspectorPane)
	}
	row = lipgloss.NewStyle().Width(viewportWidth).MaxWidth(viewportWidth).Render(row)
	return padShellBlock(row, viewportWidth, paneFrameHeight)
}

func padShellBlock(block string, width int, height int) string {
	if height <= 0 {
		return ""
	}
	if width <= 0 {
		width = 1
	}
	trimmed := strings.TrimRight(block, "\n")
	if trimmed == "" {
		return strings.TrimRight(appTheme.SurfaceBase.Width(width).Height(height).Render(""), "\n")
	}
	lines := strings.Split(trimmed, "\n")
	currentHeight := lipgloss.Height(trimmed)
	if currentHeight >= height {
		return strings.Join(lines[:height], "\n")
	}
	padLines := make([]string, 0, height-currentHeight)
	blank := appTheme.SurfaceBase.Width(width).Render("")
	for len(padLines) < height-currentHeight {
		padLines = append(padLines, blank)
	}
	return lipgloss.JoinVertical(lipgloss.Left, trimmed, strings.Join(padLines, "\n"))
}

func (m shellModel) renderSidebar() string {
	if len(m.routes) == 0 && len(m.sessionItems) == 0 {
		return "(no routes or sessions)"
	}
	entries := m.sidebarEntries()
	cursor := m.normalizedSidebarCursor(entries)
	var surface routeSurface
	if m.breakpoint() == shellBreakpointNarrow && m.narrowSidebarOn {
		surface = m.activeShellSurfaceWithoutOverlayHeight()
	} else {
		surface = m.activeShellSurface()
	}
	width := m.planShellLayout(surface).Regions.Sidebar.Width - 2
	if width < 12 {
		width = 12
	}
	lines := make([]string, 0, len(entries)+6)
	lines = append(lines, tableHeader("Navigation"))
	lines = m.appendSidebarRouteLines(lines, cursor, width)
	lines = m.appendSidebarSessionLines(lines, entries, cursor, width)
	if m.sidebarActionEntryCount() > 0 {
		lines = append(lines, "", tableHeader("Actions"))
		lines = m.appendSidebarActionLines(lines, entries, cursor, width)
	}
	return strings.Join(lines, "\n")
}

func padShellFrame(block string, width int, height int) string {
	if height <= 0 {
		return ""
	}
	if width <= 0 {
		width = 1
	}
	trimmed := strings.TrimRight(block, "\n")
	if trimmed == "" {
		return strings.TrimRight(lipgloss.NewStyle().Width(width).Height(height).Render(""), "\n")
	}
	rawLines := strings.Split(trimmed, "\n")
	if len(rawLines) >= height {
		return strings.Join(rawLines[:height], "\n")
	}
	padCount := height - len(rawLines)
	lineStyle := lipgloss.NewStyle().Width(width).MaxWidth(width).Height(1).MaxHeight(1)
	padded := make([]string, 0, height)
	for i := 0; i < padCount; i++ {
		padded = append(padded, lineStyle.Render(""))
	}
	padded = append(padded, rawLines...)
	return strings.Join(padded, "\n")
}
