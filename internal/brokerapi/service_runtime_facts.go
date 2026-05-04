package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Service) RecordRuntimeFacts(runID string, facts launcherbackend.RuntimeFactsSnapshot) error {
	normalizedRunID := strings.TrimSpace(runID)
	if normalizedRunID == "" {
		return fmt.Errorf("run id is required")
	}
	if embeddedRunID := strings.TrimSpace(facts.LaunchReceipt.RunID); embeddedRunID != "" && embeddedRunID != normalizedRunID {
		return fmt.Errorf("runtime facts launch_receipt.run_id %q does not match requested run id %q", embeddedRunID, normalizedRunID)
	}
	facts = normalizeRuntimeFactsSnapshot(normalizedRunID, facts)
	evidence, lifecycle, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		return err
	}
	if err := s.store.RecordRuntimeEvidenceState(normalizedRunID, facts, evidence, lifecycle); err != nil {
		return err
	}
	persistedFacts, persistedEvidence, _, _, ok := s.store.RuntimeEvidenceState(normalizedRunID)
	if !ok {
		return fmt.Errorf("persisted runtime evidence state missing for run %q", normalizedRunID)
	}
	if err := s.syncRunStatusFromRuntimeFacts(normalizedRunID, persistedFacts); err != nil {
		return err
	}
	runtimeSupportState := runtimeAuditSupportState(persistedEvidence, s.currentInstanceBackendPosture().InstanceID, normalizedRunID, s.PolicyDecisionRefsForRun(normalizedRunID), s.listApprovals())
	runnerAdvisory, _ := s.RunnerAdvisory(normalizedRunID)
	if err := s.SyncSessionExecutionFromRunRuntime(normalizedRunID, persistedFacts, runnerAdvisory, s.now().UTC()); err != nil {
		return err
	}
	if err := s.emitRuntimeEvidenceAuditEvents(normalizedRunID, persistedFacts, persistedEvidence, runtimeSupportState); err != nil {
		return err
	}
	return nil
}

func (s *Service) RuntimeFacts(runID string) launcherbackend.RuntimeFactsSnapshot {
	facts, evidence, lifecycle, _, ok := s.store.RuntimeEvidenceState(runID)
	if !ok {
		return launcherbackend.DefaultRuntimeFacts(runID)
	}
	facts = normalizeRuntimeFactsSnapshot(runID, facts)
	applyPersistedLifecycle(&facts, lifecycle)
	facts.LaunchReceipt.ProvisioningPosture = authoritativeRuntimeProvisioningPosture(facts.LaunchReceipt.ProvisioningPosture, evidence)
	return facts
}

func (s *Service) RuntimeEvidence(runID string) launcherbackend.RuntimeEvidenceSnapshot {
	_, evidence, _, _, ok := s.store.RuntimeEvidenceState(runID)
	if !ok {
		return launcherbackend.RuntimeEvidenceSnapshot{}
	}
	return evidence
}

func normalizeRuntimeFactsSnapshot(runID string, input launcherbackend.RuntimeFactsSnapshot) launcherbackend.RuntimeFactsSnapshot {
	facts := input
	facts.LaunchReceipt = facts.LaunchReceipt.Normalized()
	facts.PostHandshakeAttestationInput = launcherbackend.NormalizePostHandshakeRuntimeAttestationInput(facts.PostHandshakeAttestationInput)
	facts.HardeningPosture = normalizeRuntimeHardeningPosture(facts.HardeningPosture, facts.LaunchReceipt)
	facts.TerminalReport = normalizeRuntimeTerminalReport(facts.TerminalReport)
	if facts.LaunchReceipt.RunID == "" {
		facts.LaunchReceipt.RunID = runID
	}
	return facts
}

func applyPersistedLifecycle(facts *launcherbackend.RuntimeFactsSnapshot, lifecycle launcherbackend.RuntimeLifecycleState) {
	if facts == nil {
		return
	}
	if lifecycle.BackendLifecycle != nil {
		normalized := lifecycle.BackendLifecycle.Normalized()
		facts.LaunchReceipt.Lifecycle = &normalized
	}
	if strings.TrimSpace(lifecycle.ProvisioningPosture) != "" {
		facts.LaunchReceipt.ProvisioningPosture = lifecycle.ProvisioningPosture
	}
	facts.LaunchReceipt.ProvisioningPostureDegraded = lifecycle.ProvisioningPostureDegraded
	facts.LaunchReceipt.ProvisioningDegradedReasons = append([]string{}, lifecycle.ProvisioningDegradedReasons...)
	facts.LaunchReceipt.LaunchFailureReasonCode = strings.TrimSpace(lifecycle.LaunchFailureReasonCode)
}

func authoritativeRuntimeProvisioningPosture(current string, evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	posture := strings.TrimSpace(current)
	if posture != launcherbackend.ProvisioningPostureAttested {
		return posture
	}
	attestationPosture, _ := launcherbackend.DeriveAttestationPostureFromEvidence(evidence)
	if attestationPosture == launcherbackend.AttestationPostureValid {
		return launcherbackend.ProvisioningPostureAttested
	}
	return launcherbackend.ProvisioningPostureTOFU
}

