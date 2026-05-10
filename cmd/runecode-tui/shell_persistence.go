package main

import (
	"sort"
	"strings"
)

const (
	maxPersistedPinnedSessions  = 50
	maxPersistedRecentSessions  = 8
	maxPersistedRecentObjects   = 20
	maxPersistedLastSessionByWS = 50
)

func (m *shellModel) persistWorkbenchState() {
	if m.workbench == nil {
		return
	}
	m.workbench.Write(m.workbenchScope, m.currentWorkbenchState())
}

func (m *shellModel) flushWorkbenchState() {
	if flusher, ok := m.workbench.(shellWorkbenchStateFlusher); ok {
		flusher.Flush()
	}
}

func (m *shellModel) workbenchFlushError() error {
	if flusher, ok := m.workbench.(interface{ LastError() error }); ok {
		return flusher.LastError()
	}
	return nil
}

func (m *shellModel) currentWorkbenchState() workbenchLocalState {
	return workbenchLocalState{
		SidebarVisible:     m.sidebarVisible,
		InspectorVisible:   m.inspectorOn,
		InspectorMode:      normalizePresentationMode(m.preferredMode),
		ThemePreset:        normalizeThemePreset(m.themePreset),
		LeaderKey:          strings.TrimSpace(m.leaderKeyConfig),
		LastRouteID:        m.currentRouteID(),
		LastSessionID:      m.activeSessionID,
		LastSessionByWS:    cloneSessionMapWithLimit(m.lastSessionByWS, maxPersistedLastSessionByWS),
		PinnedSessions:     m.persistedPinnedSessionRefs(),
		RecentSessions:     m.persistedRecentSessionRefs(),
		RecentObjects:      cloneRecentObjectsWithLimit(m.recentObjects, maxPersistedRecentObjects),
		ViewedActivity:     cloneViewedActivity(m.viewedActivity),
		SidebarPaneRatio:   clampPaneRatio(m.sidebarRatio),
		InspectorPaneRatio: clampPaneRatio(m.inspectorRatio),
		SidebarCollapsed:   m.sidebarFolded,
		InspectorCollapsed: m.inspectorFolded,
	}
}

func (m *shellModel) restoreWorkbenchState() {
	if m.workbench == nil {
		return
	}
	state := m.workbench.Read(m.workbenchScope)
	if isZeroWorkbenchState(state) {
		return
	}
	m.restoreWorkbenchLayoutState(state)
	m.restoreWorkbenchRouteAndTheme(state)
	m.restoreWorkbenchSessionState(state)
	m.restoreWorkbenchRecentState(state)
	m.publishShellPreferencesToCurrentRoute()
	m.refreshObjectIndexFromShellState()
}

func (m *shellModel) restoreWorkbenchLayoutState(state workbenchLocalState) {
	m.sidebarVisible = state.SidebarVisible
	m.inspectorOn = state.InspectorVisible
	m.sidebarRatio = restorePaneRatio(state.SidebarPaneRatio, 0.22)
	m.inspectorRatio = restorePaneRatio(state.InspectorPaneRatio, 0.30)
	m.sidebarFolded = state.SidebarCollapsed
	m.inspectorFolded = state.InspectorCollapsed
}

func restorePaneRatio(raw float64, fallback float64) float64 {
	ratio := clampPaneRatio(raw)
	if ratio <= 0 {
		return fallback
	}
	return ratio
}

func (m *shellModel) restoreWorkbenchRouteAndTheme(state workbenchLocalState) {
	m.preferredMode = normalizePresentationMode(state.InspectorMode)
	if m.preferredMode == "" {
		m.preferredMode = presentationRendered
	}
	if rid := m.validatedRouteID(state.LastRouteID); rid != "" {
		m.location.Primary = shellObjectLocation{RouteID: rid, Object: workbenchObjectRef{Kind: "route", ID: string(rid)}}
		m.nav.SelectByRouteID(rid)
	}
	m.themePreset = normalizeThemePreset(state.ThemePreset)
	if m.themePreset == "" {
		m.themePreset = themePresetDark
	}
	appTheme = newTheme(m.themePreset)
	if err := m.setLeaderKey(strings.TrimSpace(state.LeaderKey)); err != nil && strings.TrimSpace(state.LeaderKey) != "" {
		m.leaderKeyInvalid = strings.TrimSpace(state.LeaderKey)
		m.toasts.Push(toastWarn, "Persisted leader key invalid; using default space leader.")
	}
}

func (m *shellModel) validatedRouteID(rid routeID) routeID {
	rid = routeID(strings.TrimSpace(string(rid)))
	if rid == "" {
		return ""
	}
	if _, ok := m.routeModels[rid]; ok {
		return rid
	}
	for _, def := range m.routes {
		if def.ID == rid {
			return rid
		}
	}
	return routeChat
}

