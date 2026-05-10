package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
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
	width := boundedOverlayListWidth(m.width)
	b.WriteString(renderOverlaySearchPrompt(m.palette.query) + "\n")
	if m.palette.MatchCount() == 0 {
		if m.palette.entriesLoading || m.palette.filterLoading {
			b.WriteString(muted("Updating matches..."))
			b.WriteString("\n")
			return b.String()
		}
		b.WriteString(muted("No matches. Keep typing or press esc to close."))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(tableHeader("Suggested actions"))
	b.WriteString("\n")
	b.WriteString(renderBoundedListWindowed(boundedListWindowedSpec{
		TotalRows:     m.palette.MatchCount(),
		SelectedRow:   m.palette.selectedIndex,
		Width:         width,
		Height:        10,
		GapMarker:     "...",
		ApplySelected: true,
		ActiveFill:    true,
		RenderRow: func(index int) boundedListRow {
			entry, ok := m.palette.MatchEntry(index)
			if !ok {
				return boundedListRow{}
			}
			return boundedListRow{Text: paletteMatchLineBounded(entry, width), Selectable: true}
		},
	}))
	b.WriteString("\n")
	return b.String()
}

func (m shellModel) renderSessionQuickSwitcher() string {
	b := strings.Builder{}
	width := boundedOverlayListWidth(m.width)
	b.WriteString(renderOverlaySearchPrompt(m.sessions.query) + "\n")
	if len(m.sessions.matches) == 0 {
		b.WriteString(muted("No matches. Press esc to close."))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(tableHeader("Recent sessions"))
	b.WriteString("\n")
	b.WriteString(renderBoundedListWindowed(boundedListWindowedSpec{
		TotalRows:     len(m.sessions.matches),
		SelectedRow:   m.sessions.selectedIndex,
		Width:         width,
		Height:        8,
		GapMarker:     "...",
		ApplySelected: true,
		ActiveFill:    true,
		RenderRow: func(index int) boundedListRow {
			return boundedListRow{Text: m.renderSessionQuickSwitcherRow(index, width), Selectable: true}
		},
	}))
	b.WriteString("\n")
	return b.String()
}

func (m shellModel) renderSessionQuickSwitcherRow(index int, width int) string {
	s := m.sessions.matches[index]
	marker := " "
	if index == m.sessions.selectedIndex {
		marker = "▶"
	}
	sessionLabel := s.Identity.SessionID
	if m.watch.projection.Activity.Active.Kind == "session" && strings.TrimSpace(m.watch.projection.Activity.Active.ID) != "" && m.watch.projection.Activity.Active.ID == s.Identity.SessionID {
		sessionLabel = "● " + sessionLabel
	}
	preview := truncateText(sanitizeUIText(s.LastActivityPreview), 58)
	lineParts := []string{fmt.Sprintf("%s %s", marker, sessionLabel), sessionHighLevelCue(s)}
	if counts := strings.TrimSpace(compactSessionSwitcherCounts(s)); counts != "" {
		lineParts = append(lineParts, "["+counts+"]")
	}
	line := clipDisplayText(strings.Join(lineParts, "  "), width)
	detailParts := make([]string, 0, 3)
	if workspace := strings.TrimSpace(s.Identity.WorkspaceID); workspace != "" {
		detailParts = append(detailParts, "Workspace "+workspace)
	}
	if activityKind := strings.TrimSpace(s.LastActivityKind); activityKind != "" {
		detailParts = append(detailParts, "Recent activity "+activityKind)
	}
	if preview != "" {
		detailParts = append(detailParts, preview)
	}
	detail := "    " + strings.Join(detailParts, "  •  ")
	if strings.TrimSpace(detail) == "" {
		detail = "    No recent activity yet."
	}
	if s.HasIncompleteTurn {
		detail += "  •  Needs follow-up"
	}
	return compactLines(line, muted(clipDisplayText(detail, width)))
}

func compactSessionSwitcherCounts(summary brokerapi.SessionSummary) string {
	parts := make([]string, 0, 2)
	if summary.LinkedRunCount > 0 {
		parts = append(parts, countNoun(summary.LinkedRunCount, "run", "runs"))
	}
	if summary.LinkedApprovalCount > 0 {
		parts = append(parts, countNoun(summary.LinkedApprovalCount, "approval", "approvals"))
	}
	if len(parts) == 0 {
		return "idle"
	}
	return strings.Join(parts, " • ")
}

func (m shellModel) renderLeaderWhichKey() string {
	b := strings.Builder{}
	b.WriteString(tableHeader("Leader Mode") + " " + neutralBadge("start="+m.keys.LeaderStart.label()) + "\n")
	b.WriteString("Sequence: " + m.leader.SequenceLabel() + "\n")
	b.WriteString("Press esc to abort.\n")
	if len(m.leader.prefix) == 0 {
		b.WriteString("Help: " + m.keys.LeaderStart.label() + " leader mode; tab next focus area; shift+tab previous focus area.\n")
		b.WriteString("      ctrl+p opens quick jump palette; ctrl+j opens session quick switcher.\n")
	}
	choices := m.leader.Choices()
	if len(choices) == 0 {
		b.WriteString(muted("No valid next keys."))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(tableHeader("Valid next keys"))
	b.WriteString("\n")
	b.WriteString(renderBoundedListWindowed(boundedListWindowedSpec{
		TotalRows: len(choices),
		Width:     boundedOverlayListWidth(m.width),
		Height:    10,
		GapMarker: "...",
		RenderRow: func(index int) boundedListRow {
			return boundedListRow{Text: leaderChoiceLine(choices[index])}
		},
	}))
	b.WriteString("\n")
	return b.String()
}

func leaderChoiceLine(choice shellLeaderChoice) string {
	suffix := ""
	if choice.Completes {
		suffix = " " + infoBadge("exec")
	}
	return " • " + choice.Key + " — " + defaultPlaceholder(choice.Label, "(group)") + " — " + defaultPlaceholder(choice.Description, "") + suffix
}

func (m shellModel) renderQuitConfirmDialog() string {
	reason := strings.TrimSpace(m.quitConfirm.reason)
	if reason == "" {
		reason = "active local entry"
	}
	b := strings.Builder{}
	b.WriteString(tableHeader("Quit Confirmation") + " " + warnBadge("entry active") + "\n")
	b.WriteString("Quitting now will discard " + reason + ".\n")
	b.WriteString("Press enter or y to quit anyway.\n")
	b.WriteString("Press esc or n to continue editing.\n")
	b.WriteString(muted("Emergency escape hatch remains available: press ctrl+c twice."))
	return b.String()
}

func boundedOverlayListWidth(viewportWidth int) int {
	if viewportWidth <= 0 {
		return 80
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
	startY = shellTopStatusHeight + shellSyncHealthHeight + shellPaneSpacerHeight + 3
	endY = startY + m.sidebarMouseRowCount() - 1
	return startY, endY
}

func (m shellModel) sidebarMouseRowCount() int {
	return len(m.sidebarMouseRows())
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
	rows := m.sidebarMouseRows()
	idx := mouseY - startY
	if idx < 0 || idx >= len(rows) {
		return 0, false
	}
	entryIdx := rows[idx]
	if entryIdx < 0 {
		return 0, false
	}
	return entryIdx, true
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
	if m.watch.projection.Activity.State != shellActivityStateRunning && m.watch.projection.Activity.State != shellActivityStateWaiting {
		return ""
	}
	if m.watch.projection.Activity.State == shellActivityStateWaiting {
		return warnBadge("WAITING")
	}
	return infoBadge("WORKING")
}

func (m shellModel) renderRunningIndicator() string {
	if m.watch.projection.Activity.State != shellActivityStateRunning {
		return ""
	}
	frames := []string{"⠁", "⠂", "⠄", "⠂", "⠁", "⠈", "⠐", "⠈"}
	label := "working"
	if target := strings.TrimSpace(humanShellActivityTarget(m.watch.projection.Activity.Active)); target != "" {
		label = "working on " + target
	}
	return infoBadge(frames[m.activityFrame%len(frames)] + " " + label)
}

func (m shellModel) renderPrimaryWorkbenchActions() string {
	parts := make([]string, 0, 3)
	if _, ok := m.actions.definitionByID("shell.open_action_center"); ok {
		parts = append(parts, "Action Center")
	}
	if _, ok := m.actions.definitionByID("shell.open_approvals"); ok {
		parts = append(parts, "Approvals")
	}
	if _, ok := m.actions.definitionByID("shell.open_palette"); ok {
		if m.width > 0 && m.width < shellMediumMinWidth {
			parts = append(parts, "Cmds ^P/:")
		} else {
			parts = append(parts, "Commands ctrl+p/:")
		}
	}
	if m.width > 0 && m.width < shellMediumMinWidth && len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, " · ")
}

func renderOverlaySearchPrompt(query string) string {
	return appTheme.TextSecondary.Render("› Search") + "  " + paletteSearchText(query)
}
