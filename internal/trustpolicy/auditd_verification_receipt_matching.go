package trustpolicy

import (
	"encoding/json"
	"fmt"
)

func processReceiptByKind(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict, sealDigest *Digest, sealPayload AuditSegmentSealPayload, anchorReceipts int, validAnchorForSeal bool) (int, bool) {
	if ok, done := processReceiptTarget(index, input, report, receipt, sealDigest, sealPayload, anchorReceipts, validAnchorForSeal); done {
		return ok.anchorReceipts, ok.validAnchor
	}
	if receipt.AuditReceiptKind == "anchor" {
		return processAnchorReceipt(index, input, report, receipt, anchorReceipts, validAnchorForSeal)
	}
	if receipt.AuditReceiptKind == "import" || receipt.AuditReceiptKind == "restore" {
		if err := verifyImportRestoreConsistency(receipt, *sealDigest, sealPayload); err != nil {
			addHardFailure(report, AuditVerificationReasonImportRestoreProvenanceInconsistent, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] import/restore consistency failed: %v", index, err), input.Segment.Header.SegmentID, nil)
		}
	}
	return anchorReceipts, validAnchorForSeal
}

func processAnchorReceipt(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict, anchorReceipts int, validAnchorForSeal bool) (int, bool) {
	if !anchorReceiptFamilyMatchesSeal(receipt) {
		addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] subject_family must be audit_segment_seal", index), input.Segment.Header.SegmentID, nil)
		return anchorReceipts, validAnchorForSeal
	}
	maybeAddPassphraseAnchorDegraded(index, input, report, receipt)
	return anchorReceipts + 1, true
}

func maybeAddPassphraseAnchorDegraded(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict) {
	payload := anchorReceiptPayload{}
	if err := json.Unmarshal(receipt.ReceiptPayload, &payload); err != nil || payload.PresenceMode != "passphrase" {
		return
	}
	addDegraded(report, AuditVerificationReasonAnchorPassphrasePresenceDegraded, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] uses passphrase presence mode which is degraded assurance", index), input.Segment.Header.SegmentID, nil)
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
	if receiptTargetsKnownHistoricalSeal(receipt, *sealDigest, input.KnownSealDigests) {
		return receiptProcessingState{anchorReceipts: anchorReceipts, validAnchor: validAnchor}, true
	}
	addMismatchedReceiptFailure(index, input, report, receipt)
	return receiptProcessingState{anchorReceipts: anchorReceipts, validAnchor: validAnchor}, true
}

func receiptRequiresCurrentSealMatch(receipt auditReceiptPayloadStrict) bool {
	return receipt.AuditReceiptKind == "anchor" || receipt.AuditReceiptKind == "import" || receipt.AuditReceiptKind == "restore" || receipt.AuditReceiptKind == "reconciliation"
}

func receiptTargetsKnownHistoricalSeal(receipt auditReceiptPayloadStrict, currentSealDigest Digest, knownSealDigests []Digest) bool {
	receiptDigest := mustDigestIdentity(receipt.SubjectDigest)
	if receiptDigest == "" {
		return false
	}
	if receiptDigest == mustDigestIdentity(currentSealDigest) {
		return false
	}
	for _, known := range knownSealDigests {
		if receiptDigest == mustDigestIdentity(known) {
			return true
		}
	}
	return false
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
