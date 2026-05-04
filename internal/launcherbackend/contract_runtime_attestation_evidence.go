package launcherbackend

import "strings"

const (
	attestationReasonCodeReplayDetected             = "attestation_replay_detected"
	attestationReasonCodeSourceKindInvalid          = "attestation_source_kind_invalid"
	attestationReasonCodeIdentityBindingInvalid     = "attestation_identity_binding_invalid"
	attestationReasonCodeMeasurementDigestInvalid   = "attestation_measurement_digest_invalid"
	attestationReasonCodeSessionValidationRequired  = "attestation_session_validation_required"
	attestationReasonCodePostHandshakeInputRequired = "attestation_post_handshake_input_required"
	attestationReasonCodeRuntimeEvidenceRequired    = "attestation_runtime_evidence_required"
	attestationReasonCodeFreshnessMaterialMissing   = "attestation_freshness_material_missing"
	attestationReasonCodeFreshnessBindingMissing    = "attestation_freshness_binding_missing"
	attestationReasonCodeFreshnessStale             = "attestation_freshness_stale"
	attestationReasonCodeEvidenceRequired           = "attestation_evidence_required"
	attestationReasonCodeVerificationRequired       = "attestation_verification_required"
	attestationReasonCodeVerificationNotValid       = "attestation_verification_not_valid"
	trustedRuntimeAttestationVerifierPolicyID       = "runtime_asset_admission_identity"
	trustedRuntimeAttestationRulesVersion           = "trusted-runtime-v1"
)

func buildIsolateAttestationEvidence(receipt BackendLaunchReceipt, postHandshake *PostHandshakeRuntimeAttestationInput, launch LaunchRuntimeEvidence) (*IsolateAttestationEvidence, *IsolateAttestationVerificationRecord, error) {
	attestationInput := derivePostHandshakeRuntimeAttestationInput(postHandshake)
	if !hasIsolateAttestationEvidence(attestationInput) {
		return nil, nil, nil
	}
	evidence := isolateAttestationEvidenceFromPostHandshakeInput(*attestationInput, launch)
	digest, err := canonicalSHA256Digest(isolateAttestationEvidenceDigestInput(*evidence), "isolate attestation evidence")
	if err != nil {
		return nil, nil, err
	}
	evidence.EvidenceDigest = digest

	verification, err := isolateAttestationVerificationFromPostHandshakeInput(*attestationInput, launch.EvidenceDigest, digest)
	if err != nil {
		return nil, nil, err
	}
	return evidence, verification, nil
}

func withTrustedAttestationVerificationDefaults(input PostHandshakeRuntimeAttestationInput) PostHandshakeRuntimeAttestationInput {
	if strings.TrimSpace(input.VerifierPolicyID) == "" {
		input.VerifierPolicyID = trustedRuntimeAttestationVerifierPolicyID
	}
	if strings.TrimSpace(input.VerifierPolicyDigest) == "" {
		input.VerifierPolicyDigest = trustedVerificationPolicyDigest(input)
	}
	if strings.TrimSpace(input.VerificationRulesProfileVersion) == "" {
		input.VerificationRulesProfileVersion = trustedRuntimeAttestationRulesVersion
	}
	if strings.TrimSpace(input.VerificationResult) == "" || input.VerificationResult == AttestationVerificationResultUnknown {
		input.VerificationResult = AttestationVerificationResultValid
	}
	if strings.TrimSpace(input.ReplayVerdict) == "" || input.ReplayVerdict == AttestationReplayVerdictUnknown {
		input.ReplayVerdict = AttestationReplayVerdictOriginal
	}
	return input
}

func derivePostHandshakeRuntimeAttestationInput(postHandshake *PostHandshakeRuntimeAttestationInput) *PostHandshakeRuntimeAttestationInput {
	return NormalizePostHandshakeRuntimeAttestationInput(postHandshake)
}

func applyAttestationFailClosedPolicy(receipt BackendLaunchReceipt, postHandshake *PostHandshakeRuntimeAttestationInput, evidence *RuntimeEvidenceSnapshot) {
	if evidence == nil || !requiresAttestationVerification(receipt, postHandshake) {
		return
	}
	if failed := attestationFailClosedVerification(receipt, postHandshake, evidence); failed != nil {
		evidence.AttestationVerification = failed
		return
	}
	reasonCodes := attestationReasonCodesForEvidence(evidence)
	if len(reasonCodes) == 0 {
		promoteAttestedProvisioningPosture(evidence)
		return
	}
	evidence.AttestationVerification.VerificationResult = AttestationVerificationResultInvalid
	evidence.AttestationVerification.ReasonCodes = reasonCodes
}

func attestationFailClosedVerification(receipt BackendLaunchReceipt, postHandshake *PostHandshakeRuntimeAttestationInput, evidence *RuntimeEvidenceSnapshot) *IsolateAttestationVerificationRecord {
	if !hasValidatedSessionForAttestation(receipt, evidence) {
		return invalidAttestationVerificationForRequiredEvidence("", []string{attestationReasonCodeSessionValidationRequired}, AttestationReplayVerdictUnknown)
	}
	if verification := missingPostHandshakeVerification(postHandshake, evidence); verification != nil {
		return verification
	}
	if verification := missingRuntimeEvidenceVerification(postHandshake, evidence); verification != nil {
		return verification
	}
	if verification := missingAttestationEvidenceVerification(evidence); verification != nil {
		return verification
	}
	if evidence.AttestationVerification == nil {
		return invalidAttestationVerificationForRequiredEvidence(evidence.Attestation.EvidenceDigest, []string{attestationReasonCodeVerificationRequired}, AttestationReplayVerdictUnknown)
	}
	return nil
}

