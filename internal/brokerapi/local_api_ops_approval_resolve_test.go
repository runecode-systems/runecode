package brokerapi

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func digestForBrokerTest(seed string) string {
	base := strings.Repeat(seed, 64)
	if len(base) > 64 {
		base = base[:64]
	}
	for len(base) < 64 {
		base += "0"
	}
	return "sha256:" + base
}

func TestHandleRunListRejectsAdmissionFailure(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-admission", Limit: 10}, RequestContext{AdmissionErr: errors.New("peer credentials unavailable")})
	if errResp == nil || errResp.Error.Code != "broker_api_auth_admission_denied" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalResolvePersistsRunnerApprovalWaitHintWithSupersession(t *testing.T) {
	s, oldRequestEnv, oldDecisionEnv, newApprovalID := setupServiceWithSupersededStageSignOffApprovals(t)
	oldApprovalID := approvalIDForBrokerTest(t, oldRequestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, oldApprovalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-stage-signoff-hint", ApprovalID: oldApprovalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-stage"), RunID: "run-stage", StageID: "stage-1", ActionKind: "stage_summary_sign_off", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: "sha256:" + strings.Repeat("d", 64), Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *oldRequestEnv, SignedApprovalDecision: *oldDecisionEnv}
	if _, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{}); errResp != nil {
		t.Fatalf("HandleApprovalResolve returned error: %+v", errResp)
	}
	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-stage-hint", RunID: "run-stage"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	waits, ok := runResp.Run.AdvisoryState["approval_waits"].(map[string]artifacts.RunnerApproval)
	if !ok {
		t.Fatalf("advisory_state.approval_waits = %T, want map[string]artifacts.RunnerApproval", runResp.Run.AdvisoryState["approval_waits"])
	}
	first := waits[oldApprovalID]
	if first.Status != "superseded" || first.SupersededByApproval != newApprovalID {
		t.Fatalf("unexpected supersession binding: %+v", first)
	}
}

func TestApprovalResolveAndAuditReadinessVersionOperations(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.ResolutionStatus != "resolved" || resolveResp.Approval.Status != "consumed" || resolveResp.ResolutionReasonCode != "approval_consumed" || resolveResp.ApprovedArtifact == nil {
		t.Fatalf("unexpected resolve response: %+v", resolveResp)
	}
	assertApprovalAndAuditReadEndpoints(t, s, approvalID)
	assertVersionAndLogEndpoints(t, s)
}

func TestApprovalResolveDenyDoesNotPromote(t *testing.T) {
	testApprovalResolveNonApproveOutcome(t, "deny", "denied", "approval_denied")
}
func TestApprovalResolveExpiredDoesNotPromote(t *testing.T) {
	testApprovalResolveNonApproveOutcome(t, "expired", "expired", "approval_expired")
}
func TestApprovalResolveCancelledDoesNotPromote(t *testing.T) {
	testApprovalResolveNonApproveOutcome(t, "cancelled", "cancelled", "approval_cancelled")
}

func TestApprovalResolveRejectsUnknownDecisionOutcome(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixtureAndOutcome(t, "not_supported")
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-unknown", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalResolveFailsClosedWhenCanonicalMirrorAtomicWriteFails(t *testing.T) {
	s, root := newApprovalResolveAtomicFailureService(t)
	approvalID, unapprovedDigest, requestEnv, decisionEnv := seedAtomicFailureApprovalFixture(t, s)
	breakRunnerSnapshotPathForAtomicFailure(t, root)
	resolveReq := atomicFailureResolveRequest(t, s, approvalID, unapprovedDigest, requestEnv, decisionEnv)
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_storage_write_failed" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
	stored := mustApprovalGet(t, s, approvalID)
	if stored.Status != "pending" {
		t.Fatalf("approval status = %q, want pending after rollback", stored.Status)
	}
}

func newApprovalResolveAtomicFailureService(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	ledgerRoot := root + "/audit-ledger"
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	s, err := NewServiceWithConfig(root, ledgerRoot, APIConfig{})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	return s, root
}

