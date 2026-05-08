package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
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
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-missing", SessionID: "sess-trigger-missing", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: defaultWorkflowRoutingForTriggerTests(), UserMessageContentText: "hello"}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleSessionExecutionTrigger expected blocked posture error")
	}
	if errResp.Error.Code != "project_substrate_operation_blocked" {
		t.Fatalf("error code = %q, want project_substrate_operation_blocked", errResp.Error.Code)
	}
}

func TestSessionExecutionTriggerFailsClosedForBlockedProjectSubstratePostures(t *testing.T) {
	testCases := []struct {
		name        string
		posture     string
		reasonCodes []string
	}{
		{name: "invalid", posture: projectsubstrate.CompatibilityPostureInvalid, reasonCodes: []string{"project_substrate_invalid"}},
		{name: "non-verified", posture: projectsubstrate.CompatibilityPostureNonVerified, reasonCodes: []string{"project_substrate_non_verified"}},
		{name: "unsupported-too-old", posture: projectsubstrate.CompatibilityPostureUnsupportedTooOld, reasonCodes: []string{"project_substrate_unsupported_too_old"}},
		{name: "unsupported-too-new", posture: projectsubstrate.CompatibilityPostureUnsupportedTooNew, reasonCodes: []string{"project_substrate_unsupported_too_new"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newBrokerAPIServiceForTests(t, APIConfig{})
			seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-blocked-"+tc.name, "sess-trigger-blocked-"+tc.name)
			s.discoverProjectSubstrateFn = func() (projectsubstrate.DiscoveryResult, error) {
				return projectsubstrate.DiscoveryResult{Compatibility: projectsubstrate.CompatibilityAssessment{Posture: tc.posture, NormalOperationAllowed: false, BlockedReasonCodes: tc.reasonCodes}}, nil
			}
			_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-blocked-" + tc.name, SessionID: "sess-trigger-blocked-" + tc.name, TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: defaultWorkflowRoutingForTriggerTests(), UserMessageContentText: "hello"}, RequestContext{})
			if errResp == nil {
				t.Fatalf("HandleSessionExecutionTrigger expected blocked posture error for %s", tc.posture)
			}
			if errResp.Error.Code != "project_substrate_operation_blocked" {
				t.Fatalf("error code = %q, want project_substrate_operation_blocked", errResp.Error.Code)
			}
		})
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
	authority, ok, err := s.ActiveRunPlanAuthority(exec.PrimaryRunID)
	if err != nil {
		t.Fatalf("ActiveRunPlanAuthority returned error: %v", err)
	}
	if !ok {
		t.Fatal("active trusted run plan authority missing for session execution run")
	}
	if strings.TrimSpace(authority.PlanID) == "" || strings.TrimSpace(authority.RunPlanDigest) == "" {
		t.Fatalf("active run plan authority invalid: %+v", authority)
	}
}

func TestSessionExecutionTriggerFailsClosedOnOverlappingMutationBearingStarts(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-active", "sess-trigger-active")
	first := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-active-1", SessionID: "sess-trigger-active", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "first"})
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-active-2", SessionID: "sess-trigger-active", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: defaultWorkflowRoutingForTriggerTests(), UserMessageContentText: "second"}, RequestContext{})
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
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-multiwait-2", SessionID: "sess-trigger-multiwait", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: defaultWorkflowRoutingForTriggerTests(), UserMessageContentText: "second"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_session_execution_overlap_blocked")
}

func TestSessionExecutionTriggerInteractiveAndAutonomousShareWorkflowRoutingContract(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-routing-chat", "sess-trigger-routing-chat")
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-routing-auto", "sess-trigger-routing-auto")
	routing := &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "draft_promote_apply"}
	interactive := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-routing-chat", SessionID: "sess-trigger-routing-chat", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: routing, UserMessageContentText: "chat"})
	if _, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{SessionID: "sess-trigger-routing-chat", TurnID: interactive.TurnID, ExecutionState: "completed", OccurredAt: s.currentTimestamp()}); err != nil {
		t.Fatalf("UpdateSessionTurnExecution returned error: %v", err)
	}
	autonomous := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-routing-auto", SessionID: "sess-trigger-routing-auto", TriggerSource: "autonomous_background", RequestedOperation: "start", WorkflowRouting: routing, UserMessageContentText: "auto"})
	if interactive.WorkflowRouting.WorkflowFamily != autonomous.WorkflowRouting.WorkflowFamily || interactive.WorkflowRouting.WorkflowOperation != autonomous.WorkflowRouting.WorkflowOperation {
		t.Fatalf("workflow routing mismatch: interactive=%+v autonomous=%+v", interactive.WorkflowRouting, autonomous.WorkflowRouting)
	}
}