func missingPostHandshakeVerification(postHandshake *PostHandshakeRuntimeAttestationInput, evidence *RuntimeEvidenceSnapshot) *IsolateAttestationVerificationRecord {
	if NormalizePostHandshakeRuntimeAttestationInput(postHandshake) != nil {
		return nil
	}
	return invalidAttestationVerificationForRequiredEvidence("", requiredEvidenceReasonCodes(attestationReasonCodePostHandshakeInputRequired, evidence), AttestationReplayVerdictUnknown)
}

func missingRuntimeEvidenceVerification(postHandshake *PostHandshakeRuntimeAttestationInput, evidence *RuntimeEvidenceSnapshot) *IsolateAttestationVerificationRecord {
	if postHandshake != nil && postHandshake.RuntimeEvidenceCollected {
		return nil
	}
	return invalidAttestationVerificationForRequiredEvidence("", requiredEvidenceReasonCodes(attestationReasonCodeRuntimeEvidenceRequired, evidence), AttestationReplayVerdictUnknown)
}

func missingAttestationEvidenceVerification(evidence *RuntimeEvidenceSnapshot) *IsolateAttestationVerificationRecord {
	if evidence != nil && evidence.Attestation != nil {
		return nil
	}
	return invalidAttestationVerificationForRequiredEvidence("", []string{attestationReasonCodeEvidenceRequired}, AttestationReplayVerdictUnknown)
}

func requiredEvidenceReasonCodes(requiredReason string, evidence *RuntimeEvidenceSnapshot) []string {
	reasonCodes := []string{requiredReason}
	if evidence == nil || evidence.Attestation == nil {
		reasonCodes = append(reasonCodes, attestationReasonCodeEvidenceRequired)
	}
	return reasonCodes
}

func requiresAttestationVerification(receipt BackendLaunchReceipt, postHandshake *PostHandshakeRuntimeAttestationInput) bool {
	normalized := receipt.Normalized()
	if normalized.ProvisioningPosture == ProvisioningPostureAttested {
		return true
	}
	if NormalizePostHandshakeRuntimeAttestationInput(postHandshake) != nil {
		return true
	}
	if normalized.AttestationEvidenceSourceKind != AttestationSourceKindUnknown || normalized.AttestationMeasurementProfile != "" {
		return true
	}
	if normalized.AttestationVerificationResult != AttestationVerificationResultUnknown {
		return true
	}
	if strings.TrimSpace(normalized.AttestationVerifierPolicyID) != "" || strings.TrimSpace(normalized.AttestationVerifierPolicyDigest) != "" {
		return true
	}
	return false
}

func hasValidatedSessionForAttestation(receipt BackendLaunchReceipt, evidence *RuntimeEvidenceSnapshot) bool {
	if evidence == nil || evidence.Session == nil {
		return false
	}
	security := receipt.SessionSecurity
	if security == nil {
		return false
	}
	if !security.MutuallyAuthenticated || !security.Encrypted || !security.ProofOfPossessionVerified {
		return false
	}
	return evidence.Session.LaunchContextDigest != "" && evidence.Session.HandshakeTranscriptHash != "" && evidence.Session.IsolateSessionKeyIDValue != ""
}

func promoteAttestedProvisioningPosture(evidence *RuntimeEvidenceSnapshot) {
	if evidence == nil {
		return
	}
	evidence.Launch.ProvisioningPosture = ProvisioningPostureAttested
	if evidence.Session != nil {
		evidence.Session.ProvisioningPosture = ProvisioningPostureAttested
	}
}

func attestationReasonCodesForEvidence(evidence *RuntimeEvidenceSnapshot) []string {
	reasonCodes := make([]string, 0, 4)
	reasonCodes = append(reasonCodes, evidence.AttestationVerification.ReasonCodes...)
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultValid {
		reasonCodes = append(reasonCodes, attestationReasonCodeVerificationNotValid)
	}
	if evidence.AttestationVerification.ReplayVerdict != AttestationReplayVerdictOriginal {
		reasonCodes = append(reasonCodes, attestationReasonCodeVerificationNotValid)
	}
	if evidence.AttestationVerification.ReplayVerdict == AttestationReplayVerdictReplay {
		reasonCodes = append(reasonCodes, attestationReasonCodeReplayDetected)
	}
	if !measurementProfileAcceptsSourceKind(evidence.Attestation.MeasurementProfile, evidence.Attestation.AttestationSourceKind) {
		reasonCodes = append(reasonCodes, attestationReasonCodeSourceKindInvalid)
	}
	if !attestationIdentityMatchesLaunchEvidence(evidence.Launch, evidence.Attestation) {
		reasonCodes = append(reasonCodes, attestationReasonCodeIdentityBindingInvalid)
	}
	if !attestationMeasurementIdentityMatchesEvidence(evidence.Attestation) {
		reasonCodes = append(reasonCodes, attestationReasonCodeMeasurementDigestInvalid)
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

func ReconcileRuntimeEvidenceAttestation(receipt BackendLaunchReceipt, postHandshake *PostHandshakeRuntimeAttestationInput, evidence *RuntimeEvidenceSnapshot) error {
	if evidence == nil {
		return nil
	}
	applyAttestationFailClosedPolicy(receipt.Normalized(), NormalizePostHandshakeRuntimeAttestationInput(postHandshake), evidence)
	return finalizeAttestationVerificationRecord(evidence)
}
