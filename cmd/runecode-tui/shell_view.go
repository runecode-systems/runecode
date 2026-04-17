package main

import (
	"fmt"
	"strings"
)

func (m shellModel) View() string {
	if m.quitting {
		return "Goodbye from runecode-tui.\n"
	}

	surface := m.activeShellSurface()
	b := strings.Builder{}
	m.writeShellFrame(&b, surface)
	m.writeShellToast(&b)
	m.writeShellOverlays(&b, surface)
	m.writeShellFooter(&b)
	return b.String()
}

func (m shellModel) writeShellFrame(b *strings.Builder, surface routeSurface) {
	b.WriteString(m.renderTopStatus(surface))
	b.WriteString("\n")
	b.WriteString(m.renderSyncHealth())
	b.WriteString("\n")
	b.WriteString(m.renderBreadcrumbs(surface))
	b.WriteString("\n")
	b.WriteString(m.renderBackstack())
	b.WriteString("\n\n")
	b.WriteString(m.renderShellPanes(surface))
	b.WriteString("\n")
	b.WriteString(m.renderBottomStrip(surface))
	b.WriteString("\n")
	b.WriteString(m.renderStatusSurface(surface))
	b.WriteString("\n")
}

func (m shellModel) writeShellToast(b *strings.Builder) {
	if toast := strings.TrimSpace(m.toasts.Latest()); toast != "" {
		b.WriteString("Toast: ")
		b.WriteString(toast)
		b.WriteString("\n")
	}
}

func (m shellModel) writeShellOverlays(b *strings.Builder, surface routeSurface) {
	m.writePaletteOverlay(b)
	m.writeSessionOverlay(b)
	m.writeNarrowSidebarOverlay(b)
	m.writeNarrowInspectorOverlay(b, surface)
}

func (m shellModel) writePaletteOverlay(b *strings.Builder) {
	if !m.palette.IsOpen() {
		return
	}
	b.WriteString(m.renderOverlayStack())
	b.WriteString("\n")
	b.WriteString(centeredOverlayBlock(overlayIDQuickJump, m.renderPalette()))
	b.WriteString("\n")
}

func (m shellModel) writeSessionOverlay(b *strings.Builder) {
	if !m.sessions.IsOpen() {
		return
	}
	b.WriteString(m.renderOverlayStack())
	b.WriteString("\n")
	b.WriteString(centeredOverlayBlock(overlayIDSessions, m.renderSessionQuickSwitcher()))
	b.WriteString("\n")
}

func (m shellModel) writeNarrowSidebarOverlay(b *strings.Builder) {
	if !m.narrowSidebarOn || m.breakpoint() != shellBreakpointNarrow {
		return
	}
	b.WriteString(m.renderOverlayStack())
	b.WriteString("\n")
	b.WriteString(centeredOverlayBlock(overlayIDSidebar, m.renderSidebar()))
	b.WriteString("\n")
}

