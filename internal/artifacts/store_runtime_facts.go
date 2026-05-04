package artifacts

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (s *Store) RecordRuntimeEvidenceState(runID string, facts launcherbackend.RuntimeFactsSnapshot, evidence launcherbackend.RuntimeEvidenceSnapshot, lifecycle launcherbackend.RuntimeLifecycleState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return fmt.Errorf("run id is required")
	}
	facts.LaunchReceipt = facts.LaunchReceipt.Normalized()
	facts.PostHandshakeAttestationInput = launcherbackend.NormalizePostHandshakeRuntimeAttestationInput(facts.PostHandshakeAttestationInput)
	facts.HardeningPosture = facts.HardeningPosture.Normalized()
	facts.TerminalReport = normalizeRuntimeTerminalReport(facts.TerminalReport)
	evidence = s.applyCachedAttestationVerificationLocked(evidence)
	if err := launcherbackend.ReconcileRuntimeEvidenceAttestation(facts.LaunchReceipt, facts.PostHandshakeAttestationInput, &evidence); err != nil {
		return err
	}
	reconcileAuthoritativeProvisioningPosture(&facts, &evidence)
	s.upsertAttestationVerificationCacheLocked(evidence)
	s.state.RuntimeFactsByRun[trimmedRunID] = facts
	s.state.RuntimeEvidenceByRun[trimmedRunID] = evidence
	s.state.RuntimeLifecycleByRun[trimmedRunID] = lifecycle
	if _, ok := s.state.RuntimeAuditStateByRun[trimmedRunID]; !ok {
		s.state.RuntimeAuditStateByRun[trimmedRunID] = RuntimeAuditEmissionState{}
	}
	if _, ok := s.state.Runs[trimmedRunID]; !ok {
		s.state.Runs[trimmedRunID] = "active"
	}
	_ = s.upsertSessionRuntimeBindingLocked(trimmedRunID, facts)
	return s.saveStateLocked()
}

func (s *Store) RuntimeEvidenceState(runID string) (launcherbackend.RuntimeFactsSnapshot, launcherbackend.RuntimeEvidenceSnapshot, launcherbackend.RuntimeLifecycleState, RuntimeAuditEmissionState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return launcherbackend.RuntimeFactsSnapshot{}, launcherbackend.RuntimeEvidenceSnapshot{}, launcherbackend.RuntimeLifecycleState{}, RuntimeAuditEmissionState{}, false
	}
	facts, ok := s.state.RuntimeFactsByRun[trimmedRunID]
	if !ok {
		return launcherbackend.RuntimeFactsSnapshot{}, launcherbackend.RuntimeEvidenceSnapshot{}, launcherbackend.RuntimeLifecycleState{}, RuntimeAuditEmissionState{}, false
	}
	evidence := s.state.RuntimeEvidenceByRun[trimmedRunID]
	evidence = s.applyCachedAttestationVerificationLocked(evidence)
	reconcileAuthoritativeProvisioningPosture(&facts, &evidence)
	lifecycle := s.state.RuntimeLifecycleByRun[trimmedRunID]
	auditState := s.state.RuntimeAuditStateByRun[trimmedRunID]
	return facts, evidence, lifecycle, auditState, true
}

func (s *Store) MarkRuntimeAuditEventEmitted(runID, eventType, evidenceDigest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return fmt.Errorf("run id is required")
	}
	state := s.state.RuntimeAuditStateByRun[trimmedRunID]
	switch eventType {
	case "isolate_session_started":
		state.LastIsolateSessionStartedDigest = evidenceDigest
	case "isolate_session_bound":
		state.LastIsolateSessionBoundDigest = evidenceDigest
	case "runtime_launch_admission":
		state.LastRuntimeLaunchAdmissionDigest = evidenceDigest
	case "runtime_launch_denied":
		state.LastRuntimeLaunchDeniedDigest = evidenceDigest
	default:
		return fmt.Errorf("unsupported runtime audit event type %q", eventType)
	}
	s.state.RuntimeAuditStateByRun[trimmedRunID] = state
	return s.saveStateLocked()
}

func (s *Store) UpdateRuntimeLifecycleState(runID string, lifecycle launcherbackend.RuntimeLifecycleState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return fmt.Errorf("run id is required")
	}
	facts, existed := s.runtimeFactsForLifecycleUpdateLocked(trimmedRunID)
	applyLifecycleToRuntimeFacts(&facts, &lifecycle)
	evidence, projectedLifecycle, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		if !existed {
			return s.persistLifecycleFallbackLocked(trimmedRunID, facts, lifecycle)
		}
		return err
	}
	evidence = s.applyCachedAttestationVerificationLocked(evidence)
	if err := launcherbackend.ReconcileRuntimeEvidenceAttestation(facts.LaunchReceipt, facts.PostHandshakeAttestationInput, &evidence); err != nil {
		return err
	}
	reconcileAuthoritativeProvisioningPosture(&facts, &evidence)
	s.upsertAttestationVerificationCacheLocked(evidence)
	s.state.RuntimeFactsByRun[trimmedRunID] = facts
	s.state.RuntimeEvidenceByRun[trimmedRunID] = evidence
	s.state.RuntimeLifecycleByRun[trimmedRunID] = projectedLifecycle
	if _, ok := s.state.Runs[trimmedRunID]; !ok {
		s.state.Runs[trimmedRunID] = "active"
	}
	_ = s.upsertSessionRuntimeBindingLocked(trimmedRunID, facts)
	return s.saveStateLocked()
}

