package main

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func (m shellModel) loadSessionWorkspaceCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.SessionList(ctx, 50)
		if err != nil {
			return sessionWorkspaceLoadedMsg{err: err}
		}
		return sessionWorkspaceLoadedMsg{sessions: resp.Sessions}
	}
}

func (m *shellModel) applySessionWorkspaceLoaded(msg sessionWorkspaceLoadedMsg) {
	m.sessionLoading = false
	if msg.err != nil {
		m.sessionLoadError = safeUIErrorText(msg.err)
		m.refreshObjectIndexFromShellState()
		return
	}
	m.sessionLoadError = ""
	m.sessionItems = append([]brokerapi.SessionSummary(nil), msg.sessions...)
	m.rememberSessionWorkspaces(m.sessionItems)
	m.sessions = m.sessions.UpdateSessions(m.sessionItems)
	m.ensureActiveSessionSelection()
	m.sessionSelected = selectedSessionIndex(m.sessionItems, m.activeSessionID)
	m.syncSidebarCursorToLocation()
	m.trackActiveSessionState()
	m.refreshObjectIndexFromShellState()
	m.persistWorkbenchState()
}

func (m *shellModel) rememberSessionWorkspaces(items []brokerapi.SessionSummary) {
	for _, s := range items {
		sid := strings.TrimSpace(s.Identity.SessionID)
		if sid == "" {
			continue
		}
		m.rememberSessionWorkspace(sid, s.Identity.WorkspaceID)
	}
}

func (m *shellModel) ensureActiveSessionSelection() {
	if m.activeSessionID == "" {
		m.activeSessionID = m.defaultSessionSelection()
	}
	if selectedSessionIndex(m.sessionItems, m.activeSessionID) < 0 && len(m.sessionItems) > 0 {
		m.activeSessionID = m.sessionItems[0].Identity.SessionID
	}
}

func (m *shellModel) defaultSessionSelection() string {
	if len(m.sessionItems) == 0 {
		return ""
	}
	first := m.sessionItems[0]
	fallback := first.Identity.SessionID
	ws := strings.TrimSpace(first.Identity.WorkspaceID)
	if ws == "" {
		return fallback
	}
	remembered := strings.TrimSpace(m.lastSessionByWS[ws])
	if remembered == "" || selectedSessionIndex(m.sessionItems, remembered) < 0 {
		return fallback
	}
	return remembered
}

func (m *shellModel) trackActiveSessionState() {
	if m.activeSessionID == "" {
		return
	}
	m.trackRecentSession(m.activeSessionID)
	m.trackRecentObject(workbenchObjectRef{Kind: "session", ID: m.activeSessionID, WorkspaceID: m.workspaceForSession(m.activeSessionID), SessionID: m.activeSessionID})
	m.markSessionViewed(m.activeSessionID)
	if ws := m.workspaceForSession(m.activeSessionID); ws != "" {
		m.lastSessionByWS[ws] = m.activeSessionID
	}
}

func (m *shellModel) moveSessionSelection(delta int) {
	m.moveSidebarCursor(delta)
}

func (m *shellModel) toggleSelectedSessionPin() {
	entry, ok := m.selectedSidebarEntry()
	if !ok || entry.Kind != sidebarEntrySession {
		return
	}
	sid := strings.TrimSpace(entry.Session.Identity.SessionID)
	if sid == "" {
		return
	}
	if _, ok := m.pinnedSessions[sid]; ok {
		delete(m.pinnedSessions, sid)
		m.toasts.Push(toastInfo, "Session unpinned: "+sid)
		return
	}
	m.pinnedSessions[sid] = struct{}{}
	m.toasts.Push(toastInfo, "Session pinned: "+sid)
}

func (m shellModel) activateSelectedSessionFromSidebar() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedSidebarEntry()
	if !ok || entry.Kind != sidebarEntrySession {
		return m, nil
	}
	sid := strings.TrimSpace(entry.Session.Identity.SessionID)
	if sid == "" {
		return m, nil
	}
	m.syncSidebarCursorToSessionID(sid)
	return m.activateSessionFromSidebarByID(sid)
}

func (m shellModel) activateSessionFromSidebarByID(sessionID string) (tea.Model, tea.Cmd) {
	updated, cmd := m.applyPaletteAction(paletteActionMsg{Verb: verbOpen, Target: paletteTarget{Kind: "session", SessionID: strings.TrimSpace(sessionID)}})
	shell, ok := updated.(shellModel)
	if ok {
		shell.toasts.Push(toastInfo, "Active session switched: "+shell.activeSessionID)
		return shell, cmd
	}
	return updated, cmd
}

func (m *shellModel) trackRecentSession(sessionID string) {
	if strings.TrimSpace(sessionID) == "" {
		return
	}
	filtered := []string{sessionID}
	for _, sid := range m.recentSessions {
		if sid != sessionID {
			filtered = append(filtered, sid)
		}
		if len(filtered) >= 8 {
			break
		}
	}
	m.recentSessions = filtered
	m.refreshObjectIndexFromShellState()
}

func (m *shellModel) trackRecentObject(ref workbenchObjectRef) {
	ref.Kind = strings.TrimSpace(strings.ToLower(ref.Kind))
	ref.ID = strings.TrimSpace(ref.ID)
	if ref.Kind == "" || ref.ID == "" {
		return
	}
	ref.WorkspaceID = strings.TrimSpace(ref.WorkspaceID)
	ref.SessionID = strings.TrimSpace(ref.SessionID)
	out := []workbenchObjectRef{ref}
	for _, item := range m.recentObjects {
		if item.Kind == ref.Kind && item.ID == ref.ID {
			continue
		}
		out = append(out, item)
		if len(out) >= 20 {
			break
		}
	}
	m.recentObjects = out
	m.refreshObjectIndexFromShellState()
}

func (m *shellModel) workspaceForSession(sessionID string) string {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return ""
	}
	if ws := strings.TrimSpace(m.sessionWorkspace[sid]); ws != "" {
		return ws
	}
	for _, s := range m.sessionItems {
		if s.Identity.SessionID == sid {
			return strings.TrimSpace(s.Identity.WorkspaceID)
		}
	}
	return ""
}

func (m *shellModel) markSessionViewed(sessionID string) {
	if strings.TrimSpace(sessionID) == "" {
		return
	}
	for _, s := range m.sessionItems {
		if s.Identity.SessionID == sessionID {
			m.viewedActivity[sessionID] = s.LastActivityAt
			return
		}
	}
}

func sortedSessionKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func cloneViewedActivity(viewed map[string]string) map[string]string {
	if viewed == nil {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(viewed))
	for key, value := range viewed {
		cloned[key] = value
	}
	return cloned
}
