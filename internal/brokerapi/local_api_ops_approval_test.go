package brokerapi

import (
	"context"
	"errors"
	"testing"
	"time"
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
