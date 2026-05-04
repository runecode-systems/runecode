package auditd

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const auditEvidenceIndexSchemaVersion = 1

func (l *Ledger) BuildIndex() (derivedIndex, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rebuildAndPersistDerivedIndexLocked()
}

func (l *Ledger) rebuildAndPersistDerivedIndexLocked() (derivedIndex, error) {
	index, err := l.rebuildDerivedIndexLocked()
	if err != nil {
		return derivedIndex{}, err
	}
	if err := l.saveDerivedIndexLocked(index); err != nil {
		return derivedIndex{}, err
	}
	if state, stateErr := l.loadState(); stateErr == nil {
		state.LastIndexedRecordCount = index.TotalRecords
		_ = l.saveState(state)
	}
	return index, nil
}

func (l *Ledger) rebuildDerivedIndexLocked() (derivedIndex, error) {
	segments, err := l.listSegments()
	if err != nil {
		return derivedIndex{}, err
	}
	index := newDerivedIndex(l.nowFn())
	if err := indexSegments(&index, segments); err != nil {
		return derivedIndex{}, err
	}
	if err := l.attachSealMetadataLocked(&index); err != nil {
		return derivedIndex{}, err
	}
	if err := l.attachLatestVerificationReportDigestLocked(&index); err != nil {
		return derivedIndex{}, err
	}
	index = normalizeDerivedIndex(index)
	return index, nil
}

func segmentTimelinePointers(segment trustpolicy.AuditSegmentFilePayload) ([]TimelinePointer, error) {
	pointers := make([]TimelinePointer, 0, len(segment.Frames))
	for frameIndex, frame := range segment.Frames {
		pointer, ok, err := frameTimelinePointer(segment.Header.SegmentID, frameIndex, frame)
		if err != nil {
			return nil, err
		}
		if ok {
			pointers = append(pointers, pointer)
		}
	}
	return pointers, nil
}

func frameTimelinePointer(segmentID string, frameIndex int, frame trustpolicy.AuditSegmentRecordFrame) (TimelinePointer, bool, error) {
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return TimelinePointer{}, false, err
	}
	if envelope.PayloadSchemaID != trustpolicy.AuditEventSchemaID {
		return TimelinePointer{}, false, nil
	}
	event := trustpolicy.AuditEventPayload{}
	if err := json.Unmarshal(envelope.Payload, &event); err != nil {
		return TimelinePointer{}, false, err
	}
	identity, _ := frame.RecordDigest.Identity()
	pointer := TimelinePointer{SegmentID: segmentID, FrameIndex: frameIndex, RecordDigest: identity, EmitterStreamID: event.EmitterStreamID, Sequence: event.Seq, OccurredAt: event.OccurredAt}
	if event.Scope != nil {
		pointer.RunID = event.Scope["run_id"]
	}
	return pointer, true, nil
}

func (l *Ledger) noteAppendedFrameInDerivedIndexLocked(segmentID string, frameIndex int, frame trustpolicy.AuditSegmentRecordFrame) error {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		return err
	}
	recordDigest, err := frame.RecordDigest.Identity()
	if err != nil {
		return err
	}
	if index.RecordDigestLookup == nil {
		index.RecordDigestLookup = map[string]RecordLookup{}
	}
	index.TotalRecords++
	index.RecordDigestLookup[recordDigest] = RecordLookup{SegmentID: segmentID, FrameIndex: frameIndex}
	if pointer, ok, err := frameTimelinePointer(segmentID, frameIndex, frame); err != nil {
		return err
	} else if ok {
		index.RunTimeline = append(index.RunTimeline, pointer)
	}
	index.LastIndexedSegmentID = segmentID
	index.BuiltAt = l.nowFn().UTC().Format(time.RFC3339)
	return l.saveDerivedIndexLocked(index)
}

func (l *Ledger) noteSealedSegmentInDerivedIndexLocked(sealEnvelopeDigest trustpolicy.Digest, seal trustpolicy.AuditSegmentSealPayload) error {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		return err
	}
	sealDigestIdentity, err := sealEnvelopeDigest.Identity()
	if err != nil {
		return err
	}
	if index.SegmentSealLookup == nil {
		index.SegmentSealLookup = map[string]SegmentSealLookup{}
	}
	if index.SealChainIndexLookup == nil {
		index.SealChainIndexLookup = map[string]string{}
	}
	index.SegmentSealLookup[seal.SegmentID] = SegmentSealLookup{SealDigest: sealDigestIdentity, SealChainIndex: seal.SealChainIndex}
	index.SealChainIndexLookup[strconv.FormatInt(seal.SealChainIndex, 10)] = sealDigestIdentity
	index.BuiltAt = l.nowFn().UTC().Format(time.RFC3339)
	return l.saveDerivedIndexLocked(index)
}

func (l *Ledger) notePersistedVerificationReportInDerivedIndexLocked(reportDigest trustpolicy.Digest) error {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		return err
	}
	reportID, err := reportDigest.Identity()
	if err != nil {
		return err
	}
	index.LatestVerificationReportDigest = reportID
	index.BuiltAt = l.nowFn().UTC().Format(time.RFC3339)
	return l.saveDerivedIndexLocked(index)
}