func TestSessionExecutionTriggerRejectsInvalidWorkflowRoutingValues(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-routing-invalid", "sess-trigger-routing-invalid")
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-routing-invalid", SessionID: "sess-trigger-routing-invalid", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "invalid", WorkflowOperation: "approved_change_implementation"}, UserMessageContentText: "hello"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_validation_schema_invalid")
}

func TestSessionExecutionTriggerAllowsDraftRoutingOperations(t *testing.T) {
	for _, op := range []string{"change_draft", "spec_draft"} {
		s := newBrokerAPIServiceForTests(t, APIConfig{})
		sessionID := "sess-trigger-draft-routing-" + op
		seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-draft-routing-"+op, sessionID)
		mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-draft-routing-" + op, SessionID: sessionID, TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: op}, UserMessageContentText: "hello"})
	}
}

func TestSessionExecutionTriggerMaterializesTypedDraftArtifactsForSupportedDraftOperations(t *testing.T) {
	for _, tc := range []struct {
		name           string
		operation      string
		requestID      string
		sessionID      string
		message        string
		artifactRef    string
		schemaID       string
		identityField  string
		identityPrefix string
	}{
		{name: "change draft", operation: sessionWorkflowOperationChangeDraft, requestID: "req-session-trigger-change-draft-artifact", sessionID: "sess-trigger-change-draft-artifact", message: "Draft CHG phase 3a artifact path", artifactRef: "change_draft_artifact", schemaID: "runecode.protocol.v0.RuneContextChangeDraftArtifact", identityField: "change_id", identityPrefix: "CHG-"},
		{name: "spec draft", operation: sessionWorkflowOperationSpecDraft, requestID: "req-session-trigger-spec-draft-artifact", sessionID: "sess-trigger-spec-draft-artifact", message: "Spec phase 3a artifact path", artifactRef: "spec_draft_artifact", schemaID: "runecode.protocol.v0.RuneContextSpecDraftArtifact", identityField: "spec_id", identityPrefix: "spec-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newBrokerAPIServiceForTests(t, APIConfig{})
			assertSupportedDraftArtifacts(t, s, tc)
		})
	}
}

func assertSupportedDraftArtifacts(t *testing.T, s *Service, tc struct {
	name           string
	operation      string
	requestID      string
	sessionID      string
	message        string
	artifactRef    string
	schemaID       string
	identityField  string
	identityPrefix string
}) {
	t.Helper()
	s.sessionExecutionRunner = launchSessionExecutionRunnerCompleteInProcessForTests
	seedSessionRuntimeFactsForOpsTest(t, s, "run-"+tc.operation, tc.sessionID)
	ack := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: tc.requestID, SessionID: tc.sessionID, TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: tc.operation}, UserMessageContentText: tc.message})
	if ack.ExecutionState != "running" {
		t.Fatalf("ack execution_state = %q, want running", ack.ExecutionState)
	}
	getResp := mustSessionGet(t, s, tc.requestID+"-get", tc.sessionID)
	if getResp.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing after draft run")
	}
	exec := getResp.Session.LatestTurnExecution
	if exec.ExecutionState != "completed" {
		t.Fatalf("latest execution_state after real path = %q, want completed", exec.ExecutionState)
	}
	if got := exec.WorkflowRouting.WorkflowOperation; got != tc.operation {
		t.Fatalf("latest workflow_operation = %q, want %q", got, tc.operation)
	}
	assertDraftArtifactsForExecution(t, s, exec.PrimaryRunID, tc.artifactRef, tc.schemaID, tc.identityField, tc.identityPrefix, tc.message)
}

func assertDraftArtifactsForExecution(t *testing.T, s *Service, runID, artifactRef, schemaID, identityField, identityPrefix, message string) {
	t.Helper()
	promptStepID := "session_execution/" + strings.TrimSuffix(artifactRef, "_artifact") + "_prompt"
	requireSessionExecutionLinkedArtifactByStepAndSchema(t, s, runID, promptStepID, "", message)
	artifact := requireSessionExecutionLinkedArtifactByStepAndSchema(t, s, runID, "session_execution/"+artifactRef, schemaID, "")
	identity := stringValueFromMap(artifact, identityField)
	if !strings.HasPrefix(identity, identityPrefix) {
		t.Fatalf("%s = %q, want prefix %q", identityField, identity, identityPrefix)
	}
	if got := digestObjectValueFromMap(artifact, "validated_project_substrate_digest"); got == "" {
		t.Fatalf("typed draft artifact missing validated_project_substrate_digest: %+v", artifact)
	}
}
func TestSessionExecutionTriggerRejectsMutationBearingDraftRouting(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-draft-mutation", "sess-trigger-draft-mutation")
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-draft-mutation", SessionID: "sess-trigger-draft-mutation", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "change_draft", BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "change_draft_artifact", ArtifactDigest: digestForBrokerTest("x")}}}, UserMessageContentText: "hello"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_validation_schema_invalid")
}

