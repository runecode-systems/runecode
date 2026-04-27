package artifacts

func validateRestoredDependencyCache(next *StoreState) error {
	if err := validateRestoredDependencyCacheUnitArtifacts(next); err != nil {
		return err
	}
	if err := validateRestoredDependencyCacheBatchReferences(next); err != nil {
		return err
	}
	rebuiltByRequest := rebuildDependencyCacheByRequest(next.DependencyCacheUnits)
	return validateRestoredDependencyCacheRequestIndex(next.DependencyCacheByRequest, rebuiltByRequest)
}

func validateRestoredDependencyCacheUnitArtifacts(next *StoreState) error {
	for _, unit := range next.DependencyCacheUnits {
		if err := ensureDependencyCacheUnitArtifactsPresent(next.Artifacts, unit); err != nil {
			return err
		}
	}
	return nil
}

func validateRestoredDependencyCacheBatchReferences(next *StoreState) error {
	for _, batch := range next.DependencyCacheBatches {
		for _, resolvedUnitDigest := range batch.ResolvedUnitDigests {
			if err := validateRestoredDependencyCacheBatchUnit(next, batch, resolvedUnitDigest); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRestoredDependencyCacheBatchUnit(next *StoreState, batch DependencyCacheBatchRecord, resolvedUnitDigest string) error {
	unit, ok := next.DependencyCacheUnits[resolvedUnitDigest]
	if !ok {
		return ErrDependencyCacheIncompleteState
	}
	if !next.Policy.DependencyCachePolicy.RetainCanonicalBeforeDerived {
		return nil
	}
	return ensureCanonicalDependencyArtifactsPresent(next.Artifacts, batch, unit)
}

func rebuildDependencyCacheByRequest(units map[string]DependencyCacheResolvedUnitRecord) map[string][]string {
	rebuiltByRequest := map[string][]string{}
	for resolvedUnitDigest, unit := range units {
		rebuiltByRequest[unit.RequestDigest] = uniqueSortedStrings(append(rebuiltByRequest[unit.RequestDigest], resolvedUnitDigest))
	}
	return rebuiltByRequest
}

func validateRestoredDependencyCacheRequestIndex(indexedByRequest, rebuiltByRequest map[string][]string) error {
	if len(rebuiltByRequest) != len(indexedByRequest) {
		return ErrDependencyCacheIncompleteState
	}
	for requestDigest, indexedDigests := range indexedByRequest {
		if err := validateRestoredDependencyCacheRequestIndexEntry(requestDigest, indexedDigests, rebuiltByRequest); err != nil {
			return err
		}
	}
	return nil
}

func validateRestoredDependencyCacheRequestIndexEntry(requestDigest string, indexedDigests []string, rebuiltByRequest map[string][]string) error {
	rebuiltDigests, ok := rebuiltByRequest[requestDigest]
	if !ok {
		return ErrDependencyCacheIncompleteState
	}
	normalizedIndexedDigests := uniqueSortedStrings(append([]string{}, indexedDigests...))
	if len(normalizedIndexedDigests) != len(indexedDigests) {
		return ErrDependencyCacheIncompleteState
	}
	if !equalOrderedStrings(normalizedIndexedDigests, rebuiltDigests) {
		return ErrDependencyCacheIncompleteState
	}
	return nil
}

func equalOrderedStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
