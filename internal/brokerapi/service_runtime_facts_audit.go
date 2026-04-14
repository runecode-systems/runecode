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
	if err := s.emitRuntimeSessionStartedAuditEvent(runID, evidence, facts); err != nil {
		return err
	}
	if err := s.emitRuntimeSessionBoundAuditEvent(runID, evidence, facts); err != nil {
		return err
	}
	return nil
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
