package artifacts

import "sort"

type dependencyCacheBatchStaging struct {
	stagedUnits            map[string]DependencyCacheResolvedUnitRecord
	stagedByRequest        map[string][]string
	priorUnits             map[string]DependencyCacheResolvedUnitRecord
	priorUnitPresence      map[string]bool
	priorByRequest         map[string][]string
	priorByRequestPresence map[string]bool
	unitDigests            []string
}

func newDependencyCacheBatchStaging(size int) dependencyCacheBatchStaging {
	return dependencyCacheBatchStaging{
		stagedUnits:            make(map[string]DependencyCacheResolvedUnitRecord, size),
		stagedByRequest:        make(map[string][]string, size),
		priorUnits:             make(map[string]DependencyCacheResolvedUnitRecord, size),
		priorUnitPresence:      make(map[string]bool, size),
		priorByRequest:         make(map[string][]string, size),
		priorByRequestPresence: make(map[string]bool, size),
		unitDigests:            make([]string, 0, size),
	}
}

func (s *Store) prepareDependencyCacheBatchLocked(units []DependencyCacheResolvedUnitRecord) (dependencyCacheBatchStaging, error) {
	staging := newDependencyCacheBatchStaging(len(units))
	for _, unit := range units {
		if err := s.stageDependencyCacheUnitLocked(&staging, unit); err != nil {
			return dependencyCacheBatchStaging{}, err
		}
	}
	sort.Strings(staging.unitDigests)
	return staging, nil
}

func (s *Store) stageDependencyCacheUnitLocked(staging *dependencyCacheBatchStaging, unit DependencyCacheResolvedUnitRecord) error {
	if err := validateDependencyCacheResolvedUnitRecord(unit); err != nil {
		return err
	}
	if err := ensureDependencyCacheBatchUnitArtifactsPresent(s.state.Artifacts, unit); err != nil {
		return err
	}
	s.captureDependencyCacheBatchPriorStateLocked(staging, unit)
	staging.stagedUnits[unit.ResolvedUnitDigest] = cloneDependencyResolvedUnitRecord(unit)
	staging.stagedByRequest[unit.RequestDigest] = stagedDependencyCacheRequestDigests(
		s.state.DependencyCacheByRequest[unit.RequestDigest],
		staging.stagedByRequest[unit.RequestDigest],
		unit.ResolvedUnitDigest,
	)
	staging.unitDigests = append(staging.unitDigests, unit.ResolvedUnitDigest)
	return nil
}

func ensureDependencyCacheBatchUnitArtifactsPresent(records map[string]ArtifactRecord, unit DependencyCacheResolvedUnitRecord) error {
	if _, ok := records[unit.ManifestDigest]; !ok {
		return ErrDependencyCacheUnverifiableIdentity
	}
	for _, payloadDigest := range unit.PayloadDigest {
		rec, ok := records[payloadDigest]
		if !ok {
			return ErrDependencyCacheIncompleteState
		}
		if rec.Reference.DataClass != DataClassDependencyPayloadUnit {
			return ErrDependencyCacheUnverifiableIdentity
		}
	}
	return nil
}

func (s *Store) captureDependencyCacheBatchPriorStateLocked(staging *dependencyCacheBatchStaging, unit DependencyCacheResolvedUnitRecord) {
	if _, seen := staging.priorUnits[unit.ResolvedUnitDigest]; !seen {
		prior, ok := s.state.DependencyCacheUnits[unit.ResolvedUnitDigest]
		staging.priorUnits[unit.ResolvedUnitDigest] = cloneDependencyResolvedUnitRecord(prior)
		staging.priorUnitPresence[unit.ResolvedUnitDigest] = ok
	}
	if _, seen := staging.priorByRequest[unit.RequestDigest]; seen {
		return
	}
	prior := append([]string{}, s.state.DependencyCacheByRequest[unit.RequestDigest]...)
	_, ok := s.state.DependencyCacheByRequest[unit.RequestDigest]
	staging.priorByRequest[unit.RequestDigest] = prior
	staging.priorByRequestPresence[unit.RequestDigest] = ok
}

func stagedDependencyCacheRequestDigests(existing, staged []string, resolvedUnitDigest string) []string {
	combined := append([]string{}, existing...)
	combined = append(combined, staged...)
	combined = append(combined, resolvedUnitDigest)
	return uniqueSortedStrings(combined)
}

func buildDependencyCacheBatchRecord(batch DependencyCacheBatchRecord, unitDigests []string) (DependencyCacheBatchRecord, error) {
	batchCopy := cloneDependencyBatchRecord(batch)
	batchCopy.ResolvedUnitDigests = append([]string{}, unitDigests...)
	if err := validateDependencyCacheBatchRecord(batchCopy); err != nil {
		return DependencyCacheBatchRecord{}, err
	}
	return batchCopy, nil
}

func (s *Store) applyDependencyCacheBatchStagingLocked(batchRequestDigest string, batch DependencyCacheBatchRecord, staging dependencyCacheBatchStaging) {
	for digest, unit := range staging.stagedUnits {
		s.state.DependencyCacheUnits[digest] = unit
	}
	for requestDigest, resolvedDigests := range staging.stagedByRequest {
		s.state.DependencyCacheByRequest[requestDigest] = append([]string{}, resolvedDigests...)
	}
	s.state.DependencyCacheBatches[batchRequestDigest] = batch
}

func (s *Store) rollbackDependencyCacheBatchStagingLocked(batchRequestDigest string, priorBatch DependencyCacheBatchRecord, hadPriorBatch bool, staging dependencyCacheBatchStaging) {
	for digest := range staging.stagedUnits {
		if staging.priorUnitPresence[digest] {
			s.state.DependencyCacheUnits[digest] = staging.priorUnits[digest]
			continue
		}
		delete(s.state.DependencyCacheUnits, digest)
	}
	for requestDigest := range staging.stagedByRequest {
		if staging.priorByRequestPresence[requestDigest] {
			s.state.DependencyCacheByRequest[requestDigest] = append([]string{}, staging.priorByRequest[requestDigest]...)
			continue
		}
		delete(s.state.DependencyCacheByRequest, requestDigest)
	}
	if hadPriorBatch {
		s.state.DependencyCacheBatches[batchRequestDigest] = priorBatch
		return
	}
	delete(s.state.DependencyCacheBatches, batchRequestDigest)
}
