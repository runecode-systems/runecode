package auditd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) validateSealForSegment(segment trustpolicy.AuditSegmentFilePayload, state ledgerState, envelope trustpolicy.SignedObjectEnvelope) error {
	if err := l.verifySealEnvelopeSignatureLocked(envelope); err != nil {
		return err
	}
	seal, err := decodeAndValidateSealEnvelope(envelope)
	if err != nil {
		return err
	}
	if err := validateSealFrameAlignment(segment, seal); err != nil {
		return err
	}
	recordDigests, err := segmentRecordDigests(segment)
	if err != nil {
		return err
	}
	raw, err := l.rawSegmentFramedBytes(segment)
	if err != nil {
		return err
	}
	if err := trustpolicy.VerifyOrderedAuditSegmentMerkleRoot(recordDigests, seal.MerkleRoot); err != nil {
		return err
	}
	if err := trustpolicy.VerifySegmentFileHash(raw, seal.SegmentFileHash); err != nil {
		return err
	}
	return trustpolicy.ValidateAuditSegmentSealChainLink(seal, previousSealDigestFromState(state))
}

func (l *Ledger) verifySealEnvelopeSignatureLocked(envelope trustpolicy.SignedObjectEnvelope) error {
	verifierRecords := []trustpolicy.VerifierRecord{}
	if err := readJSONFile(filepath.Join(l.rootDir, "contracts", "verifier-records.json"), &verifierRecords); err != nil {
		return fmt.Errorf("load verifier records for seal verification: %w", err)
	}
	registry, err := trustpolicy.NewVerifierRegistry(verifierRecords)
	if err != nil {
		return fmt.Errorf("build verifier registry for seal verification: %w", err)
	}
	if err := trustpolicy.VerifySignedEnvelope(envelope, registry, trustpolicy.EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   trustpolicy.AuditSegmentSealSchemaID,
		ExpectedPayloadVersion:    trustpolicy.AuditSegmentSealSchemaVersion,
	}); err != nil {
		return fmt.Errorf("verify segment seal signature: %w", err)
	}
	return nil
}

func decodeAndValidateSealEnvelope(envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.AuditSegmentSealPayload, error) {
	if envelope.PayloadSchemaID != trustpolicy.AuditSegmentSealSchemaID || envelope.PayloadSchemaVersion != trustpolicy.AuditSegmentSealSchemaVersion {
		return trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("segment seal envelope payload schema mismatch")
	}
	seal := trustpolicy.AuditSegmentSealPayload{}
	if err := json.Unmarshal(envelope.Payload, &seal); err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("decode seal payload: %w", err)
	}
	if err := trustpolicy.ValidateAuditSegmentSealPayload(seal); err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, err
	}
	return seal, nil
}

func validateSealFrameAlignment(segment trustpolicy.AuditSegmentFilePayload, seal trustpolicy.AuditSegmentSealPayload) error {
	if seal.SegmentID != segment.Header.SegmentID {
		return fmt.Errorf("seal segment_id %q does not match open segment %q", seal.SegmentID, segment.Header.SegmentID)
	}
	if int64(len(segment.Frames)) != seal.EventCount {
		return fmt.Errorf("seal event_count %d does not match frame count %d", seal.EventCount, len(segment.Frames))
	}
	if got, _ := segment.Frames[0].RecordDigest.Identity(); got != mustID(seal.FirstRecordDigest) {
		return fmt.Errorf("seal first_record_digest mismatch")
	}
	if got, _ := segment.Frames[len(segment.Frames)-1].RecordDigest.Identity(); got != mustID(seal.LastRecordDigest) {
		return fmt.Errorf("seal last_record_digest mismatch")
	}
	return nil
}

func segmentRecordDigests(segment trustpolicy.AuditSegmentFilePayload) ([]trustpolicy.Digest, error) {
	digests := make([]trustpolicy.Digest, 0, len(segment.Frames))
	for _, frame := range segment.Frames {
		digests = append(digests, frame.RecordDigest)
	}
	return digests, nil
}

func previousSealDigestFromState(state ledgerState) *trustpolicy.Digest {
	if state.LastSealEnvelopeDigest == "" {
		return nil
	}
	return &trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimPrefix(state.LastSealEnvelopeDigest, "sha256:")}
}

func mustID(d trustpolicy.Digest) string {
	id, _ := d.Identity()
	return id
}
