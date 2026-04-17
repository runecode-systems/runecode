package main

import (
	"fmt"
	"strings"
)

func (m shellModel) renderOverlayStack() string {
	if len(m.overlays) == 0 {
		return tableHeader("Overlay stack") + " none"
	}
	return tableHeader("Overlay stack") + " " + strings.Join(m.overlays, " -> ")
}

func (m shellModel) renderPalette() string {
	b := strings.Builder{}
	b.WriteString("Workbench Command Surface (: / ctrl+p)\n")
	b.WriteString("Verbs: open, inspect, jump, back\n")
	b.WriteString(fmt.Sprintf("Query: %q\n", m.palette.query))
	if len(m.palette.matches) == 0 {
		b.WriteString("No matches. Press esc to close.\n")
		return b.String()
	}
	b.WriteString("Matches:\n")
	for i, entry := range m.palette.matches {
		b.WriteString(paletteMatchLine(entry, i == m.palette.selectedIndex))
		b.WriteString("\n")
	}
	return b.String()
}

func (m shellModel) renderSessionQuickSwitcher() string {
	b := strings.Builder{}
	b.WriteString("Session Quick Switcher (ctrl+j)\n")
	b.WriteString(fmt.Sprintf("Query: %q\n", m.sessions.query))
	if len(m.sessions.matches) == 0 {
		b.WriteString("No matches. Press esc to close.\n")
		return b.String()
	}
	b.WriteString("Matches:\n")
	for i, s := range m.sessions.matches {
		marker := " "
		if i == m.sessions.selectedIndex {
			marker = ">"
		}
		sessionLabel := s.Identity.SessionID
		if m.activity.Active.Kind == "session" && strings.TrimSpace(m.activity.Active.ID) != "" && m.activity.Active.ID == s.Identity.SessionID {
			sessionLabel = "▶ " + sessionLabel
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
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func (m shellModel) paletteStartY() int {
	return 4
}

func (m shellModel) sidebarYRange() (startY int, endY int) {
	startY = 5
	endY = startY + len(m.routes) - 1
	return startY, endY
}

func (m shellModel) sidebarIndexAtMouse(mouseX int, mouseY int) (int, bool) {
	if mouseX < 0 || mouseX > 24 {
		return 0, false
	}
	startY, endY := m.sidebarYRange()
	if mouseY < startY || mouseY > endY {
		return 0, false
	}
	idx := mouseY - startY
	if idx < 0 || idx >= len(m.routes) {
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
	if m.activity.State != shellActivityStateRunning {
		return ""
	}
	if strings.TrimSpace(m.activity.Active.Kind) == "" || strings.TrimSpace(m.activity.Active.ID) == "" {
		return infoBadge("ACTIVE")
	}
	return infoBadge(fmt.Sprintf("ACTIVE %s=%s", m.activity.Active.Kind, m.activity.Active.ID))
}

func (m shellModel) renderRunningIndicator() string {
	if m.activity.State != shellActivityStateRunning {
		return ""
	}
	frames := []string{"⠁", "⠂", "⠄", "⠂", "⠁", "⠈", "⠐", "⠈"}
	label := "running"
	if strings.TrimSpace(m.activity.Active.Kind) != "" && strings.TrimSpace(m.activity.Active.ID) != "" {
		label = fmt.Sprintf("running %s:%s", m.activity.Active.Kind, m.activity.Active.ID)
	}
	return infoBadge(frames[m.activityFrame%len(frames)] + " " + label)
}
