package auditd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

// RecordInclusionByDigest resolves trusted record-inclusion material for one canonical record digest.
func (l *Ledger) RecordInclusionByDigest(recordDigest string) (AuditRecordInclusion, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	normalizedDigest, err := normalizedRecordDigestIdentity(recordDigest)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	lookup, ok, err := l.lookupRecordDigestLocked(normalizedDigest, false)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	if !ok {
		return AuditRecordInclusion{}, false, nil
	}
	return l.recordInclusionFromLookupLocked(normalizedDigest, lookup)
}

func (l *Ledger) recordInclusionFromLookupLocked(recordDigest string, lookup RecordLookup) (AuditRecordInclusion, bool, error) {
	segment, frame, err := l.recordInclusionSegmentFrameLocked(recordDigest, lookup)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	envelopeDigestIdentity, err := recordEnvelopeDigestIdentity(frame)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	recordDigests, recordDigestIdentities, err := segmentRecordDigestIdentities(segment)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	merkleRoot, merkleRootIdentity, err := recordInclusionMerkleMaterial(recordDigests)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	inclusion := newAuditRecordInclusion(recordDigest, envelopeDigestIdentity, lookup, len(segment.Frames), merkleRootIdentity, recordDigestIdentities)
	if err := l.attachSegmentSealInclusionLocked(&inclusion, lookup.SegmentID, merkleRoot); err != nil {
		return AuditRecordInclusion{}, false, err
	}
	return inclusion, true, nil
}

func (l *Ledger) recordInclusionSegmentFrameLocked(recordDigest string, lookup RecordLookup) (trustpolicy.AuditSegmentFilePayload, trustpolicy.AuditSegmentRecordFrame, error) {
	segment, err := l.loadSegment(lookup.SegmentID)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.AuditSegmentRecordFrame{}, err
	}
	if lookup.FrameIndex < 0 || lookup.FrameIndex >= len(segment.Frames) {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.AuditSegmentRecordFrame{}, fmt.Errorf("record lookup out of bounds for segment %q", lookup.SegmentID)
	}
	frame := segment.Frames[lookup.FrameIndex]
	if err := validateRecordInclusionFrame(recordDigest, frame); err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.AuditSegmentRecordFrame{}, err
	}
	return segment, frame, nil
}

func validateRecordInclusionFrame(recordDigest string, frame trustpolicy.AuditSegmentRecordFrame) error {
	matches, err := frameRecordDigestMatches(frame, recordDigest)
	if err != nil {
		return err
	}
	if !matches {
		return fmt.Errorf("record inclusion lookup mismatch for digest %q", recordDigest)
	}
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return err
	}
	return verifyFrameRecordDigest(frame, envelope)
}

func recordEnvelopeDigestIdentity(frame trustpolicy.AuditSegmentRecordFrame) (string, error) {
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return "", err
	}
	envelopeDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return "", err
	}
	return envelopeDigest.Identity()
}

func recordInclusionMerkleMaterial(recordDigests []trustpolicy.Digest) (trustpolicy.Digest, string, error) {
	merkleRoot, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot(recordDigests)
	if err != nil {
		return trustpolicy.Digest{}, "", err
	}
	merkleRootIdentity, err := merkleRoot.Identity()
	if err != nil {
		return trustpolicy.Digest{}, "", err
	}
	return merkleRoot, merkleRootIdentity, nil
}

func newAuditRecordInclusion(recordDigest string, envelopeDigestIdentity string, lookup RecordLookup, recordCount int, merkleRootIdentity string, recordDigestIdentities []string) AuditRecordInclusion {
	return AuditRecordInclusion{
		RecordDigest:         recordDigest,
		RecordEnvelopeDigest: envelopeDigestIdentity,
		SegmentID:            lookup.SegmentID,
		FrameIndex:           lookup.FrameIndex,
		SegmentRecordCount:   recordCount,
		OrderedMerkle: AuditRecordInclusionOrderedMerkleLookup{
			Profile:              trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1,
			LeafIndex:            lookup.FrameIndex,
			LeafCount:            recordCount,
			SegmentMerkleRoot:    merkleRootIdentity,
			SegmentRecordDigests: recordDigestIdentities,
		},
	}
}

