package artifacts

import "strings"

func (s *Store) RecordDependencyCacheBatch(batch DependencyCacheBatchRecord, units []DependencyCacheResolvedUnitRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(units) == 0 {
		return ErrDependencyCacheIncompleteState
	}
	staging, err := s.prepareDependencyCacheBatchLocked(units)
	if err != nil {
		return err
	}
	batchCopy, err := buildDependencyCacheBatchRecord(batch, staging.unitDigests)
	if err != nil {
		return err
	}
	priorBatch, hadPriorBatch := s.state.DependencyCacheBatches[batch.BatchRequestDigest]
	s.applyDependencyCacheBatchStagingLocked(batch.BatchRequestDigest, batchCopy, staging)
	if err := s.saveStateLocked(); err != nil {
		s.rollbackDependencyCacheBatchStagingLocked(batch.BatchRequestDigest, priorBatch, hadPriorBatch, staging)
		return err
	}
	return nil
}

func (s *Store) DependencyCacheHit(req DependencyCacheHitRequest) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _, hit, err := s.dependencyCacheLookupLocked(req)
	if err != nil {
		return false, err
	}
	return hit, nil
}

func (s *Store) DependencyCacheLookup(req DependencyCacheHitRequest) (DependencyCacheBatchRecord, DependencyCacheResolvedUnitRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dependencyCacheLookupLocked(req)
}

func (s *Store) DependencyCacheResolvedUnitByRequest(requestDigest string) (DependencyCacheResolvedUnitRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isValidDigest(requestDigest) {
		return DependencyCacheResolvedUnitRecord{}, false, ErrInvalidDigest
	}
	return s.dependencyCacheResolvedUnitByRequestLocked(requestDigest)
}

func (s *Store) DependencyCacheHandoffByRequest(req DependencyCacheHandoffRequest) (DependencyCacheHandoff, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isValidDigest(req.RequestDigest) {
		return DependencyCacheHandoff{}, false, ErrInvalidDigest
	}
	if strings.TrimSpace(req.ConsumerRole) == "" {
		return DependencyCacheHandoff{}, false, ErrFlowDenied
	}
	unit, ok, err := s.dependencyCacheResolvedUnitByRequestLocked(req.RequestDigest)
	if err != nil {
		return DependencyCacheHandoff{}, false, err
	}
	if !ok {
		return DependencyCacheHandoff{}, false, nil
	}
	if err := s.enforceFlowPolicyLocked(FlowCheckRequest{
		ProducerRole: "dependency-fetch",
		ConsumerRole: strings.TrimSpace(req.ConsumerRole),
		DataClass:    DataClassDependencyResolvedUnit,
		Digest:       unit.ManifestDigest,
		IsEgress:     false,
	}); err != nil {
		return DependencyCacheHandoff{}, false, err
	}
	if s.state.Policy.DependencyCachePolicy.MaterializedTreesDerivedNonCanonical && strings.TrimSpace(unit.MaterializationState) != "derived_read_only" {
		return DependencyCacheHandoff{}, false, ErrDependencyCacheIncompleteState
	}
	return DependencyCacheHandoff{
		RequestDigest:       unit.RequestDigest,
		ResolvedUnitDigest:  unit.ResolvedUnitDigest,
		ManifestDigest:      unit.ManifestDigest,
		PayloadDigests:      append([]string{}, unit.PayloadDigest...),
		MaterializationMode: "derived_read_only",
		HandoffMode:         "broker_internal_artifact_handoff",
	}, true, nil
}

func (s *Store) dependencyCacheResolvedUnitByRequestLocked(requestDigest string) (DependencyCacheResolvedUnitRecord, bool, error) {
	linkedUnits := s.state.DependencyCacheByRequest[requestDigest]
	if len(linkedUnits) == 0 {
		return DependencyCacheResolvedUnitRecord{}, false, nil
	}
	if len(linkedUnits) != 1 {
		return DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheAmbiguousReuse
	}
	unit, ok := s.state.DependencyCacheUnits[linkedUnits[0]]
	if !ok {
		return DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheIncompleteState
	}
	if unit.RequestDigest != requestDigest {
		return DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheAmbiguousReuse
	}
	if err := ensureDependencyCacheUnitArtifactsPresent(s.state.Artifacts, unit); err != nil {
		return DependencyCacheResolvedUnitRecord{}, false, err
	}
	return cloneDependencyResolvedUnitRecord(unit), true, nil
}

func (s *Store) dependencyCacheLookupLocked(req DependencyCacheHitRequest) (DependencyCacheBatchRecord, DependencyCacheResolvedUnitRecord, bool, error) {
	if err := validateDependencyCacheHitRequest(req); err != nil {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, err
	}
	batch, ok := s.state.DependencyCacheBatches[req.BatchRequestDigest]
	if !ok {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, nil
	}
	if batch.ResolutionState != "complete" {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheIncompleteState
	}
	if hasMaterializationDigestsWithoutCanonical(batch) {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheIncompleteState
	}
	unit, ok := s.state.DependencyCacheUnits[req.ResolvedUnitDigest]
	if !ok {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, nil
	}
	if unit.RequestDigest != req.RequestDigest {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheAmbiguousReuse
	}
	if !containsString(batch.ResolvedUnitDigests, req.ResolvedUnitDigest) {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheIncompleteState
	}
	linkedUnits := s.state.DependencyCacheByRequest[req.RequestDigest]
	if len(linkedUnits) != 1 || linkedUnits[0] != req.ResolvedUnitDigest {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheAmbiguousReuse
	}
	if err := ensureDependencyCacheUnitArtifactsPresent(s.state.Artifacts, unit); err != nil {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, err
	}
	if s.state.Policy.DependencyCachePolicy.RetainCanonicalBeforeDerived {
		if err := ensureCanonicalDependencyArtifactsPresent(s.state.Artifacts, batch, unit); err != nil {
			return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, err
		}
	}
	return cloneDependencyBatchRecord(batch), cloneDependencyResolvedUnitRecord(unit), true, nil
}
