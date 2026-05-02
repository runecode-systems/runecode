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
	segments, err := l.listSegments()
	if err != nil {
		return AuditRecordInclusion{}, false, err
	}
	for _, segment := range segments {
		inclusion, found, err := l.auditRecordInclusionForSegmentLocked(segment, normalizedDigest, recordDigestValue)
		if err != nil {
			return AuditRecordInclusion{}, false, err
		}
		if found {
			return inclusion, true, nil
		}
	}
	return AuditRecordInclusion{}, false, nil
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

func (l *Ledger) auditRecordInclusionForSegmentLocked(segment trustpolicy.AuditSegmentFilePayload, normalizedDigest string, recordDigestValue trustpolicy.Digest) (AuditRecordInclusion, bool, error) {
	for index, frame := range segment.Frames {
		matches, err := frameRecordDigestMatches(frame, normalizedDigest)
		if err != nil {
			return AuditRecordInclusion{}, false, err
		}
		if !matches {
			continue
		}
		inclusion, err := l.buildAuditRecordInclusionLocked(segment, frame, index, recordDigestValue)
		if err != nil {
			return AuditRecordInclusion{}, false, err
		}
		return inclusion, true, nil
	}
	return AuditRecordInclusion{}, false, nil
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
