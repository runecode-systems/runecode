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
		"ws=ws-1",
		"kind=chat_message",
		"preview=\"hello world\"",
		"incomplete=true",
		"cue=active",
		"runs=2",
		"approvals=1",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected %q in %q", want, line)
		}
	}
}

func TestSessionQuickSwitcherIncludesCanonicalSessionMetadata(t *testing.T) {
	m := newShellModel()
	m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: []brokerapi.SessionSummary{{
		Identity:            brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"},
		LastActivityAt:      "2026-01-03T00:00:00Z",
		LastActivityKind:    "run_progress",
		LastActivityPreview: "preview",
		HasIncompleteTurn:   false,
		LinkedRunCount:      1,
		LinkedApprovalCount: 0,
		Status:              "active",
	}}})
	m.sessions = m.sessions.Open(m.sessionItems)
	v := m.renderSessionQuickSwitcher()
	for _, want := range []string{"session-1", "ws=ws-1", "activity=2026-01-03T00:00:00Z/run_progress", "runs=1 approvals=0"} {
		if !strings.Contains(v, want) {
			t.Fatalf("expected %q in switcher view %q", want, v)
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
