package brokerapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/trustpolicy"
)

func TestSessionExecutionTriggerVerificationSmokePathProducesInspectableEvidence(t *testing.T) {
	s, repoRoot := newVerificationSmokeService(t)
	changeDraftDigest, specDraftDigest, changeID := runVerificationSmokeDraftAndPromoteFlow(t, s, repoRoot)
	inputSetArtifactDigest, inputSetDigest, finalExec := runVerificationSmokeApprovedImplementation(t, s, repoRoot, changeID)
	assertVerificationSmokeRunAndArtifactSurfaces(t, s, finalExec.PrimaryRunID)
	assertVerificationSmokeAuditEvidenceSurfaces(t, s, changeDraftDigest, specDraftDigest, inputSetArtifactDigest, inputSetDigest)
}

func newVerificationSmokeService(t *testing.T) (*Service, string) {
	t.Helper()
	repoRoot := t.TempDir()
	writeProjectSubstrateAnchors(t, repoRoot, "0.1.0-alpha.14", "verified", "runecontext")
	storeRoot := t.TempDir()
	ledgerRoot := t.TempDir()
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	s, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repoRoot})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	s.sessionExecutionRunner = launchSessionExecutionRunnerCompleteInProcessForTests
	seedSessionRuntimeFactsForOpsTest(t, s, "run-verification-smoke", "sess-verification-smoke")
	return s, repoRoot
}

func runVerificationSmokeDraftAndPromoteFlow(t *testing.T, s *Service, repoRoot string) (string, string, string) {
	t.Helper()
	changeDraftDigest, changeID, changeProposalText := verificationSmokeChangeDraft(t, s)
	applyVerificationSmokeDraft(t, s, "change_draft_artifact", changeDraftDigest, "Apply reviewed change draft")
	requireFileContents(t, repoRoot, filepath.ToSlash(filepath.Join("runecontext/changes", changeID, "proposal.md")), changeProposalText)

	specDraftDigest, specID, specDraftText := verificationSmokeSpecDraft(t, s)
	applyVerificationSmokeDraft(t, s, "spec_draft_artifact", specDraftDigest, "Apply reviewed spec draft")
	requireFileContents(t, repoRoot, filepath.ToSlash(filepath.Join("runecontext/specs", specID+".md")), specDraftText)

	return changeDraftDigest, specDraftDigest, changeID
}

func applyVerificationSmokeDraft(t *testing.T, s *Service, artifactRef, digest, message string) {
	t.Helper()
	mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: verificationSmokePromoteRequestID(artifactRef), SessionID: "sess-verification-smoke", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationDraftPromoteApply, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: artifactRef, ArtifactDigest: digest}}}, UserMessageContentText: message})
}

func verificationSmokePromoteRequestID(artifactRef string) string {
	if artifactRef == "change_draft_artifact" {
		return "req-verification-smoke-change-promote"
	}
	return "req-verification-smoke-spec-promote"
}

func verificationSmokeChangeDraft(t *testing.T, s *Service) (string, string, string) {
	t.Helper()
	changeAck := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-change-draft", SessionID: "sess-verification-smoke", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationChangeDraft}, UserMessageContentText: "Verification smoke change draft"})
	if changeAck.ExecutionState != "running" {
		t.Fatalf("change draft ack execution_state = %q, want running", changeAck.ExecutionState)
	}
	changeGet := mustSessionGet(t, s, "req-verification-smoke-change-draft-get", "sess-verification-smoke")
	if changeGet.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing after change draft")
	}
	changeExec := changeGet.Session.LatestTurnExecution
	changeArtifact := requireSessionExecutionLinkedArtifactByStepAndSchema(t, s, changeExec.PrimaryRunID, "session_execution/change_draft_artifact", "runecode.protocol.v0.RuneContextChangeDraftArtifact", "")
	changeDraftDigest := digestForRunStep(t, s, changeExec.PrimaryRunID, "session_execution/change_draft_artifact")
	changeID := stringValueFromMap(changeArtifact, "change_id")
	changeProposalDigest := digestObjectValueFromMap(changeArtifact, "artifact_digest")
	changeProposalText := mustArtifactText(t, s, changeProposalDigest)
	return changeDraftDigest, changeID, changeProposalText
}

