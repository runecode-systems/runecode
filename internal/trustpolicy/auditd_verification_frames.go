package trustpolicy

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func verifySegmentFramesAndEvents(input AuditVerificationInput, registry *VerifierRegistry, report *AuditVerificationReportPayload) ([]Digest, []SignedObjectEnvelope, []time.Time) {
	streams := map[string]streamState{}
	frameDigests := make([]Digest, 0, len(input.Segment.Frames))
	frameEnvelopes := make([]SignedObjectEnvelope, 0, len(input.Segment.Frames))
	eventTimes := make([]time.Time, 0, len(input.Segment.Frames))

	for index := range input.Segment.Frames {
		verified, ok := verifySegmentFrame(index, input, registry, report, streams)
		if !ok {
			continue
		}
		streams[verified.event.EmitterStreamID] = streamState{seq: verified.event.Seq, digest: verified.recordDigest}
		frameDigests = append(frameDigests, verified.recordDigest)
		frameEnvelopes = append(frameEnvelopes, verified.envelope)
		eventTimes = append(eventTimes, verified.eventTime)
	}

	return frameDigests, frameEnvelopes, eventTimes
}

type verifiedSegmentFrame struct {
	recordDigest Digest
	envelope     SignedObjectEnvelope
	event        AuditEventPayload
	eventTime    time.Time
}

func verifySegmentFrame(index int, input AuditVerificationInput, registry *VerifierRegistry, report *AuditVerificationReportPayload, streams map[string]streamState) (verifiedSegmentFrame, bool) {
	frame := input.Segment.Frames[index]
	frameRecordDigest := frame.RecordDigest

	rawEnvelopeBytes, ok := decodeFrameEnvelopeBytes(index, frame, input, report, frameRecordDigest)
	if !ok {
		return verifiedSegmentFrame{}, false
	}

	if !validateFrameCanonicalDigest(index, input, report, frameRecordDigest, rawEnvelopeBytes) {
		return verifiedSegmentFrame{}, false
	}

	envelope, ok := decodeFrameEnvelope(index, input, report, frameRecordDigest, rawEnvelopeBytes)
	if !ok {
		return verifiedSegmentFrame{}, false
	}

	eventTime, event, ok := verifyFrameEnvelopeAndEvent(index, input, registry, report, frameRecordDigest, envelope)
	if !ok {
		return verifiedSegmentFrame{}, false
	}

	if !verifyFrameStreamContinuity(index, input, report, streams, event, frameRecordDigest) {
		return verifiedSegmentFrame{}, false
	}

	return verifiedSegmentFrame{recordDigest: frameRecordDigest, envelope: envelope, event: event, eventTime: eventTime}, true
}

func decodeFrameEnvelopeBytes(index int, frame AuditSegmentRecordFrame, input AuditVerificationInput, report *AuditVerificationReportPayload, frameRecordDigest Digest) ([]byte, bool) {
	rawEnvelopeBytes, err := base64.StdEncoding.DecodeString(frame.CanonicalSignedEnvelopeBytes)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentFrameByteLengthMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d canonical_signed_envelope_bytes is not base64: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return nil, false
	}
	if int64(len(rawEnvelopeBytes)) != frame.ByteLength {
		addHardFailure(report, AuditVerificationReasonSegmentFrameByteLengthMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d byte_length=%d does not match decoded envelope bytes=%d", index, frame.ByteLength, len(rawEnvelopeBytes)), input.Segment.Header.SegmentID, &frameRecordDigest)
		return nil, false
	}
	return rawEnvelopeBytes, true
}

