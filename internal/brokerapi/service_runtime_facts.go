package brokerapi

import (
	"fmt"
	"slices"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) RecordRuntimeFacts(runID string, facts launcherbackend.RuntimeFactsSnapshot) error {
	normalizedRunID := strings.TrimSpace(runID)
	if normalizedRunID == "" {
		return fmt.Errorf("run id is required")
	}
	facts = normalizeRuntimeFactsSnapshot(normalizedRunID, facts)
	evidence, lifecycle, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		return err
	}
	if err := s.store.RecordRuntimeEvidenceState(normalizedRunID, facts, evidence, lifecycle); err != nil {
		return err
	}
	if err := s.emitRuntimeEvidenceAuditEvents(normalizedRunID, facts, evidence); err != nil {
		return err
	}
	return nil
}

func (s *Service) RuntimeFacts(runID string) launcherbackend.RuntimeFactsSnapshot {
	facts, _, lifecycle, _, ok := s.store.RuntimeEvidenceState(runID)
	if !ok {
		return launcherbackend.DefaultRuntimeFacts(runID)
	}
	facts = normalizeRuntimeFactsSnapshot(runID, facts)
	applyPersistedLifecycle(&facts, lifecycle)
	return facts
}

func normalizeRuntimeFactsSnapshot(runID string, input launcherbackend.RuntimeFactsSnapshot) launcherbackend.RuntimeFactsSnapshot {
	facts := input
	facts.LaunchReceipt = facts.LaunchReceipt.Normalized()
	facts.HardeningPosture = normalizeRuntimeHardeningPosture(facts.HardeningPosture)
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
	if len(lifecycle.ProvisioningDegradedReasons) > 0 {
		facts.LaunchReceipt.ProvisioningDegradedReasons = append([]string{}, lifecycle.ProvisioningDegradedReasons...)
	}
	if strings.TrimSpace(lifecycle.LaunchFailureReasonCode) != "" {
		facts.LaunchReceipt.LaunchFailureReasonCode = lifecycle.LaunchFailureReasonCode
	}
}

