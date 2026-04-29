package artifacts

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Store) RecordRunPlanAuthority(authority RunPlanAuthorityRecord, compilation RunPlanCompilationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	authority, compilation, err := prepareRunPlanAuthorityPersistence(authority, compilation)
	if err != nil {
		return err
	}
	if err := s.validateRunPlanAuthorityPersistence(authority, compilation); err != nil {
		return err
	}

	key := runPlanStateKey(authority.RunID, authority.PlanID)
	if err := ensureRunPlanAuthorityDigestConflictFree(key, authority, s.state.RunPlanAuthorities); err != nil {
		return err
	}

	applyRunPlanAuthorityTimestamps(s.nowFn().UTC(), &authority, &compilation)
	s.state.RunPlanAuthorities[key] = authority
	s.state.RunPlanCompilations[key] = compilation
	rebuildRunPlanIndexesLocked(&s.state)
	return s.saveStateLocked()
}

func prepareRunPlanAuthorityPersistence(authority RunPlanAuthorityRecord, compilation RunPlanCompilationRecord) (RunPlanAuthorityRecord, RunPlanCompilationRecord, error) {
	authority = normalizeRunPlanAuthorityRecord(authority)
	compilation = normalizeRunPlanCompilationRecord(compilation)
	bindingDigest, recordDigest, err := computeRunPlanCompilationDigests(compilation)
	if err != nil {
		return RunPlanAuthorityRecord{}, RunPlanCompilationRecord{}, err
	}
	if compilation.BindingDigest == "" {
		compilation.BindingDigest = bindingDigest
	}
	if compilation.RecordDigest == "" {
		compilation.RecordDigest = recordDigest
	}
	return authority, compilation, nil
}

func (s *Store) validateRunPlanAuthorityPersistence(authority RunPlanAuthorityRecord, compilation RunPlanCompilationRecord) error {
	if err := validateRunPlanAuthorityRecord(authority); err != nil {
		return err
	}
	record, ok := s.state.Artifacts[authority.RunPlanDigest]
	if !ok {
		return fmt.Errorf("run plan artifact %q not found", authority.RunPlanDigest)
	}
	if err := validateRunPlanCompilationRecord(compilation); err != nil {
		return err
	}
	if err := validateRunPlanAuthorityCompilationBinding(authority, compilation); err != nil {
		return err
	}
	return validateRunPlanAuthorityArtifactConsistency(authority, compilation, record, s.storeIO)
}

func ensureRunPlanAuthorityDigestConflictFree(key string, authority RunPlanAuthorityRecord, existingByKey map[string]RunPlanAuthorityRecord) error {
	existing, ok := existingByKey[key]
	if !ok {
		return nil
	}
	if strings.TrimSpace(existing.RunPlanDigest) == strings.TrimSpace(authority.RunPlanDigest) {
		return nil
	}
	return fmt.Errorf("run plan authority %q already exists with different run_plan_digest", key)
}

func applyRunPlanAuthorityTimestamps(now time.Time, authority *RunPlanAuthorityRecord, compilation *RunPlanCompilationRecord) {
	if authority.RecordedAt.IsZero() {
		authority.RecordedAt = now
	}
	if compilation.RecordedAt.IsZero() {
		compilation.RecordedAt = now
	}
}

func (s *Store) RunPlanAuthority(runID, planID string) (RunPlanAuthorityRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.state.RunPlanAuthorities[runPlanStateKey(runID, planID)]
	if !ok {
		return RunPlanAuthorityRecord{}, false
	}
	return cloneRunPlanAuthorityRecord(rec), true
}

func (s *Store) ActiveRunPlanAuthority(runID string) (RunPlanAuthorityRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return RunPlanAuthorityRecord{}, false, nil
	}
	planIDs := append([]string{}, s.state.RunPlanRefsByRun[runID]...)
	if len(planIDs) == 0 {
		return RunPlanAuthorityRecord{}, false, nil
	}
	authorities := make([]RunPlanAuthorityRecord, 0, len(planIDs))
	for _, planID := range planIDs {
		rec, ok := s.state.RunPlanAuthorities[runPlanStateKey(runID, planID)]
		if !ok {
			continue
		}
		authorities = append(authorities, rec)
	}
	if len(authorities) == 0 {
		return RunPlanAuthorityRecord{}, false, nil
	}
	selected, ok, err := selectActiveRunPlanAuthorityRecord(authorities)
	if err != nil {
		return RunPlanAuthorityRecord{}, false, err
	}
	if !ok {
		return RunPlanAuthorityRecord{}, false, nil
	}
	return cloneRunPlanAuthorityRecord(selected), true, nil
}

func (s *Store) RunPlanCompilationRecord(runID, planID string) (RunPlanCompilationRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.state.RunPlanCompilations[runPlanStateKey(runID, planID)]
	if !ok {
		return RunPlanCompilationRecord{}, false
	}
	return rec, true
}

func (s *Store) RunPlanCompilationRecordByCacheKey(cacheKey string) (RunPlanCompilationRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stateKey, ok := s.state.RunPlanCompilationByCacheKey[strings.TrimSpace(cacheKey)]
	if !ok {
		return RunPlanCompilationRecord{}, false
	}
	rec, ok := s.state.RunPlanCompilations[stateKey]
	if !ok {
		return RunPlanCompilationRecord{}, false
	}
	return rec, true
}