func seedAtomicFailureApprovalFixture(t *testing.T, s *Service) (string, string, *trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	unapprovedDigest := "sha256:" + strings.Repeat("f", 64)
	if _, putErr := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64), CreatedByRole: "workspace", RunID: "run-approval", StepID: "step-1"}); putErr != nil {
		t.Fatalf("Put unapproved returned error: %v", putErr)
	}
	actionHash := promotionActionHashForBrokerTests(unapprovedDigest, "repo/file.txt", "abc123", "tool-v1", "human")
	requestEnv, decisionEnv, verifiers := signedApprovalArtifactsForBrokerTestsWithOutcome(t, "human", unapprovedDigest, "deny")
	for _, verifier := range verifiers {
		if putErr := putTrustedVerifierRecordForService(s, verifier); putErr != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", putErr)
		}
	}
	approvalID, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	approvalPayload := decodeApprovalPayloadMap(requestEnv.Payload)
	now := time.Now().UTC()
	if err := s.RecordPolicyDecision("run-approval", "", policyengine.PolicyDecision{SchemaID: "runecode.protocol.v0.PolicyDecision", SchemaVersion: "0.3.0", DecisionOutcome: policyengine.DecisionRequireHumanApproval, PolicyReasonCode: "approval_required", ManifestHash: digestFromPayloadField(approvalPayload, "manifest_hash"), ActionRequestHash: actionHash, PolicyInputHashes: []string{digestForBrokerTest("4")}, RelevantArtifactHashes: []string{unapprovedDigest}, DetailsSchemaID: "runecode.protocol.details.policy.evaluation.v0", Details: map[string]any{"precedence": "approval_profile_moderate"}, RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.moderate.workspace_write.v0", RequiredApproval: map[string]any{"approval_trigger_code": "excerpt_promotion", "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "scope": map[string]any{"schema_id": "runecode.protocol.v0.ApprovalBoundScope", "schema_version": "0.1.0", "workspace_id": workspaceIDForRun("run-approval"), "run_id": "run-approval", "stage_id": "artifact_flow", "step_id": "step-1", "action_kind": "promotion"}, "changes_if_approved": "Promote reviewed file excerpts for downstream use.", "approval_ttl_seconds": 1200}}); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	if err := s.RecordApproval(artifacts.ApprovalRecord{ApprovalID: approvalID, Status: "pending", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", RequestedAt: now, ExpiresAt: func() *time.Time { t := now.Add(time.Hour); return &t }(), ApprovalTriggerCode: "excerpt_promotion", ChangesIfApproved: approvalChangesIfApprovedDefault, ApprovalAssuranceLevel: "reauthenticated", PresenceMode: "hardware_touch", PolicyDecisionHash: "", ManifestHash: digestFromPayloadField(approvalPayload, "manifest_hash"), ActionRequestHash: actionHash, RelevantArtifactHashes: []string{unapprovedDigest}, SourceDigest: unapprovedDigest, RequestDigest: approvalID, RequestEnvelope: requestEnv}); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return approvalID, unapprovedDigest, requestEnv, decisionEnv
}

func breakRunnerSnapshotPathForAtomicFailure(t *testing.T, root string) {
	t.Helper()
	if err := os.Remove(root + "/" + "runner_state.snapshot.json"); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove runner snapshot returned error: %v", err)
	}
	if err := os.Mkdir(root+"/"+"runner_state.snapshot.json", 0o700); err != nil {
		t.Fatalf("mkdir runner snapshot path returned error: %v", err)
	}
}

func atomicFailureResolveRequest(t *testing.T, s *Service, approvalID, unapprovedDigest string, requestEnv, decisionEnv *trustpolicy.SignedObjectEnvelope) ApprovalResolveRequest {
	t.Helper()
	storedApproval := mustApprovalGet(t, s, approvalID)
	return ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-atomic-failure", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: storedApproval.PolicyDecisionHash}, UnapprovedDigest: unapprovedDigest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
}

func mustApprovalGet(t *testing.T, s *Service, approvalID string) artifacts.ApprovalRecord {
	t.Helper()
	storedApproval, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing approval", approvalID)
	}
	return storedApproval
}

func TestApprovalResolveFailsWhenAuditEmitterUnavailable(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	s.auditor = nil
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-no-auditor", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "gateway_failure" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalResolveRejectsApproverPrincipalMismatch(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-approver-mismatch", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapproved.Digest, Approver: "different-human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalResolveUsesSingleCapturedTimeForDecidedAndConsumed(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	fixedNow := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return fixedNow })
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-time-binding", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	want := fixedNow.Format(time.RFC3339)
	if resolveResp.Approval.DecidedAt != want || resolveResp.Approval.ConsumedAt != want {
		t.Fatalf("unexpected decided/consumed timestamps: %+v", resolveResp.Approval)
	}
}