func validateFrameCanonicalDigest(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, frameRecordDigest Digest, rawEnvelopeBytes []byte) bool {
	canonicalEnvelopeBytes, err := jsoncanonicalizer.Transform(rawEnvelopeBytes)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentFrameDigestMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d envelope canonicalization failed: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return false
	}
	if !bytes.Equal(rawEnvelopeBytes, canonicalEnvelopeBytes) {
		addHardFailure(report, AuditVerificationReasonSegmentFrameDigestMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d envelope bytes are not canonical RFC8785 JCS", index), input.Segment.Header.SegmentID, &frameRecordDigest)
		return false
	}
	sum := sha256.Sum256(canonicalEnvelopeBytes)
	computedDigest := Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}
	computedID, _ := computedDigest.Identity()
	expectedID, digestErr := frameRecordDigest.Identity()
	if digestErr != nil {
		addHardFailure(report, AuditVerificationReasonSegmentFrameDigestMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d digest invalid: %v", index, digestErr), input.Segment.Header.SegmentID, &frameRecordDigest)
		return false
	}
	if computedID != expectedID {
		addHardFailure(report, AuditVerificationReasonSegmentFrameDigestMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d record_digest mismatch: got %q want %q", index, computedID, expectedID), input.Segment.Header.SegmentID, &frameRecordDigest)
		return false
	}
	return true
}

func decodeFrameEnvelope(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, frameRecordDigest Digest, rawEnvelopeBytes []byte) (SignedObjectEnvelope, bool) {
	envelope := SignedObjectEnvelope{}
	if err := json.Unmarshal(rawEnvelopeBytes, &envelope); err != nil {
		addHardFailure(report, AuditVerificationReasonSegmentFrameDigestMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d canonical envelope decode failed: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return SignedObjectEnvelope{}, false
	}
	return envelope, true
}

func verifyFrameEnvelopeAndEvent(index int, input AuditVerificationInput, registry *VerifierRegistry, report *AuditVerificationReportPayload, frameRecordDigest Digest, envelope SignedObjectEnvelope) (time.Time, AuditEventPayload, bool) {
	eventTime, verifierRecord, verifyErr := verifyEnvelopeHistoricallyAdmissible(envelope, registry, AuditEventSchemaID, AuditEventSchemaVersion)
	if verifyErr != nil {
		addHardFailure(report, AuditVerificationReasonDetachedSignatureInvalid, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d detached signature invalid: %v", index, verifyErr), input.Segment.Header.SegmentID, &frameRecordDigest)
		return time.Time{}, AuditEventPayload{}, false
	}
	if admissibleErr := checkHistoricalAdmissibility(verifierRecord, eventTime); admissibleErr != nil {
		addHardFailure(report, AuditVerificationReasonSignerHistoricallyInadmissible, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d signer historically inadmissible: %v", index, admissibleErr), input.Segment.Header.SegmentID, &frameRecordDigest)
		return time.Time{}, AuditEventPayload{}, false
	}
	if isVerifierCurrentlyDegraded(verifierRecord, eventTime) {
		addDegraded(report, AuditVerificationReasonSignerCurrentlyRevokedOrCompromised, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d signer is currently %s after event time", index, verifierRecord.Status), input.Segment.Header.SegmentID, &frameRecordDigest)
	}

	event, ok := decodeAndValidateFrameEvent(index, input, report, frameRecordDigest, envelope)
	if !ok {
		return time.Time{}, AuditEventPayload{}, false
	}
	return eventTime, event, true
}

func decodeAndValidateFrameEvent(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, frameRecordDigest Digest, envelope SignedObjectEnvelope) (AuditEventPayload, bool) {
	event, err := decodeAuditEventPayload(envelope.Payload)
	if err != nil {
		addHardFailure(report, AuditVerificationReasonEventContractMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d event decode failed: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return AuditEventPayload{}, false
	}
	entry, ok := validateFrameEventContract(index, input, report, frameRecordDigest, event)
	if !ok {
		return AuditEventPayload{}, false
	}
	if !validateFrameEventSignerEvidence(index, input, report, frameRecordDigest, event, envelope.Signature, entry) {
		return AuditEventPayload{}, false
	}
	return event, true
}

func validateFrameEventContract(index int, input AuditVerificationInput, report *AuditVerificationReportPayload, frameRecordDigest Digest, event AuditEventPayload) (AuditEventContractCatalogEntry, bool) {
	if err := validateAuditEventPayloadShape(event); err != nil {
		addHardFailure(report, AuditVerificationReasonEventContractMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d event payload shape invalid: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return AuditEventContractCatalogEntry{}, false
	}
	if err := validateAuditEventPayloadHash(event); err != nil {
		addHardFailure(report, AuditVerificationReasonEventContractMismatch, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d event payload hash invalid: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return AuditEventContractCatalogEntry{}, false
	}
	entry, err := validateAuditEventAgainstCatalog(event, input.EventContractCatalog)
	if err != nil {
		reason := AuditVerificationReasonEventContractMismatch
		if strings.Contains(err.Error(), "no event-contract entry") {
			reason = AuditVerificationReasonEventContractMissing
		}
		addHardFailure(report, reason, AuditVerificationDimensionIntegrity, fmt.Sprintf("frame %d event contract mismatch: %v", index, err), input.Segment.Header.SegmentID, &frameRecordDigest)
		return AuditEventContractCatalogEntry{}, false
	}
	return entry, true
}
