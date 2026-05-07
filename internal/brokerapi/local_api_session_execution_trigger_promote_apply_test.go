package brokerapi

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestSessionExecutionTriggerDraftPromoteApplyWritesCanonicalChangeDraft(t *testing.T) {
	repoRoot, s := newSessionExecutionTriggerWorkflowService(t, "run-change-promote", "sess-change-promote")
	draft := runChangeDraftForPromoteApply(t, s, "sess-change-promote", "req-change-promote-draft", "Draft change promote apply path")

	mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-change-promote-apply", SessionID: "sess-change-promote", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationDraftPromoteApply, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "change_draft_artifact", ArtifactDigest: draft.digest}}}, UserMessageContentText: "apply reviewed change draft"})

	requireFileContents(t, repoRoot, filepath.ToSlash(filepath.Join("runecontext/changes", draft.changeOrSpecID, "proposal.md")), draft.draftText)
	assertDraftPromoteApplyExecution(t, s, "req-change-promote-apply-get", "sess-change-promote")
	assertDraftPromoteApplyAuditEvent(t, s, draft.digest)
}

func TestSessionExecutionTriggerDraftPromoteApplyWritesCanonicalSpecDraft(t *testing.T) {
	repoRoot, s := newSessionExecutionTriggerWorkflowService(t, "run-spec-promote", "sess-spec-promote")
	draft := runSpecDraftForPromoteApply(t, s, "sess-spec-promote", "req-spec-promote-draft", "Spec promote apply path")

	mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-spec-promote-apply", SessionID: "sess-spec-promote", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationDraftPromoteApply, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "spec_draft_artifact", ArtifactDigest: draft.digest}}}, UserMessageContentText: "apply reviewed spec draft"})

	requireFileContents(t, repoRoot, filepath.ToSlash(filepath.Join("runecontext/specs", draft.changeOrSpecID+".md")), draft.draftText)
	assertDraftPromoteApplyExecution(t, s, "req-spec-promote-apply-get", "sess-spec-promote")
	assertDraftPromoteApplyAuditEvent(t, s, draft.digest)
}

func TestSessionExecutionTriggerApprovedImplementationAppliesWorkspaceAndLifecycleMetadataMutations(t *testing.T) {
	repoRoot, s := newSessionExecutionTriggerWorkflowService(t, "run-approved-impl", "sess-approved-impl")
	inputSetArtifactDigest, inputSetDigest, proposalText, tasksText := seedApprovedImplementationWorkspaceMutationFixture(t, s)
	ack := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-approved-impl", SessionID: "sess-approved-impl", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationApprovedImplementation, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "implementation_input_set", ArtifactDigest: inputSetArtifactDigest}}}, UserMessageContentText: "apply approved implementation"})

	if ack.ExecutionState != "running" {
		t.Fatalf("ack execution_state = %q, want running", ack.ExecutionState)
	}
	requireFileContents(t, repoRoot, filepath.ToSlash(filepath.Join("runecontext/changes", "CHG-approved-impl", "proposal.md")), proposalText)
	requireFileContents(t, repoRoot, filepath.ToSlash(filepath.Join("runecontext/changes", "CHG-approved-impl", "tasks.md")), tasksText)
	assertApprovedImplementationExecution(t, s, "req-approved-impl-get", "sess-approved-impl")
	assertApprovedImplementationAuditEvent(t, s, inputSetArtifactDigest, inputSetDigest)
}

func TestSessionExecutionTriggerSessionDetailProjectsExecutionOwnedLinksForCompletedWorkflowLoop(t *testing.T) {
	_, s := newSessionExecutionTriggerWorkflowService(t, "run-trigger-loop-links", "sess-trigger-loop-links")
	draft := runChangeDraftForPromoteApply(t, s, "sess-trigger-loop-links", "req-trigger-loop-links-draft", "Loop inspectability draft")
	mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-trigger-loop-links-apply", SessionID: "sess-trigger-loop-links", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationDraftPromoteApply, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "change_draft_artifact", ArtifactDigest: draft.digest}}}, UserMessageContentText: "apply inspected draft"})
	assertCompletedWorkflowLoopProjectsOwnedLinks(t, s, draft.digest, draft.primaryRunID)
}

type draftPromoteApplyFixture struct {
	digest         string
	changeOrSpecID string
	draftText      string
	primaryRunID   string
}

func newSessionExecutionTriggerWorkflowService(t *testing.T, runID, sessionID string) (string, *Service) {
	t.Helper()
	repoRoot := t.TempDir()
	writeProjectSubstrateAnchors(t, repoRoot, "0.1.0-alpha.14", "verified", "runecontext")
	s := newBrokerAPIServiceForTests(t, APIConfig{RepositoryRoot: repoRoot})
	seedSessionRuntimeFactsForOpsTest(t, s, runID, sessionID)
	return repoRoot, s
}

