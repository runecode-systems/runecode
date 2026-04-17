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

	surface := m.activeShellSurface()
	layout := m.planShellLayout(surface)
	viewportWidth, viewportHeight := normalizedShellViewport(m.width, m.height)
	workbench := m.renderShellWorkbench(surface, layout, viewportWidth, viewportHeight)
	root := appTheme.SurfaceBase.
		Width(viewportWidth).
		MaxWidth(viewportWidth).
		Height(viewportHeight).
		MaxHeight(viewportHeight)
	return root.Render(workbench)
}

func (m shellModel) renderShellWorkbench(surface routeSurface, layout shellLayoutPlan, viewportWidth int, viewportHeight int) string {
	overlayBody, _ := m.overlayBodyWithHeight(surface, layout, viewportHeight)
	b := strings.Builder{}
	m.writeShellFrame(&b, surface, layout)
	m.writeShellFooter(&b)
	frame := constrainShellBlock(strings.TrimRight(b.String(), "\n"), viewportWidth, m.availableShellHeight())
	if strings.TrimSpace(overlayBody) == "" {
		return lipgloss.JoinVertical(lipgloss.Left, frame)
	}
	return lipgloss.JoinVertical(lipgloss.Left, frame, overlayBody)
}

func (m shellModel) writeShellFrame(b *strings.Builder, surface routeSurface, layout shellLayoutPlan) {
	viewportWidth, _ := normalizedShellViewport(m.width, m.height)
	b.WriteString(constrainShellBlock(m.renderTopStatus(surface, layout), viewportWidth, shellTopStatusHeight))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(m.renderSyncHealth(), viewportWidth, shellSyncHealthHeight))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(m.renderBreadcrumbs(surface), viewportWidth, shellBreadcrumbHeight))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(m.renderHistory(), viewportWidth, shellHistoryHeight))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock("", viewportWidth, shellPaneSpacerHeight))
	b.WriteString("\n")
	b.WriteString(m.renderShellPanes(surface, layout))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(m.renderBottomStrip(surface), viewportWidth, layout.Regions.Bottom.Height))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(m.renderStatusSurface(surface), viewportWidth, layout.Regions.Status.Height))
	b.WriteString("\n")
}

func (m shellModel) writeShellFooter(b *strings.Builder) {
	viewportWidth, _ := normalizedShellViewport(m.width, m.height)
	b.WriteString(constrainShellBlock(renderHelp(m.keys, m.palette.IsOpen() || m.sessions.IsOpen()), viewportWidth, 1))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(muted(localBrokerBoundaryPosture()), viewportWidth, 1))
	b.WriteString("\n")
	b.WriteString(constrainShellBlock(muted("Trust boundary: typed broker contracts only; no CLI scraping or daemon-private path modeling."), viewportWidth, 1))
	b.WriteString("\n")
}

func (m shellModel) overlayBodyWithHeight(surface routeSurface, layout shellLayoutPlan, viewportHeight int) (string, int) {
	parts := []string{}
	if toast := strings.TrimSpace(m.toasts.Latest()); toast != "" {
		parts = append(parts, "Toast: "+sanitizeUIText(toast))
	}
	overlay := ""
	switch {
	case m.palette.IsOpen():
		overlay = compactLines(m.renderOverlayStack(), centeredOverlayBlock(overlayIDQuickJump, compactLines("Return focus: "+m.overlayReturn.Label(), m.renderPalette()), normalizedOverlayWidth(m.width)))
	case m.sessions.IsOpen():
		overlay = compactLines(m.renderOverlayStack(), centeredOverlayBlock(overlayIDSessions, compactLines("Return focus: "+m.overlayReturn.Label(), m.renderSessionQuickSwitcher()), normalizedOverlayWidth(m.width)))
	case m.narrowSidebarOn && m.breakpoint() == shellBreakpointNarrow:
		overlay = compactLines(m.renderOverlayStack(), centeredOverlayBlock(overlayIDSidebar, compactLines("Return focus: "+m.overlayReturn.Label(), m.renderSidebar()), normalizedOverlayWidth(m.width)))
	case m.narrowInspectOn && m.breakpoint() == shellBreakpointNarrow:
		inspector := strings.TrimSpace(surface.Regions.Inspector.Body)
		if inspector != "" && routeInspectorAvailable(surface) && m.inspectorOn {
			title := strings.TrimSpace(surface.Regions.Inspector.Title)
			if title == "" {
				title = "Inspector"
			}
			overlay = compactLines(m.renderOverlayStack(), centeredOverlayBlock(overlayIDInspector, compactLines("Return focus: "+m.overlayReturn.Label(), title, inspector), normalizedOverlayWidth(m.width)))
		} else if !layout.InspectorVisible && !routeInspectorAvailable(surface) {
			overlay = centeredOverlayBlock(overlayIDInspector, compactLines("Return focus: "+m.overlayReturn.Label(), "Inspector unavailable for current route."), normalizedOverlayWidth(m.width))
		}
	}
	if strings.TrimSpace(overlay) != "" {
		parts = append(parts, overlay)
	}
	if len(parts) == 0 {
		return "", 0
	}
	maxOverlayHeight := viewportHeight / 2
	if maxOverlayHeight < 1 {
		maxOverlayHeight = 1
	}
	content := constrainShellBlock(strings.Join(parts, "\n"), normalizedOverlayWidth(m.width), maxOverlayHeight)
	return content, lipgloss.Height(content)
}

