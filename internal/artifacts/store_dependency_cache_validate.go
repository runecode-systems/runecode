package artifacts

import (
	"fmt"
	"strings"
)

func ensureDependencyCacheUnitArtifactsPresent(artifacts map[string]ArtifactRecord, unit DependencyCacheResolvedUnitRecord) error {
	if unit.IntegrityState != "verified" {
		return ErrDependencyCacheUnverifiableIdentity
	}
	manifestRecord, ok := artifacts[unit.ManifestDigest]
	if !ok || manifestRecord.Reference.DataClass != DataClassDependencyResolvedUnit {
		return ErrDependencyCacheUnverifiableIdentity
	}
	for _, payloadDigest := range unit.PayloadDigest {
		payloadRecord, ok := artifacts[payloadDigest]
		if !ok {
			return ErrDependencyCacheIncompleteState
		}
		if payloadRecord.Reference.DataClass != DataClassDependencyPayloadUnit {
			return ErrDependencyCacheUnverifiableIdentity
		}
	}
	return nil
}

func hasMaterializationDigestsWithoutCanonical(batch DependencyCacheBatchRecord) bool {
	return len(batch.MaterializationDigest) > 0 && len(batch.ResolvedUnitDigests) == 0
}

func ensureCanonicalDependencyArtifactsPresent(artifacts map[string]ArtifactRecord, batch DependencyCacheBatchRecord, unit DependencyCacheResolvedUnitRecord) error {
	batchRec, ok := artifacts[batch.BatchManifestDigest]
	if !ok {
		return ErrDependencyCacheIncompleteState
	}
	if batchRec.Reference.DataClass != DataClassDependencyBatchManifest {
		return ErrDependencyCacheUnverifiableIdentity
	}
	unitRec, ok := artifacts[unit.ManifestDigest]
	if !ok {
		return ErrDependencyCacheIncompleteState
	}
	if unitRec.Reference.DataClass != DataClassDependencyResolvedUnit {
		return ErrDependencyCacheUnverifiableIdentity
	}
	return nil
}

func validateDependencyCacheHitRequest(req DependencyCacheHitRequest) error {
	if !isValidDigest(req.BatchRequestDigest) {
		return ErrInvalidDigest
	}
	if !isValidDigest(req.ResolvedUnitDigest) {
		return ErrInvalidDigest
	}
	if !isValidDigest(req.RequestDigest) {
		return ErrInvalidDigest
	}
	return nil
}

func validateDependencyCacheBatchRecord(batch DependencyCacheBatchRecord) error {
	if err := validateDependencyCacheBatchCoreDigests(batch); err != nil {
		return err
	}
	if batch.ResolutionState != "complete" {
		return ErrDependencyCacheIncompleteState
	}
	if batch.CacheOutcome != "hit_exact" && batch.CacheOutcome != "miss_filled" {
		return fmt.Errorf("invalid cache outcome")
	}
	if err := validateDigestList(batch.ResolvedUnitDigests); err != nil {
		return err
	}
	if err := validateDigestList(batch.MaterializationDigest); err != nil {
		return err
	}
	if len(batch.ResolvedUnitDigests) == 0 {
		return ErrDependencyCacheIncompleteState
	}
	return nil
}

func validateDependencyCacheBatchCoreDigests(batch DependencyCacheBatchRecord) error {
	if !isValidDigest(batch.BatchRequestDigest) || !isValidDigest(batch.BatchManifestDigest) || !isValidDigest(batch.LockfileDigest) || !isValidDigest(batch.RequestSetDigest) {
		return ErrInvalidDigest
	}
	return nil
}

func validateDependencyCacheResolvedUnitRecord(unit DependencyCacheResolvedUnitRecord) error {
	if !isValidDigest(unit.ResolvedUnitDigest) || !isValidDigest(unit.RequestDigest) || !isValidDigest(unit.ManifestDigest) {
		return ErrInvalidDigest
	}
	if strings.TrimSpace(unit.IntegrityState) != "verified" {
		return ErrDependencyCacheUnverifiableIdentity
	}
	if strings.TrimSpace(unit.MaterializationState) != "derived_read_only" {
		return ErrDependencyCacheIncompleteState
	}
	if len(unit.PayloadDigest) == 0 {
		return ErrDependencyCacheIncompleteState
	}
	return validateDigestList(unit.PayloadDigest)
}

func validateDigestList(digests []string) error {
	for _, digest := range digests {
		if !isValidDigest(digest) {
			return ErrInvalidDigest
		}
	}
	return nil
}

func cloneDependencyBatchRecord(batch DependencyCacheBatchRecord) DependencyCacheBatchRecord {
	batch.ResolvedUnitDigests = append([]string{}, batch.ResolvedUnitDigests...)
	batch.MaterializationDigest = append([]string{}, batch.MaterializationDigest...)
	return batch
}

func cloneDependencyResolvedUnitRecord(unit DependencyCacheResolvedUnitRecord) DependencyCacheResolvedUnitRecord {
	unit.PayloadDigest = append([]string{}, unit.PayloadDigest...)
	return unit
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
