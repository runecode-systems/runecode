package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
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
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	count := 0
	for _, event := range events {
		if event.Type == "session_execution_trigger_submitted" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("session_execution_trigger_submitted events = %d, want 1", count)
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

func TestSessionExecutionTriggerStartCreatesSessionAndBrokerOwnedRunBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ack := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-create-start", SessionID: "sess-trigger-create", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "create and run"})
	if ack.TurnID == "" {
		t.Fatal("turn_id is empty")
	}
	getResp := mustSessionGet(t, s, "req-session-trigger-create-get", "sess-trigger-create")
	if getResp.Session.Summary.Identity.CreatedByRunID == "" {
		t.Fatal("created_by_run_id is empty")
	}
	exec := requireCurrentSessionExecution(t, getResp.Session)
	if exec.PrimaryRunID == "" {
		t.Fatal("primary_run_id is empty")
	}
	if exec.PrimaryRunID != getResp.Session.Summary.Identity.CreatedByRunID {
		t.Fatalf("primary_run_id = %q, want created_by_run_id %q", exec.PrimaryRunID, getResp.Session.Summary.Identity.CreatedByRunID)
	}
}

func TestSessionExecutionTriggerFailsClosedOnOverlappingMutationBearingStarts(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-active", "sess-trigger-active")
	first := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-active-1", SessionID: "sess-trigger-active", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "first"})
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-active-2", SessionID: "sess-trigger-active", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "second"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_session_execution_overlap_blocked")
	getResp := mustSessionGet(t, s, "req-session-trigger-active-get", "sess-trigger-active")
	if len(getResp.Session.PendingTurnExecutions) != 1 {
		t.Fatalf("pending_turn_executions len = %d, want 1", len(getResp.Session.PendingTurnExecutions))
	}
	if got := getResp.Session.PendingTurnExecutions[0].TurnID; got != first.TurnID {
		t.Fatalf("remaining pending turn_id = %q, want %q", got, first.TurnID)
	}
}

func TestSessionExecutionTriggerSharesStartSurfaceAcrossAutonomousAndInteractivePaths(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-multiwait", "sess-trigger-multiwait")
	first := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-multiwait-1", SessionID: "sess-trigger-multiwait", TriggerSource: "autonomous_background", RequestedOperation: "start", AutonomyPosture: "operator_guided", UserMessageContentText: "first"})
	if first.ExecutionState != "waiting" {
		t.Fatalf("autonomous execution_state = %q, want waiting", first.ExecutionState)
	}
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-multiwait-2", SessionID: "sess-trigger-multiwait", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "second"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_session_execution_overlap_blocked")
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
	resp, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-resume-continue", SessionID: "sess-blocked-resume", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionExecutionTrigger returned error: %+v", errResp)
	}
	if resp.ExecutionState != "running" {
		t.Fatalf("execution_state = %q, want running", resp.ExecutionState)
	}
}

func TestSessionExecutionTriggerAutonomousOperatorGuidedStartsWaitingForOperatorInput(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-autonomous", "sess-trigger-autonomous")
	ack := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-autonomous", SessionID: "sess-trigger-autonomous", TriggerSource: "autonomous_background", RequestedOperation: "start", AutonomyPosture: "operator_guided", UserMessageContentText: "background step"})
	if ack.ExecutionState != "waiting" {
		t.Fatalf("execution_state = %q, want waiting", ack.ExecutionState)
	}
	getResp := mustSessionGet(t, s, "req-session-trigger-autonomous-get", "sess-trigger-autonomous")
	if getResp.Session.CurrentTurnExecution == nil {
		t.Fatal("current_turn_execution missing")
	}
	if getResp.Session.CurrentTurnExecution.WaitKind != "operator_input" {
		t.Fatalf("wait_kind = %q, want operator_input", getResp.Session.CurrentTurnExecution.WaitKind)
	}
	if getResp.Session.CurrentTurnExecution.WaitState != "waiting_operator_input" {
		t.Fatalf("wait_state = %q, want waiting_operator_input", getResp.Session.CurrentTurnExecution.WaitState)
	}
}

