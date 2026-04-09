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
	if errResp == nil {
		t.Fatal("HandleRunList error = nil, want typed auth admission error")
	}
	if errResp.Error.Code != "broker_api_auth_admission_denied" {
		t.Fatalf("error code = %q, want broker_api_auth_admission_denied", errResp.Error.Code)
	}
}

func TestApprovalResolveAndAuditReadinessVersionOperations(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)

	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion"}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.ResolutionStatus != "resolved" {
		t.Fatalf("resolution_status = %q, want resolved", resolveResp.ResolutionStatus)
	}
	if resolveResp.Approval.Status != "consumed" {
		t.Fatalf("approval status = %q, want consumed", resolveResp.Approval.Status)
	}
	if resolveResp.ResolutionReasonCode != "approval_consumed" {
		t.Fatalf("resolution_reason_code = %q, want approval_consumed", resolveResp.ResolutionReasonCode)
	}
	if resolveResp.ApprovedArtifact == nil {
		t.Fatal("approved_artifact = nil, want artifact for approve outcome")
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

	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-unknown", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion"}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleApprovalResolve expected broker_approval_state_invalid for unknown decision outcome")
	}
	if errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("error code = %q, want broker_approval_state_invalid", errResp.Error.Code)
	}
}

func TestApprovalResolveSucceedsWhenAuditEmitterUnavailable(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	s.auditor = nil
	approvalID := approvalIDForBrokerTest(t, requestEnv)

	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-no-auditor", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion"}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.Approval.Status != "consumed" {
		t.Fatalf("approval status = %q, want consumed", resolveResp.Approval.Status)
	}
}

func testApprovalResolveNonApproveOutcome(t *testing.T, outcome, wantStatus, wantReasonCode string) {
	t.Helper()
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixtureAndOutcome(t, outcome)
	approvalID := approvalIDForBrokerTest(t, requestEnv)

	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-" + outcome, ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion"}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve(%s) error response: %+v", outcome, errResp)
	}
	if resolveResp.ResolutionStatus != "resolved" {
		t.Fatalf("resolution_status = %q, want resolved", resolveResp.ResolutionStatus)
	}
	if resolveResp.ResolutionReasonCode != wantReasonCode {
		t.Fatalf("resolution_reason_code = %q, want %q", resolveResp.ResolutionReasonCode, wantReasonCode)
	}
	if resolveResp.Approval.Status != wantStatus {
		t.Fatalf("approval status = %q, want %q", resolveResp.Approval.Status, wantStatus)
	}
	if resolveResp.ApprovedArtifact != nil {
		t.Fatalf("approved_artifact = %+v, want nil for %s", resolveResp.ApprovedArtifact, outcome)
	}

	listResp, listErr := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-list-" + outcome, RunID: "run-approval"}, RequestContext{})
	if listErr != nil {
		t.Fatalf("HandleApprovalList(%s) error response: %+v", outcome, listErr)
	}
	if len(listResp.Approvals) != 1 {
		t.Fatalf("approval list len = %d, want 1", len(listResp.Approvals))
	}
	if listResp.Approvals[0].Status != wantStatus {
		t.Fatalf("approval list status = %q, want %q", listResp.Approvals[0].Status, wantStatus)
	}
}

func TestApprovalResolveStageSummarySignOffConsumesCurrentBinding(t *testing.T) {
	s, requestEnv, decisionEnv := setupServiceWithStageSignOffApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)

	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-stage-signoff-resolve", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-stage"), RunID: "run-stage", StageID: "stage-1", ActionKind: "stage_summary_sign_off"}, UnapprovedDigest: "sha256:" + strings.Repeat("d", 64), Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.Approval.Status != "consumed" {
		t.Fatalf("approval status = %q, want consumed", resolveResp.Approval.Status)
	}
	if resolveResp.ResolutionReasonCode != "approval_consumed" {
		t.Fatalf("resolution_reason_code = %q, want approval_consumed", resolveResp.ResolutionReasonCode)
	}
	if resolveResp.ApprovedArtifact != nil {
		t.Fatalf("approved_artifact = %+v, want nil for stage sign-off", resolveResp.ApprovedArtifact)
	}
}

