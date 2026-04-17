package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m shellModel) applyPaletteAction(action paletteActionMsg) (tea.Model, tea.Cmd) {
	switch action.Verb {
	case verbBack:
		return m.navigateBack()
	case verbOpen:
		return m.applyPaletteOpen(action.Target)
	case verbInspect:
		return m.applyPaletteInspect(action.Target)
	case verbJump:
		return m.applyPaletteJump(action.Target)
	default:
		return m, nil
	}
}

func (m shellModel) applyPaletteOpen(target paletteTarget) (tea.Model, tea.Cmd) {
	m.closeNarrowOverlaysForPaletteTarget()
	return m.applyPaletteTargetByKind(target, verbOpen)
}

func (m shellModel) applyPaletteJump(target paletteTarget) (tea.Model, tea.Cmd) {
	m.closeNarrowOverlaysForPaletteTarget()
	return m.applyPaletteTargetByKind(target, verbJump)
}

func (m shellModel) applyPaletteInspect(target paletteTarget) (tea.Model, tea.Cmd) {
	m.closeNarrowOverlaysForPaletteTarget()
	loc, cmd := m.locationForPaletteTarget(target)
	if loc.Primary.RouteID == "" {
		return m, cmd
	}
	m.location.Inspector = &loc.Primary
	m.copyActionIndex = 0
	m.syncSidebarCursorToLocation()
	m.setFocus(focusContent)
	if m.breakpoint() == shellBreakpointNarrow && m.inspectorOn {
		m.beginOverlaySession()
		m.narrowInspectOn = true
		m.narrowSidebarOn = false
		m.setFocus(focusInspector)
		m.syncOverlayStack()
	}
	m.persistWorkbenchState()
	return m, cmd
}

func (m shellModel) navigateBack() (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	prev := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.location = prev
	m.nav.SelectByRouteID(prev.Primary.RouteID)
	m.syncSidebarCursorToLocation()
	m.setFocus(focusContent)
	m.persistWorkbenchState()
	return m, m.activateCurrentRouteCmd()
}

