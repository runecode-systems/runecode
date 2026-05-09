package main

import (
	"strings"
	"testing"
)

func TestShellBottomStripSelectionHintUsesCtrlT(t *testing.T) {
	m := newShellModel()
	m.width = 150
	v := m.View()
	if !strings.Contains(v, "Broker-owned truth stays in Status, inspectors, and command discovery") {
		t.Fatalf("expected calm diagnostic hint in bottom strip, got %q", v)
	}
}

func TestShellChromeShowsCalmWorkbenchSummary(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	m.activeSessionID = "session-1"
	m.watch.projection.Activity = shellActivitySemantics{State: shellActivityStateRunning, Active: shellActivityFocus{Kind: "session", ID: "session-1"}}

	v := m.View()
	for _, want := range []string{"ROUTE", "Sidebar focus", "Working on session session-1", "Action Center", "Approvals", "Commands ctrl+p/:", "Quick action: Quit RuneCode"} {
		if !strings.Contains(v, want) {
			t.Fatalf("expected calm chrome cue %q in view, got %q", want, v)
		}
	}
	for _, retired := range []string{"Local broker API only via broker local IPC; OS peer auth is broker-enforced where supported", "Next actions:", "No route composer or status actions for this screen."} {
		if strings.Contains(v, retired) {
			t.Fatalf("did not expect dense or retired chrome text %q in view, got %q", retired, v)
		}
	}
}
