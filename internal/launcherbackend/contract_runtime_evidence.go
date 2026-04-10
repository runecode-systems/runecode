package launcherbackend

import "fmt"

func SplitRuntimeFactsEvidenceAndLifecycle(facts RuntimeFactsSnapshot) (RuntimeEvidenceSnapshot, RuntimeLifecycleState, error) {
	receipt := facts.LaunchReceipt.Normalized()
	hardening := facts.HardeningPosture.Normalized()
	if err := hardening.Validate(); err != nil {
		return RuntimeEvidenceSnapshot{}, RuntimeLifecycleState{}, fmt.Errorf("hardening_posture: %w", err)
	}
	evidence, err := buildRuntimeEvidenceSnapshot(receipt, hardening, facts.TerminalReport)
	if err != nil {
		return RuntimeEvidenceSnapshot{}, RuntimeLifecycleState{}, err
	}
	state := RuntimeLifecycleState{
		BackendLifecycle:            cloneLifecycle(receipt.Lifecycle),
		ProvisioningPosture:         receipt.ProvisioningPosture,
		ProvisioningPostureDegraded: receipt.ProvisioningPostureDegraded,
		ProvisioningDegradedReasons: uniqueSortedStrings(receipt.ProvisioningDegradedReasons),
		LaunchFailureReasonCode:     receipt.LaunchFailureReasonCode,
	}
	return evidence, state, nil
}

func buildRuntimeEvidenceSnapshot(receipt BackendLaunchReceipt, hardening AppliedHardeningPosture, terminal *BackendTerminalReport) (RuntimeEvidenceSnapshot, error) {
	launch, err := buildLaunchRuntimeEvidence(receipt)
	if err != nil {
		return RuntimeEvidenceSnapshot{}, err
	}
	hardeningEvidence, err := buildHardeningRuntimeEvidence(hardening)
	if err != nil {
		return RuntimeEvidenceSnapshot{}, err
	}
	bundle := RuntimeEvidenceSnapshot{Launch: launch, Hardening: hardeningEvidence}
	session, err := buildSessionRuntimeEvidence(receipt)
	if err != nil {
		return RuntimeEvidenceSnapshot{}, err
	}
	if session != nil {
		bundle.Session = session
	}
	if terminal != nil {
		normalized := terminal.Normalized()
		if err := normalized.Validate(); err != nil {
			return RuntimeEvidenceSnapshot{}, fmt.Errorf("terminal_report: %w", err)
		}
		digest, err := canonicalSHA256Digest(normalized, "terminal runtime evidence")
		if err != nil {
			return RuntimeEvidenceSnapshot{}, err
		}
		bundle.Terminal = &TerminalRuntimeEvidence{Report: normalized, EvidenceDigest: digest}
	}
	return bundle, nil
}

func buildLaunchRuntimeEvidence(receipt BackendLaunchReceipt) (LaunchRuntimeEvidence, error) {
	evidence := launchRuntimeEvidenceFromReceipt(receipt)
	digest, err := canonicalSHA256Digest(launchRuntimeEvidenceDigestInput(evidence), "launch runtime evidence")
	if err != nil {
		return LaunchRuntimeEvidence{}, err
	}
	evidence.EvidenceDigest = digest
	return evidence, nil
}

func launchRuntimeEvidenceFromReceipt(receipt BackendLaunchReceipt) LaunchRuntimeEvidence {
	return LaunchRuntimeEvidence{
		RunID:                        receipt.RunID,
		StageID:                      receipt.StageID,
		RoleInstanceID:               receipt.RoleInstanceID,
		RoleFamily:                   receipt.RoleFamily,
		RoleKind:                     receipt.RoleKind,
		BackendKind:                  receipt.BackendKind,
		IsolationAssuranceLevel:      receipt.IsolationAssuranceLevel,
		ProvisioningPosture:          receipt.ProvisioningPosture,
		IsolateID:                    receipt.IsolateID,
		SessionID:                    receipt.SessionID,
		SessionNonce:                 receipt.SessionNonce,
		LaunchContextDigest:          receipt.LaunchContextDigest,
		HandshakeTranscriptHash:      receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     receipt.IsolateSessionKeyIDValue,
		HostingNodeID:                receipt.HostingNodeID,
		TransportKind:                receipt.TransportKind,
		HypervisorImplementation:     receipt.HypervisorImplementation,
		AccelerationKind:             receipt.AccelerationKind,
		QEMUProvenance:               cloneQEMUProvenance(receipt.QEMUProvenance),
		RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
		RuntimeImageSignerRef:        receipt.RuntimeImageSignerRef,
		RuntimeImageSignatureDigest:  receipt.RuntimeImageSignatureDigest,
		BootComponentDigestByName:    cloneStringMap(receipt.BootComponentDigestByName),
		BootComponentDigests:         uniqueSortedStrings(receipt.BootComponentDigests),
		AttachmentPlanSummary:        cloneAttachmentPlanSummary(receipt.AttachmentPlanSummary),
		WorkspaceEncryptionPosture:   cloneWorkspaceEncryptionPosture(receipt.WorkspaceEncryptionPosture),
		CachePosture:                 cloneCachePosture(receipt.CachePosture),
		CacheEvidence:                cloneCacheEvidence(receipt.CacheEvidence),
	}
}

