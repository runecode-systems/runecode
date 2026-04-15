package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestHandleAuditAnchorSegmentSuccessPersistsReceiptAndVerification(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:          "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         "req-anchor-success",
		SealDigest:        sealDigest,
		ExportReceiptCopy: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error: %+v", errResp)
	}
	if resp.AnchoringStatus != "ok" {
		t.Fatalf("anchoring_status = %q, want ok", resp.AnchoringStatus)
	}
	if resp.ReceiptDigest == nil {
		t.Fatal("receipt_digest missing")
	}
	if resp.VerificationReportDigest == nil {
		t.Fatal("verification_report_digest missing")
	}
	if strings.TrimSpace(resp.ExportedReceiptRef) == "" {
		t.Fatal("exported_receipt_ref missing")
	}
	assertAnchorReceiptSidecarExists(t, ledgerRoot, *resp.ReceiptDigest)
}

func TestHandleAuditAnchorSegmentExportCopyIsOptional(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-sidecar-only",
		SealDigest:    sealDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error: %+v", errResp)
	}
	if resp.AnchoringStatus != "ok" {
		t.Fatalf("anchoring_status = %q, want ok", resp.AnchoringStatus)
	}
	if resp.ReceiptDigest == nil {
		t.Fatal("receipt_digest missing")
	}
	assertAnchorReceiptSidecarExists(t, ledgerRoot, *resp.ReceiptDigest)
	if got := strings.TrimSpace(resp.ExportedReceiptRef); got != "" {
		t.Fatalf("exported_receipt_ref = %q, want empty when export_receipt_copy=false", got)
	}
	if artifacts := service.List(); len(artifacts) != 0 {
		t.Fatalf("artifact store should remain unchanged when export_receipt_copy=false, got %d records", len(artifacts))
	}
}

func TestHandleAuditAnchorSegmentDoesNotMutateSegmentBytesOrSealIdentity(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	immutability := captureAnchorImmutabilityBaseline(t, service, ledgerRoot)

	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-immutability",
		SealDigest:    immutability.sealDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error: %+v", errResp)
	}
	if resp.AnchoringStatus != "ok" {
		t.Fatalf("anchoring_status = %q, want ok", resp.AnchoringStatus)
	}
	assertAnchorImmutabilityAfterAnchor(t, service, immutability)
}

func TestHandleAuditAnchorSegmentExportFailureDoesNotFailAnchoring(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	service.store = nil

	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:          "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         "req-anchor-export-best-effort",
		SealDigest:        sealDigest,
		ExportReceiptCopy: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error: %+v", errResp)
	}
	if resp.AnchoringStatus != "ok" {
		t.Fatalf("anchoring_status = %q, want ok", resp.AnchoringStatus)
	}
	if resp.ReceiptDigest == nil {
		t.Fatal("receipt_digest missing")
	}
	assertAnchorReceiptSidecarExists(t, ledgerRoot, *resp.ReceiptDigest)
	if got := strings.TrimSpace(resp.ExportedReceiptRef); got != "" {
		t.Fatalf("exported_receipt_ref = %q, want empty when export copy write fails", got)
	}
}

func TestHandleAuditAnchorSegmentSignerPresenceValidationFailsClosed(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	t.Setenv("RUNE_AUDIT_ANCHOR_PRESENCE_MODE", "invalid_presence")
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-fail-degraded",
		SealDigest:    sealDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error response: %+v", errResp)
	}
	if resp.AnchoringStatus != "failed" {
		t.Fatalf("anchoring_status = %q, want failed", resp.AnchoringStatus)
	}
	if resp.FailureCode != "anchor_receipt_invalid" {
		t.Fatalf("failure_code = %q, want anchor_receipt_invalid", resp.FailureCode)
	}
	if resp.ReceiptDigest != nil {
		t.Fatalf("receipt_digest = %+v, want nil", resp.ReceiptDigest)
	}
}

func TestHandleAuditAnchorSegmentInvalidAnchorSemanticReturnsFailed(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	t.Setenv("RUNE_AUDIT_ANCHOR_KEY_PROTECTION_POSTURE", "invalid_posture")
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-fail-closed",
		SealDigest:    sealDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error response: %+v", errResp)
	}
	if resp.AnchoringStatus != "failed" {
		t.Fatalf("anchoring_status = %q, want failed", resp.AnchoringStatus)
	}
	if resp.FailureCode != "anchor_receipt_invalid" {
		t.Fatalf("failure_code = %q, want anchor_receipt_invalid", resp.FailureCode)
	}
}

func TestHandleAuditAnchorSegmentWithConsumedSignedApprovalContext(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	decisionDigest := mustSeedConsumedApprovalForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-anchor-consumed-approval",
		SealDigest:             sealDigest,
		ApprovalDecisionDigest: &decisionDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error: %+v", errResp)
	}
	if resp.AnchoringStatus != "ok" {
		t.Fatalf("anchoring_status = %q, want ok", resp.AnchoringStatus)
	}
	if resp.ReceiptDigest == nil {
		t.Fatal("receipt_digest missing")
	}
	assertAnchorReceiptSidecarExists(t, ledgerRoot, *resp.ReceiptDigest)
	receipt := mustReadAnchorReceiptSidecar(t, ledgerRoot, *resp.ReceiptDigest)
	payload := mustAnchorReceiptPayload(t, receipt)
	if got := strings.TrimSpace(payload.ApprovalAssurance); got != "reauthenticated" {
		t.Fatalf("approval_assurance_level = %q, want reauthenticated", got)
	}
	if payload.ApprovalDecision == nil || mustDigestIdentityForAnchorTest(*payload.ApprovalDecision) != mustDigestIdentityForAnchorTest(decisionDigest) {
		t.Fatalf("approval_decision_digest = %+v, want %q", payload.ApprovalDecision, mustDigestIdentityForAnchorTest(decisionDigest))
	}
}

func TestHandleAuditAnchorSegmentFailsClosedOnUnconsumedApproval(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	decisionDigest := mustSeedPendingApprovalForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-anchor-unconsumed-approval",
		SealDigest:             sealDigest,
		ApprovalDecisionDigest: &decisionDigest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error: %+v", errResp)
	}
	if resp.AnchoringStatus != "failed" {
		t.Fatalf("anchoring_status = %q, want failed", resp.AnchoringStatus)
	}
	if resp.FailureCode != "anchor_receipt_invalid" {
		t.Fatalf("failure_code = %q, want anchor_receipt_invalid", resp.FailureCode)
	}
}

func TestHandleAuditAnchorSegmentFailsClosedOnApprovalAssuranceMismatch(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	decisionDigest := mustSeedConsumedApprovalForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-anchor-approval-assurance-mismatch",
		SealDigest:             sealDigest,
		ApprovalDecisionDigest: &decisionDigest,
		ApprovalAssuranceLevel: "hardware_backed",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error: %+v", errResp)
	}
	if resp.AnchoringStatus != "failed" {
		t.Fatalf("anchoring_status = %q, want failed", resp.AnchoringStatus)
	}
	if resp.FailureCode != "anchor_receipt_invalid" {
		t.Fatalf("failure_code = %q, want anchor_receipt_invalid", resp.FailureCode)
	}
}

type anchorReceiptPayloadForTest struct {
	ApprovalAssurance string              `json:"approval_assurance_level,omitempty"`
	ApprovalDecision  *trustpolicy.Digest `json:"approval_decision_digest,omitempty"`
}
