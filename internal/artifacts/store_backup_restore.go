package artifacts

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
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
	return normalizeState(next), nil
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
		func() error { return loadRestoredGitRemotePrepared(next, manifest.GitRemotePrepared) },
		func() error {
			return loadRestoredRuntimeState(next, manifest.RuntimeFactsByRun, manifest.RuntimeEvidenceByRun, manifest.AttestationVerificationCache, manifest.RuntimeLifecycleByRun, manifest.RuntimeAuditStateByRun, manifest.RunnerAdvisoryByRun)
		},
		func() error {
			return loadRestoredDependencyCache(next, manifest.DependencyCacheBatches, manifest.DependencyCacheUnits)
		},
		func() error { return validateRestoredDependencyCache(next) },
		func() error { return loadRestoredProviderProfiles(next, manifest.ProviderProfiles) },
		func() error { return loadRestoredProviderSetupSessions(next, manifest.ProviderSetupSessions) },
		func() error {
			return loadRestoredRunPlans(next, manifest.RunPlanAuthorities, manifest.RunPlanCompilations, ioStore)
		},
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
		Artifacts:                    map[string]ArtifactRecord{},
		DependencyCacheBatches:       map[string]DependencyCacheBatchRecord{},
		DependencyCacheUnits:         map[string]DependencyCacheResolvedUnitRecord{},
		DependencyCacheByRequest:     map[string][]string{},
		Sessions:                     map[string]SessionDurableState{},
		Approvals:                    map[string]ApprovalRecord{},
		RunApprovalRefs:              map[string][]string{},
		PolicyDecisions:              map[string]PolicyDecisionRecord{},
		RunPolicyDecisionRefs:        map[string][]string{},
		GitRemotePrepared:            map[string]GitRemotePreparedMutationRecord{},
		RunGitRemotePreparedRefs:     map[string][]string{},
		RuntimeFactsByRun:            map[string]launcherbackend.RuntimeFactsSnapshot{},
		RuntimeEvidenceByRun:         map[string]launcherbackend.RuntimeEvidenceSnapshot{},
		AttestationVerificationCache: map[string]launcherbackend.IsolateAttestationVerificationRecord{},
		RuntimeLifecycleByRun:        map[string]launcherbackend.RuntimeLifecycleState{},
		RuntimeAuditStateByRun:       map[string]RuntimeAuditEmissionState{},
		RunnerAdvisoryByRun:          map[string]RunnerAdvisoryState{},
		ProviderProfiles:             map[string]ProviderProfileDurableState{},
		ProviderSetupSessions:        map[string]ProviderSetupSessionDurableState{},
		RunPlanAuthorities:           map[string]RunPlanAuthorityRecord{},
		RunPlanRefsByRun:             map[string][]string{},
		RunPlanCompilations:          map[string]RunPlanCompilationRecord{},
		RunPlanCompilationByCacheKey: map[string]string{},
		Policy:                       manifest.Policy,
		Runs:                         runs,
		PromotionEventsByActor:       map[string][]time.Time{},
		LastAuditSequence:            lastAuditSequence,
		StorageProtectionPosture:     manifest.StorageProtection,
	}
}

func loadRestoredGitRemotePrepared(next *StoreState, records []GitRemotePreparedMutationRecord) error {
	for i, rec := range records {
		normalized := cloneGitRemotePreparedRecord(rec)
		if err := validateGitRemotePreparedRecord(normalized); err != nil {
			return fmt.Errorf("git remote prepared restore index %d: %w", i, err)
		}
		next.GitRemotePrepared[normalized.PreparedMutationID] = normalized
	}
	rebuildRunGitRemotePreparedRefsLocked(next)
	return nil
}

func loadRestoredRuntimeState(next *StoreState, factsByRun map[string]launcherbackend.RuntimeFactsSnapshot, evidenceByRun map[string]launcherbackend.RuntimeEvidenceSnapshot, verificationCache map[string]launcherbackend.IsolateAttestationVerificationRecord, lifecycleByRun map[string]launcherbackend.RuntimeLifecycleState, auditStateByRun map[string]RuntimeAuditEmissionState, advisoryByRun map[string]RunnerAdvisoryState) error {
	if err := loadRestoredRuntimeFacts(next, factsByRun); err != nil {
		return err
	}
	restorableEvidence, err := deriveRestorableRuntimeEvidence(next.RuntimeFactsByRun)
	if err != nil {
		return err
	}
	if err := loadRestoredRuntimeEvidence(next, evidenceByRun, restorableEvidence); err != nil {
		return err
	}
	if err := loadRestoredAttestationVerificationCache(next, verificationCache); err != nil {
		return err
	}
	if err := loadRestoredRuntimeLifecycle(next, lifecycleByRun); err != nil {
		return err
	}
	if err := loadRestoredRuntimeAuditState(next, auditStateByRun); err != nil {
		return err
	}
	return loadRestoredRunnerAdvisory(next, advisoryByRun)
}

func loadRestoredRuntimeFacts(next *StoreState, factsByRun map[string]launcherbackend.RuntimeFactsSnapshot) error {
	for runID, facts := range factsByRun {
		trimmedRunID, err := validateRestoredRuntimeRunID(runID, "runtime facts")
		if err != nil {
			return err
		}
		next.RuntimeFactsByRun[trimmedRunID] = cloneRuntimeFactsSnapshot(facts)
	}
	return nil
}

func loadRestoredRuntimeLifecycle(next *StoreState, lifecycleByRun map[string]launcherbackend.RuntimeLifecycleState) error {
	for runID, lifecycle := range lifecycleByRun {
		trimmedRunID, err := validateRestoredRuntimeRunID(runID, "runtime lifecycle")
		if err != nil {
			return err
		}
		next.RuntimeLifecycleByRun[trimmedRunID] = lifecycle
	}
	return nil
}

func loadRestoredRuntimeAuditState(next *StoreState, auditStateByRun map[string]RuntimeAuditEmissionState) error {
	for runID, auditState := range auditStateByRun {
		trimmedRunID, err := validateRestoredRuntimeRunID(runID, "runtime audit state")
		if err != nil {
			return err
		}
		next.RuntimeAuditStateByRun[trimmedRunID] = auditState
	}
	return nil
}

func loadRestoredRunnerAdvisory(next *StoreState, advisoryByRun map[string]RunnerAdvisoryState) error {
	for runID, advisory := range advisoryByRun {
		trimmedRunID, err := validateRestoredRuntimeRunID(runID, "runner advisory")
		if err != nil {
			return err
		}
		next.RunnerAdvisoryByRun[trimmedRunID] = copyRunnerAdvisoryState(advisory)
	}
	return nil
}

func validateRestoredRuntimeRunID(runID, context string) (string, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return "", fmt.Errorf("%s restore run id is required", context)
	}
	return trimmedRunID, nil
}
