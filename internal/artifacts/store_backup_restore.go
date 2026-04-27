package artifacts

import (
	"fmt"
	"strings"
	"time"
)

func stateFromBackup(manifest BackupManifest, lastAuditSequence int64, ioStore *storeIO) (StoreState, error) {
	if err := validatePolicy(manifest.Policy); err != nil {
		return StoreState{}, err
	}
	next := newStateFromBackup(manifest, lastAuditSequence)
	if err := loadRestoredStateRecords(&next, manifest, ioStore); err != nil {
		return StoreState{}, err
	}
	if err := validateRestoredStateLinks(&next); err != nil {
		return StoreState{}, err
	}
	return next, nil
}

func loadRestoredStateRecords(next *StoreState, manifest BackupManifest, ioStore *storeIO) error {
	unapprovedByDigest, err := loadRestoredArtifacts(next, manifest.Artifacts, ioStore)
	if err != nil {
		return err
	}
	if err := validateApprovedRestores(next.Artifacts, unapprovedByDigest); err != nil {
		return err
	}
	loaders := []func() error{
		func() error { return loadRestoredApprovals(next, manifest.Approvals) },
		func() error { return loadRestoredSessions(next, manifest.Sessions) },
		func() error { return loadRestoredPolicyDecisions(next, manifest.PolicyDecisions) },
		func() error {
			return loadRestoredDependencyCache(next, manifest.DependencyCacheBatches, manifest.DependencyCacheUnits)
		},
		func() error { return validateRestoredDependencyCache(next) },
		func() error { return loadRestoredProviderProfiles(next, manifest.ProviderProfiles) },
		func() error { return loadRestoredProviderSetupSessions(next, manifest.ProviderSetupSessions) },
	}
	for _, loader := range loaders {
		if err := loader(); err != nil {
			return err
		}
	}
	return nil
}

func validateRestoredStateLinks(next *StoreState) error {
	return validateRestoredApprovalPolicyDecisionLinks(next.Approvals, next.PolicyDecisions)
}

func newStateFromBackup(manifest BackupManifest, lastAuditSequence int64) StoreState {
	runs := make(map[string]string, len(manifest.Runs))
	for runID, status := range manifest.Runs {
		runs[runID] = status
	}
	return StoreState{
		Artifacts:                map[string]ArtifactRecord{},
		DependencyCacheBatches:   map[string]DependencyCacheBatchRecord{},
		DependencyCacheUnits:     map[string]DependencyCacheResolvedUnitRecord{},
		DependencyCacheByRequest: map[string][]string{},
		Sessions:                 map[string]SessionDurableState{},
		Approvals:                map[string]ApprovalRecord{},
		RunApprovalRefs:          map[string][]string{},
		PolicyDecisions:          map[string]PolicyDecisionRecord{},
		RunPolicyDecisionRefs:    map[string][]string{},
		ProviderProfiles:         map[string]ProviderProfileDurableState{},
		ProviderSetupSessions:    map[string]ProviderSetupSessionDurableState{},
		Policy:                   manifest.Policy,
		Runs:                     runs,
		PromotionEventsByActor:   map[string][]time.Time{},
		LastAuditSequence:        lastAuditSequence,
		StorageProtectionPosture: manifest.StorageProtection,
	}
}

func loadRestoredProviderProfiles(next *StoreState, records []ProviderProfileDurableState) error {
	for i, rec := range records {
		normalized := cloneProviderProfileDurableState(rec)
		if strings.TrimSpace(normalized.ProviderProfileID) == "" {
			return fmt.Errorf("provider profile id is required at restore index %d", i)
		}
		next.ProviderProfiles[normalized.ProviderProfileID] = normalized
	}
	return nil
}

func loadRestoredProviderSetupSessions(next *StoreState, records []ProviderSetupSessionDurableState) error {
	for i, rec := range records {
		normalized := cloneProviderSetupSessionDurableState(rec)
		if strings.TrimSpace(normalized.SetupSessionID) == "" {
			return fmt.Errorf("provider setup session id is required at restore index %d", i)
		}
		next.ProviderSetupSessions[normalized.SetupSessionID] = normalized
	}
	return nil
}

