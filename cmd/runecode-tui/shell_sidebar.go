package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type sidebarEntryKind string

const (
	sidebarEntryRoute   sidebarEntryKind = "route"
	sidebarEntrySession sidebarEntryKind = "session"
)

type sidebarEntry struct {
	Kind    sidebarEntryKind
	Route   routeDefinition
	Session brokerapi.SessionSummary
}

func (m shellModel) sidebarEntries() []sidebarEntry {
	entries := make([]sidebarEntry, 0, len(m.routes)+len(m.sessionItems))
	for _, route := range m.routes {
		entries = append(entries, sidebarEntry{Kind: sidebarEntryRoute, Route: route})
	}
	if m.sessionLoading || strings.TrimSpace(m.sessionLoadError) != "" {
		return entries
	}
	for _, session := range sortedSessionDirectorySummaries(m.sessionItems, m.recentSessions) {
		entries = append(entries, sidebarEntry{Kind: sidebarEntrySession, Session: session})
	}
	return entries
}

func (m *shellModel) normalizeSidebarCursor() {
	entries := m.sidebarEntries()
	if len(entries) == 0 {
		m.sidebarCursor = 0
		return
	}
	if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
	if m.sidebarCursor >= len(entries) {
		m.sidebarCursor = len(entries) - 1
	}
}

func (m *shellModel) moveSidebarCursor(delta int) {
	entries := m.sidebarEntries()
	if len(entries) == 0 {
		m.sidebarCursor = 0
		return
	}
	if delta >= 0 {
		m.sidebarCursor = (m.sidebarCursor + 1) % len(entries)
		return
	}
	m.sidebarCursor--
	if m.sidebarCursor < 0 {
		m.sidebarCursor = len(entries) - 1
	}
}

func (m shellModel) selectedSidebarEntry() (sidebarEntry, bool) {
	entries := m.sidebarEntries()
	if len(entries) == 0 {
		return sidebarEntry{}, false
	}
	idx := m.sidebarCursor
	if idx < 0 {
		idx = 0
	}
	if idx >= len(entries) {
		idx = len(entries) - 1
	}
	return entries[idx], true
}

func (m *shellModel) syncSidebarCursorToLocation() {
	entries := m.sidebarEntries()
	if len(entries) == 0 {
		m.sidebarCursor = 0
		return
	}
	currentRoute := m.currentRouteID()
	for i, entry := range entries {
		switch entry.Kind {
		case sidebarEntryRoute:
			if entry.Route.ID == currentRoute {
				m.sidebarCursor = i
				return
			}
		}
	}
	m.normalizeSidebarCursor()
}

func (m *shellModel) syncSidebarCursorToSessionID(sessionID string) {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return
	}
	entries := m.sidebarEntries()
	for i, entry := range entries {
		if entry.Kind != sidebarEntrySession {
			continue
		}
		if strings.TrimSpace(entry.Session.Identity.SessionID) == sid {
			m.sidebarCursor = i
			return
		}
	}
}

func (m shellModel) normalizedSidebarCursor(entries []sidebarEntry) int {
	if len(entries) == 0 {
		return 0
	}
	cursor := m.sidebarCursor
	if cursor < 0 {
		return 0
	}
	if cursor >= len(entries) {
		return len(entries) - 1
	}
	return cursor
}

func (m shellModel) appendSidebarRouteLines(lines []string, cursor int, width int) []string {
	for i, r := range m.routes {
		selected := i == cursor
		active := r.ID == m.currentRouteID()
		marker := " "
		if selected {
			marker = ">"
		} else if active {
			marker = "*"
		}
		lines = append(lines, renderSelectableRow(fmt.Sprintf("%s %d %s", marker, r.Index, r.Label), width, selected, active && !selected))
	}
	return lines
}

func (m shellModel) appendSidebarSessionLines(lines []string, entries []sidebarEntry, cursor int, width int) []string {
	if m.sessionLoading {
		return append(lines, "", tableHeader("Sessions"), "  loading canonical session directory...")
	}
	if strings.TrimSpace(m.sessionLoadError) != "" {
		return append(lines, "", tableHeader("Sessions"), "  load failed: "+m.sessionLoadError)
	}
	lines = append(lines, "", tableHeader("Sessions"))
	recentOrder := recentSessionOrder(m.recentSessions)
	for i, entry := range entries {
		if entry.Kind != sidebarEntrySession {
			continue
		}
		selected := i == cursor
		item := sessionDirectoryLine(entry.Session, m.activeSessionID, m.pinnedSessions, recentOrder, m.viewedActivity, m.watch.projection.Activity.Active)
		marker := " "
		if selected {
			marker = ">"
		}
		active := strings.TrimSpace(entry.Session.Identity.SessionID) != "" && strings.TrimSpace(entry.Session.Identity.SessionID) == strings.TrimSpace(m.activeSessionID)
		lines = append(lines, renderSelectableRow(fmt.Sprintf("%s %s", marker, item), width, selected, active && !selected))
	}
	return lines
}
