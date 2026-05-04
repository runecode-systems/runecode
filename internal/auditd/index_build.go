package auditd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func newDerivedIndex(now time.Time) derivedIndex {
	return derivedIndex{
		SchemaVersion:        auditEvidenceIndexSchemaVersion,
		BuiltAt:              now.UTC().Format(time.RFC3339),
		RecordDigestLookup:   map[string]RecordLookup{},
		SegmentSealLookup:    map[string]SegmentSealLookup{},
		SealChainIndexLookup: map[string]string{},
	}
}

func indexSegments(index *derivedIndex, segments []trustpolicy.AuditSegmentFilePayload) error {
	for _, segment := range segments {
		if err := indexSegmentFrames(index, segment); err != nil {
			return err
		}
		if err := indexSegmentTimeline(index, segment); err != nil {
			return err
		}
		index.TotalRecords += len(segment.Frames)
		if len(segment.Frames) > 0 {
			index.LastIndexedSegmentID = segment.Header.SegmentID
		}
	}
	return nil
}

func indexSegmentFrames(index *derivedIndex, segment trustpolicy.AuditSegmentFilePayload) error {
	for frameIndex, frame := range segment.Frames {
		recordDigest, err := frame.RecordDigest.Identity()
		if err != nil {
			return err
		}
		index.RecordDigestLookup[recordDigest] = RecordLookup{SegmentID: segment.Header.SegmentID, FrameIndex: frameIndex}
	}
	return nil
}

func indexSegmentTimeline(index *derivedIndex, segment trustpolicy.AuditSegmentFilePayload) error {
	pointers, err := segmentTimelinePointers(segment)
	if err != nil {
		return err
	}
	index.RunTimeline = append(index.RunTimeline, pointers...)
	return nil
}

func (l *Ledger) attachSealMetadataLocked(index *derivedIndex) error {
	seals, err := l.listSealMetadataLocked()
	if err != nil {
		return err
	}
	for _, seal := range seals {
		if err := addSealMetadata(index, seal); err != nil {
			return err
		}
	}
	return nil
}

func addSealMetadata(index *derivedIndex, seal sealMetadata) error {
	if err := validateConflictingSegmentSeal(index, seal); err != nil {
		return err
	}
	chainKey := strconv.FormatInt(seal.SealChainIndex, 10)
	if existingDigest, exists := index.SealChainIndexLookup[chainKey]; exists && existingDigest != seal.SealDigestIdentity {
		return fmt.Errorf("canonical seal metadata conflict: multiple seals share chain index %d", seal.SealChainIndex)
	}
	index.SegmentSealLookup[seal.SegmentID] = SegmentSealLookup{SealDigest: seal.SealDigestIdentity, SealChainIndex: seal.SealChainIndex}
	index.SealChainIndexLookup[chainKey] = seal.SealDigestIdentity
	return nil
}

func validateConflictingSegmentSeal(index *derivedIndex, seal sealMetadata) error {
	if existing, exists := index.SegmentSealLookup[seal.SegmentID]; exists {
		if existing.SealDigest != seal.SealDigestIdentity || existing.SealChainIndex != seal.SealChainIndex {
			return fmt.Errorf("canonical seal metadata conflict for segment %q", seal.SegmentID)
		}
	}
	return nil
}

func (l *Ledger) attachLatestVerificationReportDigestLocked(index *derivedIndex) error {
	latestReportDigest, err := l.discoverLatestVerificationReportDigestLockedWithError()
	if err != nil {
		return err
	}
	index.LatestVerificationReportDigest = latestReportDigest
	return nil
}
