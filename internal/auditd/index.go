package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) BuildIndex() (derivedIndex, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	segments, err := l.listSegments()
	if err != nil {
		return derivedIndex{}, err
	}
	index, err := buildDerivedIndex(segments, l.nowFn())
	if err != nil {
		return derivedIndex{}, err
	}
	if err := writeCanonicalJSONFile(filepath.Join(l.rootDir, indexDirName, indexFileName), index); err != nil {
		return derivedIndex{}, err
	}
	state, err := l.recoverAndPersistStateLocked()
	if err == nil {
		state.LastIndexedRecordCount = index.TotalRecords
		_ = l.saveState(state)
	}
	return index, nil
}

func buildDerivedIndex(segments []trustpolicy.AuditSegmentFilePayload, now time.Time) (derivedIndex, error) {
	index := derivedIndex{BuiltAt: now.UTC().Format(time.RFC3339)}
	for _, segment := range segments {
		pointers, err := segmentTimelinePointers(segment)
		if err != nil {
			return derivedIndex{}, err
		}
		index.RunTimeline = append(index.RunTimeline, pointers...)
		index.TotalRecords += len(pointers)
	}
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

func (l *Ledger) indexStatusLocked() (indexed int, total int, err error) {
	state, readErr := l.loadState()
	if readErr == nil {
		indexed = state.LastIndexedRecordCount
	}
	segments, err := l.listSegments()
	if err != nil {
		return 0, 0, err
	}
	for _, segment := range segments {
		total += len(segment.Frames)
	}
	if indexed > total {
		indexed = 0
	}
	return indexed, total, nil
}

func hasVerificationInputs(l *Ledger) bool {
	if err := validateVerificationInputs(l); err != nil {
		return false
	}
	return true
}

func validateVerificationInputs(l *Ledger) error {
	contractsDir := filepath.Join(l.rootDir, "contracts")
	eventCatalogPath := filepath.Join(contractsDir, "event-contract-catalog.json")
	verifierRecordsPath := filepath.Join(contractsDir, "verifier-records.json")
	if !fileExists(eventCatalogPath) {
		return fmt.Errorf("missing event contract catalog")
	}
	if !fileExists(verifierRecordsPath) {
		return fmt.Errorf("missing verifier records")
	}
	catalog := trustpolicy.AuditEventContractCatalog{}
	if err := readJSONFile(eventCatalogPath, &catalog); err != nil {
		return err
	}
	if err := trustpolicy.ValidateAuditEventContractCatalogForRuntime(catalog); err != nil {
		return err
	}
	verifierRecords := []trustpolicy.VerifierRecord{}
	if err := readJSONFile(verifierRecordsPath, &verifierRecords); err != nil {
		return err
	}
	if _, err := trustpolicy.NewVerifierRegistry(verifierRecords); err != nil {
		return err
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
