package launcherbackend

const (
	attestationReasonCodeReplayDetected           = "attestation_replay_detected"
	attestationReasonCodeFreshnessMaterialMissing = "attestation_freshness_material_missing"
	attestationReasonCodeFreshnessBindingMissing  = "attestation_freshness_binding_missing"
	attestationReasonCodeFreshnessStale           = "attestation_freshness_stale"
	attestationReasonCodeEvidenceRequired         = "attestation_evidence_required"
	attestationReasonCodeVerificationRequired     = "attestation_verification_required"
	attestationReasonCodeVerificationNotValid     = "attestation_verification_not_valid"
)

func buildIsolateAttestationEvidence(receipt BackendLaunchReceipt, launch LaunchRuntimeEvidence) (*IsolateAttestationEvidence, *IsolateAttestationVerificationRecord, error) {
	if !hasIsolateAttestationEvidence(receipt) {
		return nil, nil, nil
	}
	evidence := isolateAttestationEvidenceFromReceipt(receipt, launch)
	digest, err := canonicalSHA256Digest(isolateAttestationEvidenceDigestInput(*evidence), "isolate attestation evidence")
	if err != nil {
		return nil, nil, err
	}
	evidence.EvidenceDigest = digest

	verification, err := isolateAttestationVerificationFromReceipt(receipt, launch.EvidenceDigest, digest)
	if err != nil {
		return nil, nil, err
	}
	return evidence, verification, nil
}

func hasIsolateAttestationEvidence(receipt BackendLaunchReceipt) bool {
	return receipt.RunID != "" && receipt.IsolateID != "" && receipt.SessionID != "" &&
		receipt.SessionNonce != "" && receipt.HandshakeTranscriptHash != "" &&
		receipt.IsolateSessionKeyIDValue != "" && receipt.RuntimeImageDescriptorDigest != "" &&
		receipt.RuntimeImageBootProfile != "" && receipt.AttestationEvidenceSourceKind != "" &&
		receipt.AttestationMeasurementProfile != ""
}

func isolateAttestationEvidenceFromReceipt(receipt BackendLaunchReceipt, launch LaunchRuntimeEvidence) *IsolateAttestationEvidence {
	return &IsolateAttestationEvidence{
		RunID:                        receipt.RunID,
		IsolateID:                    receipt.IsolateID,
		SessionID:                    receipt.SessionID,
		SessionNonce:                 receipt.SessionNonce,
		HandshakeTranscriptHash:      receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     receipt.IsolateSessionKeyIDValue,
		LaunchRuntimeEvidenceDigest:  launch.EvidenceDigest,
		RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      receipt.RuntimeImageBootProfile,
		BootComponentDigests:         uniqueSortedStrings(receipt.BootComponentDigests),
		AttestationSourceKind:        receipt.AttestationEvidenceSourceKind,
		MeasurementProfile:           receipt.AttestationMeasurementProfile,
		FreshnessMaterial:            uniqueSortedStrings(receipt.AttestationFreshnessMaterial),
		FreshnessBindingClaims:       uniqueSortedStrings(receipt.AttestationFreshnessBindingClaims),
		EvidenceClaimsDigest:         receipt.AttestationEvidenceClaimsDigest,
	}
}

func isolateAttestationEvidenceDigestInput(evidence IsolateAttestationEvidence) isolateAttestationEvidenceDigestFields {
	return isolateAttestationEvidenceDigestFields{
		RunID:                        evidence.RunID,
		IsolateID:                    evidence.IsolateID,
		SessionID:                    evidence.SessionID,
		SessionNonce:                 evidence.SessionNonce,
		HandshakeTranscriptHash:      evidence.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     evidence.IsolateSessionKeyIDValue,
		LaunchRuntimeEvidenceDigest:  evidence.LaunchRuntimeEvidenceDigest,
		RuntimeImageDescriptorDigest: evidence.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      evidence.RuntimeImageBootProfile,
		BootComponentDigests:         evidence.BootComponentDigests,
		AttestationSourceKind:        evidence.AttestationSourceKind,
		MeasurementProfile:           evidence.MeasurementProfile,
		FreshnessMaterial:            evidence.FreshnessMaterial,
		FreshnessBindingClaims:       evidence.FreshnessBindingClaims,
		EvidenceClaimsDigest:         evidence.EvidenceClaimsDigest,
	}
}

