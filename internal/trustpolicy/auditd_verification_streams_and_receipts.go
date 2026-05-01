package trustpolicy

import (
	"fmt"
	"strings"
	"time"
)

func validateFrameEventSignerEvidence(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, frameRecordDigest Digest, event AuditEventPayload, signature SignatureBlock, entry AuditEventContractCatalogEntry) bool {
	if len(event.SignerEvidenceRefs) > 0 && len(input.SignerEvidence) == 0 {
		addHardFailure(report, AuditVerificationReasonSignerEvidenceMissing, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d event references signer evidence but verification input has none", index), input.Segment.Header.SegmentID, &frameRecordDigest)
		return false
	}
	if err := validateSignerEvidenceRefs(event, signature, entry, input.SignerEvidence); err != nil {
		reason := AuditVerificationReasonSignerEvidenceInvalid
		if strings.Contains(err.Error(), "missing signer evidence") {
			reason = AuditVerificationReasonSignerEvidenceMissing
		}
		addHardFailure(report, reason, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d signer evidence invalid: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return false
	}
	return true
}

func verifyFrameStreamContinuity(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, streams map[string]streamState, event AuditEventPayload, frameRecordDigest Digest) bool {
	if err := enforceStreamContinuity(streams, event, frameRecordDigest); err != nil {
		reason := AuditVerificationReasonStreamPreviousHashMismatch
		if strings.Contains(err.Error(), "gap") {
			reason = AuditVerificationReasonStreamSequenceGap
		}
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "rollback") {
			reason = AuditVerificationReasonStreamSequenceRollbackOrDuplicate
		}
		addHardFailure(report, reason, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d stream continuity check failed: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return false
	}
	return true
}

func enforceStreamContinuity(streams map[string]streamState, event AuditEventPayload, currentDigest Digest) error {
	previous, seen := streams[event.EmitterStreamID]
	if !seen {
		if event.Seq != 1 {
			return fmt.Errorf("first stream event must use seq=1")
		}
		if event.PreviousEventHash != nil {
			return fmt.Errorf("first stream event must not include previous_event_hash")
		}
		return nil
	}
	if event.Seq == previous.seq {
		return fmt.Errorf("duplicate seq %d detected", event.Seq)
	}
	if event.Seq < previous.seq {
		return fmt.Errorf("rollback seq %d < %d detected", event.Seq, previous.seq)
	}
	if event.Seq > previous.seq+1 {
		return fmt.Errorf("gap detected: seq %d follows %d", event.Seq, previous.seq)
	}
	if event.PreviousEventHash == nil {
		return fmt.Errorf("expected previous_event_hash for non-first stream event")
	}
	if mustDigestIdentity(*event.PreviousEventHash) != mustDigestIdentity(previous.digest) {
		return fmt.Errorf("previous_event_hash mismatch")
	}
	_ = currentDigest
	return nil
}

func verifyReceipts(input AuditVerificationInput, registry *VerifierRegistry, sealDigest *Digest, sealPayload AuditSegmentSealPayload, report *AuditVerificationReportPayload) {
	anchorReceipts := 0
	validAnchorForSeal := false

	for index := range input.ReceiptEnvelopes {
		receipt, ok := verifyAndDecodeReceipt(index, input, registry, report)
		if !ok {
			continue
		}
		if sealDigest == nil && receiptRequiresCurrentSealMatch(receipt) {
			addReceiptFailureForMissingSeal(index, input, report, receipt)
			continue
		}
		anchorReceipts, validAnchorForSeal = processReceiptByKind(index, input, report, receipt, sealDigest, sealPayload, anchorReceipts, validAnchorForSeal)
	}

	if sealDigest == nil {
		return
	}
	if anchorReceipts == 0 {
		addDegraded(report, AuditVerificationReasonAnchorReceiptMissing, AuditVerificationDimensionAnchoring, "no anchor receipts present for sealed segment", input.Segment.Header.SegmentID, nil)
		return
	}
	if !validAnchorForSeal {
		addDegraded(report, AuditVerificationReasonAnchorReceiptMissing, AuditVerificationDimensionAnchoring, "anchor receipts are present but none target this segment seal digest", input.Segment.Header.SegmentID, nil)
	}

	evaluateExternalAnchorEvidence(input, report, sealDigest)
}

func addReceiptFailureForMissingSeal(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict) {
	if receipt.AuditReceiptKind == "anchor" {
		addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] cannot be validated because segment seal verification failed", index), input.Segment.Header.SegmentID, nil)
		return
	}
	if receipt.AuditReceiptKind == "reconciliation" {
		addHardFailure(report, AuditVerificationReasonReceiptInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] cannot be validated because segment seal verification failed", index), input.Segment.Header.SegmentID, nil)
		return
	}
	addHardFailure(report, AuditVerificationReasonImportRestoreProvenanceInconsistent, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] cannot be validated because segment seal verification failed", index), input.Segment.Header.SegmentID, nil)
}

func verifyAndDecodeReceipt(index int, input AuditVerificationInput, registry *VerifierRegistry, report *AuditVerificationReportPayload) (auditReceiptPayloadStrict, bool) {
	envelope := input.ReceiptEnvelopes[index]
	anchorReceipt := envelopeDeclaresAnchorReceipt(envelope)
	receiptTime, verifierRecord, ok := verifyReceiptEnvelope(index, input.Segment.Header.SegmentID, envelope, registry, anchorReceipt, report)
	if !ok {
		return auditReceiptPayloadStrict{}, false
	}
	if !verifyReceiptSignerAdmissibility(index, input.Segment.Header.SegmentID, anchorReceipt, verifierRecord, receiptTime, report) {
		return auditReceiptPayloadStrict{}, false
	}
	return decodeAndValidateVerifiedReceipt(index, input.Segment.Header.SegmentID, envelope.Payload, verifierRecord, report)
}

func verifyReceiptEnvelope(index int, segmentID string, envelope SignedObjectEnvelope, registry *VerifierRegistry, anchorReceipt bool, report *AuditVerificationReportPayload) (time.Time, VerifierRecord, bool) {
	receiptTime, verifierRecord, err := verifyEnvelopeHistoricallyAdmissible(envelope, registry, AuditReceiptSchemaID, AuditReceiptSchemaVersion)
	if err != nil {
		addReceiptValidationFailure(report, segmentID, index, anchorReceipt, fmt.Sprintf("signature invalid: %v", err))
		return time.Time{}, VerifierRecord{}, false
	}
	return receiptTime, verifierRecord, true
}

func verifyReceiptSignerAdmissibility(index int, segmentID string, anchorReceipt bool, verifierRecord VerifierRecord, receiptTime time.Time, report *AuditVerificationReportPayload) bool {
	if err := checkHistoricalAdmissibility(verifierRecord, receiptTime); err != nil {
		if anchorReceipt {
			addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] signer historically inadmissible: %v", index, err), segmentID, nil)
		} else {
			addHardFailure(report, AuditVerificationReasonSignerHistoricallyInadmissible, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] signer historically inadmissible: %v", index, err), segmentID, nil)
		}
		return false
	}
	if isVerifierCurrentlyDegraded(verifierRecord, receiptTime) {
		addDegraded(report, AuditVerificationReasonSignerCurrentlyRevokedOrCompromised, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] signer is currently %s", index, verifierRecord.Status), segmentID, nil)
	}
	return true
}
