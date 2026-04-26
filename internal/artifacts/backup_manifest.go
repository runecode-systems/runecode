package artifacts

import (
	"sort"
	"time"
)

func buildBackupManifest(state StoreState, exportedAt time.Time) BackupManifest {
	manifest := newBackupManifest(state, exportedAt)
	populateBackupManifestCollections(&manifest, state)
	sortBackupManifestCollections(&manifest)
	return manifest
}

func newBackupManifest(state StoreState, exportedAt time.Time) BackupManifest {
	return BackupManifest{
		Schema:                 "runecode.backup.artifacts.v1",
		ExportedAt:             exportedAt,
		StorageProtection:      state.StorageProtectionPosture,
		Policy:                 state.Policy,
		Runs:                   map[string]string{},
		Artifacts:              make([]ArtifactRecord, 0, len(state.Artifacts)),
		DependencyCacheBatches: make([]DependencyCacheBatchRecord, 0, len(state.DependencyCacheBatches)),
		DependencyCacheUnits:   make([]DependencyCacheResolvedUnitRecord, 0, len(state.DependencyCacheUnits)),
		Sessions:               make([]SessionDurableState, 0, len(state.Sessions)),
		PolicyDecisions:        make([]PolicyDecisionRecord, 0, len(state.PolicyDecisions)),
		Approvals:              make([]ApprovalRecord, 0, len(state.Approvals)),
		ProviderProfiles:       make([]ProviderProfileDurableState, 0, len(state.ProviderProfiles)),
		ProviderSetupSessions:  make([]ProviderSetupSessionDurableState, 0, len(state.ProviderSetupSessions)),
	}
}

func populateBackupManifestCollections(manifest *BackupManifest, state StoreState) {
	for runID, status := range state.Runs {
		manifest.Runs[runID] = status
	}
	for _, rec := range state.Artifacts {
		manifest.Artifacts = append(manifest.Artifacts, rec)
	}
	for _, rec := range state.DependencyCacheBatches {
		manifest.DependencyCacheBatches = append(manifest.DependencyCacheBatches, cloneDependencyBatchRecord(rec))
	}
	for _, rec := range state.DependencyCacheUnits {
		manifest.DependencyCacheUnits = append(manifest.DependencyCacheUnits, cloneDependencyResolvedUnitRecord(rec))
	}
	for _, rec := range state.Sessions {
		manifest.Sessions = append(manifest.Sessions, copySessionDurableState(rec))
	}
	for _, rec := range state.PolicyDecisions {
		manifest.PolicyDecisions = append(manifest.PolicyDecisions, rec)
	}
	for _, rec := range state.Approvals {
		manifest.Approvals = append(manifest.Approvals, rec)
	}
	manifest.ProviderProfiles = append(manifest.ProviderProfiles, sortedProviderProfiles(state.ProviderProfiles)...)
	manifest.ProviderSetupSessions = append(manifest.ProviderSetupSessions, sortedProviderSetupSessions(state.ProviderSetupSessions)...)
}

func sortBackupManifestCollections(manifest *BackupManifest) {
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Reference.Digest < manifest.Artifacts[j].Reference.Digest
	})
	sort.Slice(manifest.Sessions, func(i, j int) bool {
		return manifest.Sessions[i].SessionID < manifest.Sessions[j].SessionID
	})
	sort.Slice(manifest.DependencyCacheBatches, func(i, j int) bool {
		return manifest.DependencyCacheBatches[i].BatchRequestDigest < manifest.DependencyCacheBatches[j].BatchRequestDigest
	})
	sort.Slice(manifest.DependencyCacheUnits, func(i, j int) bool {
		return manifest.DependencyCacheUnits[i].ResolvedUnitDigest < manifest.DependencyCacheUnits[j].ResolvedUnitDigest
	})
	sort.Slice(manifest.PolicyDecisions, func(i, j int) bool {
		return manifest.PolicyDecisions[i].Digest < manifest.PolicyDecisions[j].Digest
	})
	sort.Slice(manifest.Approvals, func(i, j int) bool {
		return manifest.Approvals[i].ApprovalID < manifest.Approvals[j].ApprovalID
	})
	sort.Slice(manifest.ProviderProfiles, func(i, j int) bool {
		return manifest.ProviderProfiles[i].ProviderProfileID < manifest.ProviderProfiles[j].ProviderProfileID
	})
	sort.Slice(manifest.ProviderSetupSessions, func(i, j int) bool {
		return manifest.ProviderSetupSessions[i].SetupSessionID < manifest.ProviderSetupSessions[j].SetupSessionID
	})
}