func normalizeRuntimeHardeningPosture(input launcherbackend.AppliedHardeningPosture, receipt launcherbackend.BackendLaunchReceipt) launcherbackend.AppliedHardeningPosture {
	hardening := input.Normalized()
	hardening = normalizeContainerMVPHardeningPosture(hardening, receipt)
	if err := hardening.Validate(); err != nil {
		hardening = launcherbackend.DefaultAppliedHardeningPosture()
		hardening.DegradedReasons = append(hardening.DegradedReasons, "hardening_posture_invalid")
		hardening = hardening.Normalized()
	}
	return hardening
}

func normalizeContainerMVPHardeningPosture(posture launcherbackend.AppliedHardeningPosture, receipt launcherbackend.BackendLaunchReceipt) launcherbackend.AppliedHardeningPosture {
	if receipt.BackendKind != launcherbackend.BackendKindContainer {
		return posture
	}
	reasons := append([]string{}, posture.DegradedReasons...)
	reasons = appendContainerRoleScopeReason(reasons, receipt.RoleFamily)
	reasons = appendContainerRootlessReasons(reasons, posture.RootlessPosture)
	reasons = appendContainerKernelAndFsReasons(reasons, posture)
	reasons = appendContainerNetworkReasons(reasons, posture)
	posture.DegradedReasons = reasons
	return posture.Normalized()
}

func normalizeRuntimeTerminalReport(input *launcherbackend.BackendTerminalReport) *launcherbackend.BackendTerminalReport {
	if input == nil {
		return nil
	}
	normalized := input.Normalized()
	if err := normalized.Validate(); err != nil {
		normalized.TerminationKind = launcherbackend.BackendTerminationKindFailed
		normalized.FailClosed = true
		normalized.FallbackPosture = launcherbackend.BackendFallbackPostureNoAutomaticFallback
		normalized.FailureReasonCode = launcherbackend.BackendErrorCodeTerminalReportInvalid
		normalized = normalized.Normalized()
	}
	return &normalized
}

func (s *Service) RecordRuntimeLifecycleState(runID string, lifecycle launcherbackend.RuntimeLifecycleState) error {
	normalizedRunID := strings.TrimSpace(runID)
	if normalizedRunID == "" {
		return fmt.Errorf("run id is required")
	}
	if lifecycle.BackendLifecycle != nil {
		normalized := lifecycle.BackendLifecycle.Normalized()
		lifecycle.BackendLifecycle = &normalized
	}
	lifecycle.ProvisioningDegradedReasons = uniqueSortedStrings(lifecycle.ProvisioningDegradedReasons)
	if err := s.store.UpdateRuntimeLifecycleState(normalizedRunID, lifecycle); err != nil {
		return err
	}
	if err := s.syncRunStatusFromRuntimeLifecycle(normalizedRunID, lifecycle); err != nil {
		return err
	}
	return nil
}

func (s *Service) syncRunStatusFromRuntimeFacts(runID string, facts launcherbackend.RuntimeFactsSnapshot) error {
	status, ok := runStatusFromRuntimeFacts(facts)
	if !ok {
		return nil
	}
	return s.SetRunStatus(runID, status)
}

func (s *Service) syncRunStatusFromRuntimeLifecycle(runID string, lifecycle launcherbackend.RuntimeLifecycleState) error {
	status, ok := runStatusFromRuntimeLifecycle(lifecycle)
	if !ok {
		return nil
	}
	return s.SetRunStatus(runID, status)
}

func runStatusFromRuntimeFacts(facts launcherbackend.RuntimeFactsSnapshot) (string, bool) {
	if facts.TerminalReport != nil {
		switch facts.TerminalReport.TerminationKind {
		case launcherbackend.BackendTerminationKindCompleted:
			return "completed", true
		case launcherbackend.BackendTerminationKindFailed:
			return "failed", true
		}
	}
	receipt := facts.LaunchReceipt.Normalized()
	if strings.TrimSpace(receipt.LaunchFailureReasonCode) != "" {
		return "failed", true
	}
	if !hasAuthoritativeRuntimeLifecycle(receipt) {
		return "", false
	}
	return runStatusFromBackendLifecycleState(receipt.Lifecycle.CurrentState, "")
}

func runStatusFromRuntimeLifecycle(lifecycle launcherbackend.RuntimeLifecycleState) (string, bool) {
	if strings.TrimSpace(lifecycle.LaunchFailureReasonCode) != "" {
		return "failed", true
	}
	if lifecycle.BackendLifecycle == nil {
		return "", false
	}
	return runStatusFromBackendLifecycleState(lifecycle.BackendLifecycle.CurrentState, "")
}

func runStatusFromBackendLifecycleState(currentState string, failureReason string) (string, bool) {
	if strings.TrimSpace(failureReason) != "" {
		return "failed", true
	}
	switch currentState {
	case launcherbackend.BackendLifecycleStatePlanned:
		return "pending", true
	case launcherbackend.BackendLifecycleStateLaunching, launcherbackend.BackendLifecycleStateStarted, launcherbackend.BackendLifecycleStateBinding:
		return "starting", true
	case launcherbackend.BackendLifecycleStateActive, launcherbackend.BackendLifecycleStateTerminating:
		return "active", true
	case launcherbackend.BackendLifecycleStateTerminated:
		return "completed", true
	default:
		return "", false
	}
}

func hasAuthoritativeRuntimeLifecycle(receipt launcherbackend.BackendLaunchReceipt) bool {
	if receipt.Lifecycle == nil {
		return false
	}
	if strings.TrimSpace(receipt.LaunchFailureReasonCode) != "" {
		return true
	}
	return receipt.BackendKind != launcherbackend.BackendKindUnknown
}