func (m shellModel) writeNarrowInspectorOverlay(b *strings.Builder, surface routeSurface) {
	if !m.narrowInspectOn || m.breakpoint() != shellBreakpointNarrow {
		return
	}
	inspector := strings.TrimSpace(surface.Inspector)
	if inspector != "" && m.inspectorOn {
		b.WriteString(m.renderOverlayStack())
		b.WriteString("\n")
		b.WriteString(centeredOverlayBlock(overlayIDInspector, inspector))
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func (m shellModel) writeShellFooter(b *strings.Builder) {
	b.WriteString(renderHelp(m.keys, m.palette.IsOpen() || m.sessions.IsOpen()))
	b.WriteString("\n")
	b.WriteString(muted(localBrokerBoundaryPosture()))
	b.WriteString("\n")
	b.WriteString(muted("Trust boundary: typed broker contracts only; no CLI scraping or daemon-private path modeling."))
	b.WriteString("\n")
}

func (m shellModel) renderTopStatus(surface routeSurface) string {
	selection := "off"
	mouseCapture := "on"
	if m.selectionMode {
		selection = "on"
		mouseCapture = "off"
	}
	return compactLines(
		appTheme.AppTitle.Render("Runecode TUI α shell")+" "+neutralBadge("THEME "+string(m.themePreset)),
		fmt.Sprintf("Top status | route=%s breakpoint=%s focus=%s sidebar=%t inspector=%t overlays=%d active_session=%s selection=%s mouse_capture=%s activity=%s %s", m.routeLabel(m.currentID), m.breakpoint(), m.focus.Label(), m.effectiveSidebarVisible(), m.shouldShowInspector(surface), len(m.overlays), defaultPlaceholder(m.activeSessionID, "none"), selection, mouseCapture, renderShellActivityState(m.activity.State), m.renderRunningIndicator()),
		fmt.Sprintf("Layout(wide): sidebar=%.0f%% inspector=%.0f%% collapsed=(sidebar:%t inspector:%t)", clampPaneRatio(m.sidebarRatio)*100, clampPaneRatio(m.inspectorRatio)*100, m.sidebarFolded, m.inspectorFolded),
	)
}

func (m shellModel) renderSyncHealth() string {
	text := "Shell sync health: " + renderShellSyncState(m.watchHealth.State)
	if strings.TrimSpace(m.activity.Active.ID) != "" {
		text += " " + infoBadge(fmt.Sprintf("active_%s=%s", m.activity.Active.Kind, m.activity.Active.ID))
	}
	if strings.TrimSpace(m.watchHealth.ErrorText) != "" {
		text += " " + muted("("+m.watchHealth.ErrorText+")")
	}
	return text
}

func (m shellModel) renderBreadcrumbs(surface routeSurface) string {
	breadcrumbs := surface.Breadcrumbs
	if len(breadcrumbs) == 0 {
		breadcrumbs = []string{"Home", m.routeLabel(m.currentID)}
	}
	return "Breadcrumbs: " + strings.Join(breadcrumbs, " > ")
}

func (m shellModel) renderBackstack() string {
	if len(m.backstack) == 0 {
		return muted("Backstack: empty")
	}
	items := make([]string, 0, len(m.backstack))
	for _, id := range m.backstack {
		items = append(items, m.routeLabel(id))
	}
	return muted("Backstack: " + strings.Join(items, " <- "))
}

func (m shellModel) renderShellPanes(surface routeSurface) string {
	mainTitle := strings.TrimSpace(surface.MainTitle)
	mainHeader := "Main pane"
	if mainTitle != "" {
		mainHeader += " — " + mainTitle
	}
	if m.breakpoint() == shellBreakpointWide {
		mainHeader += fmt.Sprintf(" (%.0f%%)", (1.0-clampPaneRatio(m.sidebarRatio)-clampPaneRatio(m.inspectorRatio))*100)
	}
	if activity := strings.TrimSpace(m.renderPaneActivityMarker()); activity != "" {
		mainHeader += " " + activity
	}
	parts := []string{framedPaneBlock(mainHeader, strings.TrimSpace(surface.Main), m.focus == focusContent)}
	if modes := renderModeSwitchTabs(surface.ModeTabs, surface.ActiveTab); strings.TrimSpace(modes) != "" {
		parts = append(parts, modes)
	}
	if m.effectiveSidebarVisible() {
		sidebarTitle := "Sidebar"
		if m.breakpoint() == shellBreakpointWide {
			sidebarTitle += fmt.Sprintf(" (%.0f%%)", clampPaneRatio(m.sidebarRatio)*100)
		}
		parts = append([]string{framedPaneBlock(sidebarTitle, m.renderSidebar(), m.focus == focusNav)}, parts...)
	}
	if m.shouldShowInspector(surface) {
		inspectorTitle := strings.TrimSpace(surface.InspectorTitle)
		if inspectorTitle == "" {
			inspectorTitle = "Inspector pane"
		}
		if m.breakpoint() == shellBreakpointWide {
			inspectorTitle += fmt.Sprintf(" (%.0f%%)", clampPaneRatio(m.inspectorRatio)*100)
		}
		parts = append(parts, framedPaneBlock(inspectorTitle, strings.TrimSpace(surface.Inspector), false))
	}
	return compactLines(parts...)
}

func (m shellModel) renderSidebar() string {
	if len(m.routes) == 0 && len(m.sessionItems) == 0 {
		return "(no routes or sessions)"
	}
	lines := make([]string, 0, len(m.routes)+len(m.sessionItems)+6)
	lines = append(lines, tableHeader("Navigation"))
	for i, r := range m.routes {
		selected := r.ID == m.currentID || i == m.nav.selectedIndex
		marker := " "
		if selected {
			marker = ">"
		}
		lines = append(lines, selectedLine(selected, fmt.Sprintf("%s %d %s", marker, r.Index, r.Label)))
	}
	if m.sessionLoading {
		lines = append(lines, "", tableHeader("Sessions"), "  loading canonical session directory...")
		return strings.Join(lines, "\n")
	}
	if strings.TrimSpace(m.sessionLoadError) != "" {
		lines = append(lines, "", tableHeader("Sessions"), "  load failed: "+m.sessionLoadError)
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "", tableHeader("Sessions"))
	items := sessionDirectoryItems(m.sessionItems, m.activeSessionID, m.pinnedSessions, m.recentSessions, m.viewedActivity, m.activity.Active)
	for i, item := range items {
		selected := i == m.sessionSelected
		marker := " "
		if selected {
			marker = ">"
		}
		lines = append(lines, selectedLine(selected, fmt.Sprintf("%s %s", marker, item)))
	}
	return strings.Join(lines, "\n")
}

func (m shellModel) renderBottomStrip(surface routeSurface) string {
	bottom := strings.TrimSpace(surface.BottomStrip)
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
		m.renderRouteCopyActions(),
		selectionHint,
	)
}

func (m shellModel) renderStatusSurface(surface routeSurface) string {
	status := strings.TrimSpace(surface.Status)
	if status == "" {
		status = fmt.Sprintf("route=%s scroll=%d", m.routeLabel(m.currentID), m.scroll)
	}
	selection := "selection=off"
	if m.selectionMode {
		selection = "selection=on"
	}
	return "Status: " + status + " | " + selection + " | clipboard=" + m.clipboard.IntegrationHint()
}

func (m shellModel) renderRouteCopyActions() string {
	actions := m.activeShellSurface().CopyActions
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
