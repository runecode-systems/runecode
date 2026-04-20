package brokerapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func assertAnchorSuccessArtifacts(t *testing.T, service *Service, ledgerRoot string, sealDigest trustpolicy.Digest, resp AuditAnchorSegmentResponse) {
	t.Helper()
	if strings.TrimSpace(resp.ProjectContextID) == "" {
		t.Fatal("project_context_identity_digest missing")
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
	assertAnchorReceiptProvenance(t, exported.Reference, sealDigest, *resp.ReceiptDigest, resp.VerificationReportDigest)
	assertAnchorReceiptSidecarExists(t, ledgerRoot, *resp.ReceiptDigest)
	assertAnchorVerificationReportSidecarExists(t, ledgerRoot, *resp.VerificationReportDigest)
}

func assertAnchorReceiptProvenance(t *testing.T, ref artifacts.ArtifactReference, sealDigest trustpolicy.Digest, receiptDigest trustpolicy.Digest, verificationDigest *trustpolicy.Digest) {
	t.Helper()
	if got := strings.TrimSpace(ref.ProvenanceReceiptHash); got != mustDigestIdentityForAnchorTest(receiptDigest) {
		t.Fatalf("exported receipt provenance_receipt_hash = %q, want receipt digest %q", got, mustDigestIdentityForAnchorTest(receiptDigest))
	}
	if got := strings.TrimSpace(ref.ProvenanceReceiptHash); got == mustDigestIdentityForAnchorTest(sealDigest) {
		t.Fatalf("exported receipt provenance_receipt_hash = %q, must not use seal digest %q", got, mustDigestIdentityForAnchorTest(sealDigest))
	}
	if verificationDigest != nil && ref.Digest == mustDigestIdentityForAnchorTest(*verificationDigest) {
		t.Fatalf("exported receipt digest must not alias verification report digest %q", mustDigestIdentityForAnchorTest(*verificationDigest))
	}
}

func assertAnchorReceiptRecorderIdentity(t *testing.T, ledgerRoot string, receiptDigest trustpolicy.Digest) {
	t.Helper()
	envelope := mustReadAnchorReceiptSidecar(t, ledgerRoot, receiptDigest)
	receipt := map[string]any{}
	if err := json.Unmarshal(envelope.Payload, &receipt); err != nil {
		t.Fatalf("Unmarshal anchor receipt payload returned error: %v", err)
	}
	recorder, ok := receipt["recorder"].(map[string]any)
	if !ok {
		t.Fatalf("recorder = %+v, want object", receipt["recorder"])
	}
	if got := strings.TrimSpace(stringValueForAnchorTest(recorder, "principal_id")); got != "auditd" {
		t.Fatalf("recorder.principal_id = %q, want auditd", got)
	}
	if got := strings.TrimSpace(stringValueForAnchorTest(recorder, "instance_id")); got != "auditd-1" {
		t.Fatalf("recorder.instance_id = %q, want auditd-1", got)
	}
}
