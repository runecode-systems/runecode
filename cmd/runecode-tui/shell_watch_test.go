package main

import (
	"errors"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestShellWatchManagerFamilySpecificFailureProjectsDegradedHealth(t *testing.T) {
	manager := newShellWatchManager()
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	manager.applyTransport(shellWatchTransportLoadedMsg{
		ObservedAt: now,
		Run: shellWatchRunTransportResult{
			Err: errors.New("watch_timeout"),
		},
		Approval: shellWatchApprovalTransportResult{
			Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &brokerapi.ApprovalSummary{ApprovalID: "ap-1"}}},
		},
		Session: shellWatchSessionTransportResult{
			Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1"}}}},
		},
	})

	health := manager.projection.Health
	if health.State != shellSyncStateDegraded {
		t.Fatalf("expected degraded health, got %s", health.State)
	}
	if len(health.DegradedFamilies) != 1 || health.DegradedFamilies[0] != "run_watch" {
		t.Fatalf("expected run_watch degraded family, got %+v", health.DegradedFamilies)
	}
	if manager.projection.Live.runWatch.lastStatus != "watch_error" {
		t.Fatalf("expected run watch fallback status, got %q", manager.projection.Live.runWatch.lastStatus)
	}
	if manager.projection.Live.approvalWatch.lastStatus != "ok" {
		t.Fatalf("expected approval watch ok status, got %q", manager.projection.Live.approvalWatch.lastStatus)
	}
}

func TestShellWatchManagerHealthTransitionsAndBackoff(t *testing.T) {
	manager := newShellWatchManager()
	now := time.Date(2026, 4, 16, 13, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	manager.applyTransport(shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Err: errors.New("rpc_failed")}})
	if manager.families[shellWatchFamilyRun].StreamState != shellWatchStreamDegraded {
		t.Fatalf("expected first non-dial failure to be degraded, got %s", manager.families[shellWatchFamilyRun].StreamState)
	}

	now = now.Add(1 * time.Second)
	manager.applyTransport(shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Err: errors.New("rpc_failed")}})
	if manager.families[shellWatchFamilyRun].StreamState != shellWatchStreamReconnecting {
		t.Fatalf("expected repeated non-dial failure to reconnect, got %s", manager.families[shellWatchFamilyRun].StreamState)
	}
	if manager.nextPollDelay() < shellWatchRetryMinDelay {
		t.Fatalf("expected next poll delay >= min retry delay, got %s", manager.nextPollDelay())
	}

	now = now.Add(2 * time.Second)
	manager.applyTransport(shellWatchTransportLoadedMsg{
		ObservedAt: now,
		Run:        shellWatchRunTransportResult{Err: errors.New("local_ipc_dial_error")},
		Approval:   shellWatchApprovalTransportResult{Err: errors.New("local_ipc_dial_error")},
		Session:    shellWatchSessionTransportResult{Err: errors.New("local_ipc_dial_error")},
	})
	if manager.projection.Health.State != shellSyncStateDisconnected {
		t.Fatalf("expected disconnected health, got %s", manager.projection.Health.State)
	}
}

func TestShellWatchManagerProjectsActivityFromTypedFamilyReduction(t *testing.T) {
	manager := newShellWatchManager()
	now := time.Date(2026, 4, 16, 14, 0, 0, 0, time.UTC)

	manager.applyTransport(shellWatchTransportLoadedMsg{
		ObservedAt: now,
		Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{
			{EventType: "run_watch_snapshot", Seq: 1, Run: &brokerapi.RunSummary{RunID: "run-1", LifecycleState: "active"}},
			{EventType: "run_watch_upsert", Seq: 2, Run: &brokerapi.RunSummary{RunID: "run-1", LifecycleState: "active"}},
		}},
		Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{
			{EventType: "approval_watch_snapshot", Seq: 1, Approval: &brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "completed"}},
		}},
		Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{
			{EventType: "session_watch_snapshot", Seq: 1, Session: &brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1"}, Status: "idle"}},
		}},
	})

	if manager.projection.Activity.State != shellActivityStateRunning {
		t.Fatalf("expected running activity state, got %s", manager.projection.Activity.State)
	}
	if manager.projection.Activity.Active.Kind != "run" || manager.projection.Activity.Active.ID != "run-1" {
		t.Fatalf("expected active run focus run-1, got %+v", manager.projection.Activity.Active)
	}
	if len(manager.projection.Feed) == 0 {
		t.Fatal("expected non-empty projected feed")
	}
	if manager.reduction.runs["run-1"].RunID != "run-1" {
		t.Fatal("expected run cache populated from typed reduction")
	}
}
