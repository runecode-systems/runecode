package brokerapi

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestExternalAnchorMutationExecuteFailsClosedOnApprovalBindingMismatch(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareExternalAnchorExecuteFixture(t, "run-anchor-execute-mismatch", "sha256:"+strings.Repeat("7", 64))
	wrongRequest := trustpolicy.Digest{HashAlg: requestDigest.HashAlg, Hash: strings.Repeat("a", len(requestDigest.Hash))}
	errResp := executeExternalAnchorMutationError(t, s, preparedID, approvalID, wrongRequest, decisionDigest, leaseID, "req-anchor-execute-mismatch")
	assertExternalAnchorExecuteError(t, errResp, "broker_approval_state_invalid", "approval_request_hash")
	assertExternalAnchorNotStartedState(t, s, preparedID)
}

func TestExternalAnchorMutationExecuteFailsClosedWhenStoredTypedRequestHashDrifts(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareExternalAnchorExecuteFixture(t, "run-anchor-execute-drift", "sha256:"+strings.Repeat("8", 64))
	tamperExternalAnchorPreparedRecord(t, s, preparedID, func(rec *artifacts.ExternalAnchorPreparedMutationRecord) {
		rec.TypedRequest["outbound_payload_digest"] = digestObject("sha256:" + strings.Repeat("e", 64))
	})
	errResp := executeExternalAnchorMutationError(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-drift")
	assertExternalAnchorExecuteError(t, errResp, "broker_approval_state_invalid", "stored typed request hash")
	assertExternalAnchorNotStartedState(t, s, preparedID)
	if rec, ok := s.ExternalAnchorPreparedGet(preparedID); !ok || rec.LastExecuteRequestID != "" {
		t.Fatalf("last_execute_request_id=%q, want empty on fail-closed pre-execution failure", rec.LastExecuteRequestID)
	}
}

func TestExternalAnchorMutationExecuteFailsClosedOnTargetIdentityBindingDrift(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareExternalAnchorExecuteFixture(t, "run-anchor-execute-target-drift", "sha256:"+strings.Repeat("9", 64))
	tamperExternalAnchorPreparedRecord(t, s, preparedID, func(rec *artifacts.ExternalAnchorPreparedMutationRecord) {
		rec.TypedRequest["target_descriptor_digest"] = digestObject("sha256:" + strings.Repeat("f", 64))
		rec.TypedRequestHash = mustCanonicalExternalAnchorTypedRequestHash(t, rec.TypedRequest)
	})
	errResp := executeExternalAnchorMutationError(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-target-drift")
	assertExternalAnchorExecuteError(t, errResp, "broker_approval_state_invalid", "target descriptor identity")
}

func TestExternalAnchorMutationExecuteRequiresValidTargetAuthLeasePosture(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, _ := prepareExternalAnchorExecuteFixture(t, "run-anchor-execute-lease-posture", "sha256:"+strings.Repeat("6", 64))
	errResp := executeExternalAnchorMutationError(t, s, preparedID, approvalID, requestDigest, decisionDigest, "", "req-anchor-execute-missing-lease")
	assertExternalAnchorExecuteError(t, errResp, "broker_validation_schema_invalid", "target_auth_lease_id")
	invalidLeaseID := mustIssueExternalAnchorNonGatewayLease(t, s, "run-anchor-execute-lease-posture")
	errResp = executeExternalAnchorMutationError(t, s, preparedID, approvalID, requestDigest, decisionDigest, invalidLeaseID, "req-anchor-execute-invalid-lease")
	assertExternalAnchorExecuteError(t, errResp, "broker_approval_state_invalid", "target auth lease retrieval failed")
}

func assertExternalAnchorExecuteError(t *testing.T, errResp *ErrorResponse, wantCode, wantMessage string) {
	t.Helper()
	if errResp == nil {
		t.Fatal("expected execute error")
	}
	if errResp.Error.Code != wantCode {
		t.Fatalf("error.code=%q, want %q", errResp.Error.Code, wantCode)
	}
	if !strings.Contains(errResp.Error.Message, wantMessage) {
		t.Fatalf("error.message=%q, want substring %q", errResp.Error.Message, wantMessage)
	}
}
