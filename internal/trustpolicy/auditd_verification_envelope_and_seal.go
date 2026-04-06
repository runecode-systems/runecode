package trustpolicy

import (
	"encoding/json"
	"fmt"
	"time"
)

func verifySegmentSeal(input AuditVerificationInput, registry *VerifierRegistry, frameDigests []Digest, report *AuditVerificationReportPayload) (*Digest, AuditSegmentSealPayload, error) {
	sealTime, verifierRecord, err := verifyEnvelopeHistoricallyAdmissible(input.SegmentSealEnvelope, registry, AuditSegmentSealSchemaID, AuditSegmentSealSchemaVersion)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("segment seal envelope invalid: %v", err), input.Segment.Header.SegmentID, nil)
		return nil, AuditSegmentSealPayload{}, err
	}
	if err := checkHistoricalAdmissibility(verifierRecord, sealTime); err != nil {
		addHardFailure(report, AuditVerificationReasonSignerHistoricallyInadmissible, AuditVerificationDimensionIntegrity, fmt.Sprintf("segment seal signer historically inadmissible: %v", err), input.Segment.Header.SegmentID, nil)
		return nil, AuditSegmentSealPayload{}, err
	}
	if isVerifierCurrentlyDegraded(verifierRecord, sealTime) {
		addDegraded(report, AuditVerificationReasonSignerCurrentlyRevokedOrCompromised, AuditVerificationDimensionIntegrity, fmt.Sprintf("segment seal signer is currently %s after seal time", verifierRecord.Status), input.Segment.Header.SegmentID, nil)
	}

	seal := AuditSegmentSealPayload{}
	if err := json.Unmarshal(input.SegmentSealEnvelope.Payload, &seal); err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("decode segment seal payload: %v", err), input.Segment.Header.SegmentID, nil)
		return nil, AuditSegmentSealPayload{}, err
	}
	if err := ValidateAuditSegmentSealPayload(seal); err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, err.Error(), input.Segment.Header.SegmentID, nil)
		return nil, AuditSegmentSealPayload{}, err
	}
	validateSegmentSealAgainstFrames(input, report, frameDigests, seal)
	validateSegmentSealHashesAndChain(input, report, frameDigests, seal)

	sealEnvelopeDigest, err := ComputeSignedEnvelopeAuditRecordDigest(input.SegmentSealEnvelope)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("segment seal digest compute failed: %v", err), input.Segment.Header.SegmentID, nil)
		return nil, seal, err
	}
	return &sealEnvelopeDigest, seal, nil
}

func validateSegmentSealHashesAndChain(input AuditVerificationInput, report *AuditVerificationReportPayload, frameDigests []Digest, seal AuditSegmentSealPayload) {
	if err := VerifyOrderedAuditSegmentMerkleRoot(frameDigests, seal.MerkleRoot); err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentMerkleRootMismatch, AuditVerificationDimensionIntegrity, err.Error(), input.Segment.Header.SegmentID, nil)
	}
	if err := VerifySegmentFileHash(input.RawFramedSegmentBytes, seal.SegmentFileHash); err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentFileHashMismatch, AuditVerificationDimensionIntegrity, err.Error(), input.Segment.Header.SegmentID, nil)
	}
	if err := ValidateAuditSegmentSealChainLink(seal, input.PreviousSealEnvelopeHash); err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentSealChainMismatch, AuditVerificationDimensionIntegrity, err.Error(), input.Segment.Header.SegmentID, nil)
	}
}

func validateSegmentSealAgainstFrames(input AuditVerificationInput, report *AuditVerificationReportPayload, frameDigests []Digest, seal AuditSegmentSealPayload) {
	if seal.SegmentID != input.Segment.Header.SegmentID {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("segment seal segment_id=%q does not match segment header segment_id=%q", seal.SegmentID, input.Segment.Header.SegmentID), input.Segment.Header.SegmentID, nil)
	}
	if int64(len(frameDigests)) != seal.EventCount {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("segment seal event_count=%d does not match frame count=%d", seal.EventCount, len(frameDigests)), input.Segment.Header.SegmentID, nil)
	}
	if len(frameDigests) == 0 {
		return
	}
	if got, _ := frameDigests[0].Identity(); got != mustDigestIdentity(seal.FirstRecordDigest) {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, "segment seal first_record_digest does not match first frame digest", input.Segment.Header.SegmentID, &frameDigests[0])
	}
	last := frameDigests[len(frameDigests)-1]
	if got, _ := last.Identity(); got != mustDigestIdentity(seal.LastRecordDigest) {
		addHardFailure(report, AuditVerificationReasonSegmentSealInvalid, AuditVerificationDimensionIntegrity, "segment seal last_record_digest does not match last frame digest", input.Segment.Header.SegmentID, &last)
	}
}

