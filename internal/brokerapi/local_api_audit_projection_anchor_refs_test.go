package brokerapi

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestProjectAuditReceiptRecordDetailAddsAnchorApprovalAndWitnessReferences(t *testing.T) {
	detail := &AuditRecordDetail{}
	receipt := &trustpolicy.AuditReceiptOperationalView{
		AuditReceiptKind:    "anchor",
		SubjectDigest:       trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('a')},
		ApprovalDecision:    &trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('b')},
		AnchorWitnessDigest: &trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('c')},
	}

	if err := projectAuditReceiptRecordDetail(detail, receipt); err != nil {
		t.Fatalf("projectAuditReceiptRecordDetail returned error: %v", err)
	}
	if detail.Summary != "Audit receipt (anchor) recorded [posture=local_only_completed]." {
		t.Fatalf("summary=%q, want local-only anchor posture summary", detail.Summary)
	}

	assertHasLinkedReference(t, detail.LinkedReferences, "audit_record", "subject_segment_seal")
	assertHasLinkedReference(t, detail.LinkedReferences, "approval", "approval_decision")
	assertHasLinkedReference(t, detail.LinkedReferences, "artifact", "anchor_witness")
}

func TestProjectAuditTimelineEntryAddsAnchorApprovalAndWitnessReferences(t *testing.T) {
	view := trustpolicy.AuditOperationalView{
		RecordDigest: trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('d')},
		Receipt: &trustpolicy.AuditReceiptOperationalView{
			AuditReceiptKind:               "anchor",
			SubjectDigest:                  trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('a')},
			ApprovalDecision:               &trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('b')},
			AnchorWitnessDigest:            &trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('c')},
			ExternalTargetKind:             "transparency_log",
			ExternalRuntimeAdapter:         "transparency_log_v0",
			ExternalTargetDescriptorDigest: &trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('e')},
			ExternalProofKind:              "transparency_log_receipt_v0",
			ExternalProofSchema:            "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0",
			ExternalProofDigest:            &trustpolicy.Digest{HashAlg: "sha256", Hash: repeatHexChar('f')},
		},
	}

	entry, ok := projectAuditTimelineEntry(view, map[string]AuditRecordVerificationPosture{})
	if !ok {
		t.Fatal("projectAuditTimelineEntry returned ok=false")
	}
	if entry.Summary != "Audit receipt (anchor) recorded [posture=external_completed]." {
		t.Fatalf("summary=%q, want external-completed anchor posture summary", entry.Summary)
	}
	assertHasLinkedReference(t, entry.LinkedReferences, "audit_record", "subject_segment_seal")
	assertHasLinkedReference(t, entry.LinkedReferences, "approval", "approval_decision")
	assertHasLinkedReference(t, entry.LinkedReferences, "artifact", "anchor_witness")
}

func assertHasLinkedReference(t *testing.T, refs []AuditRecordLinkedReference, kind string, relation string) {
	t.Helper()
	for _, ref := range refs {
		if ref.ReferenceKind == kind && ref.Relation == relation {
			return
		}
	}
	t.Fatalf("linked_references missing kind=%q relation=%q: %+v", kind, relation, refs)
}

func repeatHexChar(ch byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}
