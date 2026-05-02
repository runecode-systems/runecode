package auditd

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

// SignedEnvelopeByRecordDigest resolves one signed audit envelope by stable record digest identity.
func (l *Ledger) SignedEnvelopeByRecordDigest(recordDigest string) (trustpolicy.SignedObjectEnvelope, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	normalizedDigest, err := normalizedRecordDigestIdentity(recordDigest)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, false, err
	}
	lookup, ok, err := l.lookupRecordDigestLocked(normalizedDigest, false)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, false, err
	}
	if !ok {
		return trustpolicy.SignedObjectEnvelope{}, false, nil
	}
	segment, err := l.loadSegment(lookup.SegmentID)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, false, err
	}
	if lookup.FrameIndex < 0 || lookup.FrameIndex >= len(segment.Frames) {
		return trustpolicy.SignedObjectEnvelope{}, false, fmt.Errorf("record lookup out of bounds for segment %q", lookup.SegmentID)
	}
	frame := segment.Frames[lookup.FrameIndex]
	matches, err := frameRecordDigestMatches(frame, normalizedDigest)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, false, err
	}
	if !matches {
		return trustpolicy.SignedObjectEnvelope{}, false, fmt.Errorf("record lookup mismatch for digest %q", normalizedDigest)
	}
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, false, err
	}
	if err := verifyFrameRecordDigest(frame, envelope); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, false, err
	}
	return envelope, true, nil
}

func normalizedRecordDigestIdentity(recordDigest string) (string, error) {
	recordDigest = strings.TrimSpace(recordDigest)
	if err := validateRecordDigestIdentity(recordDigest); err != nil {
		return "", err
	}
	return recordDigest, nil
}

func signedEnvelopeByRecordDigestFromSegments(segments []trustpolicy.AuditSegmentFilePayload, recordDigest string) (trustpolicy.SignedObjectEnvelope, bool, error) {
	for _, segment := range segments {
		envelope, found, err := signedEnvelopeByRecordDigestFromFrames(segment.Frames, recordDigest)
		if err != nil {
			return trustpolicy.SignedObjectEnvelope{}, false, err
		}
		if found {
			return envelope, true, nil
		}
	}
	return trustpolicy.SignedObjectEnvelope{}, false, nil
}

func signedEnvelopeByRecordDigestFromFrames(frames []trustpolicy.AuditSegmentRecordFrame, recordDigest string) (trustpolicy.SignedObjectEnvelope, bool, error) {
	for _, frame := range frames {
		matches, err := frameRecordDigestMatches(frame, recordDigest)
		if err != nil {
			return trustpolicy.SignedObjectEnvelope{}, false, err
		}
		if !matches {
			continue
		}
		envelope, err := decodeFrameEnvelope(frame)
		if err != nil {
			return trustpolicy.SignedObjectEnvelope{}, false, err
		}
		if err := verifyFrameRecordDigest(frame, envelope); err != nil {
			return trustpolicy.SignedObjectEnvelope{}, false, err
		}
		return envelope, true, nil
	}
	return trustpolicy.SignedObjectEnvelope{}, false, nil
}

func frameRecordDigestMatches(frame trustpolicy.AuditSegmentRecordFrame, expected string) (bool, error) {
	identity, err := frame.RecordDigest.Identity()
	if err != nil {
		return false, fmt.Errorf("invalid persisted frame record_digest: %w", err)
	}
	return identity == expected, nil
}

func validateRecordDigestIdentity(recordDigest string) error {
	parts := strings.SplitN(recordDigest, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("record_digest must use identity form sha256:<64 lowercase hex>")
	}
	if _, err := (trustpolicy.Digest{HashAlg: parts[0], Hash: parts[1]}).Identity(); err != nil {
		return fmt.Errorf("record_digest: %w", err)
	}
	return nil
}
