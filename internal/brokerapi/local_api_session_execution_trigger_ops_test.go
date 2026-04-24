package brokerapi

import (
	"context"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func TestSessionExecutionTriggerReturnsTypedAckAndSupportsIdempotency(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger", "sess-trigger")
	baseReq := SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-1", SessionID: "sess-trigger", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "please continue", IdempotencyKey: "idem-trigger-1"}
	ack1 := mustSessionExecutionTrigger(t, s, baseReq)
	if ack1.EventType != "session_execution_trigger_ack" {
		t.Fatalf("event_type = %q, want session_execution_trigger_ack", ack1.EventType)
	}
	if ack1.TriggerID == "" {
		t.Fatal("trigger_id is empty")
	}
	replayReq := baseReq
	replayReq.RequestID = "req-session-trigger-2"
	ack2 := mustSessionExecutionTrigger(t, s, replayReq)
	if ack2.Seq != ack1.Seq {
		t.Fatalf("replay seq = %d, want %d", ack2.Seq, ack1.Seq)
	}
	if ack2.TriggerID != ack1.TriggerID {
		t.Fatalf("replay trigger_id = %q, want %q", ack2.TriggerID, ack1.TriggerID)
	}
}

func TestSessionExecutionTriggerRejectsInvalidTriggerSource(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-invalid", "sess-trigger-invalid")
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-invalid", SessionID: "sess-trigger-invalid", TriggerSource: "invalid", RequestedOperation: "start"}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleSessionExecutionTrigger expected validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestSessionExecutionTriggerFailsClosedWhenProjectSubstrateMissing(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-missing", "sess-trigger-missing")
	s.discoverProjectSubstrateFn = func() (projectsubstrate.DiscoveryResult, error) {
		return projectsubstrate.DiscoveryResult{Compatibility: projectsubstrate.CompatibilityAssessment{Posture: projectsubstrate.CompatibilityPostureMissing, NormalOperationAllowed: false, BlockedReasonCodes: []string{"project_substrate_missing"}}}, nil
	}
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-missing", SessionID: "sess-trigger-missing", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "hello"}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleSessionExecutionTrigger expected blocked posture error")
	}
	if errResp.Error.Code != "project_substrate_operation_blocked" {
		t.Fatalf("error code = %q, want project_substrate_operation_blocked", errResp.Error.Code)
	}
}

func TestSessionExecutionTriggerAllowsDistinctWaitingVocabularyAndControlSeparation(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-controls", "sess-trigger-controls")
	ack := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-controls", SessionID: "sess-trigger-controls", TriggerSource: "interactive_user", RequestedOperation: "start", ApprovalProfile: "moderate", AutonomyPosture: "balanced", UserMessageContentText: "continue"})
	if ack.ApprovalProfile != "moderate" {
		t.Fatalf("approval_profile = %q, want moderate", ack.ApprovalProfile)
	}
	if ack.AutonomyPosture != "balanced" {
		t.Fatalf("autonomy_posture = %q, want balanced", ack.AutonomyPosture)
	}
	if ack.ExecutionState != "running" {
		t.Fatalf("execution_state = %q, want running", ack.ExecutionState)
	}
	getResp := mustSessionGet(t, s, "req-session-trigger-controls-get", "sess-trigger-controls")
	if getResp.Session.CurrentTurnExecution == nil {
		t.Fatal("current_turn_execution missing")
	}
	if getResp.Session.CurrentTurnExecution.WaitState != "" {
		t.Fatalf("wait_state = %q, want empty for running", getResp.Session.CurrentTurnExecution.WaitState)
	}
}

func TestSessionExecutionTriggerEnforcesSingleActiveTurnExecution(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-active", "sess-trigger-active")
	_ = mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-active-1", SessionID: "sess-trigger-active", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "first"})
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-active-2", SessionID: "sess-trigger-active", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "second"}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleSessionExecutionTrigger expected active turn conflict")
	}
	if errResp.Error.Code != "broker_session_execution_active_turn_exists" {
		t.Fatalf("error code = %q, want broker_session_execution_active_turn_exists", errResp.Error.Code)
	}
}

func TestSessionExecutionTriggerProjectsSessionRunAndSnapshotBindings(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-links", "sess-trigger-links")
	seedSessionExecutionTriggerProjectionLinks(t, s)
	ack := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-links-trigger", SessionID: "sess-trigger-links", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "go"})
	if ack.TurnID == "" {
		t.Fatal("turn_id is empty")
	}
	getResp := mustSessionGet(t, s, "req-session-trigger-links-get", "sess-trigger-links")
	exec := requireCurrentSessionExecution(t, getResp.Session)
	assertSessionExecutionBindings(t, exec)
}

func TestSessionExecutionTriggerContinueFailsClosedOnDigestDriftAndProjectsBlockedTurn(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-drift", "sess-trigger-drift")
	_ = mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-drift-start", SessionID: "sess-trigger-drift", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "start"})
	bound := requireBoundExecutionDigest(t, mustSessionGet(t, s, "req-session-trigger-drift-get-start", "sess-trigger-drift").Session)
	driftDigest := digestForBrokerTest("session-trigger-drift")
	if driftDigest == bound {
		t.Fatal("test setup expected drift digest to differ from bound digest")
	}
	s.discoverProjectSubstrateFn = func() (projectsubstrate.DiscoveryResult, error) {
		return projectsubstrate.DiscoveryResult{Snapshot: projectsubstrate.ValidationSnapshot{ValidatedSnapshotDigest: driftDigest, ProjectContextIdentityDigest: driftDigest}, Compatibility: projectsubstrate.CompatibilityAssessment{Posture: projectsubstrate.CompatibilityPostureSupportedCurrent, NormalOperationAllowed: true}}, nil
	}
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-drift-continue", SessionID: "sess-trigger-drift", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue"}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleSessionExecutionTrigger expected drift blocked error")
	}
	if errResp.Error.Code != "broker_session_execution_project_context_drift" {
		t.Fatalf("error code = %q, want broker_session_execution_project_context_drift", errResp.Error.Code)
	}
	assertSessionExecutionBlockedProjection(t, mustSessionGet(t, s, "req-session-trigger-drift-get-blocked", "sess-trigger-drift").Session, "project_substrate_digest_drift")
}