func isolateAttestationVerificationFromReceipt(receipt BackendLaunchReceipt, launchEvidenceDigest, attestationEvidenceDigest string) (*IsolateAttestationVerificationRecord, error) {
	if receipt.AttestationVerifierPolicyID == "" && receipt.AttestationVerifierPolicyDigest == "" && receipt.AttestationVerificationResult == "" {
		return nil, nil
	}
	replayIdentityDigest, err := canonicalSHA256Digest(isolateAttestationReplayIdentityInput(receipt, launchEvidenceDigest, attestationEvidenceDigest), "isolate attestation replay identity")
	if err != nil {
		return nil, err
	}
	verification := &IsolateAttestationVerificationRecord{
		AttestationEvidenceDigest:       attestationEvidenceDigest,
		ReplayIdentityDigest:            replayIdentityDigest,
		VerifierPolicyID:                receipt.AttestationVerifierPolicyID,
		VerifierPolicyDigest:            receipt.AttestationVerifierPolicyDigest,
		VerificationRulesProfileVersion: receipt.AttestationVerificationRulesVersion,
		VerificationTimestamp:           receipt.AttestationVerificationTimestamp,
		VerificationResult:              receipt.AttestationVerificationResult,
		ReasonCodes:                     uniqueSortedStrings(receipt.AttestationVerificationReasonCodes),
		ReplayVerdict:                   receipt.AttestationReplayVerdict,
		DerivedMeasurementDigests:       []string{receipt.AttestationEvidenceClaimsDigest},
	}
	if verification.DerivedMeasurementDigests[0] == "" {
		verification.DerivedMeasurementDigests = nil
	}
	digest, err := canonicalSHA256Digest(isolateAttestationVerificationDigestInput(*verification), "isolate attestation verification")
	if err != nil {
		return nil, err
	}
	verification.VerificationDigest = digest
	return verification, nil
}

func isolateAttestationReplayIdentityInput(receipt BackendLaunchReceipt, launchEvidenceDigest, attestationEvidenceDigest string) isolateAttestationReplayIdentityFields {
	return isolateAttestationReplayIdentityFields{
		RunID:                     receipt.RunID,
		IsolateID:                 receipt.IsolateID,
		SessionID:                 receipt.SessionID,
		SessionNonce:              receipt.SessionNonce,
		HandshakeTranscriptHash:   receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:  receipt.IsolateSessionKeyIDValue,
		LaunchEvidenceDigest:      launchEvidenceDigest,
		AttestationEvidenceDigest: attestationEvidenceDigest,
		MeasurementProfile:        receipt.AttestationMeasurementProfile,
	}
}

func isolateAttestationVerificationDigestInput(verification IsolateAttestationVerificationRecord) isolateAttestationVerificationRecordDigestFields {
	return isolateAttestationVerificationRecordDigestFields{
		AttestationEvidenceDigest:       verification.AttestationEvidenceDigest,
		ReplayIdentityDigest:            verification.ReplayIdentityDigest,
		VerifierPolicyID:                verification.VerifierPolicyID,
		VerifierPolicyDigest:            verification.VerifierPolicyDigest,
		VerificationRulesProfileVersion: verification.VerificationRulesProfileVersion,
		VerificationTimestamp:           verification.VerificationTimestamp,
		VerificationResult:              verification.VerificationResult,
		ReasonCodes:                     verification.ReasonCodes,
		ReplayVerdict:                   verification.ReplayVerdict,
		DerivedMeasurementDigests:       verification.DerivedMeasurementDigests,
	}
}

