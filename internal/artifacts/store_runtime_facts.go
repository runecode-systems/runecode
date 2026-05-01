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
	facts.HardeningPosture = facts.HardeningPosture.Normalized()
	facts.TerminalReport = normalizeRuntimeTerminalReport(facts.TerminalReport)
	evidence = s.applyCachedAttestationVerificationLocked(evidence)
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
	facts, ok := s.state.RuntimeFactsByRun[trimmedRunID]
	if !ok {
		facts = launcherbackend.DefaultRuntimeFacts(trimmedRunID)
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
	evidence, projectedLifecycle, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		return err
	}
	evidence = s.applyCachedAttestationVerificationLocked(evidence)
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

func normalizeRuntimeTerminalReport(report *launcherbackend.BackendTerminalReport) *launcherbackend.BackendTerminalReport {
	if report == nil {
		return nil
	}
	normalized := report.Normalized()
	return &normalized
}
