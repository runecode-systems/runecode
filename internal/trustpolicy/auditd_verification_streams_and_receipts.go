package trustpolicy

import (
	"encoding/json"
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

type auditReceiptPayloadStrict struct {
	SchemaID             string          `json:"schema_id"`
	SchemaVersion        string          `json:"schema_version"`
	SubjectDigest        Digest          `json:"subject_digest"`
	AuditReceiptKind     string          `json:"audit_receipt_kind"`
	SubjectFamily        string          `json:"subject_family,omitempty"`
	Recorder             json.RawMessage `json:"recorder"`
	RecordedAt           string          `json:"recorded_at"`
	ReceiptPayloadSchema string          `json:"receipt_payload_schema_id,omitempty"`
	ReceiptPayload       json.RawMessage `json:"receipt_payload,omitempty"`
}

type importRestoreReceiptPayload struct {
	ProvenanceAction      string                     `json:"provenance_action"`
	SegmentFileHashScope  string                     `json:"segment_file_hash_scope"`
	ImportedSegments      []importRestoreSegmentLink `json:"imported_segments"`
	SourceManifestDigests []Digest                   `json:"source_manifest_digests"`
	SourceInstanceID      string                     `json:"source_instance_id,omitempty"`
}

type importRestoreSegmentLink struct {
	ImportedSegmentSealDigest Digest `json:"imported_segment_seal_digest"`
	ImportedSegmentRoot       Digest `json:"imported_segment_root"`
	SourceSegmentFileHash     Digest `json:"source_segment_file_hash"`
	LocalSegmentFileHash      Digest `json:"local_segment_file_hash"`
	ByteIdentityVerified      bool   `json:"byte_identity_verified"`
}

func verifyReceipts(input AuditVerificationInput, registry *VerifierRegistry, sealDigest *Digest, sealPayload AuditSegmentSealPayload, report *AuditVerificationReportPayload) {
	anchorReceipts := 0
	validAnchorForSeal := false

	for index := range input.ReceiptEnvelopes {
		receipt, ok := verifyAndDecodeReceipt(index, input, registry, report)
		if !ok {
			continue
		}
		anchorReceipts, validAnchorForSeal = processReceiptByKind(index, input, report, receipt, sealDigest, sealPayload, anchorReceipts, validAnchorForSeal)
	}

	if anchorReceipts == 0 {
		addDegraded(report, AuditVerificationReasonAnchorReceiptMissing, AuditVerificationDimensionAnchoring, "no anchor receipts present for sealed segment", input.Segment.Header.SegmentID, nil)
		return
	}
	if !validAnchorForSeal && len(report.HardFailures) == 0 {
		addDegraded(report, AuditVerificationReasonAnchorReceiptMissing, AuditVerificationDimensionAnchoring, "anchor receipts are present but none target this segment seal digest", input.Segment.Header.SegmentID, nil)
	}
}

func verifyAndDecodeReceipt(index int, input AuditVerificationInput, registry *VerifierRegistry, report *AuditVerificationReportPayload) (auditReceiptPayloadStrict, bool) {
	receiptEnvelope := input.ReceiptEnvelopes[index]
	receiptTime, verifierRecord, err := verifyEnvelopeHistoricallyAdmissible(receiptEnvelope, registry, AuditReceiptSchemaID, AuditReceiptSchemaVersion)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonReceiptInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] signature invalid: %v", index, err), input.Segment.Header.SegmentID, nil)
		return auditReceiptPayloadStrict{}, false
	}
	if err := checkHistoricalAdmissibility(verifierRecord, receiptTime); err != nil {
		addHardFailure(report, AuditVerificationReasonSignerHistoricallyInadmissible, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] signer historically inadmissible: %v", index, err), input.Segment.Header.SegmentID, nil)
		return auditReceiptPayloadStrict{}, false
	}
	if isVerifierCurrentlyDegraded(verifierRecord, receiptTime) {
		addDegraded(report, AuditVerificationReasonSignerCurrentlyRevokedOrCompromised, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] signer is currently %s", index, verifierRecord.Status), input.Segment.Header.SegmentID, nil)
	}

	receipt, err := decodeAuditReceiptPayload(receiptEnvelope.Payload)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonReceiptInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] decode failed: %v", index, err), input.Segment.Header.SegmentID, nil)
		return auditReceiptPayloadStrict{}, false
	}
	if err := validateAuditReceiptPayload(receipt); err != nil {
		if receipt.AuditReceiptKind == "anchor" {
			addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] invalid: %v", index, err), input.Segment.Header.SegmentID, nil)
		} else {
			addHardFailure(report, AuditVerificationReasonReceiptInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] invalid: %v", index, err), input.Segment.Header.SegmentID, nil)
		}
		return auditReceiptPayloadStrict{}, false
	}
	return receipt, true
}

