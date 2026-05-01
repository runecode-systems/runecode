package artifacts

import (
	"sort"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func buildBackupManifest(state StoreState, exportedAt time.Time) BackupManifest {
	manifest := newBackupManifest(state, exportedAt)
	populateBackupManifestCollections(&manifest, state)
	sortBackupManifestCollections(&manifest)
	return manifest
}

func newBackupManifest(state StoreState, exportedAt time.Time) BackupManifest {
	return BackupManifest{
		Schema:                       "runecode.backup.artifacts.v1",
		ExportedAt:                   exportedAt,
		StorageProtection:            state.StorageProtectionPosture,
		Policy:                       state.Policy,
		Runs:                         map[string]string{},
		Artifacts:                    make([]ArtifactRecord, 0, len(state.Artifacts)),
		DependencyCacheBatches:       make([]DependencyCacheBatchRecord, 0, len(state.DependencyCacheBatches)),
		DependencyCacheUnits:         make([]DependencyCacheResolvedUnitRecord, 0, len(state.DependencyCacheUnits)),
		Sessions:                     make([]SessionDurableState, 0, len(state.Sessions)),
		PolicyDecisions:              make([]PolicyDecisionRecord, 0, len(state.PolicyDecisions)),
		Approvals:                    make([]ApprovalRecord, 0, len(state.Approvals)),
		GitRemotePrepared:            make([]GitRemotePreparedMutationRecord, 0, len(state.GitRemotePrepared)),
		RuntimeFactsByRun:            map[string]launcherbackend.RuntimeFactsSnapshot{},
		RuntimeEvidenceByRun:         map[string]launcherbackend.RuntimeEvidenceSnapshot{},
		AttestationVerificationCache: map[string]launcherbackend.IsolateAttestationVerificationRecord{},
		RuntimeLifecycleByRun:        map[string]launcherbackend.RuntimeLifecycleState{},
		RuntimeAuditStateByRun:       map[string]RuntimeAuditEmissionState{},
		RunnerAdvisoryByRun:          map[string]RunnerAdvisoryState{},
		ProviderProfiles:             make([]ProviderProfileDurableState, 0, len(state.ProviderProfiles)),
		ProviderSetupSessions:        make([]ProviderSetupSessionDurableState, 0, len(state.ProviderSetupSessions)),
		RunPlanAuthorities:           make([]RunPlanAuthorityRecord, 0, len(state.RunPlanAuthorities)),
		RunPlanCompilations:          make([]RunPlanCompilationRecord, 0, len(state.RunPlanCompilations)),
	}
}

func populateBackupManifestCollections(manifest *BackupManifest, state StoreState) {
	populateBackupManifestPrimaryCollections(manifest, state)
	populateBackupManifestRuntimeCollections(manifest, state)
	populateBackupManifestProviderCollections(manifest, state)
	populateBackupManifestRunPlanCollections(manifest, state)
}

func populateBackupManifestPrimaryCollections(manifest *BackupManifest, state StoreState) {
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
	for _, rec := range state.GitRemotePrepared {
		manifest.GitRemotePrepared = append(manifest.GitRemotePrepared, cloneGitRemotePreparedRecord(rec))
	}
}

func populateBackupManifestRuntimeCollections(manifest *BackupManifest, state StoreState) {
	for runID, facts := range state.RuntimeFactsByRun {
		manifest.RuntimeFactsByRun[runID] = cloneRuntimeFactsSnapshot(facts)
	}
	for runID, evidence := range state.RuntimeEvidenceByRun {
		manifest.RuntimeEvidenceByRun[runID] = evidence
	}
	for key, record := range state.AttestationVerificationCache {
		manifest.AttestationVerificationCache[key] = cloneAttestationVerificationRecord(record)
	}
	for runID, lifecycle := range state.RuntimeLifecycleByRun {
		manifest.RuntimeLifecycleByRun[runID] = lifecycle
	}
	for runID, auditState := range state.RuntimeAuditStateByRun {
		manifest.RuntimeAuditStateByRun[runID] = auditState
	}
	for runID, advisory := range state.RunnerAdvisoryByRun {
		manifest.RunnerAdvisoryByRun[runID] = copyRunnerAdvisoryState(advisory)
	}
}

func populateBackupManifestProviderCollections(manifest *BackupManifest, state StoreState) {
	manifest.ProviderProfiles = append(manifest.ProviderProfiles, sortedProviderProfiles(state.ProviderProfiles)...)
	manifest.ProviderSetupSessions = append(manifest.ProviderSetupSessions, sortedProviderSetupSessions(state.ProviderSetupSessions)...)
}

func populateBackupManifestRunPlanCollections(manifest *BackupManifest, state StoreState) {
	for _, rec := range state.RunPlanAuthorities {
		manifest.RunPlanAuthorities = append(manifest.RunPlanAuthorities, cloneRunPlanAuthorityRecord(rec))
	}
	for _, rec := range state.RunPlanCompilations {
		manifest.RunPlanCompilations = append(manifest.RunPlanCompilations, rec)
	}
}

func sortBackupManifestCollections(manifest *BackupManifest) {
	sortBackupManifestCoreCollections(manifest)
	sortBackupManifestProviderCollections(manifest)
	sortBackupManifestRunPlanCollections(manifest)
}

func sortBackupManifestCoreCollections(manifest *BackupManifest) {
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Reference.Digest < manifest.Artifacts[j].Reference.Digest
	})
	sort.Slice(manifest.Sessions, func(i, j int) bool { return manifest.Sessions[i].SessionID < manifest.Sessions[j].SessionID })
	sort.Slice(manifest.DependencyCacheBatches, func(i, j int) bool {
		return manifest.DependencyCacheBatches[i].BatchRequestDigest < manifest.DependencyCacheBatches[j].BatchRequestDigest
	})
	sort.Slice(manifest.DependencyCacheUnits, func(i, j int) bool {
		return manifest.DependencyCacheUnits[i].ResolvedUnitDigest < manifest.DependencyCacheUnits[j].ResolvedUnitDigest
	})
	sort.Slice(manifest.PolicyDecisions, func(i, j int) bool { return manifest.PolicyDecisions[i].Digest < manifest.PolicyDecisions[j].Digest })
	sort.Slice(manifest.Approvals, func(i, j int) bool { return manifest.Approvals[i].ApprovalID < manifest.Approvals[j].ApprovalID })
	sort.Slice(manifest.GitRemotePrepared, func(i, j int) bool {
		return manifest.GitRemotePrepared[i].PreparedMutationID < manifest.GitRemotePrepared[j].PreparedMutationID
	})
}