func verifyEnvelopeHistoricallyAdmissible(envelope SignedObjectEnvelope, registry *VerifierRegistry, expectedSchemaID string, expectedVersion string) (time.Time, VerifierRecord, error) {
	if err := validateSignedEnvelopeMetadata(envelope); err != nil {
		return time.Time{}, VerifierRecord{}, err
	}
	if envelope.PayloadSchemaID != expectedSchemaID {
		return time.Time{}, VerifierRecord{}, fmt.Errorf("payload_schema_id %q does not match expected %q", envelope.PayloadSchemaID, expectedSchemaID)
	}
	if envelope.PayloadSchemaVersion != expectedVersion {
		return time.Time{}, VerifierRecord{}, fmt.Errorf("payload_schema_version %q does not match expected %q", envelope.PayloadSchemaVersion, expectedVersion)
	}

	verifier, err := resolveVerifierIgnoringStatus(envelope.Signature, registry)
	if err != nil {
		return time.Time{}, VerifierRecord{}, err
	}
	canonicalPayload, err := canonicalizePayload(envelope.Payload)
	if err != nil {
		return time.Time{}, VerifierRecord{}, err
	}
	signatureBytes, err := envelope.Signature.SignatureBytes()
	if err != nil {
		return time.Time{}, VerifierRecord{}, err
	}
	if err := verifySignedEnvelopeEd25519(verifier, canonicalPayload, signatureBytes); err != nil {
		return time.Time{}, VerifierRecord{}, err
	}

	timestamp, err := envelopeTimestamp(envelope)
	if err != nil {
		return time.Time{}, VerifierRecord{}, err
	}
	return timestamp, verifier, nil
}

func resolveVerifierIgnoringStatus(signature SignatureBlock, registry *VerifierRegistry) (VerifierRecord, error) {
	identity, err := signatureVerifierIdentity(signature)
	if err != nil {
		return VerifierRecord{}, err
	}
	verifier, ok := registry.byIdentity[identity]
	if !ok {
		return VerifierRecord{}, fmt.Errorf("verifier not found for %q", identity)
	}
	return verifier, nil
}

func envelopeTimestamp(envelope SignedObjectEnvelope) (time.Time, error) {
	switch envelope.PayloadSchemaID {
	case AuditEventSchemaID:
		return envelopeEventTimestamp(envelope.Payload)
	case AuditSegmentSealSchemaID:
		return envelopeSealTimestamp(envelope.Payload)
	case AuditReceiptSchemaID:
		return envelopeReceiptTimestamp(envelope.Payload)
	default:
		return time.Time{}, fmt.Errorf("unsupported payload schema for timestamp extraction %q", envelope.PayloadSchemaID)
	}
}

func envelopeEventTimestamp(payload json.RawMessage) (time.Time, error) {
	event, err := decodeAuditEventPayload(payload)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339, event.OccurredAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid occurred_at: %w", err)
	}
	return parsed, nil
}

func envelopeSealTimestamp(payload json.RawMessage) (time.Time, error) {
	seal := AuditSegmentSealPayload{}
	if err := json.Unmarshal(payload, &seal); err != nil {
		return time.Time{}, fmt.Errorf("decode segment seal payload: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339, seal.SealedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid sealed_at: %w", err)
	}
	return parsed, nil
}

func envelopeReceiptTimestamp(payload json.RawMessage) (time.Time, error) {
	receipt, err := decodeAuditReceiptPayload(payload)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339, receipt.RecordedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid recorded_at: %w", err)
	}
	return parsed, nil
}

func checkHistoricalAdmissibility(verifier VerifierRecord, signedAt time.Time) error {
	if verifier.Status == "active" || verifier.Status == "retired" {
		return nil
	}
	if verifier.Status != "revoked" && verifier.Status != "compromised" {
		return fmt.Errorf("unsupported verifier status %q", verifier.Status)
	}
	if verifier.StatusChangedAt == "" {
		return fmt.Errorf("%s verifier requires status_changed_at for historical admissibility checks", verifier.Status)
	}
	changedAt, err := time.Parse(time.RFC3339, verifier.StatusChangedAt)
	if err != nil {
		return fmt.Errorf("invalid status_changed_at: %w", err)
	}
	if !signedAt.Before(changedAt) {
		return fmt.Errorf("signature time %s is not before %s transition at %s", signedAt.Format(time.RFC3339), verifier.Status, changedAt.Format(time.RFC3339))
	}
	return nil
}

func isVerifierCurrentlyDegraded(verifier VerifierRecord, signedAt time.Time) bool {
	if verifier.Status != "revoked" && verifier.Status != "compromised" {
		return false
	}
	if verifier.StatusChangedAt == "" {
		return true
	}
	changedAt, err := time.Parse(time.RFC3339, verifier.StatusChangedAt)
	if err != nil {
		return true
	}
	return signedAt.Before(changedAt)
}