func (m shellModel) activeOverlayHeight(viewportHeight int) int {
	_, height := m.overlayBodyWithHeight(m.activeShellSurfaceWithoutOverlayHeight(), m.planShellLayout(m.activeShellSurfaceWithoutOverlayHeight()), viewportHeight)
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
	selection := "off"
	mouseCapture := "on"
	if m.selectionMode {
		selection = "on"
		mouseCapture = "off"
	}
	return compactLines(
		appTheme.AppTitle.Render("Runecode TUI α shell")+" "+neutralBadge("THEME "+string(m.themePreset)),
		fmt.Sprintf("Top status | route=%s breakpoint=%s focus=%s sidebar=%t inspector=%t overlays=%d active_session=%s selection=%s mouse_capture=%s activity=%s %s", m.routeLabel(m.currentRouteID()), layout.Breakpoint, m.focus.Label(), layout.NavigationVisible, layout.InspectorVisible, len(m.overlays), defaultPlaceholder(m.activeSessionID, "none"), selection, mouseCapture, renderShellActivityState(m.watch.projection.Activity.State), m.renderRunningIndicator()),
		fmt.Sprintf("Route caps | inspector_supported=%t inspector_enabled=%t", surface.Capabilities.Inspector.Supported, surface.Capabilities.Inspector.Enabled),
		fmt.Sprintf("Layout(wide): sidebar=%.0f%% inspector=%.0f%% collapsed=(sidebar:%t inspector:%t)", clampPaneRatio(m.sidebarRatio)*100, clampPaneRatio(m.inspectorRatio)*100, m.sidebarFolded, m.inspectorFolded),
	)
}

func (m shellModel) renderSyncHealth() string {
	text := "Shell sync health: " + renderShellSyncState(m.watch.projection.Health.State)
	if strings.TrimSpace(m.watch.projection.Activity.Active.ID) != "" {
		text += " " + infoBadge(fmt.Sprintf("active_%s=%s", m.watch.projection.Activity.Active.Kind, m.watch.projection.Activity.Active.ID))
	}
	if strings.TrimSpace(m.watch.projection.Health.ErrorText) != "" {
		text += " " + muted("("+sanitizeUIText(m.watch.projection.Health.ErrorText)+")")
	}
	return text
}

func (m shellModel) renderBreadcrumbs(surface routeSurface) string {
	breadcrumbs := surface.Chrome.Breadcrumbs
	if len(breadcrumbs) == 0 {
		breadcrumbs = []string{"Home", m.routeLabel(m.currentRouteID())}
	}
	return "Breadcrumbs: " + strings.Join(breadcrumbs, " > ")
}

