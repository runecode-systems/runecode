package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestSessionDirectoryItemsRenderRequiredMetadataAndLocalMarkers(t *testing.T) {
	sessions := []brokerapi.SessionSummary{
		{
			Identity:            brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"},
			LastActivityAt:      "2026-01-03T00:00:00Z",
			LastActivityKind:    "chat_message",
			LastActivityPreview: "hello world",
			HasIncompleteTurn:   true,
			LinkedRunCount:      2,
			LinkedApprovalCount: 1,
			Status:              "active",
		},
	}
	pinned := map[string]struct{}{"session-1": {}}
	recents := []string{"session-1"}
	viewed := map[string]string{"session-1": "2026-01-02T00:00:00Z"}
	items := sessionDirectoryItems(sessions, "session-1", pinned, recents, viewed, shellActivityFocus{Kind: "session", ID: "session-1"})
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	line := items[0]
	for _, want := range []string{
		"session-1",
		"[active,pin,recent,new,running]",
		"workspace ws-1",
		"active",
		"2 run(s)",
		"1 approval(s)",
		"hello world",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected %q in %q", want, line)
		}
	}
}

func TestSessionSidebarLineSanitizesRenderedSessionID(t *testing.T) {
	rawSessionID := " session-1\nspoof\r\x1b[31m"
	line := sessionSidebarLine(
		brokerapi.SessionSummary{
			Identity:       brokerapi.SessionIdentity{SessionID: rawSessionID},
			Status:         "active",
			LastActivityAt: "2026-01-03T00:00:00Z",
		},
		rawSessionID,
		map[string]struct{}{rawSessionID: {}},
		map[string]int{rawSessionID: 0},
		map[string]string{rawSessionID: ""},
		shellActivityFocus{Kind: "session", ID: rawSessionID},
	)

	if !strings.Contains(line, sanitizeUIText(rawSessionID)) {
		t.Fatalf("expected sanitized session ID in %q", line)
	}
	for _, unsafe := range []string{"\n", "\r", "\x1b"} {
		if strings.Contains(line, unsafe) {
			t.Fatalf("expected sidebar line to exclude %q, got %q", unsafe, line)
		}
	}
	for _, want := range []string{"active", "pinned", "new", "live"} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected %q marker in %q", want, line)
		}
	}
}

func TestSessionDirectoryLineUsesRawSessionIDForViewedLookup(t *testing.T) {
	rawSessionID := "session-1[31m"
	lastViewed := "2026-01-03T00:00:00Z"
	line := sessionDirectoryLine(
		brokerapi.SessionSummary{
			Identity:       brokerapi.SessionIdentity{SessionID: rawSessionID, WorkspaceID: "ws-1"},
			LastActivityAt: lastViewed,
			Status:         "active",
		},
		"",
		nil,
		nil,
		map[string]string{rawSessionID: lastViewed},
		shellActivityFocus{},
	)

	if strings.Contains(line, "new") {
		t.Fatalf("expected viewed state to use raw session ID and suppress new marker, got %q", line)
	}
	if !strings.Contains(line, sanitizeUIText(rawSessionID)) {
		t.Fatalf("expected sanitized session ID in %q", line)
	}
}

func TestSessionQuickSwitcherIncludesCanonicalSessionMetadata(t *testing.T) {
	m := newShellModel()
	rawSessionID := " session-1\nspoof\r\x1b[31m"
	rawWorkspaceID := " ws-1\nspoof\r\x1b[31m"
	rawActivityKind := " token=secret123\nrun_progress\x00"
	m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: []brokerapi.SessionSummary{{
		Identity:            brokerapi.SessionIdentity{SessionID: rawSessionID, WorkspaceID: rawWorkspaceID},
		LastActivityAt:      "2026-01-03T00:00:00Z",
		LastActivityKind:    rawActivityKind,
		LastActivityPreview: "preview",
		HasIncompleteTurn:   false,
		LinkedRunCount:      1,
		LinkedApprovalCount: 0,
		Status:              "active",
	}}})
	m.watch.projection.Activity.Active = shellActivityFocus{Kind: "session", ID: rawSessionID}
	m.sessions = m.sessions.Open(m.sessionItems)
	v := m.renderSessionQuickSwitcher()
	for _, want := range []string{"● " + sanitizeUIText(rawSessionID), "Workspace " + sanitizeUIText(rawWorkspaceID), "Recent activity " + sanitizeUIText(rawActivityKind), "1 run"} {
		if !strings.Contains(v, want) {
			t.Fatalf("expected %q in switcher view %q", want, v)
		}
	}
	for _, unsafe := range []string{"\nspoof", "\r", "\x1b", "secret123"} {
		if strings.Contains(v, unsafe) {
			t.Fatalf("expected quick switcher view to exclude %q, got %q", unsafe, v)
		}
	}
}

func TestSessionQuickSwitcherBoundsMatchListWithGapMarkers(t *testing.T) {
	m := newShellModel()
	m.width = 90
	sessions := make([]brokerapi.SessionSummary, 0, 10)
	for i := 0; i < 10; i++ {
		sessions = append(sessions, brokerapi.SessionSummary{
			Identity:            brokerapi.SessionIdentity{SessionID: fmt.Sprintf("session-%d", i), WorkspaceID: "ws-1"},
			LastActivityAt:      "2026-01-03T00:00:00Z",
			LastActivityKind:    "chat_message",
			LastActivityPreview: "preview",
			HasIncompleteTurn:   false,
		})
	}
	m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: sessions})
	m.sessions = m.sessions.Open(m.sessionItems)
	m.sessions.selectedIndex = 5

	v := m.renderSessionQuickSwitcher()
	if strings.Count(v, "\n...\n") < 1 {
		t.Fatalf("expected bounded match gap marker in quick switcher view, got %q", v)
	}
	if strings.Contains(v, "session-0") && strings.Contains(v, "session-9") {
		t.Fatalf("expected bounded render to omit at least one edge row, got %q", v)
	}
}

func TestSessionSwitcherFilteringUsesCachedNormalizedSessionText(t *testing.T) {
	sessions := []brokerapi.SessionSummary{
		{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, LastActivityKind: "chat_message", LastActivityPreview: "alpha preview"},
		{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-2"}, LastActivityKind: "run_progress", LastActivityPreview: "Needs Follow-Up"},
	}
	m := newSessionSwitcherModel().Open(sessions)
	if got := len(m.normalizedSessions); got != len(sessions) {
		t.Fatalf("expected %d cached normalized sessions, got %d", len(sessions), got)
	}

	m.query = "WS-2"
	m.rebuildMatches()
	if len(m.matches) != 1 || m.matches[0].Identity.SessionID != "session-2" {
		t.Fatalf("expected workspace match for session-2, got %+v", m.matches)
	}

	m.query = "FOLLOW-UP"
	m.rebuildMatches()
	if len(m.matches) != 1 || m.matches[0].Identity.SessionID != "session-2" {
		t.Fatalf("expected preview match for session-2, got %+v", m.matches)
	}
}
