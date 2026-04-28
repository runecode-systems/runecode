package brokerapi

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func defaultWorkflowRoutingForTriggerTests() *SessionWorkflowPackRouting {
	return &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "draft_promote_apply"}
}

func seedSessionExecutionTriggerProjectionLinks(t *testing.T, s *Service) {
	t.Helper()
	_ = mustSessionSendMessage(t, s, SessionSendMessageRequest{SchemaID: "runecode.protocol.v0.SessionSendMessageRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-links-msg", SessionID: "sess-trigger-links", Role: "user", ContentText: "link", RelatedLinks: &SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{"run-session-trigger-links"}, ApprovalIDs: []string{digestForBrokerTest("a")}, ArtifactDigests: []string{digestForBrokerTest("b")}, AuditRecordDigests: []string{digestForBrokerTest("c")}}})
}

func requireCurrentSessionExecution(t *testing.T, detail SessionDetail) *SessionTurnExecution {
	t.Helper()
	if detail.CurrentTurnExecution == nil {
		t.Fatal("current_turn_execution missing")
	}
	return detail.CurrentTurnExecution
}

func assertSessionExecutionBindings(t *testing.T, exec *SessionTurnExecution) {
	t.Helper()
	if exec.PrimaryRunID != "run-session-trigger-links" {
		t.Fatalf("primary_run_id = %q, want run-session-trigger-links", exec.PrimaryRunID)
	}
	if exec.BoundValidatedProjectSubstrateDigest == "" {
		t.Fatal("bound_validated_project_substrate_digest is empty")
	}
	if len(exec.LinkedRunIDs) == 0 || len(exec.LinkedApprovalIDs) == 0 || len(exec.LinkedArtifactDigests) == 0 || len(exec.LinkedAuditRecordDigests) == 0 {
		t.Fatalf("execution links missing: %+v", exec)
	}
}

func requireBoundExecutionDigest(t *testing.T, detail SessionDetail) string {
	t.Helper()
	exec := requireCurrentSessionExecution(t, detail)
	if exec.BoundValidatedProjectSubstrateDigest == "" {
		t.Fatal("bound_validated_project_substrate_digest missing after start")
	}
	return exec.BoundValidatedProjectSubstrateDigest
}

func blockSessionPosturePreserved(t *testing.T, blocked SessionDetail) {
	t.Helper()
	if blocked.Summary.WorkPosture != "blocked" {
		t.Fatalf("summary.work_posture = %q, want blocked", blocked.Summary.WorkPosture)
	}
	if blocked.Summary.WorkPostureReasonCode != "project_substrate_digest_drift" {
		t.Fatalf("summary.work_posture_reason_code = %q, want project_substrate_digest_drift", blocked.Summary.WorkPostureReasonCode)
	}
	if blocked.CurrentTurnExecution == nil || blocked.CurrentTurnExecution.ExecutionState != "blocked" {
		t.Fatal("current blocked execution missing after runtime facts refresh")
	}
}

func markSessionExecutionWaiting(t *testing.T, s *Service, turnID, sessionID string) {
	t.Helper()
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: sessionID, TurnID: turnID, ExecutionState: "waiting", WaitKind: "operator_input", WaitState: "waiting_operator_input", OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
}

func markSessionExecutionBlocked(t *testing.T, s *Service, turnID, sessionID string) {
	t.Helper()
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: sessionID, TurnID: turnID, ExecutionState: "blocked", WaitKind: "project_blocked", WaitState: "waiting_project_blocked", BlockedReasonCode: "project_substrate_digest_drift", OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
}

func assertSessionExecutionContinueBlocked(t *testing.T, errResp *ErrorResponse, wantCode string) {
	t.Helper()
	if errResp == nil {
		t.Fatal("expected session execution error")
	}
	if errResp.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q", errResp.Error.Code, wantCode)
	}
}