func verificationSmokeSpecDraft(t *testing.T, s *Service) (string, string, string) {
	t.Helper()
	specAck := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-spec-draft", SessionID: "sess-verification-smoke", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationSpecDraft}, UserMessageContentText: "Verification smoke spec draft"})
	if specAck.ExecutionState != "running" {
		t.Fatalf("spec draft ack execution_state = %q, want running", specAck.ExecutionState)
	}
	specGet := mustSessionGet(t, s, "req-verification-smoke-spec-draft-get", "sess-verification-smoke")
	if specGet.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing after spec draft")
	}
	specExec := specGet.Session.LatestTurnExecution
	specArtifact := requireSessionExecutionLinkedArtifactByStepAndSchema(t, s, specExec.PrimaryRunID, "session_execution/spec_draft_artifact", "runecode.protocol.v0.RuneContextSpecDraftArtifact", "")
	specDraftDigest := digestForRunStep(t, s, specExec.PrimaryRunID, "session_execution/spec_draft_artifact")
	specID := stringValueFromMap(specArtifact, "spec_id")
	specTextDigest := digestObjectValueFromMap(specArtifact, "artifact_digest")
	specDraftText := mustArtifactText(t, s, specTextDigest)
	return specDraftDigest, specID, specDraftText
}

func runVerificationSmokeApprovedImplementation(t *testing.T, s *Service, repoRoot, changeID string) (string, string, *SessionTurnExecution) {
	t.Helper()
	proposalText := fmt.Sprintf("# %s\n\n## Verification smoke\nWorkspace mutation applied from approved input set.\n", changeID)
	tasksText := "# Tasks\n\n- [x] Verification smoke implementation path applied\n"
	approvedWorkspaceDigest := digestForVerificationSmokeInput("verification-smoke-approved-workspace")
	approvedMetadataDigest := digestForVerificationSmokeInput("verification-smoke-approved-metadata")
	proposalMutationDigest := putApprovedImplementationMutationArtifactForTest(t, s, map[string]any{"target_path": filepath.ToSlash(filepath.Join("runecontext/changes", changeID, "proposal.md")), "content": proposalText, "content_digest": digestObject(digestForVerificationSmokeInput(proposalText)), "write_mode": "update"})
	tasksMutationDigest := putApprovedImplementationMutationArtifactForTest(t, s, map[string]any{"target_path": filepath.ToSlash(filepath.Join("runecontext/changes", changeID, "tasks.md")), "content": tasksText, "content_digest": digestObject(digestForVerificationSmokeInput(tasksText)), "write_mode": "create"})
	payload := approvedImplementationInputSetFixture(t, s, []string{approvedWorkspaceDigest, approvedMetadataDigest, proposalMutationDigest, tasksMutationDigest}, []string{proposalMutationDigest}, []string{tasksMutationDigest})
	inputSetDigest, ok := approvedImplementationInputSetDigest(payload)
	if !ok {
		t.Fatal("approvedImplementationInputSetDigest returned invalid verification fixture digest")
	}
	inputSetArtifactDigest := putApprovedImplementationInputSetForTest(t, s, payload)
	implAck := mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-approved-implementation", SessionID: "sess-verification-smoke", TriggerSource: "interactive_user", RequestedOperation: "start", WorkflowRouting: &SessionWorkflowPackRouting{SchemaID: "runecode.protocol.v0.SessionWorkflowPackRouting", SchemaVersion: "0.1.0", WorkflowFamily: "runecontext", WorkflowOperation: sessionWorkflowOperationApprovedImplementation, BoundInputArtifacts: []SessionWorkflowPackBoundInputArtifact{{ArtifactRef: "implementation_input_set", ArtifactDigest: inputSetArtifactDigest}}}, UserMessageContentText: "Apply approved implementation"})
	if implAck.ExecutionState != "running" {
		t.Fatalf("approved implementation ack execution_state = %q, want running", implAck.ExecutionState)
	}
	requireFileContents(t, repoRoot, filepath.ToSlash(filepath.Join("runecontext/changes", changeID, "proposal.md")), proposalText)
	requireFileContents(t, repoRoot, filepath.ToSlash(filepath.Join("runecontext/changes", changeID, "tasks.md")), tasksText)
	post := mustSessionGet(t, s, "req-verification-smoke-post", "sess-verification-smoke")
	if post.Session.LatestTurnExecution == nil {
		t.Fatal("latest_turn_execution missing after approved implementation")
	}
	finalExec := post.Session.LatestTurnExecution
	if got := finalExec.WorkflowRouting.WorkflowOperation; got != sessionWorkflowOperationApprovedImplementation {
		t.Fatalf("latest workflow_operation = %q, want %q", got, sessionWorkflowOperationApprovedImplementation)
	}
	if finalExec.ExecutionState != "completed" {
		t.Fatalf("latest execution_state = %q, want completed", finalExec.ExecutionState)
	}
	if len(finalExec.LinkedApprovalIDs) < 2 {
		t.Fatalf("linked_approval_ids len = %d, want at least 2", len(finalExec.LinkedApprovalIDs))
	}
	return inputSetArtifactDigest, inputSetDigest, finalExec
}