func segmentRecordDigestIdentities(segment trustpolicy.AuditSegmentFilePayload) ([]trustpolicy.Digest, []string, error) {
	recordDigests := make([]trustpolicy.Digest, 0, len(segment.Frames))
	recordDigestIdentities := make([]string, 0, len(segment.Frames))
	for frameIndex, frame := range segment.Frames {
		recordDigests = append(recordDigests, frame.RecordDigest)
		identity, err := frame.RecordDigest.Identity()
		if err != nil {
			return nil, nil, fmt.Errorf("segment frame %d record_digest invalid: %w", frameIndex, err)
		}
		recordDigestIdentities = append(recordDigestIdentities, identity)
	}
	return recordDigests, recordDigestIdentities, nil
}

func (l *Ledger) attachSegmentSealInclusionLocked(inclusion *AuditRecordInclusion, segmentID string, computedMerkleRoot trustpolicy.Digest) error {
	sealLookup, hasSeal, err := l.lookupSegmentSealLocked(segmentID, false)
	if err != nil {
		return err
	}
	if !hasSeal {
		return nil
	}
	sealPayload, err := l.loadSealPayloadByDigestIdentityLocked(sealLookup.SealDigest)
	if err != nil {
		return err
	}
	sealRootIdentity, err := validateSegmentSealInclusion(segmentID, sealLookup, sealPayload, computedMerkleRoot)
	if err != nil {
		return err
	}
	return applySegmentSealInclusion(inclusion, sealLookup, sealPayload, sealRootIdentity)
}

func validateSegmentSealInclusion(segmentID string, sealLookup SegmentSealLookup, sealPayload trustpolicy.AuditSegmentSealPayload, computedMerkleRoot trustpolicy.Digest) (string, error) {
	if sealPayload.SegmentID != segmentID {
		return "", fmt.Errorf("segment seal lookup mismatch: segment %q payload segment %q", segmentID, sealPayload.SegmentID)
	}
	if sealPayload.SealChainIndex != sealLookup.SealChainIndex {
		return "", fmt.Errorf("segment seal lookup mismatch: chain index %d payload %d", sealLookup.SealChainIndex, sealPayload.SealChainIndex)
	}
	computedIdentity, err := computedMerkleRoot.Identity()
	if err != nil {
		return "", err
	}
	sealRootIdentity, err := sealPayload.MerkleRoot.Identity()
	if err != nil {
		return "", err
	}
	if computedIdentity != sealRootIdentity {
		return "", fmt.Errorf("segment merkle root mismatch: computed %q seal %q", computedIdentity, sealRootIdentity)
	}
	return sealRootIdentity, nil
}

func applySegmentSealInclusion(inclusion *AuditRecordInclusion, sealLookup SegmentSealLookup, sealPayload trustpolicy.AuditSegmentSealPayload, sealRootIdentity string) error {
	inclusion.SegmentSealDigest = sealLookup.SealDigest
	inclusion.SegmentSealChainIndex = &sealLookup.SealChainIndex
	if sealPayload.PreviousSealDigest != nil {
		previousIdentity, err := sealPayload.PreviousSealDigest.Identity()
		if err != nil {
			return err
		}
		inclusion.PreviousSealDigest = previousIdentity
	}
	inclusion.OrderedMerkle.SegmentMerkleRoot = sealRootIdentity
	return nil
}

func (l *Ledger) loadSealPayloadByDigestIdentityLocked(identity string) (trustpolicy.AuditSegmentSealPayload, error) {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, err
	}
	path := filepath.Join(l.rootDir, sidecarDirName, sealsDirName, digest.Hash+".json")
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := readJSONFile(path, &envelope); err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, err
	}
	computedDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, err
	}
	computedIdentity, _ := computedDigest.Identity()
	if computedIdentity != identity {
		return trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("seal sidecar digest mismatch: expected %q computed %q", identity, computedIdentity)
	}
	seal := trustpolicy.AuditSegmentSealPayload{}
	if err := json.Unmarshal(envelope.Payload, &seal); err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("decode seal payload %q: %w", digest.Hash+".json", err)
	}
	if err := trustpolicy.ValidateAuditSegmentSealPayload(seal); err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("validate seal payload %q: %w", digest.Hash+".json", err)
	}
	return seal, nil
}
