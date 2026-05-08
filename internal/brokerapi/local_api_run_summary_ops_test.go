package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/runplan"
)

func TestRunSummaryUsesBuiltInWorkflowAuthorityForTypedDraftExecutionPath(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	exec, runGet := runGetForChangeDraftSummaryTest(t, s)
	assertChangeDraftRunSummary(t, runGet)
	assertRunHasTypedChangeDraftArtifact(t, s, exec.PrimaryRunID)
}

func assertChangeDraftRunSummary(t *testing.T, runGet RunGetResponse) {
	t.Helper()
	if got := strings.TrimSpace(runGet.Run.Summary.WorkflowKind); got != "builtin_rc_change_draft_v0" {
		t.Fatalf("workflow_kind = %q, want builtin_rc_change_draft_v0", got)
	}
	if runGet.Run.Summary.LifecycleState != "completed" {
		t.Fatalf("lifecycle_state = %q, want completed", runGet.Run.Summary.LifecycleState)
	}
}

func assertRunHasTypedChangeDraftArtifact(t *testing.T, s *Service, runID string) {
	t.Helper()
	requireSessionExecutionLinkedArtifactByStepAndSchema(t, s, runID, "session_execution/change_draft_artifact", "runecode.protocol.v0.RuneContextChangeDraftArtifact", "")
	if !runHasTypedChangeDraftArtifact(t, s, runID) {
		t.Fatal("typed change draft artifact not found in run-scoped artifacts")
	}
}

func runHasTypedChangeDraftArtifact(t *testing.T, s *Service, runID string) bool {
	t.Helper()
	for _, record := range s.List() {
		if record.RunID != runID || record.StepID != "session_execution/change_draft_artifact" {
			continue
		}
		payload := mustArtifactPayload(t, s, record.Reference.Digest)
		decoded := mustDecodeArtifactJSON(t, record.StepID, payload)
		if strings.TrimSpace(stringValueFromMap(decoded, "schema_id")) == "runecode.protocol.v0.RuneContextChangeDraftArtifact" {
			return true
		}
	}
	return false
}

func runGetForChangeDraftSummaryTest(t *testing.T, s *Service) (*SessionTurnExecution, RunGetResponse) {
	t.Helper()
	s.sessionExecutionRunner = launchSessionExecutionRunnerCompleteInProcessForTests
	_ = mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-run-summary-draft-path", SessionID: "sess-run-summary-draft-path", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationChangeDraft}, UserMessageContentText: "summary path draft artifact"})
	getResp := mustSessionGet(t, s, "req-run-summary-draft-path-session", "sess-run-summary-draft-path")
	if getResp.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing")
	}
	exec := getResp.Session.LatestTurnExecution
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{
		SchemaID:      "runecode.protocol.v0.RunGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-summary-draft-path-get",
		RunID:         exec.PrimaryRunID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	return exec, runGet
}

func TestRunSummaryUsesActivePlanAuthorityForApprovedImplementationPath(t *testing.T) {
	repoRoot := t.TempDir()
	writeProjectSubstrateAnchors(t, repoRoot, "0.1.0-alpha.14", "verified", "runecontext")
	s := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-summary-approved-impl", "sess-run-summary-approved-impl")

	mutationDigest := putApprovedImplementationMutationArtifactForTest(t, s, map[string]any{
		"target_path":    "runecontext/changes/CHG-approved-impl/proposal.md",
		"content":        "# CHG-approved-impl\n\nImplemented by approved workflow.\n",
		"content_digest": digestObject(artifacts.DigestBytes([]byte("# CHG-approved-impl\n\nImplemented by approved workflow.\n"))),
		"write_mode":     "create",
	})
	approvedDigest := artifacts.DigestBytes([]byte("approved-run-summary-input"))
	inputSetDigest := putApprovedImplementationInputSetForTest(t, s, approvedImplementationInputSetFixture(t, s, []string{approvedDigest, mutationDigest}, []string{mutationDigest}, nil))

	_ = mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-run-summary-approved-impl", SessionID: "sess-run-summary-approved-impl", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationApprovedImplementation, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "implementation_input_set", ArtifactDigest: inputSetDigest}}}, UserMessageContentText: "apply approved implementation"})
	getResp := mustSessionGet(t, s, "req-run-summary-approved-impl-session", "sess-run-summary-approved-impl")
	if getResp.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing")
	}
	exec := getResp.Session.LatestTurnExecution
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-summary-approved-impl-get", RunID: exec.PrimaryRunID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.WorkflowKind); got != "builtin_rc_approved_implementation_v0" {
		t.Fatalf("workflow_kind = %q, want builtin_rc_approved_implementation_v0", got)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.CurrentStageID); got == "" || got == "artifact_flow" {
		t.Fatalf("current_stage_id = %q, want plan-authoritative non-artifact_flow stage", got)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.ApprovalProfile); got != "moderate" {
		t.Fatalf("approval_profile = %q, want moderate", got)
	}
	if runGet.Run.Summary.LifecycleState != "blocked" {
		t.Fatalf("lifecycle_state = %q, want blocked from consumed approvals", runGet.Run.Summary.LifecycleState)
	}
}

