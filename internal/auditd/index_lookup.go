package auditd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (l *Ledger) LookupRecordDigest(recordDigestIdentity string) (RecordLookup, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	normalizedDigest, err := normalizedRecordDigestIdentity(recordDigestIdentity)
	if err != nil {
		return RecordLookup{}, false, err
	}
	return l.lookupRecordDigestLocked(normalizedDigest, false)
}

func (l *Ledger) lookupRecordDigestLocked(recordDigestIdentity string, refreshed bool) (RecordLookup, bool, error) {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		if errors.Is(err, errStaleLegacyDerivedIndexRepresentation) {
			return RecordLookup{}, false, err
		}
		return RecordLookup{}, false, err
	}
	lookup, ok := index.RecordDigestLookup[recordDigestIdentity]
	if !ok {
		return l.retryMissingRecordLookupLocked(recordDigestIdentity, refreshed)
	}
	if validateErr := l.validateRecordLookupAgainstCanonicalLocked(recordDigestIdentity, lookup); validateErr == nil {
		return lookup, true, nil
	}
	if refreshed {
		return RecordLookup{}, false, fmt.Errorf("record lookup mismatch after index refresh for %q", recordDigestIdentity)
	}
	if _, refreshErr := l.refreshDerivedIndexLocked("record lookup mismatch"); refreshErr != nil {
		return RecordLookup{}, false, refreshErr
	}
	return l.lookupRecordDigestLocked(recordDigestIdentity, true)
}

func (l *Ledger) retryMissingRecordLookupLocked(recordDigestIdentity string, refreshed bool) (RecordLookup, bool, error) {
	if refreshed {
		return RecordLookup{}, false, nil
	}
	if _, refreshErr := l.refreshDerivedIndexLocked("record digest not found in index"); refreshErr != nil {
		return RecordLookup{}, false, refreshErr
	}
	return l.lookupRecordDigestLocked(recordDigestIdentity, true)
}

func (l *Ledger) LookupSegmentSeal(segmentID string) (SegmentSealLookup, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	normalizedSegmentID := strings.TrimSpace(segmentID)
	if normalizedSegmentID == "" {
		return SegmentSealLookup{}, false, nil
	}
	return l.lookupSegmentSealLocked(normalizedSegmentID, false)
}

func (l *Ledger) lookupSegmentSealLocked(segmentID string, refreshed bool) (SegmentSealLookup, bool, error) {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		if errors.Is(err, errStaleLegacyDerivedIndexRepresentation) {
			return SegmentSealLookup{}, false, err
		}
		return SegmentSealLookup{}, false, err
	}
	lookup, ok := index.SegmentSealLookup[segmentID]
	if !ok {
		return l.retryMissingSegmentSealLookupLocked(segmentID, refreshed)
	}
	if validateErr := l.validateSegmentSealLookupAgainstCanonicalLocked(segmentID, lookup); validateErr == nil {
		return lookup, true, nil
	}
	if refreshed {
		return SegmentSealLookup{}, false, fmt.Errorf("segment seal lookup mismatch after index refresh for %q", segmentID)
	}
	if _, refreshErr := l.refreshDerivedIndexLocked("segment seal lookup mismatch"); refreshErr != nil {
		return SegmentSealLookup{}, false, refreshErr
	}
	return l.lookupSegmentSealLocked(segmentID, true)
}

func (l *Ledger) retryMissingSegmentSealLookupLocked(segmentID string, refreshed bool) (SegmentSealLookup, bool, error) {
	if refreshed {
		return SegmentSealLookup{}, false, nil
	}
	if _, refreshErr := l.refreshDerivedIndexLocked("segment seal not found in index"); refreshErr != nil {
		return SegmentSealLookup{}, false, refreshErr
	}
	return l.lookupSegmentSealLocked(segmentID, true)
}

func (l *Ledger) LookupSealDigestByChainIndex(sealChainIndex int64) (string, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if sealChainIndex < 0 {
		return "", false, nil
	}
	return l.lookupSealDigestByChainIndexLocked(sealChainIndex, false)
}

func (l *Ledger) lookupSealDigestByChainIndexLocked(sealChainIndex int64, refreshed bool) (string, bool, error) {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		if errors.Is(err, errStaleLegacyDerivedIndexRepresentation) {
			return "", false, err
		}
		return "", false, err
	}
	sealDigest, ok := index.SealChainIndexLookup[strconv.FormatInt(sealChainIndex, 10)]
	if !ok {
		return l.retryMissingSealChainLookupLocked(sealChainIndex, refreshed)
	}
	if validateErr := l.validateSealChainLookupAgainstCanonicalLocked(sealChainIndex, sealDigest, index); validateErr == nil {
		return sealDigest, true, nil
	}
	if refreshed {
		return "", false, fmt.Errorf("seal chain index lookup mismatch after index refresh for %d", sealChainIndex)
	}
	if _, refreshErr := l.refreshDerivedIndexLocked("seal chain index lookup mismatch"); refreshErr != nil {
		return "", false, refreshErr
	}
	return l.lookupSealDigestByChainIndexLocked(sealChainIndex, true)
}

func (l *Ledger) retryMissingSealChainLookupLocked(sealChainIndex int64, refreshed bool) (string, bool, error) {
	if refreshed {
		return "", false, nil
	}
	if _, refreshErr := l.refreshDerivedIndexLocked("seal chain index not found in index"); refreshErr != nil {
		return "", false, refreshErr
	}
	return l.lookupSealDigestByChainIndexLocked(sealChainIndex, true)
}