func TestSessionExecutionTriggerApprovedImplementationRequiresBoundInputSet(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-session-trigger-approved-missing", "sess-trigger-approved-missing")
	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-trigger-approved-missing", SessionID: "sess-trigger-approved-missing", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: "approved_change_implementation"}, UserMessageContentText: "hello"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_validation_schema_invalid")
}

func TestSessionExecutionTriggerApprovedImplementationRejectsUnapprovedMutationDigest(t *testing.T) {
	repoRoot := t.TempDir()
	writeProjectSubstrateAnchors(t, repoRoot, "0.1.0-alpha.14", "verified", "runecontext")
	s := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-approved-impl-invalid", "sess-approved-impl-invalid")

	proposalText := "# CHG-approved-impl-invalid\n"
	proposalDigest := putApprovedImplementationMutationArtifactForTest(t, s, map[string]any{
		"target_path":    "runecontext/changes/CHG-approved-impl-invalid/proposal.md",
		"content":        proposalText,
		"content_digest": digestObject(artifacts.DigestBytes([]byte(proposalText))),
		"write_mode":     "create",
	})
	fixture := approvedImplementationInputSetFixture(t, s, []string{artifacts.DigestBytes([]byte("approved-only"))}, nil, nil)
	fixture["workspace_mutation_digests"] = digestObjects([]string{proposalDigest})
	inputSetDigest := putApprovedImplementationInputSetForTest(t, s, fixture)

	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-approved-impl-invalid", SessionID: "sess-approved-impl-invalid", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationApprovedImplementation, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "implementation_input_set", ArtifactDigest: inputSetDigest}}}, UserMessageContentText: "apply approved implementation"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_storage_write_failed")
	if !strings.Contains(errResp.Error.Message, "not included in approved_input_digests") {
		t.Fatalf("error message = %q, want unapproved mutation digest detail", errResp.Error.Message)
	}
}

func TestSessionExecutionTriggerApprovedImplementationRejectsEmbeddedInputSetDigestDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeProjectSubstrateAnchors(t, repoRoot, "0.1.0-alpha.14", "verified", "runecontext")
	s := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-approved-impl-digest-drift", "sess-approved-impl-digest-drift")

	proposalText := "# CHG-approved-impl-digest-drift\n"
	proposalDigest := putApprovedImplementationMutationArtifactForTest(t, s, map[string]any{
		"target_path":    "runecontext/changes/CHG-approved-impl-digest-drift/proposal.md",
		"content":        proposalText,
		"content_digest": digestObject(artifacts.DigestBytes([]byte(proposalText))),
		"write_mode":     "create",
	})
	payload := approvedImplementationInputSetFixture(t, s, []string{proposalDigest}, []string{proposalDigest}, nil)
	payload["input_set_digest"] = digestObject("sha256:" + strings.Repeat("f", 64))
	inputSetArtifactDigest := putApprovedImplementationInputSetArtifactForTest(t, s, payload)

	_, errResp := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-approved-impl-digest-drift", SessionID: "sess-approved-impl-digest-drift", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationApprovedImplementation, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "implementation_input_set", ArtifactDigest: inputSetArtifactDigest}}}, UserMessageContentText: "apply approved implementation"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, errResp, "broker_validation_schema_invalid")
	if !strings.Contains(errResp.Error.Message, "input_set_digest drift detected") {
		t.Fatalf("error message = %q, want input_set_digest drift detail", errResp.Error.Message)
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
	_, triggerErr := s.HandleSessionExecutionTrigger(context.Background(), SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-session-blocked-trigger", SessionID: "sess-trigger-blocked-posture", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: defaultWorkflowRoutingForTriggerTests(), UserMessageContentText: "blocked"}, RequestContext{})
	assertSessionExecutionContinueBlocked(t, triggerErr, "project_substrate_operation_blocked")
}