func applyAttestationFailClosedPolicy(receipt BackendLaunchReceipt, evidence *RuntimeEvidenceSnapshot) {
	if evidence == nil || receipt.ProvisioningPosture != ProvisioningPostureAttested {
		return
	}
	if evidence.Attestation == nil {
		evidence.AttestationVerification = invalidAttestationVerificationForRequiredEvidence("", []string{attestationReasonCodeEvidenceRequired}, AttestationReplayVerdictUnknown)
		return
	}
	if evidence.AttestationVerification == nil {
		evidence.AttestationVerification = invalidAttestationVerificationForRequiredEvidence(evidence.Attestation.EvidenceDigest, []string{attestationReasonCodeVerificationRequired}, AttestationReplayVerdictUnknown)
		return
	}
	reasonCodes := attestationReasonCodesForEvidence(evidence)
	if len(reasonCodes) == 0 {
		return
	}
	evidence.AttestationVerification.VerificationResult = AttestationVerificationResultInvalid
	evidence.AttestationVerification.ReasonCodes = reasonCodes
}

func attestationReasonCodesForEvidence(evidence *RuntimeEvidenceSnapshot) []string {
	reasonCodes := make([]string, 0, 4)
	reasonCodes = append(reasonCodes, evidence.AttestationVerification.ReasonCodes...)
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultValid {
		reasonCodes = append(reasonCodes, attestationReasonCodeVerificationNotValid)
	}
	if evidence.AttestationVerification.ReplayVerdict == AttestationReplayVerdictReplay {
		reasonCodes = append(reasonCodes, attestationReasonCodeReplayDetected)
	}
	if len(evidence.Attestation.FreshnessMaterial) == 0 {
		reasonCodes = append(reasonCodes, attestationReasonCodeFreshnessMaterialMissing)
	}
	if len(evidence.Attestation.FreshnessBindingClaims) == 0 {
		reasonCodes = append(reasonCodes, attestationReasonCodeFreshnessBindingMissing)
	}
	if containsAnyReasonCode(reasonCodes, attestationReasonCodeFreshnessStale, "freshness_stale", "attestation_stale") {
		reasonCodes = append(reasonCodes, attestationReasonCodeFreshnessStale)
	}
	return uniqueSortedStrings(reasonCodes)
}

func invalidAttestationVerificationForRequiredEvidence(attestationEvidenceDigest string, reasonCodes []string, replayVerdict string) *IsolateAttestationVerificationRecord {
	return &IsolateAttestationVerificationRecord{
		AttestationEvidenceDigest: attestationEvidenceDigest,
		VerificationResult:        AttestationVerificationResultInvalid,
		ReasonCodes:               uniqueSortedStrings(reasonCodes),
		ReplayVerdict:             replayVerdict,
	}
}

func containsAnyReasonCode(values []string, expected ...string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		for _, match := range expected {
			if value == match {
				return true
			}
		}
	}
	return false
}

func finalizeAttestationVerificationRecord(evidence *RuntimeEvidenceSnapshot) error {
	if evidence == nil || evidence.AttestationVerification == nil {
		return nil
	}
	verification := evidence.AttestationVerification
	verification.ReasonCodes = uniqueSortedStrings(verification.ReasonCodes)
	if verification.AttestationEvidenceDigest == "" && evidence.Attestation != nil {
		verification.AttestationEvidenceDigest = evidence.Attestation.EvidenceDigest
	}
	if verification.ReplayIdentityDigest == "" && evidence.Attestation != nil {
		replayIdentityDigest, err := canonicalSHA256Digest(isolateAttestationReplayIdentityFields{
			RunID:                     evidence.Attestation.RunID,
			IsolateID:                 evidence.Attestation.IsolateID,
			SessionID:                 evidence.Attestation.SessionID,
			SessionNonce:              evidence.Attestation.SessionNonce,
			HandshakeTranscriptHash:   evidence.Attestation.HandshakeTranscriptHash,
			IsolateSessionKeyIDValue:  evidence.Attestation.IsolateSessionKeyIDValue,
			LaunchEvidenceDigest:      evidence.Attestation.LaunchRuntimeEvidenceDigest,
			AttestationEvidenceDigest: evidence.Attestation.EvidenceDigest,
			MeasurementProfile:        evidence.Attestation.MeasurementProfile,
		}, "isolate attestation replay identity")
		if err != nil {
			return err
		}
		verification.ReplayIdentityDigest = replayIdentityDigest
	}
	digest, err := canonicalSHA256Digest(isolateAttestationVerificationDigestInput(*verification), "isolate attestation verification")
	if err != nil {
		return err
	}
	verification.VerificationDigest = digest
	return nil
}
