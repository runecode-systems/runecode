package artifacts

import (
	"encoding/json"
	"os"
)

func (s *Store) recordApprovalWithRunnerMirrorLocked(record ApprovalRecord, mirror RunnerApproval) error {
	priorApproval, hadPriorApproval := s.state.Approvals[record.ApprovalID]
	priorRunApprovalRefs := copyRunApprovalRefs(s.state.RunApprovalRefs)
	priorRunnerAdvisoryByRun := copyRunnerAdvisoryByRun(s.state.RunnerAdvisoryByRun)
	priorRuns := copyRunStatuses(s.state.Runs)

	s.state.Approvals[record.ApprovalID] = record
	rebuildRunApprovalRefsLocked(&s.state)

	journalRecord, err := runnerApprovalJournalRecord(mirror)
	if err != nil {
		s.restoreApprovalWithMirrorRollback(record.ApprovalID, 0, hadPriorApproval, priorApproval, priorRunApprovalRefs, priorRunnerAdvisoryByRun, priorRuns)
		return err
	}
	accepted, err := s.appendRunnerJournalRecordLocked(journalRecord)
	if err != nil {
		s.restoreApprovalWithMirrorRollback(record.ApprovalID, journalRecord.Sequence, hadPriorApproval, priorApproval, priorRunApprovalRefs, priorRunnerAdvisoryByRun, priorRuns)
		return err
	}
	if !accepted {
		if err := s.saveStateLocked(); err != nil {
			s.restoreApprovalWithMirrorRollback(record.ApprovalID, journalRecord.Sequence, hadPriorApproval, priorApproval, priorRunApprovalRefs, priorRunnerAdvisoryByRun, priorRuns)
			return err
		}
	}
	return nil
}

func (s *Store) restoreApprovalWithMirrorRollback(approvalID string, appendedSequence int64, hadPriorApproval bool, priorApproval ApprovalRecord, priorRunApprovalRefs map[string][]string, priorRunnerAdvisoryByRun map[string]RunnerAdvisoryState, priorRuns map[string]string) {
	if appendedSequence > 0 {
		_ = truncateRunnerJournalSequence(s.rootDir, appendedSequence)
	}
	if hadPriorApproval {
		s.state.Approvals[approvalID] = priorApproval
	} else {
		delete(s.state.Approvals, approvalID)
	}
	s.state.RunApprovalRefs = priorRunApprovalRefs
	s.state.RunnerAdvisoryByRun = priorRunnerAdvisoryByRun
	s.state.Runs = priorRuns
}

func copyRunApprovalRefs(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for runID, refs := range in {
		out[runID] = append([]string(nil), refs...)
	}
	return out
}

func truncateRunnerJournalSequence(rootDir string, sequence int64) error {
	if sequence <= 0 {
		return nil
	}
	records, err := readRunnerJournalRecords(rootDir)
	if err != nil {
		return err
	}
	filtered := make([]RunnerDurableJournalRecord, 0, len(records))
	for _, rec := range records {
		if rec.Sequence == sequence {
			continue
		}
		filtered = append(filtered, rec)
	}
	return writeRunnerJournalRecords(rootDir, filtered)
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

func copyRunStatuses(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for runID, status := range in {
		out[runID] = status
	}
	return out
}