func runPlanStateKey(runID, planID string) string {
	return strings.TrimSpace(runID) + "|" + strings.TrimSpace(planID)
}

func normalizeRunPlanAuthorityRecord(rec RunPlanAuthorityRecord) RunPlanAuthorityRecord {
	rec.RunID = strings.TrimSpace(rec.RunID)
	rec.PlanID = strings.TrimSpace(rec.PlanID)
	rec.SupersedesPlanID = strings.TrimSpace(rec.SupersedesPlanID)
	rec.RunPlanDigest = strings.TrimSpace(rec.RunPlanDigest)
	rec.WorkflowDefinitionHash = strings.TrimSpace(rec.WorkflowDefinitionHash)
	rec.ProcessDefinitionHash = strings.TrimSpace(rec.ProcessDefinitionHash)
	rec.PolicyContextHash = strings.TrimSpace(rec.PolicyContextHash)
	rec.ProjectContextIdentityDigest = strings.TrimSpace(rec.ProjectContextIdentityDigest)
	rec.Entries = cloneRunPlanGateEntries(rec.Entries)
	return rec
}

func normalizeRunPlanCompilationRecord(rec RunPlanCompilationRecord) RunPlanCompilationRecord {
	rec.RunID = strings.TrimSpace(rec.RunID)
	rec.PlanID = strings.TrimSpace(rec.PlanID)
	rec.RunPlanDigest = strings.TrimSpace(rec.RunPlanDigest)
	rec.SupersedesPlanID = strings.TrimSpace(rec.SupersedesPlanID)
	rec.CompileCacheKey = strings.TrimSpace(rec.CompileCacheKey)
	rec.WorkflowDefinitionRef = strings.TrimSpace(rec.WorkflowDefinitionRef)
	rec.ProcessDefinitionRef = strings.TrimSpace(rec.ProcessDefinitionRef)
	rec.WorkflowDefinitionHash = strings.TrimSpace(rec.WorkflowDefinitionHash)
	rec.ProcessDefinitionHash = strings.TrimSpace(rec.ProcessDefinitionHash)
	rec.PolicyContextHash = strings.TrimSpace(rec.PolicyContextHash)
	rec.ProjectContextIdentityDigest = strings.TrimSpace(rec.ProjectContextIdentityDigest)
	rec.BindingDigest = strings.TrimSpace(rec.BindingDigest)
	rec.RecordDigest = strings.TrimSpace(rec.RecordDigest)
	return rec
}

func rebuildRunPlanRefsByRunLocked(state *StoreState) {
	state.RunPlanRefsByRun = map[string][]string{}
	for _, rec := range state.RunPlanAuthorities {
		runID := strings.TrimSpace(rec.RunID)
		planID := strings.TrimSpace(rec.PlanID)
		if runID == "" || planID == "" {
			continue
		}
		state.RunPlanRefsByRun[runID] = uniqueSortedStrings(append(state.RunPlanRefsByRun[runID], planID))
	}
}

func rebuildRunPlanCompilationCacheKeyIndexLocked(state *StoreState) {
	state.RunPlanCompilationByCacheKey = map[string]string{}
	keys := make([]string, 0, len(state.RunPlanCompilations))
	for key := range state.RunPlanCompilations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rec := state.RunPlanCompilations[key]
		cacheKey := strings.TrimSpace(rec.CompileCacheKey)
		if cacheKey == "" {
			continue
		}
		state.RunPlanCompilationByCacheKey[cacheKey] = key
	}
}

func rebuildRunPlanIndexesLocked(state *StoreState) {
	rebuildRunPlanRefsByRunLocked(state)
	rebuildRunPlanCompilationCacheKeyIndexLocked(state)
}

func reconcileRunPlanIndexesLocked(state *StoreState) bool {
	prior := copyRunPlanRefsByRun(state.RunPlanRefsByRun)
	rebuildRunPlanRefsByRunLocked(state)
	priorCacheIndex := len(state.RunPlanCompilationByCacheKey)
	rebuildRunPlanCompilationCacheKeyIndexLocked(state)
	changed := !sameRunPlanRefsByRun(prior, state.RunPlanRefsByRun)
	for key := range state.RunPlanCompilations {
		if _, ok := state.RunPlanAuthorities[key]; ok {
			continue
		}
		delete(state.RunPlanCompilations, key)
		changed = true
	}
	if priorCacheIndex != len(state.RunPlanCompilationByCacheKey) {
		changed = true
	}
	rebuildRunPlanCompilationCacheKeyIndexLocked(state)
	return changed
}

func copyRunPlanRefsByRun(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for runID, refs := range in {
		out[runID] = append([]string{}, refs...)
	}
	return out
}

func cloneRunPlanAuthorityRecord(rec RunPlanAuthorityRecord) RunPlanAuthorityRecord {
	out := rec
	out.Entries = cloneRunPlanGateEntries(rec.Entries)
	return out
}

func purgeRunPlanAuthoritiesByDigestLocked(state *StoreState, digest string) bool {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return false
	}
	changed := false
	for key, rec := range state.RunPlanAuthorities {
		if strings.TrimSpace(rec.RunPlanDigest) != digest {
			continue
		}
		delete(state.RunPlanAuthorities, key)
		delete(state.RunPlanCompilations, key)
		changed = true
	}
	if changed {
		rebuildRunPlanIndexesLocked(state)
	}
	return changed
}
