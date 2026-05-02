package auditd

import (
	"fmt"
	"strconv"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) currentSegmentEvidenceLocked() (trustpolicy.AuditSegmentFilePayload, trustpolicy.SignedObjectEnvelope, trustpolicy.AuditSegmentSealPayload, *trustpolicy.Digest, []byte, error) {
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	if state.LastSealedSegmentID == "" {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, fmt.Errorf("no sealed segment available for verification")
	}
	segment, err := l.loadSegment(state.LastSealedSegmentID)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	rawBytes, err := l.rawSegmentFramedBytes(segment)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	sealEnvelope, _, sealPayload, err := l.loadSealEnvelopeForSegmentLocked(state.LastSealedSegmentID)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	previousDigest, err := l.previousSealDigestByIndexLocked(sealPayload.SealChainIndex - 1)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, trustpolicy.SignedObjectEnvelope{}, trustpolicy.AuditSegmentSealPayload{}, nil, nil, err
	}
	return segment, sealEnvelope, sealPayload, previousDigest, rawBytes, nil
}

func (l *Ledger) loadSealEnvelopeForSegmentLocked(segmentID string) (trustpolicy.SignedObjectEnvelope, trustpolicy.Digest, trustpolicy.AuditSegmentSealPayload, error) {
	lookup, err := l.segmentSealLookupLocked(segmentID)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, err
	}
	envelope, digest, payload, err := l.readVerifiedSegmentSealEnvelope(lookup.DigestIdentity)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, err
	}
	if payload.SegmentID != segmentID {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("segment seal %s does not match requested segment %s", payload.SegmentID, segmentID)
	}
	return envelope, digest, payload, nil
}

func (l *Ledger) segmentSealLookupLocked(segmentID string) (segmentSealLookup, error) {
	if err := l.ensureProofLookupIndexLocked(); err != nil {
		return segmentSealLookup{}, err
	}
	lookup, ok := l.lookupIndex.SegmentSeals[segmentID]
	if ok {
		return lookup, nil
	}
	if err := l.refreshProofLookupIndexLocked(); err != nil {
		return segmentSealLookup{}, err
	}
	lookup, ok = l.lookupIndex.SegmentSeals[segmentID]
	if !ok {
		return segmentSealLookup{}, fmt.Errorf("no segment seal found for %s", segmentID)
	}
	return lookup, nil
}

func (l *Ledger) readVerifiedSegmentSealEnvelope(identity string) (trustpolicy.SignedObjectEnvelope, trustpolicy.Digest, trustpolicy.AuditSegmentSealPayload, error) {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, err
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := readJSONFile(sidecarPath(l.rootDir, sealsDirName, identity), &envelope); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, err
	}
	computedDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, err
	}
	if mustDigestIdentity(computedDigest) != identity {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("segment seal content digest mismatch for %s", identity)
	}
	payload, err := decodeAndValidateSealEnvelope(envelope)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, err
	}
	return envelope, digest, payload, nil
}

func (l *Ledger) previousSealDigestByIndexLocked(index int64) (*trustpolicy.Digest, error) {
	if index < 0 {
		return nil, nil
	}
	if err := l.ensureProofLookupIndexLocked(); err != nil {
		return nil, err
	}
	identity, ok := l.lookupIndex.SealByChainIndex[strconv.FormatInt(index, 10)]
	if !ok {
		if err := l.refreshProofLookupIndexLocked(); err != nil {
			return nil, err
		}
		identity, ok = l.lookupIndex.SealByChainIndex[strconv.FormatInt(index, 10)]
		if !ok {
			return nil, fmt.Errorf("missing previous seal digest at chain index %d", index)
		}
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return nil, err
	}
	return &digest, nil
}
