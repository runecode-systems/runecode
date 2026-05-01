package artifacts

import (
	"fmt"
	"reflect"
	"strings"
)

func (s *Store) ExternalAnchorPreparedUpsert(record ExternalAnchorPreparedMutationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateExternalAnchorPreparedRecord(record); err != nil {
		return err
	}
	now := s.nowFn().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	s.state.ExternalAnchorPrepared[record.PreparedMutationID] = record
	rebuildRunExternalAnchorPreparedRefsLocked(&s.state)
	return s.saveStateLocked()
}

func (s *Store) ExternalAnchorPreparedGet(preparedMutationID string) (ExternalAnchorPreparedMutationRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.state.ExternalAnchorPrepared[strings.TrimSpace(preparedMutationID)]
	return rec, ok
}

func (s *Store) ExternalAnchorPreparedTransitionLifecycle(preparedMutationID, expectedLifecycle string, mutate func(ExternalAnchorPreparedMutationRecord) ExternalAnchorPreparedMutationRecord) (ExternalAnchorPreparedMutationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	preparedMutationID = strings.TrimSpace(preparedMutationID)
	rec, ok := s.state.ExternalAnchorPrepared[preparedMutationID]
	if !ok {
		return ExternalAnchorPreparedMutationRecord{}, fmt.Errorf("prepared mutation %q not found", preparedMutationID)
	}
	if strings.TrimSpace(rec.LifecycleState) != strings.TrimSpace(expectedLifecycle) {
		return ExternalAnchorPreparedMutationRecord{}, fmt.Errorf("prepared mutation %q lifecycle_state=%q, want %q", preparedMutationID, rec.LifecycleState, expectedLifecycle)
	}

	rec = mutate(rec)
	if err := validateExternalAnchorPreparedRecord(rec); err != nil {
		return ExternalAnchorPreparedMutationRecord{}, err
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = s.nowFn().UTC()
	}
	rec.UpdatedAt = s.nowFn().UTC()
	s.state.ExternalAnchorPrepared[preparedMutationID] = rec
	rebuildRunExternalAnchorPreparedRefsLocked(&s.state)
	if err := s.saveStateLocked(); err != nil {
		return ExternalAnchorPreparedMutationRecord{}, err
	}
	return rec, nil
}

func (s *Store) ExternalAnchorPreparedRefsForRun(runID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	refs := append([]string{}, s.state.RunExternalAnchorPreparedRefs[strings.TrimSpace(runID)]...)
	if len(refs) == 0 {
		return []string{}
	}
	return refs
}

func rebuildRunExternalAnchorPreparedRefsLocked(state *StoreState) {
	state.RunExternalAnchorPreparedRefs = map[string][]string{}
	for _, rec := range state.ExternalAnchorPrepared {
		if strings.TrimSpace(rec.RunID) == "" {
			continue
		}
		state.RunExternalAnchorPreparedRefs[rec.RunID] = uniqueSortedStrings(append(state.RunExternalAnchorPreparedRefs[rec.RunID], rec.PreparedMutationID))
	}
}

func validateExternalAnchorPreparedRecord(record ExternalAnchorPreparedMutationRecord) error {
	if err := validateExternalAnchorPreparedRecordRequiredFields(record); err != nil {
		return err
	}
	if err := validateExternalAnchorPreparedRecordPrimaryDigests(record); err != nil {
		return err
	}
	if err := validateExternalAnchorPreparedRecordOptionalDigests(record); err != nil {
		return err
	}
	return validateExternalAnchorPreparedRecordExecutionState(record)
}

func validateExternalAnchorPreparedRecordRequiredFields(record ExternalAnchorPreparedMutationRecord) error {
	if strings.TrimSpace(record.PreparedMutationID) == "" {
		return fmt.Errorf("prepared mutation id is required")
	}
	if strings.TrimSpace(record.RunID) == "" {
		return fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(record.LifecycleState) == "" {
		return fmt.Errorf("lifecycle state is required")
	}
	if strings.TrimSpace(record.ExecutionState) == "" {
		return fmt.Errorf("execution state is required")
	}
	if record.TypedRequest == nil {
		return fmt.Errorf("typed request is required")
	}
	return nil
}

func validateExternalAnchorPreparedRecordPrimaryDigests(record ExternalAnchorPreparedMutationRecord) error {
	for _, digestField := range []struct {
		value string
		name  string
	}{{value: record.TypedRequestHash, name: "typed request hash"}, {value: record.ActionRequestHash, name: "action request hash"}, {value: record.PolicyDecisionHash, name: "policy decision hash"}} {
		if err := validateGitRemotePreparedDigest(digestField.value, digestField.name); err != nil {
			return err
		}
	}
	return nil
}

func validateExternalAnchorPreparedRecordOptionalDigests(record ExternalAnchorPreparedMutationRecord) error {
	for _, digestField := range []struct {
		value string
		name  string
	}{{value: record.LastExecuteAttemptSealDigest, name: "last execute attempt seal digest"}, {value: record.LastExecuteAttemptTargetID, name: "last execute attempt target descriptor digest"}, {value: record.LastExecuteAttemptRequestID, name: "last execute attempt typed request hash"}, {value: record.LastExecuteSnapshotSealID, name: "last execute snapshot seal digest"}, {value: record.LastAnchorReceiptDigest, name: "last anchor receipt digest"}, {value: record.LastAnchorEvidenceDigest, name: "last anchor evidence digest"}, {value: record.LastAnchorVerificationDigest, name: "last anchor verification digest"}, {value: record.LastAnchorProofDigest, name: "last anchor proof digest"}, {value: record.LastAnchorProviderReceipt, name: "last anchor provider receipt digest"}, {value: record.LastAnchorTranscriptDigest, name: "last anchor transcript digest"}} {
		if strings.TrimSpace(digestField.value) == "" {
			continue
		}
		if err := validateGitRemotePreparedDigest(digestField.value, digestField.name); err != nil {
			return err
		}
	}
	return nil
}

func validateExternalAnchorPreparedRecordExecutionState(record ExternalAnchorPreparedMutationRecord) error {
	if record.LastExecuteDeferredPolls < 0 {
		return fmt.Errorf("last execute deferred polls remaining must be >= 0")
	}
	return nil
}

func reconcileRunExternalAnchorPreparedRefsLocked(state *StoreState) bool {
	prior := copyRunExternalAnchorPreparedRefs(state.RunExternalAnchorPreparedRefs)
	rebuildRunExternalAnchorPreparedRefsLocked(state)
	return !reflect.DeepEqual(prior, state.RunExternalAnchorPreparedRefs)
}

func copyRunExternalAnchorPreparedRefs(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for runID, refs := range in {
		out[runID] = append([]string{}, refs...)
	}
	return out
}
