package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type sessionSwitcherModel struct {
	open          bool
	query         string
	selectedIndex int
	matches       []brokerapi.SessionSummary
	sessions      []brokerapi.SessionSummary
}

func newSessionSwitcherModel() sessionSwitcherModel {
	return sessionSwitcherModel{}
}

func (m sessionSwitcherModel) IsOpen() bool {
	return m.open
}

func (m sessionSwitcherModel) Open(sessions []brokerapi.SessionSummary) sessionSwitcherModel {
	m.open = true
	m.query = ""
	m.selectedIndex = 0
	m.sessions = append([]brokerapi.SessionSummary(nil), sessions...)
	m.rebuildMatches()
	return m
}

func (m sessionSwitcherModel) Close() sessionSwitcherModel {
	m.open = false
	return m
}

func (m sessionSwitcherModel) UpdateSessions(sessions []brokerapi.SessionSummary) sessionSwitcherModel {
	m.sessions = append([]brokerapi.SessionSummary(nil), sessions...)
	m.rebuildMatches()
	return m
}

func (m sessionSwitcherModel) SelectedSessionID() string {
	if len(m.matches) == 0 {
		return ""
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
	if m.selectedIndex >= len(m.matches) {
		m.selectedIndex = len(m.matches) - 1
	}
	return m.matches[m.selectedIndex].Identity.SessionID
}

func (m sessionSwitcherModel) Next() sessionSwitcherModel {
	if len(m.matches) == 0 {
		return m
	}
	m.selectedIndex = (m.selectedIndex + 1) % len(m.matches)
	return m
}

func (m sessionSwitcherModel) Prev() sessionSwitcherModel {
	if len(m.matches) == 0 {
		return m
	}
	m.selectedIndex--
	if m.selectedIndex < 0 {
		m.selectedIndex = len(m.matches) - 1
	}
	return m
}

func (m sessionSwitcherModel) AppendQuery(value string) sessionSwitcherModel {
	if strings.TrimSpace(value) == "" {
		return m
	}
	m.query += value
	m.rebuildMatches()
	return m
}

func (m sessionSwitcherModel) DeleteQueryRune() sessionSwitcherModel {
	if len(m.query) == 0 {
		return m
	}
	_, size := utf8.DecodeLastRuneInString(m.query)
	if size <= 0 {
		return m
	}
	m.query = m.query[:len(m.query)-size]
	m.rebuildMatches()
	return m
}

func (m *sessionSwitcherModel) rebuildMatches() {
	needle := strings.ToLower(strings.TrimSpace(m.query))
	m.matches = m.matches[:0]
	if needle == "" {
		m.matches = append(m.matches, m.sessions...)
	} else {
		for _, s := range m.sessions {
			if strings.Contains(strings.ToLower(s.Identity.SessionID), needle) ||
				strings.Contains(strings.ToLower(s.Identity.WorkspaceID), needle) ||
				strings.Contains(strings.ToLower(s.LastActivityKind), needle) ||
				strings.Contains(strings.ToLower(s.LastActivityPreview), needle) {
				m.matches = append(m.matches, s)
			}
		}
	}
	if m.selectedIndex >= len(m.matches) {
		if len(m.matches) == 0 {
			m.selectedIndex = 0
		} else {
			m.selectedIndex = len(m.matches) - 1
		}
	}
}

func parseTimestamp(ts string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(ts))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func isSessionActivityNewSinceViewed(summary brokerapi.SessionSummary, viewedAt string) bool {
	if strings.TrimSpace(summary.LastActivityAt) == "" {
		return false
	}
	if strings.TrimSpace(viewedAt) == "" {
		return true
	}
	last := parseTimestamp(summary.LastActivityAt)
	viewed := parseTimestamp(viewedAt)
	if last.IsZero() || viewed.IsZero() {
		return strings.TrimSpace(summary.LastActivityAt) != strings.TrimSpace(viewedAt)
	}
	return last.After(viewed)
}

func sessionHighLevelCue(summary brokerapi.SessionSummary) string {
	status := strings.ToLower(strings.TrimSpace(summary.Status))
	if strings.Contains(status, "fail") || strings.Contains(status, "error") {
		return "failed"
	}
	if strings.Contains(status, "degrad") {
		return "degraded"
	}
	if strings.Contains(status, "block") {
		return "blocked"
	}
	if strings.Contains(status, "wait") || strings.Contains(status, "queued") {
		return "waiting"
	}
	if summary.HasIncompleteTurn || strings.Contains(status, "active") || strings.Contains(status, "progress") || strings.Contains(status, "run") {
		return "active"
	}
	return "idle"
}

func sessionDirectoryItems(summaries []brokerapi.SessionSummary, activeSessionID string, pinned map[string]struct{}, recents []string, viewed map[string]string, active shellActivityFocus) []string {
	recentOrder := recentSessionOrder(recents)
	ordered := sortedSessionDirectorySummaries(summaries, recents)
	items := make([]string, 0, len(ordered))
	for _, s := range ordered {
		items = append(items, sessionDirectoryLine(s, activeSessionID, pinned, recentOrder, viewed, active))
	}
	return items
}

func sessionDirectoryLine(summary brokerapi.SessionSummary, activeSessionID string, pinned map[string]struct{}, recentOrder map[string]int, viewed map[string]string, active shellActivityFocus) string {
	sid := summary.Identity.SessionID
	markerText := formatSessionDirectoryMarkers(summary, activeSessionID, pinned, recentOrder, viewed[sid], active)
	return fmt.Sprintf("%s%s | ws=%s | at=%s kind=%s | preview=%q | incomplete=%t cue=%s | runs=%d approvals=%d",
		sid,
		markerText,
		summary.Identity.WorkspaceID,
		defaultPlaceholder(summary.LastActivityAt, "n/a"),
		defaultPlaceholder(summary.LastActivityKind, "n/a"),
		truncateText(summary.LastActivityPreview, 52),
		summary.HasIncompleteTurn,
		sessionHighLevelCue(summary),
		summary.LinkedRunCount,
		summary.LinkedApprovalCount,
	)
}

func recentSessionOrder(recents []string) map[string]int {
	recentOrder := map[string]int{}
	for i, sid := range recents {
		recentOrder[sid] = i
	}
	return recentOrder
}

func sortedSessionDirectorySummaries(summaries []brokerapi.SessionSummary, recents []string) []brokerapi.SessionSummary {
	ordered := append([]brokerapi.SessionSummary(nil), summaries...)
	recentOrder := recentSessionOrder(recents)
	sort.SliceStable(ordered, func(i, j int) bool {
		return sessionSortLess(ordered[i], ordered[j], recentOrder)
	})
	return ordered
}

func sessionSortLess(left, right brokerapi.SessionSummary, recentOrder map[string]int) bool {
	leftAt, rightAt := parseTimestamp(left.LastActivityAt), parseTimestamp(right.LastActivityAt)
	if !leftAt.Equal(rightAt) {
		return leftAt.After(rightAt)
	}
	leftRecent, leftOK := recentOrder[left.Identity.SessionID]
	rightRecent, rightOK := recentOrder[right.Identity.SessionID]
	if leftOK && rightOK && leftRecent != rightRecent {
		return leftRecent < rightRecent
	}
	return left.Identity.SessionID < right.Identity.SessionID
}

func formatSessionDirectoryMarkers(summary brokerapi.SessionSummary, activeSessionID string, pinned map[string]struct{}, recentOrder map[string]int, viewedAt string, active shellActivityFocus) string {
	markers := sessionDirectoryMarkers(summary, activeSessionID, pinned, recentOrder, viewedAt, active)
	if len(markers) == 0 {
		return ""
	}
	return " [" + strings.Join(markers, ",") + "]"
}

func sessionDirectoryMarkers(summary brokerapi.SessionSummary, activeSessionID string, pinned map[string]struct{}, recentOrder map[string]int, viewedAt string, active shellActivityFocus) []string {
	sid := summary.Identity.SessionID
	markers := make([]string, 0, 5)
	if sid == activeSessionID {
		markers = append(markers, "active")
	}
	if _, ok := pinned[sid]; ok {
		markers = append(markers, "pin")
	}
	if _, ok := recentOrder[sid]; ok {
		markers = append(markers, "recent")
	}
	if isSessionActivityNewSinceViewed(summary, viewedAt) {
		markers = append(markers, "new")
	}
	if active.Kind == "session" && strings.TrimSpace(active.ID) != "" && active.ID == sid {
		markers = append(markers, "running")
	}
	return markers
}

func truncateText(text string, maxLen int) string {
	trimmed := strings.TrimSpace(redactSecrets(text))
	if maxLen <= 0 {
		return trimmed
	}
	if utf8.RuneCountInString(trimmed) <= maxLen {
		return trimmed
	}
	if maxLen <= 1 {
		for _, r := range trimmed {
			return string(r)
		}
		return ""
	}
	runes := []rune(trimmed)
	return string(runes[:maxLen-1]) + "…"
}

func defaultPlaceholder(text string, placeholder string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return placeholder
	}
	return trimmed
}