func TestApprovalResolveStageSummarySignOffSupersededWhenNewerPendingExists(t *testing.T) {
	s, oldRequestEnv, oldDecisionEnv, newApprovalID := setupServiceWithSupersededStageSignOffApprovals(t)
	oldApprovalID := approvalIDForBrokerTest(t, oldRequestEnv)

	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-stage-signoff-superseded", ApprovalID: oldApprovalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-stage"), RunID: "run-stage", StageID: "stage-1", ActionKind: "stage_summary_sign_off"}, UnapprovedDigest: "sha256:" + strings.Repeat("d", 64), Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *oldRequestEnv, SignedApprovalDecision: *oldDecisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.ResolutionStatus != "no_change" {
		t.Fatalf("resolution_status = %q, want no_change", resolveResp.ResolutionStatus)
	}
	if resolveResp.ResolutionReasonCode != "approval_superseded" {
		t.Fatalf("resolution_reason_code = %q, want approval_superseded", resolveResp.ResolutionReasonCode)
	}
	if resolveResp.Approval.Status != "superseded" {
		t.Fatalf("approval status = %q, want superseded", resolveResp.Approval.Status)
	}
	if resolveResp.Approval.SupersededByApprovalID != newApprovalID {
		t.Fatalf("superseded_by_approval_id = %q, want %q", resolveResp.Approval.SupersededByApprovalID, newApprovalID)
	}
}

func TestApprovalResolveRejectsTerminalReresolution(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixtureAndOutcome(t, "deny")
	approvalID := approvalIDForBrokerTest(t, requestEnv)

	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve-initial", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: workspaceIDForRun("run-approval"), RunID: "run-approval", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion"}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	if _, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{}); errResp != nil {
		t.Fatalf("initial HandleApprovalResolve error response: %+v", errResp)
	}

	resolveReq.RequestID = "req-approval-resolve-reresolve"
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleApprovalResolve re-resolve expected terminal state error")
	}
	if errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("error code = %q, want broker_approval_state_invalid", errResp.Error.Code)
	}
}

func TestApprovalResolveRejectsWhenStoredBoundScopeFieldsOmittedByRequest(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)

	resolveReq := ApprovalResolveRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-approval-resolve-scope-omitted",
		ApprovalID:    approvalID,
		BoundScope: ApprovalBoundScope{
			SchemaID:      "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion: "0.1.0",
			ActionKind:    "promotion",
		},
		UnapprovedDigest:       unapproved.Digest,
		Approver:               "human",
		RepoPath:               "repo/file.txt",
		Commit:                 "abc123",
		ExtractorToolVersion:   "tool-v1",
		FullContentVisible:     true,
		ExplicitViewFull:       false,
		BulkRequest:            false,
		BulkApprovalConfirmed:  false,
		SignedApprovalRequest:  *requestEnv,
		SignedApprovalDecision: *decisionEnv,
	}

	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleApprovalResolve expected bound scope mismatch when stored fields are omitted")
	}
	if errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("error code = %q, want broker_approval_state_invalid", errResp.Error.Code)
	}
	if !strings.Contains(errResp.Error.Message, "bound_scope.workspace_id") {
		t.Fatalf("error message = %q, want workspace_id mismatch", errResp.Error.Message)
	}
}

func TestHandleApprovalListRejectsInFlightLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	defer release()
	_, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-approval-list-limit",
	}, RequestContext{ClientID: "client-a", LaneID: "lane-a"})
	if errResp == nil {
		t.Fatal("HandleApprovalList expected in-flight limit error")
	}
	if errResp.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("error code = %q, want broker_limit_in_flight_exceeded", errResp.Error.Code)
	}
}

func TestHandleApprovalListRejectsDeadlineExceeded(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	deadline := time.Now().Add(-time.Second)
	_, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-approval-list-timeout",
	}, RequestContext{Deadline: &deadline})
	if errResp == nil {
		t.Fatal("HandleApprovalList expected deadline error")
	}
	if errResp.Error.Code != "broker_timeout_request_deadline_exceeded" {
		t.Fatalf("error code = %q, want broker_timeout_request_deadline_exceeded", errResp.Error.Code)
	}
}

func TestApprovalListDerivesPendingFromUnapprovedArtifacts(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref := putUnapprovedExcerptArtifactForApprovalTest(t, s, "run-approval-derived", "step-1", "a")
	approvalID := createPendingApprovalFromPolicyDecision(t, s, "run-approval-derived", "step-1", ref.Digest)

	resp, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-derived-approval-list",
		RunID:         "run-approval-derived",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalList error response: %+v", errResp)
	}
	assertDerivedPendingApproval(t, s, resp.Approvals, "run-approval-derived", "step-1", approvalID)
}

