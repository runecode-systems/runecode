package main

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestChatExecutionTerminal(t *testing.T) {
	tests := []struct {
		name string
		exec brokerapi.SessionTurnExecution
		want bool
	}{
		{
			name: "terminal when terminal outcome is cancelled",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState:  "waiting",
				TerminalOutcome: "cancelled",
			},
			want: true,
		},
		{
			name: "non terminal for waiting execution state",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState: "waiting",
			},
			want: false,
		},
		{
			name: "non terminal for blocked execution state",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState: "blocked",
			},
			want: false,
		},
		{
			name: "terminal for completed execution state",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState: "completed",
			},
			want: true,
		},
		{
			name: "terminal for failed execution state",
			exec: brokerapi.SessionTurnExecution{
				ExecutionState: "failed",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatExecutionTerminal(tt.exec); got != tt.want {
				t.Fatalf("chatExecutionTerminal(%+v) = %t, want %t", tt.exec, got, tt.want)
			}
		})
	}
}

func TestChatExecutionStateCardReflectsApprovalEvidenceAndStages(t *testing.T) {
	detail := &brokerapi.SessionDetail{
		Summary:                  brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}},
		LinkedRunIDs:             []string{"run-1"},
		LinkedApprovalIDs:        []string{"ap-1"},
		LinkedArtifactDigests:    []string{"sha256:bbbb"},
		LinkedAuditRecordDigests: []string{"sha256:aaaa"},
		CurrentTurnExecution: &brokerapi.SessionTurnExecution{
			ExecutionState:        "waiting",
			WaitKind:              "approval",
			WaitState:             "waiting_approval",
			PrimaryRunID:          "run-1",
			PendingApprovalID:     "ap-1",
			LinkedRunIDs:          []string{"run-1"},
			LinkedApprovalIDs:     []string{"ap-1"},
			LinkedArtifactDigests: []string{"sha256:bbbb"},
		},
	}
	run := &brokerapi.RunDetail{
		Summary:               brokerapi.RunSummary{RunID: "run-1", WorkflowDefinitionHash: "sha256:plan", LifecycleState: "active"},
		PendingApprovalIDs:    []string{"ap-1"},
		ArtifactCountsByClass: map[string]int{"change_draft": 1},
		AuthoritativeState:    map[string]any{"workflow_projection_reason": "plan_authoritative"},
		AdvisoryState:         map[string]any{"runner": "active", "last_checkpoint": map[string]any{"checkpoint_code": "approval_wait_entered"}},
	}
	card := chatExecutionStateCard(detail, nil, run)
	if card.State != routeLoadStateApprovalRequired {
		t.Fatalf("card state = %q, want %q", card.State, routeLoadStateApprovalRequired)
	}
	if !strings.Contains(card.Reason, "Broker-known stages:") || !strings.Contains(card.Reason, "plan compiled") || !strings.Contains(card.Reason, "waiting") {
		t.Fatalf("expected stage summary in reason, got %q", card.Reason)
	}
	if !strings.Contains(card.Reason, "Current broker state: waiting / wait waiting approval") {
		t.Fatalf("expected current/latest execution summary in reason, got %q", card.Reason)
	}
	if !strings.Contains(card.Reason, "Evidence: 1 linked run • 1 approval • 1 artifact • 1 audit record") {
		t.Fatalf("expected evidence summary in reason, got %q", card.Reason)
	}
	if !strings.Contains(card.NextAction, "Approvals") {
		t.Fatalf("expected approvals follow-up, got %q", card.NextAction)
	}
}

func TestChatExecutionStageSummaryOnlyShowsBrokerOwnedStages(t *testing.T) {
	detail := &brokerapi.SessionDetail{
		Summary: brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}},
		CurrentTurnExecution: &brokerapi.SessionTurnExecution{
			ExecutionState: "running",
			PrimaryRunID:   "run-1",
		},
	}
	run := &brokerapi.RunDetail{Summary: brokerapi.RunSummary{RunID: "run-1"}}
	got := chatExecutionStageSummary(*detail.CurrentTurnExecution, detail, run)
	if strings.Contains(got, "run linked") {
		t.Fatalf("did not expect local-only run linked stage, got %q", got)
	}
	if strings.Contains(got, "waiting") {
		t.Fatalf("did not expect waiting stage without broker-owned waiting state, got %q", got)
	}
	if got != "" {
		t.Fatalf("expected no broker-owned stages, got %q", got)
	}
}

func TestChatExecutionStatusAndActionForProjectBlockedUsesStatusRemediation(t *testing.T) {
	exec := brokerapi.SessionTurnExecution{ExecutionState: "blocked", WaitKind: "project_blocked", WaitState: "waiting_project_blocked"}
	status, action := chatExecutionStatusAndAction(exec)
	if status != "Workflow cannot continue yet." {
		t.Fatalf("status = %q", status)
	}
	if !strings.Contains(action, "Open Status") {
		t.Fatalf("expected Status remediation action, got %q", action)
	}
}