func TestApprovalResolveTreatsExpiryBoundaryAsExpiredAndMatchesApprovalGetParity(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	stored := mustApprovalGet(t, s, approvalID)
	fixedNow := time.Date(2026, 4, 10, 12, 30, 0, 0, time.UTC)
	stored.ExpiresAt = &fixedNow
	if err := s.RecordApproval(stored); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	s.SetNowFuncForTests(func() time.Time { return fixedNow })

	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-expiry-boundary", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}

	updated := mustApprovalGet(t, s, approvalID)
	if updated.Status != "expired" {
		t.Fatalf("approval status = %q, want expired", updated.Status)
	}
}

func TestApprovalResolveRedactsApprovalWaitBindingHashesFromRunDetail(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 10, 13, 0, 0, 0, time.UTC)
	if err := s.RecordRunnerApprovalWait(artifacts.RunnerApproval{ApprovalID: "sha256:" + strings.Repeat("a", 64), RunID: "run-redact", StageID: "stage-1", StepID: "step-1", RoleInstanceID: "role-1", Status: "pending", ApprovalType: "exact_action", BoundActionHash: "sha256:" + strings.Repeat("b", 64), OccurredAt: now}); err != nil {
		t.Fatalf("RecordRunnerApprovalWait returned error: %v", err)
	}
	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-redact", RunID: "run-redact"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	waits, ok := runResp.Run.AdvisoryState["approval_waits"].(map[string]artifacts.RunnerApproval)
	if !ok || waits["sha256:"+strings.Repeat("a", 64)].BoundActionHash != "" {
		t.Fatalf("unexpected waits payload: %#v", runResp.Run.AdvisoryState["approval_waits"])
	}
}

func testApprovalResolveNonApproveOutcome(t *testing.T, outcome, wantStatus, wantReasonCode string) {
	t.Helper()
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixtureAndOutcome(t, outcome)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-" + outcome, ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve(%s) error response: %+v", outcome, errResp)
	}
	if resolveResp.ResolutionStatus != "resolved" || resolveResp.ResolutionReasonCode != wantReasonCode || resolveResp.Approval.Status != wantStatus || resolveResp.ApprovedArtifact != nil {
		t.Fatalf("unexpected resolve response: %+v", resolveResp)
	}
	listResp, listErr := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-list-" + outcome, RunID: "run-approval"}, RequestContext{})
	if listErr != nil || len(listResp.Approvals) != 1 || listResp.Approvals[0].Status != wantStatus {
		t.Fatalf("unexpected list response: resp=%+v err=%+v", listResp, listErr)
	}
}

func TestApprovalResolveStageSummarySignOffConsumesCurrentBinding(t *testing.T) {
	s, requestEnv, decisionEnv := setupServiceWithStageSignOffApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-stage-signoff-resolve", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-stage"), RunID: "run-stage", StageID: "stage-1", ActionKind: "stage_summary_sign_off", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: "sha256:" + strings.Repeat("d", 64), Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil || resolveResp.Approval.Status != "consumed" || resolveResp.ResolutionReasonCode != "approval_consumed" || resolveResp.ApprovedArtifact != nil {
		t.Fatalf("unexpected resolve response: resp=%+v err=%+v", resolveResp, errResp)
	}
}

func TestApprovalResolveStageSummarySignOffSupersededWhenNewerPendingExists(t *testing.T) {
	s, oldRequestEnv, oldDecisionEnv, newApprovalID := setupServiceWithSupersededStageSignOffApprovals(t)
	oldApprovalID := approvalIDForBrokerTest(t, oldRequestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, oldApprovalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-stage-signoff-superseded", ApprovalID: oldApprovalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-stage"), RunID: "run-stage", StageID: "stage-1", ActionKind: "stage_summary_sign_off", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: "sha256:" + strings.Repeat("d", 64), Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *oldRequestEnv, SignedApprovalDecision: *oldDecisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil || resolveResp.ResolutionStatus != "no_change" || resolveResp.ResolutionReasonCode != "approval_superseded" || resolveResp.Approval.Status != "superseded" || resolveResp.Approval.SupersededByApprovalID != newApprovalID {
		t.Fatalf("unexpected resolve response: resp=%+v err=%+v", resolveResp, errResp)
	}
}

