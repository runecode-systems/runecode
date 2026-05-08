package launcherdaemon

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func populateRuntimeSessionBinding(receipt *launcherbackend.BackendLaunchReceipt, binding runtimeSessionBinding) {
	if receipt == nil {
		return
	}
	receipt.ProvisioningPosture = launcherbackend.ProvisioningPostureTOFU
	receipt.IsolateID = binding.IsolateID
	receipt.SessionID = binding.SessionID
	receipt.SessionNonce = binding.SessionNonce
}

func buildPostHandshakeAttestationProgress(receipt launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord) (*launcherbackend.PostHandshakeRuntimeAttestationInput, error) {
	return buildPostHandshakeAttestationProgressFromMaterial(receipt, admission, nil)
}

func buildPostHandshakeAttestationProgressFromMaterial(receipt launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord, material *launcherbackend.RuntimePostHandshakeMaterial) (*launcherbackend.PostHandshakeRuntimeAttestationInput, error) {
	if receipt.RunID == "" {
		return nil, fmt.Errorf("receipt is required")
	}
	input, err := collectPostHandshakeRuntimeAttestationInput(&receipt, admission, material)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func recordValidatedSecureSession(receipt *launcherbackend.BackendLaunchReceipt, summary launcherbackend.SecureSessionSummary, launchContextDigest string) error {
	if receipt == nil {
		return fmt.Errorf("receipt is required")
	}
	receipt.IsolateID = summary.BindingRecord.IsolateID
	receipt.SessionID = summary.BindingRecord.SessionID
	receipt.SessionNonce = summary.BindingRecord.SessionNonce
	receipt.LaunchContextDigest = launchContextDigest
	receipt.HandshakeTranscriptHash = summary.BindingRecord.HandshakeTranscriptHash
	receipt.IsolateSessionKeyIDValue = summary.BindingRecord.IsolateKeyIDValue
	receipt.SessionSecurity = &summary.SecurityPosture
	receipt.ProvisioningPosture = summary.BindingRecord.ProvisioningMode
	receipt.AttestationVerificationResult = launcherbackend.AttestationVerificationResultUnknown
	receipt.AttestationReplayVerdict = launcherbackend.AttestationReplayVerdictUnknown
	receipt.AttestationVerificationReasonCodes = nil
	receipt.AttestationVerificationTimestamp = ""
	return nil
}

func validateSecureSessionAndBuildSummary(receipt launcherbackend.BackendLaunchReceipt, secureSession *launcherbackend.RuntimeSecureSessionMaterial) (launcherbackend.SecureSessionSummary, string, error) {
	if secureSession == nil {
		return launcherbackend.SecureSessionSummary{}, "", fmt.Errorf("runtime secure-session material is required")
	}
	if receipt.RunID == "" || receipt.IsolateID == "" || receipt.SessionID == "" || receipt.SessionNonce == "" {
		return launcherbackend.SecureSessionSummary{}, "", fmt.Errorf("session binding is required before secure session validation")
	}
	binding, err := launcherbackend.ValidateSessionHandshake(secureSession.LaunchContext, secureSession.HostHello, secureSession.IsolateHello, secureSession.SessionReady, nil)
	if err != nil {
		return launcherbackend.SecureSessionSummary{}, "", fmt.Errorf("secure session validation failed: %w", err)
	}
	if err := validateSecureSessionSummaryBinding(receipt, binding); err != nil {
		return launcherbackend.SecureSessionSummary{}, "", err
	}
	summary, err := launcherbackend.BuildSecureSessionSummary(secureSession.HostHello, secureSession.IsolateHello, secureSession.SessionReady, binding)
	if err != nil {
		return launcherbackend.SecureSessionSummary{}, "", fmt.Errorf("secure session summary failed: %w", err)
	}
	return summary, secureSession.LaunchContext.LaunchContextDigest, nil
}

func validateSecureSessionSummaryBinding(receipt launcherbackend.BackendLaunchReceipt, binding launcherbackend.SessionBindingRecord) error {
	if binding.RunID != receipt.RunID || binding.IsolateID != receipt.IsolateID || binding.SessionID != receipt.SessionID || binding.SessionNonce != receipt.SessionNonce {
		return fmt.Errorf("secure session material must bind to launch receipt session tuple")
	}
	return nil
}

func secureSessionTransportKind(value string) string {
	switch value {
	case launcherbackend.TransportKindVSock, launcherbackend.TransportKindVirtioSerial:
		return value
	default:
		return launcherbackend.TransportKindVSock
	}
}

func validateRuntimeReportedAttestationBinding(receipt *launcherbackend.BackendLaunchReceipt, input *launcherbackend.PostHandshakeRuntimeAttestationInput) error {
	if receipt == nil || input == nil {
		return nil
	}
	if err := validateRuntimeReportedRequiredFields(input); err != nil {
		return err
	}
	if err := validateRuntimeReportedSessionTuple(receipt, input); err != nil {
		return err
	}
	return validateRuntimeReportedIdentity(receipt, input)
}

func validateRuntimeReportedRequiredFields(input *launcherbackend.PostHandshakeRuntimeAttestationInput) error {
	if input == nil || !input.RuntimeEvidenceCollected {
		return nil
	}
	if input.RunID == "" || input.IsolateID == "" || input.SessionID == "" || input.SessionNonce == "" || input.LaunchContextDigest == "" || input.HandshakeTranscriptHash == "" || input.IsolateSessionKeyIDValue == "" {
		return fmt.Errorf("runtime-reported attestation input must include full validated session binding when runtime evidence is collected")
	}
	if input.RuntimeImageDescriptorDigest == "" || input.RuntimeImageBootProfile == "" {
		return fmt.Errorf("runtime-reported attestation input must include admitted runtime identity when runtime evidence is collected")
	}
	return nil
}

func validateRuntimeReportedSessionTuple(receipt *launcherbackend.BackendLaunchReceipt, input *launcherbackend.PostHandshakeRuntimeAttestationInput) error {
	checks := []struct {
		value string
		want  string
		msg   string
	}{
		{input.RunID, receipt.RunID, "runtime-reported attestation input must bind to launch receipt run_id"},
		{input.IsolateID, receipt.IsolateID, "runtime-reported attestation input must bind to launch receipt isolate_id"},
		{input.SessionID, receipt.SessionID, "runtime-reported attestation input must bind to launch receipt session_id"},
		{input.SessionNonce, receipt.SessionNonce, "runtime-reported attestation input must bind to launch receipt session_nonce"},
		{input.LaunchContextDigest, receipt.LaunchContextDigest, "runtime-reported attestation input must bind to launch receipt launch_context_digest"},
		{input.HandshakeTranscriptHash, receipt.HandshakeTranscriptHash, "runtime-reported attestation input must bind to launch receipt handshake_transcript_hash"},
		{input.IsolateSessionKeyIDValue, receipt.IsolateSessionKeyIDValue, "runtime-reported attestation input must bind to launch receipt isolate_session_key_id_value"},
	}
	for _, check := range checks {
		if check.value != "" && check.value != check.want {
			return fmt.Errorf("%s", check.msg)
		}
	}
	return nil
}

func validateRuntimeReportedIdentity(receipt *launcherbackend.BackendLaunchReceipt, input *launcherbackend.PostHandshakeRuntimeAttestationInput) error {
	if input.RuntimeImageDescriptorDigest != "" && input.RuntimeImageDescriptorDigest != receipt.RuntimeImageDescriptorDigest {
		return fmt.Errorf("runtime-reported attestation input must bind to admitted runtime image descriptor")
	}
	if input.RuntimeImageBootProfile != "" && input.RuntimeImageBootProfile != receipt.RuntimeImageBootProfile {
		return fmt.Errorf("runtime-reported attestation input must bind to admitted runtime image boot profile")
	}
	return nil
}

func recordPostHandshakeAttestationProgress(receipt *launcherbackend.BackendLaunchReceipt, input *launcherbackend.PostHandshakeRuntimeAttestationInput) error {
	if receipt == nil {
		return fmt.Errorf("receipt is required")
	}
	normalizedInput := launcherbackend.NormalizePostHandshakeRuntimeAttestationInput(input)
	if normalizedInput == nil {
		return fmt.Errorf("post-handshake runtime attestation input is required")
	}
	if err := validatePostHandshakeAttestationInputBinding(receipt, normalizedInput); err != nil {
		return err
	}
	receipt.BootComponentDigests = componentDigestValues(receipt.BootComponentDigestByName)
	receipt.AttestationEvidenceSourceKind = normalizedInput.AttestationSourceKind
	receipt.AttestationMeasurementProfile = normalizedInput.MeasurementProfile
	receipt.AttestationFreshnessMaterial = append([]string{}, normalizedInput.FreshnessMaterial...)
	receipt.AttestationFreshnessBindingClaims = append([]string{}, normalizedInput.FreshnessBindingClaims...)
	receipt.AttestationEvidenceClaimsDigest = normalizedInput.EvidenceClaimsDigest
	receipt.AttestationVerifierPolicyID = ""
	receipt.AttestationVerifierPolicyDigest = ""
	receipt.AttestationVerificationRulesVersion = ""
	receipt.AttestationVerificationTimestamp = ""
	receipt.AttestationVerificationResult = launcherbackend.AttestationVerificationResultUnknown
	receipt.AttestationReplayVerdict = launcherbackend.AttestationReplayVerdictUnknown
	receipt.AttestationVerificationReasonCodes = nil
	return nil
}

func validatePostHandshakeAttestationInputBinding(receipt *launcherbackend.BackendLaunchReceipt, input *launcherbackend.PostHandshakeRuntimeAttestationInput) error {
	if receipt == nil || input == nil {
		return fmt.Errorf("post-handshake runtime attestation input is required")
	}
	if input.RunID == "" || input.IsolateID == "" || input.SessionID == "" || input.SessionNonce == "" || input.LaunchContextDigest == "" || input.HandshakeTranscriptHash == "" || input.IsolateSessionKeyIDValue == "" {
		return fmt.Errorf("session binding is required before attestation")
	}
	if input.RuntimeImageDescriptorDigest == "" || input.RuntimeImageBootProfile == "" {
		return fmt.Errorf("runtime identity is required before attestation")
	}
	if input.RunID != receipt.RunID || input.IsolateID != receipt.IsolateID || input.SessionID != receipt.SessionID || input.SessionNonce != receipt.SessionNonce || input.LaunchContextDigest != receipt.LaunchContextDigest || input.HandshakeTranscriptHash != receipt.HandshakeTranscriptHash || input.IsolateSessionKeyIDValue != receipt.IsolateSessionKeyIDValue {
		return fmt.Errorf("post-handshake attestation input must bind to validated live session tuple")
	}
	if input.RuntimeImageDescriptorDigest != receipt.RuntimeImageDescriptorDigest || input.RuntimeImageBootProfile != receipt.RuntimeImageBootProfile {
		return fmt.Errorf("post-handshake attestation input must bind to admitted runtime identity")
	}
	return nil
}

func runtimePostHandshakeFactsUpdate(runID string, receipt launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord, hardening launcherbackend.AppliedHardeningPosture, material *launcherbackend.RuntimePostHandshakeMaterial) (RuntimeUpdate, error) {
	if material == nil || material.SecureSession == nil {
		return RuntimeUpdate{}, backendError(launcherbackend.BackendErrorCodeHandshakeFailed, "runtime secure-session material not provided")
	}
	summary, launchContextDigest, err := validateSecureSessionAndBuildSummary(receipt, material.SecureSession)
	if err != nil {
		return RuntimeUpdate{}, err
	}
	if err := recordValidatedSecureSession(&receipt, summary, launchContextDigest); err != nil {
		return RuntimeUpdate{}, err
	}
	postHandshake, err := buildPostHandshakeAttestationProgressFromMaterial(receipt, admission, material)
	if err != nil {
		return RuntimeUpdate{}, err
	}
	if err := recordPostHandshakeAttestationProgress(&receipt, postHandshake); err != nil {
		return RuntimeUpdate{}, err
	}
	return RuntimeUpdate{RunID: runID, Facts: &launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: receipt, PostHandshakeAttestationInput: postHandshake, HardeningPosture: hardening}}, nil
}
