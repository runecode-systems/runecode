package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) emitRuntimeEvidenceAuditEvents(runID string, facts launcherbackend.RuntimeFactsSnapshot, evidence launcherbackend.RuntimeEvidenceSnapshot) error {
	if s.auditor == nil || s.store == nil {
		return fmt.Errorf("broker runtime audit path unavailable")
	}
	if err := s.emitRuntimeLaunchAdmissionAuditEvent(runID, evidence, facts); err != nil {
		return err
	}
	if err := s.emitRuntimeLaunchDeniedAuditEvent(runID, evidence, facts); err != nil {
		return err
	}
	if err := s.emitRuntimeSessionStartedAuditEvent(runID, evidence, facts); err != nil {
		return err
	}
	if err := s.emitRuntimeSessionBoundAuditEvent(runID, evidence, facts); err != nil {
		return err
	}
	return nil
}

func (s *Service) emitRuntimeLaunchAdmissionAuditEvent(runID string, evidence launcherbackend.RuntimeEvidenceSnapshot, facts launcherbackend.RuntimeFactsSnapshot) error {
	if strings.TrimSpace(evidence.Launch.EvidenceDigest) == "" || strings.TrimSpace(facts.LaunchReceipt.LaunchFailureReasonCode) != "" {
		return nil
	}
	_, _, _, auditState, _ := s.store.RuntimeEvidenceState(runID)
	if auditState.LastRuntimeLaunchAdmissionDigest == evidence.Launch.EvidenceDigest {
		return nil
	}
	payload := map[string]string{
		"schema_id":                       "runecode.protocol.v0.RuntimeLaunchAdmissionPayload",
		"schema_version":                  "0.1.0",
		"run_id":                          evidence.Launch.RunID,
		"backend_kind":                    evidence.Launch.BackendKind,
		"runtime_launch_digest":           evidence.Launch.EvidenceDigest,
		"runtime_image_descriptor_digest": evidence.Launch.RuntimeImageDescriptorDigest,
	}
	if toolchainDigest := strings.TrimSpace(evidence.Launch.RuntimeToolchainDescriptorDigest); toolchainDigest != "" {
		payload["runtime_toolchain_descriptor_digest"] = toolchainDigest
	}
	if details, err := runtimeAuditDetailsForPayload("runtime_launch_admission", "runecode.protocol.v0.RuntimeLaunchAdmissionPayload", payload, evidence, facts); err != nil {
		return err
	} else if err := s.auditor.emitRuntimeLaunchAdmissionEvent(s.store, details); err != nil {
		return err
	}
	return s.store.MarkRuntimeAuditEventEmitted(runID, "runtime_launch_admission", evidence.Launch.EvidenceDigest)
}

func (s *Service) emitRuntimeLaunchDeniedAuditEvent(runID string, evidence launcherbackend.RuntimeEvidenceSnapshot, facts launcherbackend.RuntimeFactsSnapshot) error {
	reason := strings.TrimSpace(facts.LaunchReceipt.LaunchFailureReasonCode)
	if reason == "" || strings.TrimSpace(evidence.Launch.EvidenceDigest) == "" {
		return nil
	}
	_, _, _, auditState, _ := s.store.RuntimeEvidenceState(runID)
	if auditState.LastRuntimeLaunchDeniedDigest == evidence.Launch.EvidenceDigest {
		return nil
	}
	payload := map[string]string{
		"schema_id":                  "runecode.protocol.v0.RuntimeLaunchDeniedPayload",
		"schema_version":             "0.1.0",
		"run_id":                     evidence.Launch.RunID,
		"backend_kind":               evidence.Launch.BackendKind,
		"runtime_launch_digest":      evidence.Launch.EvidenceDigest,
		"launch_failure_reason_code": reason,
	}
	if details, err := runtimeAuditDetailsForPayload("runtime_launch_denied", "runecode.protocol.v0.RuntimeLaunchDeniedPayload", payload, evidence, facts); err != nil {
		return err
	} else if err := s.auditor.emitRuntimeLaunchDeniedEvent(s.store, details); err != nil {
		return err
	}
	return s.store.MarkRuntimeAuditEventEmitted(runID, "runtime_launch_denied", evidence.Launch.EvidenceDigest)
}

func (s *Service) emitRuntimeSessionStartedAuditEvent(runID string, evidence launcherbackend.RuntimeEvidenceSnapshot, facts launcherbackend.RuntimeFactsSnapshot) error {
	if evidence.Session == nil || strings.TrimSpace(evidence.Session.EvidenceDigest) == "" {
		return nil
	}
	_, _, _, auditState, _ := s.store.RuntimeEvidenceState(runID)
	marker := runtimeSessionAuditIdentityKey(evidence)
	if auditState.LastIsolateSessionStartedDigest == marker {
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
	return s.store.MarkRuntimeAuditEventEmitted(runID, "isolate_session_started", marker)
}

func (s *Service) emitRuntimeSessionBoundAuditEvent(runID string, evidence launcherbackend.RuntimeEvidenceSnapshot, facts launcherbackend.RuntimeFactsSnapshot) error {
	if evidence.Session == nil || strings.TrimSpace(evidence.Session.EvidenceDigest) == "" {
		return nil
	}
	_, _, _, auditState, _ := s.store.RuntimeEvidenceState(runID)
	marker := runtimeSessionAuditIdentityKey(evidence)
	if auditState.LastIsolateSessionBoundDigest == marker {
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
	return s.store.MarkRuntimeAuditEventEmitted(runID, "isolate_session_bound", marker)
}