func TestApprovalResolveStageSummarySignOffSupersededWhenPlanBindingChanges(t *testing.T) {
	s, oldRequestEnv, oldDecisionEnv, newApprovalID := setupServiceWithPlanScopedSupersededStageSignOffApprovals(t)
	oldApprovalID := approvalIDForBrokerTest(t, oldRequestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, oldApprovalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-stage-signoff-superseded-plan", ApprovalID: oldApprovalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-stage"), RunID: "run-stage", StageID: "stage-1", ActionKind: "stage_summary_sign_off", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: "sha256:" + strings.Repeat("d", 64), Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *oldRequestEnv, SignedApprovalDecision: *oldDecisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil || resolveResp.ResolutionStatus != "no_change" || resolveResp.ResolutionReasonCode != "approval_superseded" || resolveResp.Approval.Status != "superseded" || resolveResp.Approval.SupersededByApprovalID != newApprovalID {
		t.Fatalf("unexpected resolve response: resp=%+v err=%+v", resolveResp, errResp)
	}
}

func TestApprovalResolveBackendPostureConsumesViaGenericExactActionPath(t *testing.T) {
	s, requestEnv, decisionEnv := setupServiceWithBackendPostureApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-backend-posture-resolve", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-backend"), InstanceID: "launcher-instance-1", RunID: "run-backend", ActionKind: policyengine.ActionKindBackendPosture, PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: "sha256:" + strings.Repeat("d", 64), Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.Approval.Status != "consumed" || resolveResp.ResolutionReasonCode != "approval_consumed" || resolveResp.ApprovedArtifact != nil {
		t.Fatalf("unexpected resolve response: %+v", resolveResp)
	}
}

func TestApprovalResolveBackendPostureStaysDistinctFromStageSupersessionSemantics(t *testing.T) {
	s, oldRequestEnv, oldDecisionEnv := setupServiceWithBackendPostureApprovalFixture(t)
	newRequestEnv, _, _ := signedBackendPostureApprovalArtifactsForBrokerTests(t, "human", "container", "explicit_selection", "select_backend", "reduce_assurance", "exact_action_approval", "approve")
	newApprovalID := seedPendingBackendPostureApprovalForSignedRequest(t, s, *newRequestEnv)
	if newApprovalID == "" {
		t.Fatal("expected second pending backend-posture approval")
	}
	oldApprovalID := approvalIDForBrokerTest(t, oldRequestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, oldApprovalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-backend-posture-not-stage-superseded", ApprovalID: oldApprovalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-backend"), InstanceID: "launcher-instance-1", RunID: "run-backend", ActionKind: policyengine.ActionKindBackendPosture, PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: "sha256:" + strings.Repeat("d", 64), Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *oldRequestEnv, SignedApprovalDecision: *oldDecisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.ResolutionStatus != "resolved" || resolveResp.ResolutionReasonCode != "approval_consumed" || resolveResp.Approval.Status != "consumed" || resolveResp.Approval.SupersededByApprovalID != "" {
		t.Fatalf("unexpected resolve response: %+v", resolveResp)
	}
}

func TestApprovalResolveRejectsTerminalReresolution(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixtureAndOutcome(t, "deny")
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-initial", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	if _, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{}); errResp != nil {
		t.Fatalf("initial HandleApprovalResolve error response: %+v", errResp)
	}
	resolveReq.RequestID = "req-approval-resolve-reresolve"
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalResolveRejectsWhenStoredBoundScopeFieldsOmittedByRequest(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-scope-omitted", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", ActionKind: "promotion"}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_approval_state_invalid" || !strings.Contains(errResp.Error.Message, "bound_scope.workspace_id") {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalResolveRejectsMismatchedUnapprovedDigestBinding(t *testing.T) {
	s, resolveReq := mismatchedResolveRequest(t, "sha256:"+strings.Repeat("f", 64), "req-approval-resolve-wrong-source", "")
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_approval_state_invalid" || !strings.Contains(errResp.Error.Message, "unapproved_digest") {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalResolveRejectsWhenApprovalIDDoesNotMatchSignedRequest(t *testing.T) {
	s, resolveReq := mismatchedResolveRequest(t, "", "req-approval-resolve-wrong-id", "sha256:"+strings.Repeat("9", 64))
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_approval_state_invalid" || !strings.Contains(errResp.Error.Message, "approval_id does not match") {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func mismatchedResolveRequest(t *testing.T, unapprovedOverride, requestID, approvalIDOverride string) (*Service, ApprovalResolveRequest) {
	t.Helper()
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	if unapprovedOverride == "" {
		unapprovedOverride = unapproved.Digest
	}
	if approvalIDOverride != "" {
		approvalID = approvalIDOverride
	}
	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: requestID, ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", PolicyDecisionHash: policyDecisionHash}, UnapprovedDigest: unapprovedOverride, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	return s, resolveReq
}
