package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestHandleAuditAnchorSegmentSuccessPersistsReceiptAndVerification(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-success",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
		ExportReceiptCopy:   true,
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
	exported, err := service.Head(resp.ExportedReceiptRef)
	if err != nil {
		t.Fatalf("Head(exported_receipt_ref) returned error: %v", err)
	}
	if exported.Reference.DataClass != artifacts.DataClassAuditReceiptExportCopy {
		t.Fatalf("exported receipt data_class = %q, want %q", exported.Reference.DataClass, artifacts.DataClassAuditReceiptExportCopy)
	}
	if resp.VerificationReportDigest != nil && exported.Reference.Digest == mustDigestIdentityForAnchorTest(*resp.VerificationReportDigest) {
		t.Fatalf("exported receipt digest must not alias verification report digest %q", mustDigestIdentityForAnchorTest(*resp.VerificationReportDigest))
	}
	assertAnchorReceiptSidecarExists(t, ledgerRoot, *resp.ReceiptDigest)
	assertAnchorVerificationReportSidecarExists(t, ledgerRoot, *resp.VerificationReportDigest)
}

func TestHandleAuditAnchorSegmentExportCopyIsOptional(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-sidecar-only",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
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
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", immutability.sealDigest)

	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-immutability",
		SealDigest:          immutability.sealDigest,
		PresenceAttestation: presence,
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
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	service.store = nil

	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-export-best-effort",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
		ExportReceiptCopy:   true,
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

func TestHandleAuditAnchorSegmentFailsClosedWithoutPresenceAttestation(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-anchor-fail-missing-presence",
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

func TestHandleAuditAnchorSegmentFailsClosedWithInvalidPresenceAttestation(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	presence.AcknowledgmentToken = "deadbeef"
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-fail-invalid-presence",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
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

func TestHandleAuditAnchorSegmentSignerPresenceValidationFailsClosed(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	t.Setenv("RUNE_AUDIT_ANCHOR_PRESENCE_MODE", "invalid_presence")
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-fail-degraded",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
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
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	t.Setenv("RUNE_AUDIT_ANCHOR_KEY_PROTECTION_POSTURE", "invalid_posture")
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-fail-closed",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
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
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	decisionDigest := mustSeedConsumedApprovalForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-anchor-consumed-approval",
		SealDigest:             sealDigest,
		ApprovalDecisionDigest: &decisionDigest,
		PresenceAttestation:    presence,
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

func TestHandleAuditAnchorSegmentPolicyRequiresApprovalFailsClosedWhenMissingDecision(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	mustSeedAnchorPolicyDecision(t, service, sealDigest, "require_human_approval", "reauthenticated")
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-policy-required-missing-decision",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
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
	if resp.ReceiptDigest != nil {
		t.Fatalf("receipt_digest = %+v, want nil", resp.ReceiptDigest)
	}
}

func TestHandleAuditAnchorSegmentPolicyRequiresApprovalWithConsumedDecisionSucceeds(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	decisionDigest := mustSeedConsumedApprovalForRequiredAnchorPolicy(t, service, sealDigest, "reauthenticated")
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-anchor-policy-required-consumed-decision",
		SealDigest:             sealDigest,
		ApprovalDecisionDigest: &decisionDigest,
		PresenceAttestation:    presence,
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
}

func TestHandleAuditAnchorSegmentPolicyDoesNotRequireApprovalWithoutDecisionSucceeds(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	mustSeedAnchorPolicyDecision(t, service, sealDigest, "allow", "")
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-policy-allow-no-decision",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
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
}

func TestHandleAuditAnchorSegmentPolicyRequiredAssuranceMismatchFailsClosed(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	decisionDigest := mustSeedConsumedApprovalForRequiredAnchorPolicy(t, service, sealDigest, "hardware_backed")
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-anchor-policy-assurance-mismatch",
		SealDigest:             sealDigest,
		ApprovalDecisionDigest: &decisionDigest,
		PresenceAttestation:    presence,
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

func TestHandleAuditAnchorSegmentFailsClosedOnUnconsumedApproval(t *testing.T) {
	service, _ := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	decisionDigest := mustSeedPendingApprovalForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-anchor-unconsumed-approval",
		SealDigest:             sealDigest,
		ApprovalDecisionDigest: &decisionDigest,
		PresenceAttestation:    presence,
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
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)
	decisionDigest := mustSeedConsumedApprovalForAnchorTest(t, service)
	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-anchor-approval-assurance-mismatch",
		SealDigest:             sealDigest,
		ApprovalDecisionDigest: &decisionDigest,
		ApprovalAssuranceLevel: "hardware_backed",
		PresenceAttestation:    presence,
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

func TestHandleAuditAnchorSegmentRestartVerificationKeepsAnchoringOK(t *testing.T) {
	service, ledgerRoot := newAuditAnchorTestService(t)
	sealDigest := mustLatestSealDigestForAnchorTest(t, service)
	presence := mustAuditAnchorPresenceAttestation(t, "os_confirmation", sealDigest)

	resp, errResp := service.HandleAuditAnchorSegment(context.Background(), AuditAnchorSegmentRequest{
		SchemaID:            "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-anchor-restart-verify",
		SealDigest:          sealDigest,
		PresenceAttestation: presence,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditAnchorSegment returned error: %+v", errResp)
	}
	if resp.AnchoringStatus != "ok" {
		t.Fatalf("anchoring_status = %q, want ok", resp.AnchoringStatus)
	}

	reopened, err := auditd.Open(ledgerRoot)
	if err != nil {
		t.Fatalf("auditd.Open(restart) returned error: %v", err)
	}
	verification, err := reopened.VerifyCurrentSegmentAndPersist()
	if err != nil {
		t.Fatalf("VerifyCurrentSegmentAndPersist(restart) returned error: %v", err)
	}
	if verification.Report.AnchoringStatus != trustpolicy.AuditVerificationStatusOK {
		t.Fatalf("anchoring_status = %q, want %q", verification.Report.AnchoringStatus, trustpolicy.AuditVerificationStatusOK)
	}
	if containsReasonCodeForAnchorTest(verification.Report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid) {
		t.Fatalf("hard_failures = %v, did not want %q", verification.Report.HardFailures, trustpolicy.AuditVerificationReasonAnchorReceiptInvalid)
	}
}

func containsReasonCodeForAnchorTest(codes []string, code string) bool {
	for idx := range codes {
		if codes[idx] == code {
			return true
		}
	}
	return false
}

type anchorReceiptPayloadForTest struct {
	ApprovalAssurance string              `json:"approval_assurance_level,omitempty"`
	ApprovalDecision  *trustpolicy.Digest `json:"approval_decision_digest,omitempty"`
}
