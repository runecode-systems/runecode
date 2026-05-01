package artifacts

import (
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func normalizeState(state StoreState) StoreState {
	state = normalizePrimaryStateMaps(state)
	state = normalizeRuntimeStateMaps(state)
	if state.Policy.HandOffReferenceMode == "" {
		state.Policy = DefaultPolicy()
	} else {
		state.Policy.DependencyCachePolicy = normalizeDependencyCachePolicy(state.Policy.DependencyCachePolicy)
	}
	if state.StorageProtectionPosture == "" {
		state.StorageProtectionPosture = "encrypted_at_rest_default"
	}
	return state
}

func normalizeDependencyCachePolicy(policy DependencyCachePolicy) DependencyCachePolicy {
	if !policy.ReadOnlyArtifactsRequired {
		policy.ReadOnlyArtifactsRequired = true
	}
	if !policy.BatchManifestImmutable {
		policy.BatchManifestImmutable = true
	}
	if !policy.ResolvedUnitManifestImmutable {
		policy.ResolvedUnitManifestImmutable = true
	}
	if !policy.ResolvedPayloadImmutable {
		policy.ResolvedPayloadImmutable = true
	}
	if !policy.MaterializedTreesDerivedNonCanonical {
		policy.MaterializedTreesDerivedNonCanonical = true
	}
	if !policy.FailClosedOnAmbiguousPartialReuse {
		policy.FailClosedOnAmbiguousPartialReuse = true
	}
	if !policy.FailClosedOnIncompleteState {
		policy.FailClosedOnIncompleteState = true
	}
	if !policy.RetainCanonicalBeforeDerived {
		policy.RetainCanonicalBeforeDerived = true
	}
	return policy
}

func normalizePrimaryStateMaps(state StoreState) StoreState {
	state = normalizeCoreStateMaps(state)
	state = normalizeApprovalStateMaps(state)
	state = normalizeProviderStateMaps(state)
	return normalizePromotionStateMaps(state)
}

func normalizeCoreStateMaps(state StoreState) StoreState {
	if state.Artifacts == nil {
		state.Artifacts = map[string]ArtifactRecord{}
	}
	if state.DependencyCacheBatches == nil {
		state.DependencyCacheBatches = map[string]DependencyCacheBatchRecord{}
	}
	if state.DependencyCacheUnits == nil {
		state.DependencyCacheUnits = map[string]DependencyCacheResolvedUnitRecord{}
	}
	if state.DependencyCacheByRequest == nil {
		state.DependencyCacheByRequest = map[string][]string{}
	}
	if state.Sessions == nil {
		state.Sessions = map[string]SessionDurableState{}
	}
	if state.Runs == nil {
		state.Runs = map[string]string{}
	}
	if state.PolicyDecisions == nil {
		state.PolicyDecisions = map[string]PolicyDecisionRecord{}
	}
	if state.RunPolicyDecisionRefs == nil {
		state.RunPolicyDecisionRefs = map[string][]string{}
	}
	return state
}

func normalizeApprovalStateMaps(state StoreState) StoreState {
	if state.Approvals == nil {
		state.Approvals = map[string]ApprovalRecord{}
	}
	if state.RunApprovalRefs == nil {
		state.RunApprovalRefs = map[string][]string{}
	}
	if state.GitRemotePrepared == nil {
		state.GitRemotePrepared = map[string]GitRemotePreparedMutationRecord{}
	}
	if state.RunGitRemotePreparedRefs == nil {
		state.RunGitRemotePreparedRefs = map[string][]string{}
	}
	return state
}

func normalizeProviderStateMaps(state StoreState) StoreState {
	if state.ProviderProfiles == nil {
		state.ProviderProfiles = map[string]ProviderProfileDurableState{}
	}
	if state.ProviderSetupSessions == nil {
		state.ProviderSetupSessions = map[string]ProviderSetupSessionDurableState{}
	}
	if state.RunPlanAuthorities == nil {
		state.RunPlanAuthorities = map[string]RunPlanAuthorityRecord{}
	}
	if state.RunPlanRefsByRun == nil {
		state.RunPlanRefsByRun = map[string][]string{}
	}
	if state.RunPlanCompilations == nil {
		state.RunPlanCompilations = map[string]RunPlanCompilationRecord{}
	}
	if state.RunPlanCompilationByCacheKey == nil {
		state.RunPlanCompilationByCacheKey = map[string]string{}
	}
	return state
}

func normalizePromotionStateMaps(state StoreState) StoreState {
	if state.PromotionEventsByActor == nil {
		state.PromotionEventsByActor = map[string][]time.Time{}
	}
	return state
}

func normalizeRuntimeStateMaps(state StoreState) StoreState {
	if state.RuntimeFactsByRun == nil {
		state.RuntimeFactsByRun = map[string]launcherbackend.RuntimeFactsSnapshot{}
	}
	if state.RuntimeEvidenceByRun == nil {
		state.RuntimeEvidenceByRun = map[string]launcherbackend.RuntimeEvidenceSnapshot{}
	}
	if state.AttestationVerificationCache == nil {
		state.AttestationVerificationCache = map[string]launcherbackend.IsolateAttestationVerificationRecord{}
	}
	if state.RuntimeLifecycleByRun == nil {
		state.RuntimeLifecycleByRun = map[string]launcherbackend.RuntimeLifecycleState{}
	}
	if state.RuntimeAuditStateByRun == nil {
		state.RuntimeAuditStateByRun = map[string]RuntimeAuditEmissionState{}
	}
	if state.RunnerAdvisoryByRun == nil {
		state.RunnerAdvisoryByRun = map[string]RunnerAdvisoryState{}
	}
	return state
}