func launchRuntimeEvidenceDigestInput(evidence LaunchRuntimeEvidence) launchRuntimeEvidenceDigestFields {
	return launchRuntimeEvidenceDigestFields{
		RunID:                        evidence.RunID,
		StageID:                      evidence.StageID,
		RoleInstanceID:               evidence.RoleInstanceID,
		RoleFamily:                   evidence.RoleFamily,
		RoleKind:                     evidence.RoleKind,
		BackendKind:                  evidence.BackendKind,
		IsolationAssuranceLevel:      evidence.IsolationAssuranceLevel,
		ProvisioningPosture:          evidence.ProvisioningPosture,
		IsolateID:                    evidence.IsolateID,
		SessionID:                    evidence.SessionID,
		SessionNonce:                 evidence.SessionNonce,
		LaunchContextDigest:          evidence.LaunchContextDigest,
		HandshakeTranscriptHash:      evidence.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     evidence.IsolateSessionKeyIDValue,
		HostingNodeID:                evidence.HostingNodeID,
		TransportKind:                evidence.TransportKind,
		HypervisorImplementation:     evidence.HypervisorImplementation,
		AccelerationKind:             evidence.AccelerationKind,
		QEMUProvenance:               evidence.QEMUProvenance,
		RuntimeImageDescriptorDigest: evidence.RuntimeImageDescriptorDigest,
		RuntimeImageSignerRef:        evidence.RuntimeImageSignerRef,
		RuntimeImageSignatureDigest:  evidence.RuntimeImageSignatureDigest,
		BootComponentDigestByName:    evidence.BootComponentDigestByName,
		BootComponentDigests:         evidence.BootComponentDigests,
		AttachmentPlanSummary:        evidence.AttachmentPlanSummary,
		WorkspaceEncryptionPosture:   evidence.WorkspaceEncryptionPosture,
		CachePosture:                 evidence.CachePosture,
		CacheEvidence:                evidence.CacheEvidence,
	}
}

func buildSessionRuntimeEvidence(receipt BackendLaunchReceipt) (*SessionRuntimeEvidence, error) {
	if !hasSessionRuntimeEvidence(receipt) {
		return nil, nil
	}
	evidence := sessionRuntimeEvidenceFromReceipt(receipt)
	digest, err := canonicalSHA256Digest(sessionRuntimeEvidenceDigestInput(*evidence), "session runtime evidence")
	if err != nil {
		return nil, err
	}
	evidence.EvidenceDigest = digest
	return evidence, nil
}

func hasSessionRuntimeEvidence(receipt BackendLaunchReceipt) bool {
	return receipt.RunID != "" && receipt.IsolateID != "" && receipt.SessionID != "" && receipt.SessionNonce != "" &&
		receipt.HandshakeTranscriptHash != "" && receipt.IsolateSessionKeyIDValue != ""
}

func sessionRuntimeEvidenceFromReceipt(receipt BackendLaunchReceipt) *SessionRuntimeEvidence {
	return &SessionRuntimeEvidence{
		RunID:                    receipt.RunID,
		IsolateID:                receipt.IsolateID,
		SessionID:                receipt.SessionID,
		SessionNonce:             receipt.SessionNonce,
		LaunchContextDigest:      receipt.LaunchContextDigest,
		HandshakeTranscriptHash:  receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue: receipt.IsolateSessionKeyIDValue,
		ProvisioningPosture:      receipt.ProvisioningPosture,
		SessionSecurity:          cloneSessionSecurityPosture(receipt.SessionSecurity),
	}
}

func sessionRuntimeEvidenceDigestInput(evidence SessionRuntimeEvidence) sessionRuntimeEvidenceDigestFields {
	return sessionRuntimeEvidenceDigestFields{
		RunID:                    evidence.RunID,
		IsolateID:                evidence.IsolateID,
		SessionID:                evidence.SessionID,
		SessionNonce:             evidence.SessionNonce,
		LaunchContextDigest:      evidence.LaunchContextDigest,
		HandshakeTranscriptHash:  evidence.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue: evidence.IsolateSessionKeyIDValue,
		ProvisioningPosture:      evidence.ProvisioningPosture,
		SessionSecurity:          evidence.SessionSecurity,
	}
}

func buildHardeningRuntimeEvidence(posture AppliedHardeningPosture) (HardeningRuntimeEvidence, error) {
	digest, err := canonicalSHA256Digest(posture, "hardening runtime evidence")
	if err != nil {
		return HardeningRuntimeEvidence{}, err
	}
	return HardeningRuntimeEvidence{Posture: posture, EvidenceDigest: digest}, nil
}
