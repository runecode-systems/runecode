package brokerapi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

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
	s.now = func() time.Time { return fixedNow }
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
