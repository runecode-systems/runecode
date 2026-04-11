package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func appendRunnerJournalRecord(rootDir string, record RunnerDurableJournalRecord) error {
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(rootDir, runnerJournalFileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func writeRunnerSnapshot(rootDir string, snapshot RunnerDurableSnapshot) error {
	b, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(rootDir, runnerSnapshotFileName), b, 0o600)
}

func ensureRunnerDurableFiles(rootDir string, runs map[string]RunnerAdvisoryState, idem map[string]int64, seq int64) error {
	if err := writeRunnerSnapshot(rootDir, RunnerDurableSnapshot{
		Family:        runnerSnapshotFamily,
		SchemaVersion: runnerDurableSchemaVersion,
		LastSequence:  seq,
		Runs:          runs,
		Idempotency:   idem,
	}); err != nil {
		return err
	}
	path := filepath.Join(rootDir, runnerJournalFileName)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte{}, 0o600)
}

func scopedRunnerIdempotencyKey(record RunnerDurableJournalRecord) string {
	runID := strings.TrimSpace(record.RunID)
	key := strings.TrimSpace(record.IdempotencyKey)
	if runID == "" {
		return key
	}
	if key == "" {
		return runID + "@"
	}
	return runID + "@" + key
}