func (m *shellModel) restoreWorkbenchSessionState(state workbenchLocalState) {
	m.activeSessionID = strings.TrimSpace(state.LastSessionID)
	m.pinnedSessions = map[string]struct{}{}
	for _, ref := range state.PinnedSessions {
		if len(m.pinnedSessions) >= maxPersistedPinnedSessions {
			break
		}
		sid := strings.TrimSpace(ref.SessionID)
		if sid == "" {
			continue
		}
		m.pinnedSessions[sid] = struct{}{}
		m.rememberSessionWorkspace(sid, ref.WorkspaceID)
	}
	m.recentSessions = make([]string, 0, min(len(state.RecentSessions), maxPersistedRecentSessions))
	for _, ref := range state.RecentSessions {
		if len(m.recentSessions) >= maxPersistedRecentSessions {
			break
		}
		sid := strings.TrimSpace(ref.SessionID)
		if sid == "" {
			continue
		}
		m.recentSessions = append(m.recentSessions, sid)
		m.rememberSessionWorkspace(sid, ref.WorkspaceID)
	}
}

func (m *shellModel) rememberSessionWorkspace(sessionID string, workspaceID string) {
	if ws := strings.TrimSpace(workspaceID); ws != "" {
		m.sessionWorkspace[sessionID] = ws
		m.refreshObjectIndexFromShellState()
	}
}

func (m *shellModel) restoreWorkbenchRecentState(state workbenchLocalState) {
	m.recentObjects = cloneRecentObjectsWithLimit(state.RecentObjects, maxPersistedRecentObjects)
	m.lastSessionByWS = cloneSessionMapWithLimit(state.LastSessionByWS, maxPersistedLastSessionByWS)
	m.viewedActivity = cloneViewedActivity(state.ViewedActivity)
}

func (m *shellModel) persistedPinnedSessionRefs() []workbenchSessionRef {
	keys := sortedSessionKeys(m.pinnedSessions)
	if len(keys) > maxPersistedPinnedSessions {
		keys = keys[:maxPersistedPinnedSessions]
	}
	out := make([]workbenchSessionRef, 0, len(keys))
	for _, sid := range keys {
		out = append(out, workbenchSessionRef{WorkspaceID: strings.TrimSpace(m.sessionWorkspace[sid]), SessionID: sid})
	}
	return out
}

func (m *shellModel) persistedRecentSessionRefs() []workbenchSessionRef {
	limit := min(len(m.recentSessions), maxPersistedRecentSessions)
	out := make([]workbenchSessionRef, 0, limit)
	for _, sid := range m.recentSessions {
		if len(out) >= maxPersistedRecentSessions {
			break
		}
		sid = strings.TrimSpace(sid)
		if sid == "" {
			continue
		}
		out = append(out, workbenchSessionRef{WorkspaceID: strings.TrimSpace(m.sessionWorkspace[sid]), SessionID: sid})
	}
	return out
}

func cloneSessionMap(in map[string]string) map[string]string {
	return cloneSessionMapWithLimit(in, 0)
}

func cloneSessionMapWithLimit(in map[string]string, limit int) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if limit > 0 && len(keys) > limit {
		keys = keys[:limit]
	}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		v := in[k]
		out[k] = v
	}
	return out
}

func cloneRecentObjectsWithLimit(in []workbenchObjectRef, limit int) []workbenchObjectRef {
	if len(in) == 0 {
		return nil
	}
	if limit > 0 && len(in) > limit {
		in = in[:limit]
	}
	out := make([]workbenchObjectRef, 0, len(in))
	for _, ref := range in {
		out = append(out, workbenchObjectRef{
			Kind:        strings.TrimSpace(ref.Kind),
			ID:          strings.TrimSpace(ref.ID),
			WorkspaceID: strings.TrimSpace(ref.WorkspaceID),
			SessionID:   strings.TrimSpace(ref.SessionID),
		})
	}
	return out
}

func clampPaneRatio(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v < 0.15 {
		return 0.15
	}
	if v > 0.5 {
		return 0.5
	}
	return v
}

func isZeroWorkbenchState(state workbenchLocalState) bool {
	return state.LastRouteID == "" && state.LastSessionID == "" && len(state.PinnedSessions) == 0 && len(state.RecentSessions) == 0 && len(state.ViewedActivity) == 0 && len(state.RecentObjects) == 0 && state.ThemePreset == "" && strings.TrimSpace(state.LeaderKey) == "" && state.SidebarPaneRatio == 0 && state.InspectorPaneRatio == 0
}