func TestSessionRuntimeFactsDoNotOverwriteBlockedSessionPosture(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-blocked-preserve", "sess-blocked-preserve")
	_ = mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-preserve-start", SessionID: "sess-blocked-preserve", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "start"})
	bound := requireBoundExecutionDigest(t, mustSessionGet(t, s, "req-session-blocked-preserve-get-start", "sess-blocked-preserve").Session)
	driftDigest := digestForBrokerTest("session-blocked-preserve-drift")
	if driftDigest == bound {
		t.Fatal("test setup expected drift digest to differ from bound digest")
	}
	s.discoverProjectSubstrateFn = func() (projectsubstrate.DiscoveryResult, error) {
		return projectsubstrate.DiscoveryResult{Snapshot: projectsubstrate.ValidationSnapshot{ValidatedSnapshotDigest: driftDigest, ProjectContextIdentityDigest: driftDigest}, Compatibility: projectsubstrate.CompatibilityAssessment{Posture: projectsubstrate.CompatibilityPostureSupportedCurrent, NormalOperationAllowed: true}}, nil
	}
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-preserve-continue", SessionID: "sess-blocked-preserve", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue"}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleSessionExecutionTrigger expected drift blocked error")
	}
	if err := s.RecordRuntimeFacts("run-session-blocked-preserve", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-session-blocked-preserve", SessionID: "sess-blocked-preserve"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	blockSessionPosturePreserved(t, mustSessionGet(t, s, "req-session-blocked-preserve-get-blocked", "sess-blocked-preserve").Session)
}

func TestSessionExecutionTriggerContinueRequiresValidatedSnapshotDigest(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-continue-digest", "sess-continue-digest")
	start := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-continue-digest-start", SessionID: "sess-continue-digest", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "start"})
	markSessionExecutionWaiting(t, s, start.TurnID, "sess-continue-digest")
	s.discoverProjectSubstrateFn = func() (projectsubstrate.DiscoveryResult, error) {
		return projectsubstrate.DiscoveryResult{Snapshot: projectsubstrate.ValidationSnapshot{}, Compatibility: projectsubstrate.CompatibilityAssessment{Posture: projectsubstrate.CompatibilityPostureSupportedCurrent, NormalOperationAllowed: true}}, nil
	}
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-continue-digest-continue", SessionID: "sess-continue-digest", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "project_substrate_operation_blocked")
}

func TestSessionExecutionTriggerContinueRejectsBlockedTurnResume(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-blocked-resume", "sess-blocked-resume")
	start := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-resume-start", SessionID: "sess-blocked-resume", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "start"})
	markSessionExecutionBlocked(t, s, start.TurnID, "sess-blocked-resume")
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-resume-continue", SessionID: "sess-blocked-resume", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_session_execution_continue_missing_execution")
}

func TestSessionExecutionTriggerContinueFailsClosedWithoutResumableExecution(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-resume-missing", "sess-trigger-resume-missing")
	startAck := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-resume-missing-start", SessionID: "sess-trigger-resume-missing", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "start"})
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: "sess-trigger-resume-missing", TurnID: startAck.TurnID, ExecutionState: "completed", OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-resume-missing-continue", SessionID: "sess-trigger-resume-missing", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_session_execution_continue_missing_execution")
}

func TestBlockedProjectPostureAllowsSessionInspectionButBlocksSessionExecutionTrigger(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-blocked-posture", "sess-trigger-blocked-posture")
	s.discoverProjectSubstrateFn = func() (projectsubstrate.DiscoveryResult, error) {
		return projectsubstrate.DiscoveryResult{Compatibility: projectsubstrate.CompatibilityAssessment{Posture: projectsubstrate.CompatibilityPostureMissing, NormalOperationAllowed: false, BlockedReasonCodes: []string{"project_substrate_missing"}}}, nil
	}
	listResp, listErr := s.HandleSessionList(context.Background(), SessionListRequest{SchemaID: "runecode.protocol.v0.SessionListRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-list", Limit: 10}, RequestContext{})
	if listErr != nil {
		t.Fatalf("HandleSessionList returned error in blocked posture: %+v", listErr)
	}
	requireSingleSessionSummary(t, listResp, "sess-trigger-blocked-posture")
	_, getErr := s.HandleSessionGet(context.Background(), SessionGetRequest{SchemaID: "runecode.protocol.v0.SessionGetRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-get", SessionID: "sess-trigger-blocked-posture"}, RequestContext{})
	if getErr != nil {
		t.Fatalf("HandleSessionGet returned error in blocked posture: %+v", getErr)
	}
	_, triggerErr := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-trigger", SessionID: "sess-trigger-blocked-posture", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "blocked"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, triggerErr, "project_substrate_operation_blocked")
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