func sortBackupManifestProviderCollections(manifest *BackupManifest) {
	sort.Slice(manifest.ProviderProfiles, func(i, j int) bool {
		return manifest.ProviderProfiles[i].ProviderProfileID < manifest.ProviderProfiles[j].ProviderProfileID
	})
	sort.Slice(manifest.ProviderSetupSessions, func(i, j int) bool {
		return manifest.ProviderSetupSessions[i].SetupSessionID < manifest.ProviderSetupSessions[j].SetupSessionID
	})
}

func sortBackupManifestRunPlanCollections(manifest *BackupManifest) {
	sort.Slice(manifest.RunPlanAuthorities, func(i, j int) bool {
		return runPlanLess(manifest.RunPlanAuthorities[i].RunID, manifest.RunPlanAuthorities[i].PlanID, manifest.RunPlanAuthorities[j].RunID, manifest.RunPlanAuthorities[j].PlanID)
	})
	sort.Slice(manifest.RunPlanCompilations, func(i, j int) bool {
		return runPlanLess(manifest.RunPlanCompilations[i].RunID, manifest.RunPlanCompilations[i].PlanID, manifest.RunPlanCompilations[j].RunID, manifest.RunPlanCompilations[j].PlanID)
	})
}

func runPlanLess(leftRunID, leftPlanID, rightRunID, rightPlanID string) bool {
	if leftRunID != rightRunID {
		return leftRunID < rightRunID
	}
	return leftPlanID < rightPlanID
}

func cloneGitRemotePreparedRecord(in GitRemotePreparedMutationRecord) GitRemotePreparedMutationRecord {
	out := in
	if in.TypedRequest != nil {
		out.TypedRequest = copyMap(in.TypedRequest)
	}
	if in.DerivedSummary != nil {
		out.DerivedSummary = copyMap(in.DerivedSummary)
	}
	return out
}

func cloneRuntimeFactsSnapshot(in launcherbackend.RuntimeFactsSnapshot) launcherbackend.RuntimeFactsSnapshot {
	out := in
	out.LaunchReceipt = in.LaunchReceipt.Normalized()
	out.HardeningPosture = in.HardeningPosture.Normalized()
	out.TerminalReport = normalizeRuntimeTerminalReport(in.TerminalReport)
	return out
}
