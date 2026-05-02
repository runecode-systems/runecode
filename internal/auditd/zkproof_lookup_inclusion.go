package auditd

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type AuditRecordInclusion struct {
	SegmentID            string
	RecordDigest         trustpolicy.Digest
	RecordEnvelope       trustpolicy.SignedObjectEnvelope
	RecordIndex          int
	SegmentRecordDigests []trustpolicy.Digest
	SealEnvelopeDigest   trustpolicy.Digest
	SealPayload          trustpolicy.AuditSegmentSealPayload
}

func (l *Ledger) AuditRecordInclusion(recordDigest string) (AuditRecordInclusion, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	normalizedDigest, recordDigestValue, err := resolvedAuditRecordDigest(recordDigest)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	if err := l.ensureProofLookupIndexLocked(); err != nil {
		return AuditRecordInclusion{}, false, err
	}
	lookup, found, err := l.lookupRecordInclusionLocked(normalizedDigest)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	if !found {
		return AuditRecordInclusion{}, false, nil
	}
	segment, err := l.loadSegment(lookup.SegmentID)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	if ok, err := l.validateInclusionFrameIndexLocked(lookup, segment); err != nil || !ok {
		return AuditRecordInclusion{}, false, err
	}
	frame := segment.Frames[lookup.FrameIndex]
	matched, err := l.validateInclusionFrameDigestLocked(frame, normalizedDigest)
	if err != nil || !matched {
		return AuditRecordInclusion{}, false, err
	}
	inclusion, err := l.buildAuditRecordInclusionLocked(segment, frame, lookup.FrameIndex, recordDigestValue)
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	return inclusion, true, nil
}

func (l *Ledger) lookupRecordInclusionLocked(normalizedDigest string) (recordInclusionLookup, bool, error) {
	lookup, found := l.lookupIndex.RecordInclusions[normalizedDigest]
	if found {
		return lookup, true, nil
	}
	if err := l.refreshProofLookupIndexLocked(); err != nil {
		return recordInclusionLookup{}, false, err
	}
	lookup, found = l.lookupIndex.RecordInclusions[normalizedDigest]
	return lookup, found, nil
}

func (l *Ledger) validateInclusionFrameIndexLocked(lookup recordInclusionLookup, segment trustpolicy.AuditSegmentFilePayload) (bool, error) {
	if lookup.FrameIndex >= 0 && lookup.FrameIndex < len(segment.Frames) {
		return true, nil
	}
	if err := l.refreshProofLookupIndexLocked(); err != nil {
		return false, err
	}
	return false, nil
}

func (l *Ledger) validateInclusionFrameDigestLocked(frame trustpolicy.AuditSegmentRecordFrame, normalizedDigest string) (bool, error) {
	matches, err := frameRecordDigestMatches(frame, normalizedDigest)
	if err != nil {
		return false, err
	}
	if matches {
		return true, nil
	}
	if err := l.refreshProofLookupIndexLocked(); err != nil {
		return false, err
	}
	return false, nil
}

func resolvedAuditRecordDigest(recordDigest string) (string, trustpolicy.Digest, error) {
	normalizedDigest, err := normalizedRecordDigestIdentity(recordDigest)
	if err != nil {
		return "", trustpolicy.Digest{}, err
	}
	value, err := digestFromIdentity(normalizedDigest)
	if err != nil {
		return "", trustpolicy.Digest{}, err
	}
	return normalizedDigest, value, nil
}

func (l *Ledger) buildAuditRecordInclusionLocked(segment trustpolicy.AuditSegmentFilePayload, frame trustpolicy.AuditSegmentRecordFrame, index int, recordDigestValue trustpolicy.Digest) (AuditRecordInclusion, error) {
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return AuditRecordInclusion{}, err
	}
	if err := verifyFrameRecordDigest(frame, envelope); err != nil {
		return AuditRecordInclusion{}, err
	}
	_, sealDigest, sealPayload, err := l.loadSealEnvelopeForSegmentLocked(segment.Header.SegmentID)
	if err != nil {
		return AuditRecordInclusion{}, err
	}
	recordDigests, err := segmentRecordDigests(segment)
	if err != nil {
		return AuditRecordInclusion{}, err
	}
	return AuditRecordInclusion{
		SegmentID:            segment.Header.SegmentID,
		RecordDigest:         recordDigestValue,
		RecordEnvelope:       envelope,
		RecordIndex:          index,
		SegmentRecordDigests: recordDigests,
		SealEnvelopeDigest:   sealDigest,
		SealPayload:          sealPayload,
	}, nil
}

func (i AuditRecordInclusion) Validate() error {
	if strings.TrimSpace(i.SegmentID) == "" {
		return fmt.Errorf("segment_id is required")
	}
	if _, err := i.RecordDigest.Identity(); err != nil {
		return fmt.Errorf("record_digest: %w", err)
	}
	if _, err := i.SealEnvelopeDigest.Identity(); err != nil {
		return fmt.Errorf("seal_envelope_digest: %w", err)
	}
	if i.RecordIndex < 0 || i.RecordIndex >= len(i.SegmentRecordDigests) {
		return fmt.Errorf("record_index out of range")
	}
	return nil
}
