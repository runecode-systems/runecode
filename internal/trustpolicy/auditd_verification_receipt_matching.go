package trustpolicy

import (
	"fmt"
	"time"
)

func processReceiptByKind(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict, sealDigest *Digest, sealPayload AuditSegmentSealPayload, anchorReceipts int, validAnchorForSeal bool) (int, bool) {
	if ok, done := processReceiptTarget(index, input, report, receipt, sealDigest, sealPayload, anchorReceipts, validAnchorForSeal); done {
		return ok.anchorReceipts, ok.validAnchor
	}
	if receipt.AuditReceiptKind == "anchor" {
		if !anchorReceiptFamilyMatchesSeal(receipt) {
			addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] subject_family must be audit_segment_seal", index), input.Segment.Header.SegmentID, nil)
			return anchorReceipts, validAnchorForSeal
		}
		return anchorReceipts + 1, true
	}
	if receipt.AuditReceiptKind == "import" || receipt.AuditReceiptKind == "restore" {
		if err := verifyImportRestoreConsistency(receipt, *sealDigest, sealPayload); err != nil {
			addHardFailure(report, AuditVerificationReasonImportRestoreProvenanceInconsistent, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] import/restore consistency failed: %v", index, err), input.Segment.Header.SegmentID, nil)
		}
	}
	return anchorReceipts, validAnchorForSeal
}

func anchorReceiptFamilyMatchesSeal(receipt auditReceiptPayloadStrict) bool {
	return receipt.SubjectFamily == "audit_segment_seal"
}

type receiptProcessingState struct {
	anchorReceipts int
	validAnchor    bool
}

func processReceiptTarget(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict, sealDigest *Digest, sealPayload AuditSegmentSealPayload, anchorReceipts int, validAnchor bool) (receiptProcessingState, bool) {
	if !receiptRequiresCurrentSealMatch(receipt) || sealDigest == nil {
		return receiptProcessingState{anchorReceipts: anchorReceipts, validAnchor: validAnchor}, true
	}
	if mustDigestIdentity(receipt.SubjectDigest) == mustDigestIdentity(*sealDigest) {
		return receiptProcessingState{anchorReceipts: anchorReceipts, validAnchor: validAnchor}, false
	}
	if receiptIsHistoricalForCurrentSeal(receipt, sealPayload) {
		return receiptProcessingState{anchorReceipts: anchorReceipts, validAnchor: validAnchor}, true
	}
	addMismatchedReceiptFailure(index, input, report, receipt)
	return receiptProcessingState{anchorReceipts: anchorReceipts, validAnchor: validAnchor}, true
}

func receiptRequiresCurrentSealMatch(receipt auditReceiptPayloadStrict) bool {
	return receipt.AuditReceiptKind == "anchor" || receipt.AuditReceiptKind == "import" || receipt.AuditReceiptKind == "restore" || receipt.AuditReceiptKind == "reconciliation"
}

func receiptIsHistoricalForCurrentSeal(receipt auditReceiptPayloadStrict, sealPayload AuditSegmentSealPayload) bool {
	if receipt.RecordedAt == "" || sealPayload.SealedAt == "" {
		return false
	}
	recordedAt, err := time.Parse(time.RFC3339, receipt.RecordedAt)
	if err != nil {
		return false
	}
	sealedAt, err := time.Parse(time.RFC3339, sealPayload.SealedAt)
	if err != nil {
		return false
	}
	return recordedAt.Before(sealedAt)
}

func addMismatchedReceiptFailure(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict) {
	if receipt.AuditReceiptKind == "anchor" {
		addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] subject_digest does not match segment seal digest", index), input.Segment.Header.SegmentID, nil)
		return
	}
	if receipt.AuditReceiptKind == "import" || receipt.AuditReceiptKind == "restore" {
		addHardFailure(report, AuditVerificationReasonImportRestoreProvenanceInconsistent, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] subject_digest does not match segment seal digest", index), input.Segment.Header.SegmentID, nil)
		return
	}
	if receipt.AuditReceiptKind == "reconciliation" {
		addHardFailure(report, AuditVerificationReasonReceiptInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] subject_digest does not match segment seal digest", index), input.Segment.Header.SegmentID, nil)
	}
}
