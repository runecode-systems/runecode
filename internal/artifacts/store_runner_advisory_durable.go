package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (s *Store) appendRunnerJournalRecordLocked(record RunnerDurableJournalRecord) (bool, error) {
	runs, idem, seq, _, err := loadRunnerDurableState(s.rootDir)
	if err != nil {
		return false, err
	}
	key := scopedRunnerIdempotencyKey(record)
	if prevSeq, ok := idem[key]; ok && prevSeq > 0 {
		s.state.RunnerAdvisoryByRun = runs
		return false, nil
	}
	record.Sequence = seq + 1
	if err := appendRunnerJournalRecord(s.rootDir, record); err != nil {
		return false, err
	}
	if err := applyRunnerJournalRecord(runs, record); err != nil {
		return false, err
	}
	idem[key] = record.Sequence
	if err := writeRunnerSnapshot(s.rootDir, RunnerDurableSnapshot{
		Family:        runnerSnapshotFamily,
		SchemaVersion: runnerDurableSchemaVersion,
		LastSequence:  record.Sequence,
		Runs:          runs,
		Idempotency:   idem,
	}); err != nil {
		return false, err
	}
	s.state.RunnerAdvisoryByRun = runs
	ensureRunnerStatusExists(&s.state, record.RunID)
	if err := s.saveStateLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func loadRunnerDurableState(rootDir string) (map[string]RunnerAdvisoryState, map[string]int64, int64, bool, error) {
	runs, idem, seq, err := loadRunnerSnapshot(rootDir)
	if err != nil {
		return nil, nil, 0, false, err
	}
	records, err := readRunnerJournalRecords(rootDir)
	if err != nil {
		return nil, nil, 0, false, err
	}
	updatedSeq, err := replayRunnerJournalRecords(runs, idem, seq, records)
	if err != nil {
		return nil, nil, 0, false, err
	}
	seq = updatedSeq
	seq, needsSnapshot := reconcileRunnerSequenceFromRecords(seq, records)
	return runs, idem, seq, needsSnapshot, nil
}

func replayRunnerJournalRecords(runs map[string]RunnerAdvisoryState, idem map[string]int64, seq int64, records []RunnerDurableJournalRecord) (int64, error) {
	for _, record := range records {
		if shouldSkipRunnerJournalRecord(seq, idem, record) {
			continue
		}
		if err := applyRunnerJournalRecord(runs, record); err != nil {
			return 0, err
		}
		idem[scopedRunnerIdempotencyKey(record)] = record.Sequence
		seq = record.Sequence
	}
	return seq, nil
}

func shouldSkipRunnerJournalRecord(seq int64, idem map[string]int64, record RunnerDurableJournalRecord) bool {
	if record.Sequence <= seq {
		return true
	}
	key := scopedRunnerIdempotencyKey(record)
	prev, ok := idem[key]
	return ok && prev >= record.Sequence
}

func reconcileRunnerSequenceFromRecords(seq int64, records []RunnerDurableJournalRecord) (int64, bool) {
	if len(records) == 0 {
		return seq, false
	}
	last := records[len(records)-1].Sequence
	if last == seq {
		return seq, false
	}
	return last, true
}

func loadRunnerSnapshot(rootDir string) (map[string]RunnerAdvisoryState, map[string]int64, int64, error) {
	path := filepath.Join(rootDir, runnerSnapshotFileName)
	b, ok, err := readRunnerSnapshotBytes(path)
	if err != nil {
		return nil, nil, 0, err
	}
	if !ok {
		return map[string]RunnerAdvisoryState{}, map[string]int64{}, 0, nil
	}
	return decodeRunnerSnapshot(b)
}

func readRunnerSnapshotBytes(path string) ([]byte, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if len(b) == 0 {
		return nil, false, nil
	}
	return b, true, nil
}

func decodeRunnerSnapshot(b []byte) (map[string]RunnerAdvisoryState, map[string]int64, int64, error) {
	var snap RunnerDurableSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, nil, 0, err
	}
	normalized, err := normalizeRunnerSnapshot(snap)
	if err != nil {
		return nil, nil, 0, err
	}
	return normalized.Runs, normalized.Idempotency, normalized.LastSequence, nil
}

func normalizeRunnerSnapshot(snap RunnerDurableSnapshot) (RunnerDurableSnapshot, error) {
	if snap.SchemaVersion == 0 {
		snap.SchemaVersion = 1
	}
	if snap.SchemaVersion != runnerDurableSchemaVersion {
		return RunnerDurableSnapshot{}, fmt.Errorf("unsupported runner snapshot schema version %d", snap.SchemaVersion)
	}
	if snap.Family != "" && snap.Family != runnerSnapshotFamily {
		return RunnerDurableSnapshot{}, fmt.Errorf("unsupported runner snapshot family %q", snap.Family)
	}
	if snap.Runs == nil {
		snap.Runs = map[string]RunnerAdvisoryState{}
	}
	if snap.Idempotency == nil {
		snap.Idempotency = map[string]int64{}
	}
	return snap, nil
}

func readRunnerJournalRecords(rootDir string) ([]RunnerDurableJournalRecord, error) {
	lines, err := loadRunnerJournalLines(rootDir)
	if err != nil {
		return nil, err
	}
	records := make([]RunnerDurableJournalRecord, 0, len(lines))
	for _, line := range lines {
		rec, ok, err := parseRunnerJournalRecord(line)
		if err != nil {
			return nil, err
		}
		if ok {
			records = append(records, rec)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Sequence < records[j].Sequence })
	return records, nil
}

func loadRunnerJournalLines(rootDir string) ([]string, error) {
	path := filepath.Join(rootDir, runnerJournalFileName)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func parseRunnerJournalRecord(line string) (RunnerDurableJournalRecord, bool, error) {
	if strings.TrimSpace(line) == "" {
		return RunnerDurableJournalRecord{}, false, nil
	}
	var rec RunnerDurableJournalRecord
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		return RunnerDurableJournalRecord{}, false, err
	}
	if err := validateRunnerJournalRecordShape(&rec); err != nil {
		return RunnerDurableJournalRecord{}, false, err
	}
	return rec, true, nil
}

func validateRunnerJournalRecordShape(rec *RunnerDurableJournalRecord) error {
	if rec.SchemaVersion == 0 {
		rec.SchemaVersion = 1
	}
	if rec.SchemaVersion != runnerDurableSchemaVersion {
		return fmt.Errorf("unsupported runner journal schema version %d", rec.SchemaVersion)
	}
	if rec.Family != "" && rec.Family != runnerJournalFamily {
		return fmt.Errorf("unsupported runner journal family %q", rec.Family)
	}
	return nil
}
