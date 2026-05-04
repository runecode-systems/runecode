package launcherdaemon

import (
	"fmt"
	"time"

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
	receipt.LaunchContextDigest = binding.LaunchContextDigest
	receipt.HandshakeTranscriptHash = binding.HandshakeTranscriptHash
	receipt.IsolateSessionKeyIDValue = binding.IsolateSessionKeyIDValue
}

func upgradeReceiptAfterSecureSessionValidation(receipt *launcherbackend.BackendLaunchReceipt, spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord, now time.Time) (*launcherbackend.PostHandshakeRuntimeAttestationInput, error) {
	if receipt == nil {
		return nil, fmt.Errorf("receipt is required")
	}
	summary, launchContextDigest, err := validateSecureSessionAndBuildSummary(spec, *receipt)
	if err != nil {
		return nil, err
	}
	receipt.IsolateID = summary.BindingRecord.IsolateID
	receipt.SessionID = summary.BindingRecord.SessionID
	receipt.SessionNonce = summary.BindingRecord.SessionNonce
	receipt.LaunchContextDigest = launchContextDigest
	receipt.HandshakeTranscriptHash = summary.BindingRecord.HandshakeTranscriptHash
	receipt.IsolateSessionKeyIDValue = summary.BindingRecord.IsolateKeyIDValue
	receipt.SessionSecurity = &summary.SecurityPosture
	receipt.ProvisioningPosture = summary.BindingRecord.ProvisioningMode
	attestationInput, err := collectPostHandshakeRuntimeAttestationInput(receipt, admission)
	if err != nil {
		return nil, err
	}
	if err := applyTrustedRuntimeAttestation(receipt, attestationInput, now); err != nil {
		return nil, err
	}
	return attestationInput, nil
}

func validateSecureSessionAndBuildSummary(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt) (launcherbackend.SecureSessionSummary, string, error) {
	handshakeTuple, launchContextDigest, err := secureSessionHandshakeTuple(spec, receipt)
	if err != nil {
		return launcherbackend.SecureSessionSummary{}, "", err
	}
	binding, err := launcherbackend.ValidateSessionHandshake(handshakeTuple.launchContext, handshakeTuple.host, handshakeTuple.isolate, handshakeTuple.ready, nil)
	if err != nil {
		return launcherbackend.SecureSessionSummary{}, "", fmt.Errorf("secure session validation failed: %w", err)
	}
	summary, err := launcherbackend.BuildSecureSessionSummary(handshakeTuple.host, handshakeTuple.isolate, handshakeTuple.ready, binding)
	if err != nil {
		return launcherbackend.SecureSessionSummary{}, "", fmt.Errorf("secure session summary failed: %w", err)
	}
	return summary, launchContextDigest, nil
}

func secureSessionTransportKind(value string) string {
	switch value {
	case launcherbackend.TransportKindVSock, launcherbackend.TransportKindVirtioSerial:
		return value
	default:
		return launcherbackend.TransportKindVSock
	}
}

func collectPostHandshakeRuntimeAttestationInput(receipt *launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord) (*launcherbackend.PostHandshakeRuntimeAttestationInput, error) {
	expectedMeasurementDigests, err := canonicalTrustedRuntimeMeasurementDigests(receipt, admission)
	if err != nil {
		return nil, err
	}
	return &launcherbackend.PostHandshakeRuntimeAttestationInput{
		RunID:                        receipt.RunID,
		IsolateID:                    receipt.IsolateID,
		SessionID:                    receipt.SessionID,
		SessionNonce:                 receipt.SessionNonce,
		RuntimeEvidenceCollected:     false,
		LaunchContextDigest:          receipt.LaunchContextDigest,
		HandshakeTranscriptHash:      receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     receipt.IsolateSessionKeyIDValue,
		RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      receipt.RuntimeImageBootProfile,
		RuntimeImageVerifierRef:      receipt.RuntimeImageVerifierRef,
		AuthorityStateDigest:         receipt.AuthorityStateDigest,
		BootComponentDigestByName:    cloneMap(receipt.BootComponentDigestByName),
		BootComponentDigests:         componentDigestValues(receipt.BootComponentDigestByName),
		AttestationSourceKind:        launcherbackend.AttestationSourceKindTrustedRuntime,
		MeasurementProfile:           admission.AttestationMeasurementProfile,
		FreshnessMaterial:            []string{"session_nonce"},
		FreshnessBindingClaims:       []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"},
		EvidenceClaimsDigest:         expectedMeasurementDigests[0],
	}, nil
}

func applyTrustedRuntimeAttestation(receipt *launcherbackend.BackendLaunchReceipt, attestationInput *launcherbackend.PostHandshakeRuntimeAttestationInput, now time.Time) error {
	_ = now
	if receipt == nil {
		return fmt.Errorf("receipt is required")
	}
	if attestationInput == nil {
		return fmt.Errorf("post-handshake runtime attestation input is required")
	}
	normalizedInput := launcherbackend.NormalizePostHandshakeRuntimeAttestationInput(attestationInput)
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
	attestationInput.VerifierPolicyID = ""
	attestationInput.VerifierPolicyDigest = ""
	attestationInput.VerificationRulesProfileVersion = ""
	attestationInput.VerificationTimestamp = now.UTC().Format(time.RFC3339)
	attestationInput.VerificationResult = launcherbackend.AttestationVerificationResultUnknown
	attestationInput.VerificationReasonCodes = nil
	attestationInput.ReplayVerdict = launcherbackend.AttestationReplayVerdictUnknown
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