func processReceiptByKind(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict, sealDigest *Digest, sealPayload AuditSegmentSealPayload, anchorReceipts int, validAnchorForSeal bool) (int, bool) {
	if receipt.AuditReceiptKind == "anchor" {
		anchorReceipts++
		if validateAnchorReceipt(index, input, report, receipt, sealDigest) {
			validAnchorForSeal = true
		}
	}
	if (receipt.AuditReceiptKind == "import" || receipt.AuditReceiptKind == "restore") && sealDigest != nil {
		if err := verifyImportRestoreConsistency(receipt, *sealDigest, sealPayload); err != nil {
			addHardFailure(report, AuditVerificationReasonImportRestoreProvenanceInconsistent, AuditVerificationDimensionIntegrity, fmt.Sprintf("receipt[%d] import/restore consistency failed: %v", index, err), input.Segment.Header.SegmentID, nil)
		}
	}
	return anchorReceipts, validAnchorForSeal
}

func validateAnchorReceipt(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, receipt auditReceiptPayloadStrict, sealDigest *Digest) bool {
	if sealDigest == nil {
		return false
	}
	receiptSubject, _ := receipt.SubjectDigest.Identity()
	if receiptSubject == mustDigestIdentity(*sealDigest) {
		return true
	}
	addHardFailure(report, AuditVerificationReasonAnchorReceiptInvalid, AuditVerificationDimensionAnchoring, fmt.Sprintf("anchor receipt[%d] subject_digest does not match segment seal digest", index), input.Segment.Header.SegmentID, nil)
	return false
}

func decodeAuditReceiptPayload(raw json.RawMessage) (auditReceiptPayloadStrict, error) {
	receipt := auditReceiptPayloadStrict{}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return auditReceiptPayloadStrict{}, fmt.Errorf("decode audit receipt payload: %w", err)
	}
	return receipt, nil
}

func validateAuditReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if err := validateAuditReceiptCoreFields(receipt); err != nil {
		return err
	}
	if err := validateAuditReceiptPayloadPresence(receipt); err != nil {
		return err
	}
	if receipt.AuditReceiptKind != "import" && receipt.AuditReceiptKind != "restore" {
		return nil
	}
	return validateImportRestoreReceiptPayload(receipt)
}

func validateAuditReceiptCoreFields(receipt auditReceiptPayloadStrict) error {
	if receipt.SchemaID != AuditReceiptSchemaID {
		return fmt.Errorf("unexpected receipt schema_id %q", receipt.SchemaID)
	}
	if receipt.SchemaVersion != AuditReceiptSchemaVersion {
		return fmt.Errorf("unexpected receipt schema_version %q", receipt.SchemaVersion)
	}
	if _, err := receipt.SubjectDigest.Identity(); err != nil {
		return fmt.Errorf("subject_digest: %w", err)
	}
	if !auditVerificationCodePattern.MatchString(receipt.AuditReceiptKind) {
		return fmt.Errorf("invalid audit_receipt_kind %q", receipt.AuditReceiptKind)
	}
	if _, ok := map[string]struct{}{"anchor": {}, "import": {}, "restore": {}, "reconciliation": {}}[receipt.AuditReceiptKind]; !ok {
		return fmt.Errorf("unsupported audit_receipt_kind %q", receipt.AuditReceiptKind)
	}
	if err := validateReceiptRecorder(receipt.Recorder); err != nil {
		return err
	}
	if receipt.RecordedAt == "" {
		return fmt.Errorf("recorded_at is required")
	}
	if _, err := time.Parse(time.RFC3339, receipt.RecordedAt); err != nil {
		return fmt.Errorf("invalid recorded_at: %w", err)
	}
	return nil
}

func validateReceiptRecorder(recorder json.RawMessage) error {
	if len(recorder) == 0 {
		return fmt.Errorf("recorder is required")
	}
	identity := PrincipalIdentity{}
	if err := json.Unmarshal(recorder, &identity); err != nil {
		return fmt.Errorf("recorder must decode as principal identity: %w", err)
	}
	if identity.SchemaID != "runecode.protocol.v0.PrincipalIdentity" {
		return fmt.Errorf("recorder.schema_id must be runecode.protocol.v0.PrincipalIdentity")
	}
	if identity.SchemaVersion != "0.2.0" {
		return fmt.Errorf("recorder.schema_version must be 0.2.0")
	}
	if identity.ActorKind == "" || identity.PrincipalID == "" || identity.InstanceID == "" {
		return fmt.Errorf("recorder is required")
	}
	return nil
}
