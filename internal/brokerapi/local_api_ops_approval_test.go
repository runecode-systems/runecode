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

	resolveReq := ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-resolve", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: "workspace-1", RunID: "run-approval", StageID: "artifact_flow", ActionKind: "excerpt_promotion"}, UnapprovedDigest: unapproved.Digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
	resolveResp, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", errResp)
	}
	if resolveResp.ResolutionStatus != "resolved" {
		t.Fatalf("resolution_status = %q, want resolved", resolveResp.ResolutionStatus)
	}

	assertApprovalAndAuditReadEndpoints(t, s, approvalID)
	assertVersionAndLogEndpoints(t, s)
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
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               []byte("private excerpt"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64),
		CreatedByRole:         "workspace",
		RunID:                 "run-approval-derived",
		StepID:                "step-1",
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	resp, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-derived-approval-list",
		RunID:         "run-approval-derived",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalList error response: %+v", errResp)
	}
	if len(resp.Approvals) != 1 {
		t.Fatalf("approval count = %d, want 1", len(resp.Approvals))
	}
	approval := resp.Approvals[0]
	if approval.Status != "pending" {
		t.Fatalf("approval status = %q, want pending", approval.Status)
	}
	if approval.ApprovalTriggerCode != "excerpt_promotion" {
		t.Fatalf("approval trigger = %q, want excerpt_promotion", approval.ApprovalTriggerCode)
	}
	if approval.BoundScope.RunID != "run-approval-derived" {
		t.Fatalf("bound scope run_id = %q, want run-approval-derived", approval.BoundScope.RunID)
	}
	if approval.BoundScope.StepID != "step-1" {
		t.Fatalf("bound scope step_id = %q, want step-1", approval.BoundScope.StepID)
	}
	expectedID := shaDigestIdentity("pending-approval:" + ref.Digest)
	if approval.ApprovalID != expectedID {
		t.Fatalf("approval id = %q, want %q", approval.ApprovalID, expectedID)
	}
}

func TestApprovalGetReturnsDerivedPendingApproval(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               []byte("private excerpt"),
		ContentType:           "text/plain",
		DataClass:             artifacts.DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64),
		CreatedByRole:         "workspace",
		RunID:                 "run-approval-get",
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	approvalID := shaDigestIdentity("pending-approval:" + ref.Digest)

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
	if resp.SignedApprovalRequest != nil || resp.SignedApprovalDecision != nil {
		t.Fatal("derived pending approval should not include signed request/decision envelopes")
	}
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
