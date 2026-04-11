package artifacts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func (s *Store) recordApprovalWithRunnerMirrorLocked(record ApprovalRecord, mirror RunnerApproval) error {
	priorApproval, hadPriorApproval := s.state.Approvals[record.ApprovalID]
	priorRunApprovalRefs := copyRunApprovalRefs(s.state.RunApprovalRefs)
	priorRunnerAdvisoryByRun := copyRunnerAdvisoryByRun(s.state.RunnerAdvisoryByRun)
	priorRuns := copyRunStatuses(s.state.Runs)
	priorDurable, err := captureRunnerDurableFiles(s.rootDir)
	if err != nil {
		return err
	}

	s.state.Approvals[record.ApprovalID] = record
	rebuildRunApprovalRefsLocked(&s.state)

	journalRecord, err := runnerApprovalJournalRecord(mirror)
	if err != nil {
		return errors.Join(err, s.restoreApprovalWithMirrorRollback(record.ApprovalID, hadPriorApproval, priorApproval, priorRunApprovalRefs, priorRunnerAdvisoryByRun, priorRuns, priorDurable))
	}
	accepted, err := s.appendRunnerJournalRecordLocked(journalRecord)
	if err != nil {
		return errors.Join(err, s.restoreApprovalWithMirrorRollback(record.ApprovalID, hadPriorApproval, priorApproval, priorRunApprovalRefs, priorRunnerAdvisoryByRun, priorRuns, priorDurable))
	}
	if !accepted {
		if err := s.saveStateLocked(); err != nil {
			return errors.Join(err, s.restoreApprovalWithMirrorRollback(record.ApprovalID, hadPriorApproval, priorApproval, priorRunApprovalRefs, priorRunnerAdvisoryByRun, priorRuns, priorDurable))
		}
	}
	return nil
}

func (s *Store) restoreApprovalWithMirrorRollback(approvalID string, hadPriorApproval bool, priorApproval ApprovalRecord, priorRunApprovalRefs map[string][]string, priorRunnerAdvisoryByRun map[string]RunnerAdvisoryState, priorRuns map[string]string, priorDurable runnerDurableFiles) error {
	var rollbackErr error
	if err := restoreRunnerDurableFiles(s.rootDir, priorDurable); err != nil {
		rollbackErr = errors.Join(rollbackErr, err)
	} else {
		runs, idem, seq, _, loadErr := loadRunnerDurableState(s.rootDir)
		if loadErr != nil {
			rollbackErr = errors.Join(rollbackErr, loadErr)
		} else if err := ensureRunnerDurableFiles(s.rootDir, runs, idem, seq); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	if hadPriorApproval {
		s.state.Approvals[approvalID] = priorApproval
	} else {
		delete(s.state.Approvals, approvalID)
	}
	s.state.RunApprovalRefs = priorRunApprovalRefs
	s.state.RunnerAdvisoryByRun = priorRunnerAdvisoryByRun
	s.state.Runs = priorRuns
	return rollbackErr
}

func copyRunApprovalRefs(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for runID, refs := range in {
		out[runID] = append([]string(nil), refs...)
	}
	return out
}

func writeRunnerJournalRecords(rootDir string, records []RunnerDurableJournalRecord) error {
	path := rootDir + string(os.PathSeparator) + runnerJournalFileName
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, rec := range records {
		line, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return nil
}

type runnerDurableFiles struct {
	journalBytes   []byte
	journalExists  bool
	snapshotBytes  []byte
	snapshotExists bool
}

func captureRunnerDurableFiles(rootDir string) (runnerDurableFiles, error) {
	journalBytes, journalExists, err := readOptionalFile(filepath.Join(rootDir, runnerJournalFileName))
	if err != nil {
		return runnerDurableFiles{}, err
	}
	snapshotBytes, snapshotExists, err := readOptionalFile(filepath.Join(rootDir, runnerSnapshotFileName))
	if err != nil {
		return runnerDurableFiles{}, err
	}
	return runnerDurableFiles{journalBytes: journalBytes, journalExists: journalExists, snapshotBytes: snapshotBytes, snapshotExists: snapshotExists}, nil
}

func restoreRunnerDurableFiles(rootDir string, files runnerDurableFiles) error {
	if err := restoreOptionalFile(filepath.Join(rootDir, runnerJournalFileName), files.journalBytes, files.journalExists); err != nil {
		return err
	}
	return restoreOptionalFile(filepath.Join(rootDir, runnerSnapshotFileName), files.snapshotBytes, files.snapshotExists)
}

func readOptionalFile(path string) ([]byte, bool, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil, false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return append([]byte(nil), b...), true, nil
}

func restoreOptionalFile(path string, contents []byte, exists bool) error {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !exists {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return os.WriteFile(path, contents, 0o600)
}

func copyRunStatuses(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for runID, status := range in {
		out[runID] = status
	}
	return out
}
