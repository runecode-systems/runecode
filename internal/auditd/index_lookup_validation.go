package auditd

import "fmt"

func (l *Ledger) validateRecordLookupAgainstCanonicalLocked(recordDigest string, lookup RecordLookup) error {
	segment, err := l.loadSegment(lookup.SegmentID)
	if err != nil {
		return err
	}
	if lookup.FrameIndex < 0 || lookup.FrameIndex >= len(segment.Frames) {
		return fmt.Errorf("frame index %d outside segment %q bounds", lookup.FrameIndex, lookup.SegmentID)
	}
	frame := segment.Frames[lookup.FrameIndex]
	matches, err := frameRecordDigestMatches(frame, recordDigest)
	if err != nil {
		return err
	}
	if !matches {
		return fmt.Errorf("record digest points at mismatched frame")
	}
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return err
	}
	return verifyFrameRecordDigest(frame, envelope)
}

func (l *Ledger) validateSegmentSealLookupAgainstCanonicalLocked(segmentID string, lookup SegmentSealLookup) error {
	metadata, err := l.loadSealMetadataByDigestIdentityLocked(lookup.SealDigest)
	if err != nil {
		return err
	}
	if metadata.SegmentID != segmentID {
		return fmt.Errorf("segment seal lookup mismatch: index segment %q canonical segment %q", segmentID, metadata.SegmentID)
	}
	if metadata.SealChainIndex != lookup.SealChainIndex {
		return fmt.Errorf("segment seal lookup mismatch: index chain %d canonical chain %d", lookup.SealChainIndex, metadata.SealChainIndex)
	}
	return nil
}

func (l *Ledger) validateSealChainLookupAgainstCanonicalLocked(sealChainIndex int64, sealDigest string, index derivedIndex) error {
	metadata, err := l.loadSealMetadataByDigestIdentityLocked(sealDigest)
	if err != nil {
		return err
	}
	if metadata.SealChainIndex != sealChainIndex {
		return fmt.Errorf("seal chain lookup mismatch: index chain %d canonical chain %d", sealChainIndex, metadata.SealChainIndex)
	}
	if segmentEntry, exists := index.SegmentSealLookup[metadata.SegmentID]; exists {
		if segmentEntry.SealDigest != sealDigest || segmentEntry.SealChainIndex != sealChainIndex {
			return fmt.Errorf("seal chain lookup mismatch with segment seal lookup for %q", metadata.SegmentID)
		}
	}
	return nil
}