func (m *shellModel) closeNarrowOverlaysForPaletteTarget() {
	if m.breakpoint() != shellBreakpointNarrow {
		return
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
}

func (m shellModel) applyPaletteTargetByKind(target paletteTarget, verb navigationVerb) (tea.Model, tea.Cmd) {
	if verb == verbInspect {
		return m.applyPaletteInspect(target)
	}
	if target.Kind == "command" {
		cmd := m.commands.Execute(target.CommandID, &m)
		m.publishShellPreferencesToCurrentRoute()
		m.persistWorkbenchState()
		m.syncOverlayStack()
		return m, cmd
	}
	loc, cmd := m.locationForPaletteTarget(target)
	if loc.Primary.RouteID == "" {
		return m, cmd
	}
	if verb == verbJump {
		m.history = append(m.history, m.currentLocation())
	}
	m.location.Primary = loc.Primary
	if verb != verbInspect {
		m.location.Inspector = nil
	}
	m.copyActionIndex = 0
	m.nav.SelectByRouteID(loc.Primary.RouteID)
	if target.Kind == "session" {
		m.syncSidebarCursorToSessionID(target.SessionID)
	} else {
		m.syncSidebarCursorToLocation()
	}
	m.setFocus(focusContent)
	m.publishShellPreferencesToCurrentRoute()
	m.persistWorkbenchState()
	if m.breakpoint() == shellBreakpointNarrow {
		m.syncOverlayStack()
	}
	return m, tea.Batch(m.activateCurrentRouteCmd(), cmd)
}

func (m shellModel) locationForPaletteTarget(target paletteTarget) (shellWorkbenchLocation, tea.Cmd) {
	switch target.Kind {
	case "route":
		return m.locationForRouteTarget(target), nil
	case "session":
		return m.locationForSessionTarget(target), func() tea.Msg { return chatSelectSessionMsg{SessionID: strings.TrimSpace(target.SessionID)} }
	case "run":
		return m.locationForRunTarget(target), func() tea.Msg { return runsSelectRunMsg{RunID: strings.TrimSpace(target.RunID)} }
	case "approval":
		return m.locationForApprovalTarget(target), func() tea.Msg { return approvalsSelectMsg{ApprovalID: strings.TrimSpace(target.ApprovalID)} }
	case "action_center":
		return m.locationForActionCenterTarget(), nil
	case "artifact":
		return m.locationForArtifactTarget(target), func() tea.Msg { return artifactsSelectDigestMsg{Digest: strings.TrimSpace(target.Digest)} }
	case "audit":
		return m.locationForAuditTarget(target), func() tea.Msg { return auditSelectRecordMsg{Digest: strings.TrimSpace(target.Digest)} }
	default:
		return m.currentLocation(), nil
	}
}

func (m shellModel) locationForRouteTarget(target paletteTarget) shellWorkbenchLocation {
	rid := target.RouteID
	if rid == "" {
		rid = m.currentRouteID()
	}
	obj := workbenchObjectRef{Kind: "route", ID: string(rid)}
	m.trackRecentObject(obj)
	return shellWorkbenchLocation{Primary: shellObjectLocation{RouteID: rid, Object: obj}}
}

func (m shellModel) locationForSessionTarget(target paletteTarget) shellWorkbenchLocation {
	sid := strings.TrimSpace(target.SessionID)
	ws := m.workspaceForSession(sid)
	obj := workbenchObjectRef{Kind: "session", ID: sid, WorkspaceID: ws, SessionID: sid}
	m.trackRecentObject(obj)
	if sid != "" {
		m.activeSessionID = sid
		m.sessionSelected = selectedSessionIndex(m.sessionItems, sid)
		m.syncSidebarCursorToSessionID(sid)
		m.trackRecentSession(sid)
		m.markSessionViewed(sid)
		if ws != "" {
			m.lastSessionByWS[ws] = sid
		}
	}
	return shellWorkbenchLocation{Primary: shellObjectLocation{RouteID: routeChat, Object: obj}}
}

func (m shellModel) locationForRunTarget(target paletteTarget) shellWorkbenchLocation {
	runID := strings.TrimSpace(target.RunID)
	obj := workbenchObjectRef{Kind: "run", ID: runID, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID}
	m.trackRecentObject(obj)
	return shellWorkbenchLocation{Primary: shellObjectLocation{RouteID: routeRuns, Object: obj}}
}

func (m shellModel) locationForApprovalTarget(target paletteTarget) shellWorkbenchLocation {
	approvalID := strings.TrimSpace(target.ApprovalID)
	obj := workbenchObjectRef{Kind: "approval", ID: approvalID, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID}
	m.trackRecentObject(obj)
	return shellWorkbenchLocation{Primary: shellObjectLocation{RouteID: routeApprovals, Object: obj}}
}

func (m shellModel) locationForActionCenterTarget() shellWorkbenchLocation {
	obj := workbenchObjectRef{Kind: "route", ID: string(routeAction)}
	m.trackRecentObject(obj)
	return shellWorkbenchLocation{Primary: shellObjectLocation{RouteID: routeAction, Object: obj}}
}

func (m shellModel) locationForArtifactTarget(target paletteTarget) shellWorkbenchLocation {
	digest := strings.TrimSpace(target.Digest)
	obj := workbenchObjectRef{Kind: "artifact", ID: digest, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID}
	m.trackRecentObject(obj)
	return shellWorkbenchLocation{Primary: shellObjectLocation{RouteID: routeArtifacts, Object: obj}}
}

func (m shellModel) locationForAuditTarget(target paletteTarget) shellWorkbenchLocation {
	digest := strings.TrimSpace(target.Digest)
	obj := workbenchObjectRef{Kind: "audit", ID: digest, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID}
	m.trackRecentObject(obj)
	return shellWorkbenchLocation{Primary: shellObjectLocation{RouteID: routeAudit, Object: obj}}
}
func (m *shellModel) toggleActiveInspector() {
	m.inspectorOn = !m.inspectorOn
	if m.breakpoint() == shellBreakpointNarrow {
		if m.inspectorOn {
			m.beginOverlaySession()
			m.narrowInspectOn = true
			m.narrowSidebarOn = false
			m.setFocus(focusInspector)
		} else {
			m.narrowInspectOn = false
			m.restoreFocusAfterOverlayClose()
		}
		m.syncOverlayStack()
	}
	m.publishShellPreferencesToCurrentRoute()
	m.normalizeFocusForLayout()
	m.persistWorkbenchState()
}