func digestForVerificationSmokeInput(value string) string {
	return artifacts.DigestBytes([]byte(value))
}

func assertVerificationSmokeRunAndArtifactSurfaces(t *testing.T, s *Service, runID string) {
	t.Helper()
	runListResp, errResp := s.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-run-list", Limit: 20}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunList returned error: %+v", errResp)
	}
	if len(runListResp.Runs) < 5 {
		t.Fatalf("run list len = %d, want at least 5 workflow runs", len(runListResp.Runs))
	}
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-run-get", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet returned error: %+v", errResp)
	}
	if got := runGet.Run.Summary.WorkflowKind; got != "builtin_rc_approved_implementation_v0" {
		t.Fatalf("run summary workflow_kind = %q, want builtin_rc_approved_implementation_v0", got)
	}
	artifactListResp, errResp := s.HandleArtifactListV0(context.Background(), LocalArtifactListRequest{SchemaID: "runecode.protocol.v0.ArtifactListRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-artifact-list", RunID: runID, Limit: 20}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleArtifactListV0 returned error: %+v", errResp)
	}
	if len(artifactListResp.Artifacts) == 0 {
		t.Fatal("artifact list empty for final workflow run")
	}
}

func assertVerificationSmokeAuditEvidenceSurfaces(t *testing.T, s *Service, changeDraftDigest, specDraftDigest, inputSetArtifactDigest, inputSetDigest string) {
	t.Helper()
	auditSurface, err := s.LatestAuditVerificationSurface(50)
	if err != nil {
		t.Fatalf("LatestAuditVerificationSurface returned error: %v", err)
	}
	if len(auditSurface.Views) == 0 {
		t.Fatal("latest audit verification surface views empty")
	}
	recordDigest := auditSurface.Views[0].RecordDigest
	assertVerificationSmokeAuditRecordEvidence(t, s, recordDigest)
	assertVerificationSmokeAuditSnapshotEvidence(t, s)
	assertVerificationSmokeOfflineBundleVerification(t, s)
	assertVerificationSmokeAuditEvents(t, s, changeDraftDigest, specDraftDigest, inputSetArtifactDigest, inputSetDigest)
}