func runChangeDraftForPromoteApply(t *testing.T, s *Service, sessionID, requestID, message string) draftPromoteApplyFixture {
	t.Helper()
	ack := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: requestID, SessionID: sessionID, TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationChangeDraft}, UserMessageContentText: message})
	if ack.ExecutionState != "running" {
		t.Fatalf("change draft ack execution_state = %q, want running", ack.ExecutionState)
	}
	return loadDraftPromoteApplyFixture(t, s, sessionID, requestID+"-get", "session_execution/change_draft_artifact", "runecode.protocol.v0.RuneContextChangeDraftArtifact", "change_id")
}

func runSpecDraftForPromoteApply(t *testing.T, s *Service, sessionID, requestID, message string) draftPromoteApplyFixture {
	t.Helper()
	mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: requestID, SessionID: sessionID, TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationSpecDraft}, UserMessageContentText: message})
	return loadDraftPromoteApplyFixture(t, s, sessionID, requestID+"-get", "session_execution/spec_draft_artifact", "runecode.protocol.v0.RuneContextSpecDraftArtifact", "spec_id")
}

func loadDraftPromoteApplyFixture(t *testing.T, s *Service, sessionID, requestID, stepID, schemaID, identityField string) draftPromoteApplyFixture {
	t.Helper()
	getResp := mustSessionGet(t, s, requestID, sessionID)
	if getResp.Session.LatestTurnExecution == nil {
		t.Fatalf("latest_turn_execution missing after %s", stepID)
	}
	exec := getResp.Session.LatestTurnExecution
	artifact := requireSessionExecutionLinkedArtifactByStepAndSchema(t, s, exec.PrimaryRunID, stepID, schemaID, "")
	draftTextDigest := digestObjectValueFromMap(artifact, "artifact_digest")
	return draftPromoteApplyFixture{
		digest:         digestForRunStep(t, s, exec.PrimaryRunID, stepID),
		changeOrSpecID: stringValueFromMap(artifact, identityField),
		draftText:      mustArtifactText(t, s, draftTextDigest),
		primaryRunID:   exec.PrimaryRunID,
	}
}

func seedApprovedImplementationWorkspaceMutationFixture(t *testing.T, s *Service) (string, string, string, string) {
	t.Helper()
	proposalText := "# CHG-approved-impl\n\n## Summary\nImplemented from approved input set.\n"
	tasksText := "# Tasks\n\n- [x] Implement approved workspace mutation path\n"
	approvedWorkspaceDigest := artifacts.DigestBytes([]byte("approved-workspace-input"))
	approvedMetadataDigest := artifacts.DigestBytes([]byte("approved-metadata-input"))
	proposalDigest := putApprovedImplementationMutationArtifactForTest(t, s, map[string]any{
		"target_path":    "runecontext/changes/CHG-approved-impl/proposal.md",
		"content":        proposalText,
		"content_digest": digestObject(artifacts.DigestBytes([]byte(proposalText))),
		"write_mode":     "create",
	})
	tasksDigest := putApprovedImplementationMutationArtifactForTest(t, s, map[string]any{
		"target_path":    "runecontext/changes/CHG-approved-impl/tasks.md",
		"content":        tasksText,
		"content_digest": digestObject(artifacts.DigestBytes([]byte(tasksText))),
		"write_mode":     "create",
	})
	payload := approvedImplementationInputSetFixture(t, s, []string{approvedWorkspaceDigest, approvedMetadataDigest, proposalDigest, tasksDigest}, []string{proposalDigest}, []string{tasksDigest})
	inputSetDigest, ok := approvedImplementationInputSetDigest(payload)
	if !ok {
		t.Fatal("approvedImplementationInputSetDigest returned invalid fixture digest")
	}
	inputSetArtifactDigest := putApprovedImplementationInputSetForTest(t, s, payload)
	return inputSetArtifactDigest, inputSetDigest, proposalText, tasksText
}

func assertDraftPromoteApplyExecution(t *testing.T, s *Service, requestID, sessionID string) {
	t.Helper()
	post := mustSessionGet(t, s, requestID, sessionID)
	if post.Session.LatestTurnExecution == nil || post.Session.LatestTurnExecution.ExecutionState != "completed" {
		t.Fatalf("latest turn execution after change promote/apply = %+v, want completed", post.Session.LatestTurnExecution)
	}
	if post.Session.LatestTurnExecution.WorkflowRouting.WorkflowOperation != sessionWorkflowOperationDraftPromoteApply {
		t.Fatalf("latest workflow operation after change promote/apply = %q, want %q", post.Session.LatestTurnExecution.WorkflowRouting.WorkflowOperation, sessionWorkflowOperationDraftPromoteApply)
	}
	if approvalID := post.Session.LatestTurnExecution.PendingApprovalID; approvalID != "" {
		t.Fatalf("pending_approval_id after change promote/apply = %q, want empty", approvalID)
	}
	if len(post.Session.LatestTurnExecution.LinkedApprovalIDs) == 0 {
		t.Fatal("linked_approval_ids empty after change promote/apply")
	}
	approvalResp := mustApprovalGetResponse(t, s, requestID+"-approval", post.Session.LatestTurnExecution.LinkedApprovalIDs[0])
	if approvalResp.Approval.Status != "consumed" || approvalResp.Approval.BoundScope.ActionKind != policyengine.ActionKindWorkspaceWrite {
		t.Fatalf("unexpected promote/apply approval: %+v", approvalResp.Approval)
	}
}

