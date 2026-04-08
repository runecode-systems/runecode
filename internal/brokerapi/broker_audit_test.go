package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestBrokerRejectionPathsAreAudited(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxMessageBytes: 2048, MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})

	_, _ = s.HandleArtifactList(context.Background(), DefaultArtifactListRequest("req-auth"), RequestContext{AdmissionErr: context.Canceled})
	_, _ = s.HandleArtifactHead(context.Background(), ArtifactHeadRequest{SchemaID: "runecode.protocol.v0.BrokerArtifactHeadRequest", SchemaVersion: "0.1.0", RequestID: "req-schema", Digest: "not-a-digest"}, RequestContext{})
	oversized := DefaultArtifactPutRequest("req-size", []byte(strings.Repeat("a", 4000)), "text/plain", "spec_text", "sha256:"+strings.Repeat("1", 64), "workspace", "run-1", "step-1")
	_, _ = s.HandleArtifactPut(context.Background(), oversized, RequestContext{})
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	_, _ = s.HandleArtifactList(context.Background(), DefaultArtifactListRequest("req-limit"), RequestContext{ClientID: "client-a", LaneID: "lane-a"})
	release()

	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}

	assertBrokerRejectionAuditEvent(t, events, "req-auth", "broker_api_auth_admission_denied")
	assertBrokerRejectionAuditEvent(t, events, "req-schema", "broker_validation_schema_invalid")
	assertBrokerRejectionAuditEvent(t, events, "req-size", "broker_limit_message_size_exceeded")
	assertBrokerRejectionAuditEvent(t, events, "req-limit", "broker_limit_in_flight_exceeded")
}

func TestBrokerApprovalResolveAuditsResolution(t *testing.T) {
	s, unapproved, requestEnv, decisionEnv := setupServiceWithApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	resolveReq := brokerAuditResolveRequest(approvalID, unapproved.Digest, requestEnv, decisionEnv)
	if _, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{}); errResp != nil {
		t.Fatalf("HandleApprovalResolve returned error response: %+v", errResp)
	}
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	if !hasBrokerApprovalResolutionEvent(events, "req-approval-audit", approvalID, "approved") {
		t.Fatal("expected broker_approval_resolution audit event for approval resolve")
	}
}

func brokerAuditResolveRequest(approvalID, digest string, requestEnv, decisionEnv *trustpolicy.SignedObjectEnvelope) ApprovalResolveRequest {
	return ApprovalResolveRequest{SchemaID: "runecode.protocol.v0.ApprovalResolveRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-audit", ApprovalID: approvalID, BoundScope: ApprovalBoundScope{SchemaID: "runecode.protocol.v0.ApprovalBoundScope", SchemaVersion: "0.1.0", WorkspaceID: "workspace-1", RunID: "run-approval", StageID: "artifact_flow", ActionKind: "promotion"}, UnapprovedDigest: digest, Approver: "human", RepoPath: "repo/file.txt", Commit: "abc123", ExtractorToolVersion: "tool-v1", FullContentVisible: true, ExplicitViewFull: false, BulkRequest: false, BulkApprovalConfirmed: false, SignedApprovalRequest: *requestEnv, SignedApprovalDecision: *decisionEnv}
}

func hasBrokerApprovalResolutionEvent(events []artifacts.AuditEvent, requestID, approvalID, status string) bool {
	for _, event := range events {
		if event.Type != brokerAuditEventTypeApprovalResolution {
			continue
		}
		if event.Details["request_id"] != requestID || event.Details["approval_id"] != approvalID {
			continue
		}
		return event.Details["approval_status"] == status
	}
	return false
}

func TestBrokerRejectionFailsClosedWhenAuditPersistFails(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	if err := s.store.AppendTrustedAuditEvent("prime", "brokerapi", map[string]interface{}{"x": "y"}); err != nil {
		t.Fatalf("prime audit append returned error: %v", err)
	}
	if err := s.store.SetPolicy(artifacts.Policy{}); err == nil {
		// keep state touched to ensure store writable before forcing failure path
	}

	if s.store != nil {
		s.store = nil
	}
	errResp := s.makeError("req-audit-fail", "broker_limit_in_flight_exceeded", "transport", true, "limit")
	if errResp.Error.Code != "gateway_failure" {
		t.Fatalf("error code = %q, want gateway_failure", errResp.Error.Code)
	}
}

func assertBrokerRejectionAuditEvent(t *testing.T, events []artifacts.AuditEvent, requestID, reasonCode string) {
	t.Helper()
	for _, event := range events {
		if event.Type != brokerAuditEventTypeRejection {
			continue
		}
		if event.Details["request_id"] != requestID {
			continue
		}
		if event.Details["reason_code"] != reasonCode {
			t.Fatalf("reason_code = %v, want %s", event.Details["reason_code"], reasonCode)
		}
		return
	}
	t.Fatalf("missing broker rejection audit event for request_id=%s reason_code=%s", requestID, reasonCode)
}