func putUnapprovedExcerptArtifactForApprovalTest(t *testing.T, s *Service, runID, stepID, hashFill string) artifacts.ArtifactReference {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               []byte("private excerpt"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat(hashFill, 64),
		CreatedByRole:         "workspace",
		RunID:                 runID,
		StepID:                stepID,
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return ref
}

func assertDerivedPendingApproval(t *testing.T, s *Service, approvals []ApprovalSummary, runID, stepID, approvalID string) {
	t.Helper()
	if len(approvals) != 1 {
		t.Fatalf("approval count = %d, want 1", len(approvals))
	}
	approval := approvals[0]
	if approval.Status != "pending" {
		t.Fatalf("approval status = %q, want pending", approval.Status)
	}
	if approval.ApprovalTriggerCode != "excerpt_promotion" {
		t.Fatalf("approval trigger = %q, want excerpt_promotion", approval.ApprovalTriggerCode)
	}
	if approval.BoundScope.RunID != runID {
		t.Fatalf("bound scope run_id = %q, want %s", approval.BoundScope.RunID, runID)
	}
	if approval.BoundScope.StepID != stepID {
		t.Fatalf("bound scope step_id = %q, want %s", approval.BoundScope.StepID, stepID)
	}
	if approval.ApprovalID != approvalID {
		t.Fatalf("approval id = %q, want %q", approval.ApprovalID, approvalID)
	}
}

func TestApprovalGetReturnsDerivedPendingApproval(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	approvalID := createPendingApprovalForGetTest(t, s)
	resp, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-derived-approval-get",
		ApprovalID:    approvalID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", errResp)
	}
	if resp.Approval.ApprovalID != approvalID {
		t.Fatalf("approval id = %q, want %q", resp.Approval.ApprovalID, approvalID)
	}
	if resp.SignedApprovalRequest == nil {
		t.Fatal("pending approval should include signed approval request envelope")
	}
	if resp.SignedApprovalDecision != nil {
		t.Fatal("pending approval should not include signed approval decision envelope")
	}
	derivedID, deriveErr := approvalIDFromRequest(*resp.SignedApprovalRequest)
	if deriveErr != nil {
		t.Fatalf("approvalIDFromRequest(signed_approval_request) returned error: %v", deriveErr)
	}
	if derivedID != approvalID {
		t.Fatalf("derived approval id = %q, want %q", derivedID, approvalID)
	}
}

func createPendingApprovalForGetTest(t *testing.T, s *Service) string {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64), CreatedByRole: "workspace", RunID: "run-approval-get"})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return createPendingApprovalFromPolicyDecision(t, s, "run-approval-get", "", ref.Digest)
}

func TestHandleApprovalGetRejectsInFlightLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	defer release()

	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-approval-get-limit",
		ApprovalID:    "sha256:" + strings.Repeat("a", 64),
	}, RequestContext{ClientID: "client-a", LaneID: "lane-a"})
	if errResp == nil {
		t.Fatal("HandleApprovalGet expected in-flight limit error")
	}
	if errResp.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("error code = %q, want broker_limit_in_flight_exceeded", errResp.Error.Code)
	}
}

func TestHandleApprovalGetRejectsDeadlineExceeded(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	deadline := time.Now().Add(-time.Second)
	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-approval-get-timeout",
		ApprovalID:    "sha256:" + strings.Repeat("a", 64),
	}, RequestContext{Deadline: &deadline})
	if errResp == nil {
		t.Fatal("HandleApprovalGet expected deadline error")
	}
	if errResp.Error.Code != "broker_timeout_request_deadline_exceeded" {
		t.Fatalf("error code = %q, want broker_timeout_request_deadline_exceeded", errResp.Error.Code)
	}
}

func TestHandleApprovalGetUsesNotFoundApprovalCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-approval-get-missing",
		ApprovalID:    "sha256:" + strings.Repeat("f", 64),
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleApprovalGet expected not-found error")
	}
	if errResp.Error.Code != "broker_not_found_approval" {
		t.Fatalf("error code = %q, want broker_not_found_approval", errResp.Error.Code)
	}
}
