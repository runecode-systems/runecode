package main

import (
	"fmt"
	"strings"
)

func (m shellModel) renderOverlayStack() string {
	if len(m.overlays) == 0 {
		return tableHeader("Overlay stack") + " none"
	}
	labels := make([]string, 0, len(m.overlays))
	for _, id := range m.overlays {
		labels = append(labels, string(id))
	}
	return tableHeader("Overlay stack") + " " + strings.Join(labels, " -> ")
}

func (m shellModel) renderPalette() string {
	b := strings.Builder{}
	b.WriteString(tableHeader("Workbench Command Surface") + " " + neutralBadge("toggle=: / ctrl+p") + "\n")
	b.WriteString("Verbs: " + strings.Join([]string{infoBadge("open"), infoBadge("inspect"), infoBadge("jump"), infoBadge("back")}, " ") + "\n")
	b.WriteString(fmt.Sprintf("Query: %q\n", m.palette.query))
	if len(m.palette.matches) == 0 {
		b.WriteString(muted("No matches. Press esc to close."))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(tableHeader("Matches"))
	b.WriteString("\n")
	rows := make([]boundedListRow, 0, len(m.palette.matches))
	for _, entry := range m.palette.matches {
		rows = append(rows, boundedListRow{Text: paletteMatchLine(entry, false), Selectable: true})
	}
	b.WriteString(renderBoundedList(boundedListSpec{
		Rows:          rows,
		Selected:      m.palette.selectedIndex,
		Width:         boundedOverlayListWidth(m.width),
		Height:        8,
		GapMarker:     "...",
		PreserveGaps:  true,
		ApplySelected: true,
		ActiveFill:    true,
	}))
	b.WriteString("\n")
	return b.String()
}

func (m shellModel) renderSessionQuickSwitcher() string {
	b := strings.Builder{}
	b.WriteString(tableHeader("Session Quick Switcher") + " " + neutralBadge("toggle=ctrl+j") + "\n")
	b.WriteString(fmt.Sprintf("Query: %q\n", m.sessions.query))
	if len(m.sessions.matches) == 0 {
		b.WriteString(muted("No matches. Press esc to close."))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(tableHeader("Matches"))
	b.WriteString("\n")
	rows := make([]boundedListRow, 0, len(m.sessions.matches))
	for i, s := range m.sessions.matches {
		marker := " "
		if i == m.sessions.selectedIndex {
			marker = "▶"
		}
		sessionLabel := s.Identity.SessionID
		if m.watch.projection.Activity.Active.Kind == "session" && strings.TrimSpace(m.watch.projection.Activity.Active.ID) != "" && m.watch.projection.Activity.Active.ID == s.Identity.SessionID {
			sessionLabel = "● " + sessionLabel
		}
		line := fmt.Sprintf(" %s %s | ws=%s | activity=%s/%s | cue=%s | preview=%q | incomplete=%t | runs=%d approvals=%d",
			marker,
			sessionLabel,
			s.Identity.WorkspaceID,
			defaultPlaceholder(s.LastActivityAt, "n/a"),
			defaultPlaceholder(s.LastActivityKind, "n/a"),
			sessionHighLevelCue(s),
			truncateText(s.LastActivityPreview, 50),
			s.HasIncompleteTurn,
			s.LinkedRunCount,
			s.LinkedApprovalCount,
		)
		rows = append(rows, boundedListRow{Text: line, Selectable: true})
	}
	b.WriteString(renderBoundedList(boundedListSpec{
		Rows:          rows,
		Selected:      m.sessions.selectedIndex,
		Width:         boundedOverlayListWidth(m.width),
		Height:        8,
		GapMarker:     "...",
		PreserveGaps:  true,
		ApplySelected: true,
		ActiveFill:    true,
	}))
	b.WriteString("\n")
	return b.String()
}

func boundedOverlayListWidth(viewportWidth int) int {
	if viewportWidth <= 0 {
		return 0
	}
	width := overlayBlockWidth(viewportWidth) - 4
	if width < 1 {
		return 1
	}
	return width
}

func (m shellModel) paletteStartY() int {
	return 3
}

func (m shellModel) sidebarYRange() (startY int, endY int) {
	startY = shellTopStatusHeight + shellSyncHealthHeight + shellBreadcrumbHeight + shellHistoryHeight + shellPaneSpacerHeight + 3
	endY = startY + m.sidebarMouseRowCount() - 1
	return startY, endY
}

func (m shellModel) sidebarMouseRowCount() int {
	rows := len(m.routes)
	if m.sessionLoading || strings.TrimSpace(m.sessionLoadError) != "" {
		return rows
	}
	sessionCount := len(sortedSessionDirectorySummaries(m.sessionItems, m.recentSessions))
	if sessionCount == 0 {
		return rows
	}
	return rows + 2 + sessionCount
}

func (m shellModel) sidebarIndexAtMouse(mouseX int, mouseY int) (int, bool) {
	width := m.planShellLayout(m.activeShellSurface()).Regions.Sidebar.Width
	if width <= 0 {
		return 0, false
	}
	if mouseX < 0 || mouseX >= width {
		return 0, false
	}
	startY, endY := m.sidebarYRange()
	if mouseY < startY || mouseY > endY {
		return 0, false
	}
	idx := mouseY - startY
	entries := m.sidebarEntries()
	if idx < 0 || idx >= len(entries) {
		return 0, false
	}
	return idx, true
}

func (m shellModel) routeLabel(id routeID) string {
	for _, r := range m.routes {
		if r.ID == id {
			return r.Label
		}
	}
	if id == "" {
		return "unknown"
	}
	return string(id)
}

func (m shellModel) renderPaneActivityMarker() string {
	if m.watch.projection.Activity.State != shellActivityStateRunning {
		return ""
	}
	if strings.TrimSpace(m.watch.projection.Activity.Active.Kind) == "" || strings.TrimSpace(m.watch.projection.Activity.Active.ID) == "" {
		return infoBadge("ACTIVE")
	}
	return infoBadge(fmt.Sprintf("ACTIVE %s=%s", sanitizeUIText(m.watch.projection.Activity.Active.Kind), sanitizeUIText(m.watch.projection.Activity.Active.ID)))
}

func (m shellModel) renderRunningIndicator() string {
	if m.watch.projection.Activity.State != shellActivityStateRunning {
		return ""
	}
	frames := []string{"⠁", "⠂", "⠄", "⠂", "⠁", "⠈", "⠐", "⠈"}
	label := "running"
	if strings.TrimSpace(m.watch.projection.Activity.Active.Kind) != "" && strings.TrimSpace(m.watch.projection.Activity.Active.ID) != "" {
		label = fmt.Sprintf("running %s:%s", sanitizeUIText(m.watch.projection.Activity.Active.Kind), sanitizeUIText(m.watch.projection.Activity.Active.ID))
	}
	return infoBadge(frames[m.activityFrame%len(frames)] + " " + label)
}