func TestRunSummaryUsesPlanAuthoritativeStageAndApprovalProfileForDraftPromoteApply(t *testing.T) {
	repoRoot := t.TempDir()
	writeProjectSubstrateAnchors(t, repoRoot, "0.1.0-alpha.14", "verified", "runecontext")
	s := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	seedSessionRuntimeFactsForOpsTest(t, s, "run-summary-promote", "sess-run-summary-promote")

	mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-run-summary-promote-draft", SessionID: "sess-run-summary-promote", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationChangeDraft}, UserMessageContentText: "summary promote draft"})
	draftGet := mustSessionGet(t, s, "req-run-summary-promote-draft-get", "sess-run-summary-promote")
	if draftGet.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing after draft")
	}
	draftRunID := draftGet.Session.LatestTurnExecution.PrimaryRunID
	draftDigest := digestForRunStep(t, s, draftRunID, "session_execution/change_draft_artifact")

	mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-run-summary-promote-apply", SessionID: "sess-run-summary-promote", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationDraftPromoteApply, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "change_draft_artifact", ArtifactDigest: draftDigest}}}, UserMessageContentText: "summary promote apply"})
	post := mustSessionGet(t, s, "req-run-summary-promote-apply-get", "sess-run-summary-promote")
	if post.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing after promote/apply")
	}
	runID := post.Session.LatestTurnExecution.PrimaryRunID
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-summary-promote-get", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.WorkflowKind); got != "builtin_rc_draft_promote_v0" {
		t.Fatalf("workflow_kind = %q, want builtin_rc_draft_promote_v0", got)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.CurrentStageID); got == "" || got == "artifact_flow" {
		t.Fatalf("current_stage_id = %q, want plan-authoritative non-artifact_flow stage", got)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.ApprovalProfile); got != "moderate" {
		t.Fatalf("approval_profile = %q, want moderate", got)
	}
}

func TestRunSummaryLeavesWorkflowIdentityUnknownWithoutActivePlanAuthority(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-summary-missing-authority"
	entry := runplan.BuiltInWorkflowCatalogV0()[0]
	if err := s.SetRunStatus(runID, "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	if _, err := s.Put(artifacts.PutRequest{
		Payload:               []byte(`{"schema_id":"runecode.protocol.v0.WorkflowDefinition"}`),
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: entry.WorkflowDefinitionHash,
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 runID,
		StepID:                "session_execution/workflow_definition",
	}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-summary-missing-authority", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.WorkflowKind); got != "" {
		t.Fatalf("workflow_kind = %q, want empty without active plan authority", got)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.WorkflowDefinitionHash); got != "" {
		t.Fatalf("workflow_definition_hash = %q, want empty without active plan authority", got)
	}
	if got := strings.TrimSpace(runGet.Run.Summary.CurrentStageID); got != "" {
		t.Fatalf("current_stage_id = %q, want empty without active plan authority", got)
	}
	if got := runGet.Run.AuthoritativeState["workflow_projection_reason"]; got != "missing_active_run_plan_authority" {
		t.Fatalf("authoritative_state.workflow_projection_reason = %v, want missing_active_run_plan_authority", got)
	}
}