func assertVerificationSmokeAuditRecordEvidence(t *testing.T, s *Service, recordDigest trustpolicy.Digest) {
	t.Helper()
	recordGetResp, errResp := s.HandleAuditRecordGet(context.Background(), AuditRecordGetRequest{SchemaID: "runecode.protocol.v0.AuditRecordGetRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-audit-record-get", RecordDigest: recordDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditRecordGet returned error: %+v", errResp)
	}
	if recordGetResp.Record.RecordFamily == "" {
		t.Fatal("audit record detail missing record_family")
	}
	inclusionResp, errResp := s.HandleAuditRecordInclusionGet(context.Background(), AuditRecordInclusionGetRequest{SchemaID: "runecode.protocol.v0.AuditRecordInclusionGetRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-audit-inclusion-get", RecordDigest: recordDigest}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditRecordInclusionGet returned error: %+v", errResp)
	}
	if inclusionResp.Inclusion.SegmentID == "" {
		t.Fatal("audit inclusion missing segment_id")
	}
}

func assertVerificationSmokeAuditSnapshotEvidence(t *testing.T, s *Service) {
	t.Helper()
	snapshotResp, errResp := s.HandleAuditEvidenceSnapshotGet(context.Background(), AuditEvidenceSnapshotGetRequest{SchemaID: "runecode.protocol.v0.AuditEvidenceSnapshotGetRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-audit-snapshot"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceSnapshotGet returned error: %+v", errResp)
	}
	if len(snapshotResp.Snapshot.SegmentSealDigests) == 0 {
		t.Fatal("audit evidence snapshot missing segment_seal_digests")
	}
}

func assertVerificationSmokeOfflineBundleVerification(t *testing.T, s *Service) {
	t.Helper()
	offlineVerifyResp := exportAndVerifyWorkflowSmokeBundle(t, s)
	if offlineVerifyResp.Verification.VerificationStatus == "" {
		t.Fatal("offline verification status empty")
	}
	if len(offlineVerifyResp.Verification.VerificationReports) == 0 {
		t.Fatal("offline verification reports empty")
	}
}

func assertVerificationSmokeAuditEvents(t *testing.T, s *Service, changeDraftDigest, specDraftDigest, inputSetArtifactDigest, inputSetDigest string) {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	if !auditEventContainsValue(events, "runecontext_draft_promote_apply", "draft_artifact_digest", changeDraftDigest) {
		t.Fatalf("draft promote/apply audit event missing change draft digest %q", changeDraftDigest)
	}
	if !auditEventContainsValue(events, "runecontext_draft_promote_apply", "draft_artifact_digest", specDraftDigest) {
		t.Fatalf("draft promote/apply audit event missing spec draft digest %q", specDraftDigest)
	}
	if !auditEventContainsValue(events, "runecontext_approved_implementation_applied", "input_set_artifact_digest", inputSetArtifactDigest) {
		t.Fatalf("approved implementation audit event missing input set artifact digest %q", inputSetArtifactDigest)
	}
	if !auditEventContainsValue(events, "runecontext_approved_implementation_applied", "input_set_digest", inputSetDigest) {
		t.Fatalf("approved implementation audit event missing input set digest %q", inputSetDigest)
	}
}

func exportAndVerifyWorkflowSmokeBundle(t *testing.T, s *Service) AuditEvidenceBundleOfflineVerifyResponse {
	t.Helper()
	exportReq := AuditEvidenceBundleExportRequest{SchemaID: "runecode.protocol.v0.AuditEvidenceBundleExportRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-bundle-export", Scope: AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"}, ExportProfile: "external_relying_party_minimal", CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"}, DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "digest_metadata_only", SelectiveDisclosureApplied: true}, ArchiveFormat: "tar"}
	exportEvents, errResp := s.HandleAuditEvidenceBundleExport(context.Background(), exportReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceBundleExport returned error: %+v", errResp)
	}
	archiveBytes := gatherAuditBundleExportBytes(t, exportEvents)
	if len(archiveBytes) == 0 {
		t.Fatal("bundle export archive bytes empty")
	}
	dir := canonicalTempDir(t)
	bundlePath := filepath.Join(dir, "verification-smoke-bundle.tar")
	if err := os.WriteFile(bundlePath, archiveBytes, 0o600); err != nil {
		t.Fatalf("WriteFile(bundlePath) returned error: %v", err)
	}
	offlineVerifyResp, errResp := s.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{SchemaID: "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest", SchemaVersion: "0.1.0", RequestID: "req-verification-smoke-bundle-offline-verify", BundlePath: bundlePath, ArchiveFormat: "tar"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceBundleOfflineVerify returned error: %+v", errResp)
	}
	return offlineVerifyResp
}
