package main

import (
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