func (s *Store) runtimeFactsForLifecycleUpdateLocked(runID string) (launcherbackend.RuntimeFactsSnapshot, bool) {
	facts, ok := s.state.RuntimeFactsByRun[runID]
	if ok {
		return facts, true
	}
	return launcherbackend.DefaultRuntimeFacts(runID), false
}

func applyLifecycleToRuntimeFacts(facts *launcherbackend.RuntimeFactsSnapshot, lifecycle *launcherbackend.RuntimeLifecycleState) {
	if facts == nil || lifecycle == nil {
		return
	}
	if lifecycle.BackendLifecycle != nil {
		normalized := lifecycle.BackendLifecycle.Normalized()
		facts.LaunchReceipt.Lifecycle = &normalized
	}
	if normalized := normalizeLifecycleProvisioningPosture(lifecycle); normalized != "" {
		facts.LaunchReceipt.ProvisioningPosture = normalized
	}
	facts.LaunchReceipt.ProvisioningPostureDegraded = lifecycle.ProvisioningPostureDegraded
	facts.LaunchReceipt.ProvisioningDegradedReasons = append([]string{}, lifecycle.ProvisioningDegradedReasons...)
	facts.LaunchReceipt.LaunchFailureReasonCode = strings.TrimSpace(lifecycle.LaunchFailureReasonCode)
}

func normalizeLifecycleProvisioningPosture(lifecycle *launcherbackend.RuntimeLifecycleState) string {
	if lifecycle == nil {
		return ""
	}
	if strings.TrimSpace(lifecycle.ProvisioningPosture) == launcherbackend.ProvisioningPostureAttested {
		lifecycle.ProvisioningPosture = ""
	}
	return lifecycle.ProvisioningPosture
}

func (s *Store) persistLifecycleFallbackLocked(runID string, facts launcherbackend.RuntimeFactsSnapshot, lifecycle launcherbackend.RuntimeLifecycleState) error {
	s.state.RuntimeFactsByRun[runID] = facts
	s.state.RuntimeLifecycleByRun[runID] = lifecycle
	if _, exists := s.state.Runs[runID]; !exists {
		s.state.Runs[runID] = "active"
	}
	return s.saveStateLocked()
}

func normalizeRuntimeTerminalReport(report *launcherbackend.BackendTerminalReport) *launcherbackend.BackendTerminalReport {
	if report == nil {
		return nil
	}
	normalized := report.Normalized()
	return &normalized
}

func reconcileAuthoritativeProvisioningPosture(facts *launcherbackend.RuntimeFactsSnapshot, evidence *launcherbackend.RuntimeEvidenceSnapshot) {
	if evidence == nil {
		return
	}
	posture := authoritativeProvisioningPostureFromEvidence(*evidence)
	evidence.Launch.ProvisioningPosture = posture
	if evidence.Session != nil {
		evidence.Session.ProvisioningPosture = sessionProvisioningPostureForEvidence(*evidence, posture)
	}
	if facts != nil {
		facts.LaunchReceipt.ProvisioningPosture = posture
	}
}

func authoritativeProvisioningPostureFromEvidence(evidence launcherbackend.RuntimeEvidenceSnapshot) string {
	launchPosture := strings.TrimSpace(evidence.Launch.ProvisioningPosture)
	attestationPosture, _ := launcherbackend.DeriveAttestationPostureFromEvidence(evidence)
	if attestationPosture == launcherbackend.AttestationPostureValid {
		return launcherbackend.ProvisioningPostureAttested
	}
	if launchPosture == launcherbackend.ProvisioningPostureAttested {
		return launcherbackend.ProvisioningPostureTOFU
	}
	return launchPosture
}

func sessionProvisioningPostureForEvidence(evidence launcherbackend.RuntimeEvidenceSnapshot, launchPosture string) string {
	if evidence.Session == nil {
		return ""
	}
	if strings.TrimSpace(evidence.Session.ProvisioningPosture) != launcherbackend.ProvisioningPostureAttested {
		return evidence.Session.ProvisioningPosture
	}
	if launchPosture == launcherbackend.ProvisioningPostureAttested {
		return launcherbackend.ProvisioningPostureAttested
	}
	return launcherbackend.ProvisioningPostureTOFU
}