func assertDraftPromoteApplyAuditEvent(t *testing.T, s *Service, draftDigest string) {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	if !auditEventContainsValue(events, "runecontext_draft_promote_apply", "draft_artifact_digest", draftDigest) {
		t.Fatalf("draft promote/apply audit event missing draft digest %q", draftDigest)
	}
}

func assertApprovedImplementationExecution(t *testing.T, s *Service, requestID, sessionID string) {
	t.Helper()
	post := mustSessionGet(t, s, requestID, sessionID)
	if post.Session.LatestTurnExecution == nil || post.Session.LatestTurnExecution.ExecutionState != "completed" {
		t.Fatalf("latest turn execution after approved implementation = %+v, want completed", post.Session.LatestTurnExecution)
	}
	if post.Session.LatestTurnExecution.WorkflowRouting.WorkflowOperation != sessionWorkflowOperationApprovedImplementation {
		t.Fatalf("latest workflow operation = %q, want %q", post.Session.LatestTurnExecution.WorkflowRouting.WorkflowOperation, sessionWorkflowOperationApprovedImplementation)
	}
	if len(post.Session.LatestTurnExecution.LinkedApprovalIDs) != 2 {
		t.Fatalf("linked_approval_ids len = %d, want 2", len(post.Session.LatestTurnExecution.LinkedApprovalIDs))
	}
	for _, approvalID := range post.Session.LatestTurnExecution.LinkedApprovalIDs {
		approvalResp := mustApprovalGetResponse(t, s, requestID+"-approval-"+sessionExecutionIdentifierToken(approvalID), approvalID)
		if approvalResp.Approval.Status != "consumed" || approvalResp.Approval.BoundScope.ActionKind != policyengine.ActionKindWorkspaceWrite {
			t.Fatalf("unexpected approved implementation approval: %+v", approvalResp.Approval)
		}
	}
}

func assertApprovedImplementationAuditEvent(t *testing.T, s *Service, inputSetArtifactDigest, inputSetDigest string) {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	if !auditEventContainsValue(events, "runecontext_approved_implementation_applied", "input_set_artifact_digest", inputSetArtifactDigest) {
		t.Fatalf("approved implementation audit event missing input set artifact digest %q", inputSetArtifactDigest)
	}
	if !auditEventContainsValue(events, "runecontext_approved_implementation_applied", "input_set_digest", inputSetDigest) {
		t.Fatalf("approved implementation audit event missing input set digest %q", inputSetDigest)
	}
}

func assertCompletedWorkflowLoopProjectsOwnedLinks(t *testing.T, s *Service, draftDigest, draftRunID string) {
	t.Helper()
	post := mustSessionGet(t, s, "req-trigger-loop-links-post", "sess-trigger-loop-links")
	if post.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing after promote/apply")
	}
	exec := post.Session.LatestTurnExecution
	if exec.ExecutionState != "completed" {
		t.Fatalf("latest execution_state = %q, want completed", exec.ExecutionState)
	}
	if len(exec.LinkedApprovalIDs) == 0 {
		t.Fatal("latest linked_approval_ids empty after promote/apply")
	}
	if !strings.Contains(strings.Join(post.Session.LinkedArtifactDigests, ","), draftDigest) {
		t.Fatalf("session linked_artifact_digests = %+v, want draft digest %q included", post.Session.LinkedArtifactDigests, draftDigest)
	}
	if len(post.Session.LinkedApprovalIDs) < len(exec.LinkedApprovalIDs) {
		t.Fatalf("session linked_approval_ids = %d, want at least %d", len(post.Session.LinkedApprovalIDs), len(exec.LinkedApprovalIDs))
	}
	if !strings.Contains(strings.Join(post.Session.LinkedRunIDs, ","), draftRunID) {
		t.Fatalf("session linked_run_ids = %+v, want draft run %q included", post.Session.LinkedRunIDs, draftRunID)
	}
	if !strings.Contains(strings.Join(post.Session.LinkedRunIDs, ","), exec.PrimaryRunID) {
		t.Fatalf("session linked_run_ids = %+v, want promote/apply run %q included", post.Session.LinkedRunIDs, exec.PrimaryRunID)
	}
}