func normalizeRuntimeHardeningPosture(input launcherbackend.AppliedHardeningPosture) launcherbackend.AppliedHardeningPosture {
	hardening := input.Normalized()
	if err := hardening.Validate(); err != nil {
		hardening = launcherbackend.DefaultAppliedHardeningPosture()
		hardening.DegradedReasons = append(hardening.DegradedReasons, "hardening_posture_invalid")
		hardening = hardening.Normalized()
	}
	return hardening
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

func (s *Service) emitRuntimeEvidenceAuditEvents(runID string, facts launcherbackend.RuntimeFactsSnapshot, evidence launcherbackend.RuntimeEvidenceSnapshot) error {
	if s.auditor == nil || s.store == nil {
		return fmt.Errorf("broker runtime audit path unavailable")
	}
	if err := s.emitRuntimeSessionStartedAuditEvent(runID, evidence, facts); err != nil {
		return err
	}
	if err := s.emitRuntimeSessionBoundAuditEvent(runID, evidence, facts); err != nil {
		return err
	}
	return nil
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
	return s.store.UpdateRuntimeLifecycleState(normalizedRunID, lifecycle)
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	slices.Sort(out)
	return out
}

func (s *Service) emitRuntimeSessionStartedAuditEvent(runID string, evidence launcherbackend.RuntimeEvidenceSnapshot, facts launcherbackend.RuntimeFactsSnapshot) error {
	if evidence.Session == nil || strings.TrimSpace(evidence.Session.EvidenceDigest) == "" {
		return nil
	}
	_, _, _, auditState, _ := s.store.RuntimeEvidenceState(runID)
	if auditState.LastIsolateSessionStartedDigest == evidence.Session.EvidenceDigest {
		return nil
	}
	payload := trustpolicy.IsolateSessionStartedPayload{
		SchemaID:                      trustpolicy.IsolateSessionStartedPayloadSchemaID,
		SchemaVersion:                 trustpolicy.IsolateSessionStartedPayloadSchemaVersion,
		RunID:                         evidence.Launch.RunID,
		IsolateID:                     evidence.Launch.IsolateID,
		SessionID:                     evidence.Launch.SessionID,
		BackendKind:                   evidence.Launch.BackendKind,
		IsolationAssuranceLevel:       evidence.Launch.IsolationAssuranceLevel,
		ProvisioningPosture:           evidence.Launch.ProvisioningPosture,
		LaunchContextDigest:           evidence.Launch.LaunchContextDigest,
		HandshakeTranscriptHash:       evidence.Launch.HandshakeTranscriptHash,
		LaunchReceiptDigest:           evidence.Launch.EvidenceDigest,
		RuntimeImageDescriptorDigest:  evidence.Launch.RuntimeImageDescriptorDigest,
		AppliedHardeningPostureDigest: evidence.Hardening.EvidenceDigest,
	}
	if details, err := runtimeAuditDetailsForPayload("isolate_session_started", trustpolicy.IsolateSessionStartedPayloadSchemaID, payload, evidence, facts); err != nil {
		return err
	} else if err := s.auditor.emitLauncherRuntimeEvent(s.store, "isolate_session_started", details); err != nil {
		return err
	}
	return s.store.MarkRuntimeAuditEventEmitted(runID, "isolate_session_started", evidence.Session.EvidenceDigest)
}

func (s *Service) emitRuntimeSessionBoundAuditEvent(runID string, evidence launcherbackend.RuntimeEvidenceSnapshot, facts launcherbackend.RuntimeFactsSnapshot) error {
	if evidence.Session == nil || strings.TrimSpace(evidence.Session.EvidenceDigest) == "" {
		return nil
	}
	_, _, _, auditState, _ := s.store.RuntimeEvidenceState(runID)
	if auditState.LastIsolateSessionBoundDigest == evidence.Session.EvidenceDigest {
		return nil
	}
	payload := trustpolicy.IsolateSessionBoundPayload{
		SchemaID:                      trustpolicy.IsolateSessionBoundPayloadSchemaID,
		SchemaVersion:                 trustpolicy.IsolateSessionBoundPayloadSchemaVersion,
		RunID:                         evidence.Launch.RunID,
		IsolateID:                     evidence.Launch.IsolateID,
		SessionID:                     evidence.Launch.SessionID,
		BackendKind:                   evidence.Launch.BackendKind,
		IsolationAssuranceLevel:       evidence.Launch.IsolationAssuranceLevel,
		ProvisioningPosture:           evidence.Launch.ProvisioningPosture,
		LaunchContextDigest:           evidence.Launch.LaunchContextDigest,
		HandshakeTranscriptHash:       evidence.Launch.HandshakeTranscriptHash,
		SessionBindingDigest:          evidence.Session.EvidenceDigest,
		RuntimeImageDescriptorDigest:  evidence.Launch.RuntimeImageDescriptorDigest,
		AppliedHardeningPostureDigest: evidence.Hardening.EvidenceDigest,
	}
	if details, err := runtimeAuditDetailsForPayload("isolate_session_bound", trustpolicy.IsolateSessionBoundPayloadSchemaID, payload, evidence, facts); err != nil {
		return err
	} else if err := s.auditor.emitLauncherRuntimeEvent(s.store, "isolate_session_bound", details); err != nil {
		return err
	}
	return s.store.MarkRuntimeAuditEventEmitted(runID, "isolate_session_bound", evidence.Session.EvidenceDigest)
}

func ParseDataClass(value string) (artifacts.DataClass, error) {
	class := artifacts.DataClass(value)
	switch class {
	case artifacts.DataClassSpecText,
		artifacts.DataClassUnapprovedFileExcerpts,
		artifacts.DataClassApprovedFileExcerpts,
		artifacts.DataClassDiffs,
		artifacts.DataClassBuildLogs,
		artifacts.DataClassAuditEvents,
		artifacts.DataClassAuditVerificationReport,
		artifacts.DataClassWebQuery,
		artifacts.DataClassWebCitations:
		return class, nil
	default:
		return "", fmt.Errorf("unsupported data class %q", value)
	}
}