func TestSessionExecutionTriggerContinueRejectsWaitingApprovalUntilApprovalResolves(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID, policyDecisionHash, storedApproval := prepareSessionExecutionApprovalFixture(t, s, requestEnv)
	seedSessionRuntimeFactsForOpsTest(t, s, "run-approval", "sess-trigger-waiting-approval")
	ack := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-waiting-approval-start", SessionID: "sess-trigger-waiting-approval", TriggerSource: "autonomous_background", RequestedOperation: "start", AutonomyPosture: "balanced", UserMessageContentText: "background step"})
	if ack.ExecutionState != "running" {
		t.Fatalf("execution_state = %q, want running", ack.ExecutionState)
	}
	recordAndAssertApprovalWait(t, s, approvalID, storedApproval.ActionRequestHash)
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-waiting-approval-continue-blocked", SessionID: "sess-trigger-waiting-approval", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_session_execution_continue_waiting_approval")
	resolveSessionExecutionApprovalWait(t, s, approvalID, policyDecisionHash, unapproved.Digest, requestEnv, decisionEnv)
	resp, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-waiting-approval-continue-resolved", SessionID: "sess-trigger-waiting-approval", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionExecutionTrigger returned error: %+v", errResp)
	}
	if resp.ExecutionState != "running" {
		t.Fatalf("execution_state = %q, want running", resp.ExecutionState)
	}
}

func prepareSessionExecutionApprovalFixture(t *testing.T, s *Service, requestEnv *trustpolicy.SignedObjectEnvelope) (string, string, artifacts.ApprovalRecord) {
	t.Helper()
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	return approvalID, policyDecisionHashForStoredApproval(t, s, approvalID), mustApprovalGet(t, s, approvalID)
}

func recordAndAssertApprovalWait(t *testing.T, s *Service, approvalID, actionHash string) {
	t.Helper()
	if err := s.RecordRunnerApprovalWait(artifacts.RunnerApproval{ApprovalID: approvalID, RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", RoleInstanceID: "role-1", Status: "pending", ApprovalType: "exact_action", BoundActionHash: actionHash, OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("RecordRunnerApprovalWait returned error: %v", err)
	}
	if err := s.syncSessionExecutionForRun("run-approval", s.currentTimestamp()); err != nil {
		t.Fatalf("syncSessionExecutionForRun returned error: %v", err)
	}
	getResp := mustSessionGet(t, s, "req-session-trigger-waiting-approval-get", "sess-trigger-waiting-approval")
	exec := requireCurrentSessionExecution(t, getResp.Session)
	if exec.WaitKind != "approval" {
		t.Fatalf("wait_kind = %q, want approval", exec.WaitKind)
	}
	if exec.WaitState != "waiting_approval" {
		t.Fatalf("wait_state = %q, want waiting_approval", exec.WaitState)
	}
	if exec.PendingApprovalID != approvalID {
		t.Fatalf("pending_approval_id = %q, want %q", exec.PendingApprovalID, approvalID)
	}
}

func resolveSessionExecutionApprovalWait(t *testing.T, s *Service, approvalID, policyDecisionHash, unapprovedDigest string, requestEnv, decisionEnv *trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-waiting-approval-resolve", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: policyengine.ActionKindPromotion, PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapprovedDigest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	if _, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{}); errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	resolved := mustSessionGet(t, s, "req-session-trigger-waiting-approval-get-resolved", "sess-trigger-waiting-approval")
	resolvedExec := requireCurrentSessionExecution(t, resolved.Session)
	if resolvedExec.WaitKind != "" {
		t.Fatalf("wait_kind after resolve = %q, want empty", resolvedExec.WaitKind)
	}
}

func TestSessionExecutionTriggerContinueTargetsExplicitTurn(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-targeted", "sess-trigger-targeted")
	first := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-targeted-1", SessionID: "sess-trigger-targeted", TriggerSource: "autonomous_background", RequestedOperation: "start", AutonomyPosture: "operator_guided", UserMessageContentText: "first"})
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: "sess-trigger-targeted", TurnID: first.TurnID, ExecutionState: "waiting", WaitKind: "external_dependency", WaitState: "waiting_external_dependency", OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
	resp, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-targeted-continue", SessionID: "sess-trigger-targeted", TurnID: first.TurnID, TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue first"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionExecutionTrigger returned error: %+v", errResp)
	}
	if resp.TurnID != first.TurnID {
		t.Fatalf("continued turn_id = %q, want %q", resp.TurnID, first.TurnID)
	}
	getResp := mustSessionGet(t, s, "req-session-trigger-targeted-get", "sess-trigger-targeted")
	if len(getResp.Session.PendingTurnExecutions) != 1 {
		t.Fatalf("pending_turn_executions len = %d, want 1", len(getResp.Session.PendingTurnExecutions))
	}
	if state := getResp.Session.PendingTurnExecutions[0].ExecutionState; state != "running" {
		t.Fatalf("execution_state = %q, want running", state)
	}
}

