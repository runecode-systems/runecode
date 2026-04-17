package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m shellModel) applyPaletteAction(action paletteActionMsg) (tea.Model, tea.Cmd) {
	switch action.Verb {
	case verbBack:
		return m.navigateBack()
	case verbOpen, verbJump:
		return m.applyPaletteTarget(action.Target)
	case verbInspect:
		return m.applyPaletteInspect(action.Target)
	default:
		return m, nil
	}
}

func (m shellModel) applyPaletteInspect(target paletteTarget) (tea.Model, tea.Cmd) {
	updated, cmd := m.applyPaletteTarget(target)
	shell, ok := updated.(shellModel)
	if !ok {
		return updated, cmd
	}
	if shell.breakpoint() != shellBreakpointNarrow {
		return shell, cmd
	}
	if strings.TrimSpace(shell.activeShellSurface().Inspector) == "" || !shell.inspectorOn {
		return shell, cmd
	}
	shell.narrowInspectOn = true
	shell.narrowSidebarOn = false
	shell.syncOverlayStack()
	return shell, cmd
}

func (m shellModel) navigateBack() (tea.Model, tea.Cmd) {
	if len(m.backstack) == 0 {
		return m, nil
	}
	prev := m.backstack[len(m.backstack)-1]
	m.backstack = m.backstack[:len(m.backstack)-1]
	m.currentID = prev
	m.nav.SelectByRouteID(prev)
	m.focusManager.Set(focusContent)
	m.focus = m.focusManager.Current()
	m.persistWorkbenchState()
	return m, m.activateCurrentRouteCmd()
}

func (m shellModel) applyPaletteTarget(target paletteTarget) (tea.Model, tea.Cmd) {
	m.closeNarrowOverlaysForPaletteTarget()
	return m.applyPaletteTargetByKind(target)
}

func (m *shellModel) closeNarrowOverlaysForPaletteTarget() {
	if m.breakpoint() != shellBreakpointNarrow {
		return
	}
	m.narrowSidebarOn = false
	m.narrowInspectOn = false
}

func (m shellModel) applyPaletteTargetByKind(target paletteTarget) (tea.Model, tea.Cmd) {
	switch target.Kind {
	case "route":
		return m.applyRouteTarget(target)
	case "session":
		return m.applySessionTarget(target)
	case "run":
		return m.applyRunTarget(target)
	case "approval":
		return m.applyApprovalTarget(target)
	case "action_center":
		return m.applyActionCenterTarget()
	case "artifact":
		return m.applyArtifactTarget(target)
	case "audit":
		return m.applyAuditTarget(target)
	case "command":
		return m.applyCommandTarget(target)
	default:
		return m, nil
	}
}

func (m shellModel) applyRouteTarget(target paletteTarget) (tea.Model, tea.Cmd) {
	m.trackRecentObject(workbenchObjectRef{Kind: "route", ID: string(target.RouteID)})
	m.syncNarrowOverlayStackIfNeeded()
	return m.switchToRoute(target.RouteID, true)
}

func (m shellModel) applySessionTarget(target paletteTarget) (tea.Model, tea.Cmd) {
	m.trackRecentObject(workbenchObjectRef{Kind: "session", ID: target.SessionID, WorkspaceID: m.workspaceForSession(target.SessionID), SessionID: target.SessionID})
	m.syncNarrowOverlayStackIfNeeded()
	return m.activateSessionFromSidebarByID(target.SessionID)
}

func (m shellModel) applyRunTarget(target paletteTarget) (tea.Model, tea.Cmd) {
	m.trackRecentObject(workbenchObjectRef{Kind: "run", ID: target.RunID, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID})
	updated, cmd := m.switchToRoute(routeRuns, true)
	return withNarrowOverlaySynced(updated, m.breakpoint()), tea.Batch(cmd, func() tea.Msg { return runsSelectRunMsg{RunID: target.RunID} })
}

