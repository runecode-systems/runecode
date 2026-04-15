package brokerapi

import (
	"context"
	"strings"
	"testing"
)

func TestHandleAuditAnchorSegmentReturnsSignerUnavailableWhenSignerMissing(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	service.secretsSvc = nil
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-signer-missing",
		SealDigest:    sealDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error response: %+v", errResp)
	}
	if resp.AnchoringStatus != "failed" {
		t.Fatalf("anchoring_status = %q, want failed", resp.AnchoringStatus)
	}
	if resp.FailureCode != "anchor_signer_unavailable" {
		t.Fatalf("failure_code = %q, want anchor_signer_unavailable", resp.FailureCode)
	}
	if got := strings.TrimSpace(resp.FailureMessage); got != "audit anchor signer unavailable" {
		t.Fatalf("failure_message = %q, want audit anchor signer unavailable", got)
	}
}
