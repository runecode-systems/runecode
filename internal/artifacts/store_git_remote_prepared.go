package artifacts

import (
	"fmt"
	"reflect"
	"strings"
)

func (s *Store) GitRemotePreparedUpsert(record GitRemotePreparedMutationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateGitRemotePreparedRecord(record); err != nil {
		return err
	}
	now := s.nowFn().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	s.state.GitRemotePrepared[record.PreparedMutationID] = record
	rebuildRunGitRemotePreparedRefsLocked(&s.state)
	return s.saveStateLocked()
}

func (s *Store) GitRemotePreparedGet(preparedMutationID string) (GitRemotePreparedMutationRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.state.GitRemotePrepared[strings.TrimSpace(preparedMutationID)]
	return rec, ok
}

func (s *Store) GitRemotePreparedTransitionLifecycle(preparedMutationID, expectedLifecycle string, mutate func(GitRemotePreparedMutationRecord) GitRemotePreparedMutationRecord) (GitRemotePreparedMutationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	preparedMutationID = strings.TrimSpace(preparedMutationID)
	rec, ok := s.state.GitRemotePrepared[preparedMutationID]
	if !ok {
		return GitRemotePreparedMutationRecord{}, fmt.Errorf("prepared mutation %q not found", preparedMutationID)
	}
	if strings.TrimSpace(rec.LifecycleState) != strings.TrimSpace(expectedLifecycle) {
		return GitRemotePreparedMutationRecord{}, fmt.Errorf("prepared mutation %q lifecycle_state=%q, want %q", preparedMutationID, rec.LifecycleState, expectedLifecycle)
	}

	rec = mutate(rec)
	if err := validateGitRemotePreparedRecord(rec); err != nil {
		return GitRemotePreparedMutationRecord{}, err
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = s.nowFn().UTC()
	}
	rec.UpdatedAt = s.nowFn().UTC()
	s.state.GitRemotePrepared[preparedMutationID] = rec
	rebuildRunGitRemotePreparedRefsLocked(&s.state)
	if err := s.saveStateLocked(); err != nil {
		return GitRemotePreparedMutationRecord{}, err
	}
	return rec, nil
}

func (s *Store) GitRemotePreparedRefsForRun(runID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	refs := append([]string{}, s.state.RunGitRemotePreparedRefs[strings.TrimSpace(runID)]...)
	if len(refs) == 0 {
		return []string{}
	}
	return refs
}

func rebuildRunGitRemotePreparedRefsLocked(state *StoreState) {
	state.RunGitRemotePreparedRefs = map[string][]string{}
	for _, rec := range state.GitRemotePrepared {
		if strings.TrimSpace(rec.RunID) == "" {
			continue
		}
		state.RunGitRemotePreparedRefs[rec.RunID] = uniqueSortedStrings(append(state.RunGitRemotePreparedRefs[rec.RunID], rec.PreparedMutationID))
	}
}

func validateGitRemotePreparedRecord(record GitRemotePreparedMutationRecord) error {
	if strings.TrimSpace(record.PreparedMutationID) == "" {
		return fmt.Errorf("prepared mutation id is required")
	}
	if strings.TrimSpace(record.RunID) == "" {
		return fmt.Errorf("run id is required")
	}
	if err := validateGitRemotePreparedDigest(record.TypedRequestHash, "typed request hash"); err != nil {
		return err
	}
	if err := validateGitRemotePreparedDigest(record.ActionRequestHash, "action request hash"); err != nil {
		return err
	}
	if err := validateGitRemotePreparedDigest(record.PolicyDecisionHash, "policy decision hash"); err != nil {
		return err
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
	if record.DerivedSummary == nil {
		return fmt.Errorf("derived summary is required")
	}
	return nil
}

func validateGitRemotePreparedDigest(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	if !isValidDigest(value) {
		return fmt.Errorf("%s must be sha256 identity", field)
	}
	return nil
}

func reconcileRunGitRemotePreparedRefsLocked(state *StoreState) bool {
	prior := copyRunGitRemotePreparedRefs(state.RunGitRemotePreparedRefs)
	rebuildRunGitRemotePreparedRefsLocked(state)
	return !reflect.DeepEqual(prior, state.RunGitRemotePreparedRefs)
}

func copyRunGitRemotePreparedRefs(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for runID, refs := range in {
		out[runID] = append([]string{}, refs...)
	}
	return out
}
