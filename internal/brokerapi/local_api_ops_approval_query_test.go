package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestHandleApprovalListRejectsInFlightLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	defer release()
	_, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-list-limit"}, RequestContext{ClientID: "client-a", LaneID: "lane-a"})
	if errResp == nil || errResp.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestHandleApprovalListRejectsDeadlineExceeded(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	deadline := time.Now().Add(-time.Second)
	_, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-list-timeout"}, RequestContext{Deadline: &deadline})
	if errResp == nil || errResp.Error.Code != "broker_timeout_request_deadline_exceeded" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalListDerivesPendingFromUnapprovedArtifacts(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref := putUnapprovedExcerptArtifactForApprovalTest(t, s, "run-approval-derived", "step-1", "a")
	approvalID := createPendingApprovalFromPolicyDecision(t, s, "run-approval-derived", "step-1", ref.Digest)
	resp, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-derived-approval-list", RunID: "run-approval-derived"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalList error response: %+v", errResp)
	}
	assertDerivedPendingApproval(t, resp.Approvals, "run-approval-derived", "step-1", approvalID)
}

func putUnapprovedExcerptArtifactForApprovalTest(t *testing.T, s *Service, runID, stepID, hashFill string) artifacts.ArtifactReference {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat(hashFill, 64), CreatedByRole: "workspace", RunID: runID, StepID: stepID})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return ref
}

func assertDerivedPendingApproval(t *testing.T, approvals []ApprovalSummary, runID, stepID, approvalID string) {
	t.Helper()
	if len(approvals) != 1 {
		t.Fatalf("approval count = %d, want 1", len(approvals))
	}
	approval := approvals[0]
	if approval.Status != "pending" || approval.ApprovalTriggerCode != "excerpt_promotion" || approval.BoundScope.RunID != runID || approval.BoundScope.StepID != stepID || approval.ApprovalID != approvalID {
		t.Fatalf("unexpected approval summary: %+v", approval)
	}
}

func TestApprovalGetReturnsDerivedPendingApproval(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	approvalID := createPendingApprovalForGetTest(t, s)
	resp, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-derived-approval-get", ApprovalID: approvalID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", errResp)
	}
	if resp.Approval.ApprovalID != approvalID || resp.SignedApprovalRequest == nil || resp.SignedApprovalDecision != nil {
		t.Fatalf("unexpected approval get response: %+v", resp)
	}
	derivedID, deriveErr := approvalIDFromRequest(*resp.SignedApprovalRequest)
	if deriveErr != nil || derivedID != approvalID {
		t.Fatalf("unexpected approvalIDFromRequest output: id=%q err=%v", derivedID, deriveErr)
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
	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-limit", ApprovalID: "sha256:" + strings.Repeat("a", 64)}, RequestContext{ClientID: "client-a", LaneID: "lane-a"})
	if errResp == nil || errResp.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestHandleApprovalGetRejectsDeadlineExceeded(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	deadline := time.Now().Add(-time.Second)
	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-timeout", ApprovalID: "sha256:" + strings.Repeat("a", 64)}, RequestContext{Deadline: &deadline})
	if errResp == nil || errResp.Error.Code != "broker_timeout_request_deadline_exceeded" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestHandleApprovalGetUsesNotFoundApprovalCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-missing", ApprovalID: "sha256:" + strings.Repeat("f", 64)}, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_not_found_approval" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}