func (m shellModel) renderHistory() string {
	if len(m.history) == 0 {
		return muted("History: empty")
	}
	items := make([]string, 0, len(m.history))
	for _, loc := range m.history {
		entry := m.routeLabel(loc.Primary.RouteID)
		if id := strings.TrimSpace(loc.Primary.Object.ID); id != "" && strings.ToLower(strings.TrimSpace(loc.Primary.Object.Kind)) != "route" {
			entry += ":" + id
		}
		if loc.Inspector != nil {
			if inspectID := strings.TrimSpace(loc.Inspector.Object.ID); inspectID != "" {
				entry += " [inspect:" + inspectID + "]"
			}
		}
		items = append(items, entry)
	}
	return muted("History: " + strings.Join(items, " <- "))
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
	mainBody := strings.TrimSpace(surface.Regions.Main.Body)
	if modes := renderModeSwitchTabs(surface.Actions.ModeTabs, surface.Actions.ActiveTab); strings.TrimSpace(modes) != "" {
		mainBody = compactLines(mainBody, modes)
	}
	mainPane := renderShellPane(shellPaneSpec{Title: mainTitle, Body: mainBody, Width: routeRegionWidth(layout.Regions.Main, viewportWidth), Height: routeRegionHeight(layout.Regions.Main, viewportHeight), Focused: m.focus == focusContent, Border: shellPaneBorder{Top: true, Bottom: true, Left: true, Right: true}})
	row := mainPane
	paneFrameHeight := layout.Regions.Main.Height
	if layout.NavigationVisible {
		sidebarTitle := "Sidebar"
		if layout.Breakpoint == shellBreakpointWide {
			sidebarTitle += fmt.Sprintf(" (%.0f%%)", clampPaneRatio(m.sidebarRatio)*100)
		}
		sidebarPane := renderShellPane(shellPaneSpec{Title: sidebarTitle, Body: m.renderSidebar(), Width: routeRegionWidth(layout.Regions.Sidebar, viewportWidth/4), Height: routeRegionHeight(layout.Regions.Sidebar, viewportHeight), Focused: m.focus == focusNav, Border: shellPaneBorder{Top: true, Bottom: true, Left: true, Right: false}})
		row = joinPanesHorizontal(sidebarPane, row)
	}
	if layout.InspectorVisible {
		inspectorTitle := strings.TrimSpace(surface.Regions.Inspector.Title)
		if inspectorTitle == "" {
			inspectorTitle = "Inspector pane"
		}
		if layout.Breakpoint == shellBreakpointWide {
			inspectorTitle += fmt.Sprintf(" (%.0f%%)", clampPaneRatio(m.inspectorRatio)*100)
		}
		inspectorPane := renderShellPane(shellPaneSpec{Title: inspectorTitle, Body: strings.TrimSpace(surface.Regions.Inspector.Body), Width: routeRegionWidth(layout.Regions.Inspector, viewportWidth/3), Height: routeRegionHeight(layout.Regions.Inspector, viewportHeight), Focused: m.focus == focusInspector, Border: shellPaneBorder{Top: true, Bottom: true, Left: false, Right: true}})
		row = joinPanesHorizontal(row, inspectorPane)
	}
	row = lipgloss.NewStyle().Width(viewportWidth).MaxWidth(viewportWidth).Render(row)
	return constrainShellBlock(row, viewportWidth, paneFrameHeight)
}

func (m shellModel) renderSidebar() string {
	if len(m.routes) == 0 && len(m.sessionItems) == 0 {
		return "(no routes or sessions)"
	}
	entries := m.sidebarEntries()
	cursor := m.normalizedSidebarCursor(entries)
	lines := make([]string, 0, len(entries)+6)
	lines = append(lines, tableHeader("Navigation"))
	lines = m.appendSidebarRouteLines(lines, cursor)
	lines = m.appendSidebarSessionLines(lines, entries, cursor)
	return strings.Join(lines, "\n")
}

func (m shellModel) renderBottomStrip(surface routeSurface) string {
	bottom := strings.TrimSpace(surface.Regions.Bottom.Body)
	if bottom == "" {
		bottom = muted("Bottom strip: no route composer/status actions")
	}
	selectionHint := "Selection mode off (ctrl+t toggles; mouse capture on)."
	if m.selectionMode {
		selectionHint = "Selection mode ON (ctrl+t to exit); mouse capture disabled so drag-to-select works."
	}
	return compactLines(
		tableHeader("Bottom strip"),
		bottom,
		m.renderRouteActionHints(surface),
		m.renderRouteCopyActions(),
		selectionHint,
	)
}

func (m shellModel) renderRouteActionHints(surface routeSurface) string {
	parts := []string{}
	if len(surface.Actions.ReferenceActions) > 0 {
		parts = append(parts, fmt.Sprintf("Linked refs actionable=%d", len(surface.Actions.ReferenceActions)))
	}
	if len(surface.Actions.LocalActions) > 0 {
		parts = append(parts, fmt.Sprintf("Local actions executable=%d", len(surface.Actions.LocalActions)))
	}
	if len(parts) == 0 {
		return muted("Actionable refs/actions: none")
	}
	return "Actionable refs/actions: " + strings.Join(parts, " | ")
}

func (m shellModel) renderStatusSurface(surface routeSurface) string {
	status := strings.TrimSpace(surface.Regions.Status.Body)
	if status == "" {
		status = fmt.Sprintf("route=%s", m.routeLabel(m.currentRouteID()))
	}
	selection := "selection=off"
	if m.selectionMode {
		selection = "selection=on"
	}
	return "Status: " + status + " | " + selection + " | clipboard=" + sanitizeUIText(m.clipboard.IntegrationHint())
}

func (m shellModel) renderRouteCopyActions() string {
	actions := m.activeShellSurface().Actions.CopyActions
	if len(actions) == 0 {
		return muted("Copy actions: none (use terminal selection for long-form text).")
	}
	items := make([]string, 0, len(actions))
	for i, action := range actions {
		label := strings.TrimSpace(action.Label)
		if label == "" {
			label = strings.TrimSpace(action.ID)
		}
		if label == "" {
			label = fmt.Sprintf("copy-%d", i+1)
		}
		if i == m.copyActionIndex {
			label = "[next:" + label + "]"
		}
		items = append(items, label)
	}
	return "Copy actions (Y cycles/copies): " + strings.Join(items, " | ")
}
