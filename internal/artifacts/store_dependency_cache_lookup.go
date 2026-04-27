package artifacts

func (s *Store) dependencyCacheLookupLocked(req DependencyCacheHitRequest) (DependencyCacheBatchRecord, DependencyCacheResolvedUnitRecord, bool, error) {
	if err := validateDependencyCacheHitRequest(req); err != nil {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, err
	}
	batch, ok, err := s.lookupDependencyCacheBatchLocked(req.BatchRequestDigest)
	if err != nil || !ok {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, err
	}
	unit, ok, err := s.lookupDependencyCacheUnitLocked(req, batch)
	if err != nil || !ok {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, err
	}
	if err := s.validateDependencyCacheLookupArtifactsLocked(batch, unit); err != nil {
		return DependencyCacheBatchRecord{}, DependencyCacheResolvedUnitRecord{}, false, err
	}
	return cloneDependencyBatchRecord(batch), cloneDependencyResolvedUnitRecord(unit), true, nil
}

func (s *Store) lookupDependencyCacheBatchLocked(batchRequestDigest string) (DependencyCacheBatchRecord, bool, error) {
	batch, ok := s.state.DependencyCacheBatches[batchRequestDigest]
	if !ok {
		return DependencyCacheBatchRecord{}, false, nil
	}
	if batch.ResolutionState != "complete" || hasMaterializationDigestsWithoutCanonical(batch) {
		return DependencyCacheBatchRecord{}, false, ErrDependencyCacheIncompleteState
	}
	return batch, true, nil
}

func (s *Store) lookupDependencyCacheUnitLocked(req DependencyCacheHitRequest, batch DependencyCacheBatchRecord) (DependencyCacheResolvedUnitRecord, bool, error) {
	unit, ok := s.state.DependencyCacheUnits[req.ResolvedUnitDigest]
	if !ok {
		return DependencyCacheResolvedUnitRecord{}, false, nil
	}
	if unit.RequestDigest != req.RequestDigest {
		return DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheAmbiguousReuse
	}
	if !containsString(batch.ResolvedUnitDigests, req.ResolvedUnitDigest) {
		return DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheIncompleteState
	}
	linkedUnits := s.state.DependencyCacheByRequest[req.RequestDigest]
	if len(linkedUnits) != 1 || linkedUnits[0] != req.ResolvedUnitDigest {
		return DependencyCacheResolvedUnitRecord{}, false, ErrDependencyCacheAmbiguousReuse
	}
	return unit, true, nil
}

func (s *Store) validateDependencyCacheLookupArtifactsLocked(batch DependencyCacheBatchRecord, unit DependencyCacheResolvedUnitRecord) error {
	if err := ensureDependencyCacheUnitArtifactsPresent(s.state.Artifacts, unit); err != nil {
		return err
	}
	if !s.state.Policy.DependencyCachePolicy.RetainCanonicalBeforeDerived {
		return nil
	}
	return ensureCanonicalDependencyArtifactsPresent(s.state.Artifacts, batch, unit)
}