func (m shellModel) applyApprovalTarget(target paletteTarget) (tea.Model, tea.Cmd) {
	m.trackRecentObject(workbenchObjectRef{Kind: "approval", ID: target.ApprovalID, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID})
	updated, cmd := m.switchToRoute(routeApprovals, true)
	return withNarrowOverlaySynced(updated, m.breakpoint()), tea.Batch(cmd, func() tea.Msg { return approvalsSelectMsg{ApprovalID: target.ApprovalID} })
}

func (m shellModel) applyActionCenterTarget() (tea.Model, tea.Cmd) {
	m.trackRecentObject(workbenchObjectRef{Kind: "route", ID: string(routeAction)})
	m.syncNarrowOverlayStackIfNeeded()
	return m.switchToRoute(routeAction, true)
}

func (m shellModel) applyArtifactTarget(target paletteTarget) (tea.Model, tea.Cmd) {
	m.trackRecentObject(workbenchObjectRef{Kind: "artifact", ID: target.Digest, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID})
	updated, cmd := m.switchToRoute(routeArtifacts, true)
	return withNarrowOverlaySynced(updated, m.breakpoint()), tea.Batch(cmd, func() tea.Msg { return artifactsSelectDigestMsg{Digest: target.Digest} })
}

func (m shellModel) applyAuditTarget(target paletteTarget) (tea.Model, tea.Cmd) {
	m.trackRecentObject(workbenchObjectRef{Kind: "audit", ID: target.Digest, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID})
	updated, cmd := m.switchToRoute(routeAudit, true)
	return withNarrowOverlaySynced(updated, m.breakpoint()), tea.Batch(cmd, func() tea.Msg { return auditSelectRecordMsg{Digest: target.Digest} })
}

func (m shellModel) applyCommandTarget(target paletteTarget) (tea.Model, tea.Cmd) {
	if m.commands.Execute(target.CommandID, &m) {
		m.applyInspectorVisibilityToRoutes()
		m.applyPreferredPresentationToRoutes()
		m.persistWorkbenchState()
	}
	m.syncOverlayStack()
	return m, nil
}

func (m *shellModel) syncNarrowOverlayStackIfNeeded() {
	if m.breakpoint() == shellBreakpointNarrow {
		m.syncOverlayStack()
	}
}

func withNarrowOverlaySynced(updated tea.Model, breakpoint shellBreakpoint) tea.Model {
	shell, ok := updated.(shellModel)
	if !ok {
		return updated
	}
	if breakpoint == shellBreakpointNarrow {
		shell.syncOverlayStack()
	}
	return shell
}
func (m *shellModel) toggleActiveInspector() {
	model, ok := m.routeModels[m.currentID]
	if !ok || model == nil {
		return
	}
	switch typed := model.(type) {
	case chatRouteModel:
		typed.inspectorOn = !typed.inspectorOn
		m.inspectorOn = typed.inspectorOn
		m.routeModels[m.currentID] = typed
	case runsRouteModel:
		typed.inspectorOn = !typed.inspectorOn
		m.inspectorOn = typed.inspectorOn
		m.routeModels[m.currentID] = typed
	case approvalsRouteModel:
		typed.inspectorOn = !typed.inspectorOn
		m.inspectorOn = typed.inspectorOn
		m.routeModels[m.currentID] = typed
	case artifactsRouteModel:
		typed.inspectorOn = !typed.inspectorOn
		m.inspectorOn = typed.inspectorOn
		m.routeModels[m.currentID] = typed
	case auditRouteModel:
		typed.inspectorOn = !typed.inspectorOn
		m.inspectorOn = typed.inspectorOn
		m.routeModels[m.currentID] = typed
	default:
		return
	}
	m.applyInspectorVisibilityToRoutes()
	if m.breakpoint() == shellBreakpointNarrow {
		if m.inspectorOn {
			m.narrowInspectOn = true
			m.narrowSidebarOn = false
		} else {
			m.narrowInspectOn = false
		}
		m.syncOverlayStack()
	}
	m.persistWorkbenchState()
}