func TestSessionExecutionTriggerContinueSupportsIdempotentRetry(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-continue-idem", "sess-trigger-continue-idem")
	start := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-continue-idem-start", SessionID: "sess-trigger-continue-idem", TriggerSource: "autonomous_background", RequestedOperation: "start", AutonomyPosture: "operator_guided", UserMessageContentText: "wait first"})
	firstResp := mustSessionExecutionContinue(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-continue-idem-1", SessionID: "sess-trigger-continue-idem", TurnID: start.TurnID, TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue", IdempotencyKey: "idem-continue-1"})
	assertStoredSessionExecutionTriggerIdempotencyRecord(t, s, "sess-trigger-continue-idem", "idem-continue-1", firstResp)
	secondResp := mustSessionExecutionContinue(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-continue-idem-2", SessionID: "sess-trigger-continue-idem", TriggerSource: "resume_follow_up", RequestedOperation: "continue", UserMessageContentText: "continue", IdempotencyKey: "idem-continue-1"})
	assertSessionExecutionTriggerReplayResponse(t, secondResp, firstResp)
}

func mustSessionExecutionContinue(t *testing.T, s *Service, req SessionExecutionTriggerRequest) SessionExecutionTriggerResponse {
	t.Helper()
	resp, errResp := s.HandleSessionExecutionTrigger(context.Background(), req, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleSessionExecutionTrigger returned error: %+v", errResp)
	}
	return resp
}

func assertStoredSessionExecutionTriggerIdempotencyRecord(t *testing.T, s *Service, sessionID, key string, resp SessionExecutionTriggerResponse) {
	t.Helper()
	if resp.TriggerID == "" {
		t.Fatal("trigger_id is empty")
	}
	state, ok := s.SessionState(sessionID)
	if !ok {
		t.Fatal("SessionState missing")
	}
	record, ok := state.ExecutionTriggerIdempotencyByKey[key]
	if !ok {
		t.Fatal("continue idempotency record missing")
	}
	if record.TriggerID != resp.TriggerID {
		t.Fatalf("stored trigger_id = %q, want %q", record.TriggerID, resp.TriggerID)
	}
	if record.TurnID != resp.TurnID {
		t.Fatalf("stored turn_id = %q, want %q", record.TurnID, resp.TurnID)
	}
	if record.Seq != resp.Seq {
		t.Fatalf("stored seq = %d, want %d", record.Seq, resp.Seq)
	}
}

func assertSessionExecutionTriggerReplayResponse(t *testing.T, got, want SessionExecutionTriggerResponse) {
	t.Helper()
	if got.Seq != want.Seq {
		t.Fatalf("replay seq = %d, want %d", got.Seq, want.Seq)
	}
	if got.TurnID != want.TurnID {
		t.Fatalf("replay turn_id = %q, want %q", got.TurnID, want.TurnID)
	}
	if got.TriggerID != want.TriggerID {
		t.Fatalf("replay trigger_id = %q, want %q", got.TriggerID, want.TriggerID)
	}
}

func TestSessionExecutionTriggerRejectsUserMessageContentTextAboveSchemaLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-message-limit", "sess-trigger-message-limit")
	tooLong := strings.Repeat("x", 32769)
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-message-limit", SessionID: "sess-trigger-message-limit", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: tooLong}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_validation_schema_invalid")
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