func loadRestoredSessions(next *StoreState, records []SessionDurableState) error {
	for i, rec := range records {
		normalized := normalizeSessionDurableState(rec)
		if normalized.SessionID == "" {
			return fmt.Errorf("session id is required at restore index %d (workspace=%q)", i, normalized.WorkspaceID)
		}
		next.Sessions[normalized.SessionID] = normalized
	}
	return nil
}

func loadRestoredArtifacts(next *StoreState, records []ArtifactRecord, ioStore *storeIO) (map[string]ArtifactRecord, error) {
	unapprovedByDigest := map[string]ArtifactRecord{}
	for _, rec := range records {
		validated, err := validateRestoredRecord(rec, ioStore)
		if err != nil {
			return nil, err
		}
		next.Artifacts[validated.Reference.Digest] = validated
		if validated.Reference.DataClass == DataClassUnapprovedFileExcerpts {
			unapprovedByDigest[validated.Reference.Digest] = validated
		}
	}
	return unapprovedByDigest, nil
}

func loadRestoredApprovals(next *StoreState, records []ApprovalRecord) error {
	for _, rec := range records {
		if err := validateApprovalRecord(rec); err != nil {
			return err
		}
		if err := requirePolicyDecisionHashForBoundApproval(rec); err != nil {
			return err
		}
		next.Approvals[rec.ApprovalID] = rec
		if rec.RunID != "" {
			next.RunApprovalRefs[rec.RunID] = uniqueSortedStrings(append(next.RunApprovalRefs[rec.RunID], rec.ApprovalID))
		}
	}
	return nil
}

func loadRestoredPolicyDecisions(next *StoreState, records []PolicyDecisionRecord) error {
	for _, rec := range records {
		if err := validatePolicyDecisionRecord(rec); err != nil {
			return err
		}
		if _, canonicalPayload, err := canonicalizePolicyDecisionRecord(rec); err != nil {
			return err
		} else if err := applyComputedPolicyDecisionDigest(&rec, canonicalPayload); err != nil {
			return err
		}
		next.PolicyDecisions[rec.Digest] = rec
		if rec.RunID != "" {
			next.RunPolicyDecisionRefs[rec.RunID] = uniqueSortedStrings(append(next.RunPolicyDecisionRefs[rec.RunID], rec.Digest))
		}
	}
	return nil
}

func loadRestoredDependencyCache(next *StoreState, batches []DependencyCacheBatchRecord, units []DependencyCacheResolvedUnitRecord) error {
	for _, unit := range units {
		if err := validateDependencyCacheResolvedUnitRecord(unit); err != nil {
			return err
		}
		next.DependencyCacheUnits[unit.ResolvedUnitDigest] = cloneDependencyResolvedUnitRecord(unit)
		next.DependencyCacheByRequest[unit.RequestDigest] = uniqueSortedStrings(append(next.DependencyCacheByRequest[unit.RequestDigest], unit.ResolvedUnitDigest))
	}
	for _, batch := range batches {
		if err := validateDependencyCacheBatchRecord(batch); err != nil {
			return err
		}
		next.DependencyCacheBatches[batch.BatchRequestDigest] = cloneDependencyBatchRecord(batch)
	}
	return nil
}

func validateRestoredApprovalPolicyDecisionLinks(approvals map[string]ApprovalRecord, decisions map[string]PolicyDecisionRecord) error {
	for approvalID, rec := range approvals {
		if !approvalHasBindingKeys(&rec) {
			continue
		}
		hash := strings.TrimSpace(rec.PolicyDecisionHash)
		decision, ok := decisions[hash]
		if !ok {
			return fmt.Errorf("%w: approval %q policy decision %q not found", ErrApprovalPolicyDecisionRequired, approvalID, hash)
		}
		if decision.ManifestHash != rec.ManifestHash || decision.ActionRequestHash != rec.ActionRequestHash {
			return fmt.Errorf("%w: approval %q policy decision %q binding mismatch", ErrApprovalPolicyDecisionRequired, approvalID, hash)
		}
	}
	return nil
}
