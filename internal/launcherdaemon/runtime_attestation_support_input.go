package launcherdaemon

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func collectPostHandshakeRuntimeAttestationInput(receipt *launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord, material *launcherbackend.RuntimePostHandshakeMaterial) (*launcherbackend.PostHandshakeRuntimeAttestationInput, error) {
	expectedMeasurementDigests, err := canonicalTrustedRuntimeMeasurementDigests(receipt, admission)
	if err != nil {
		return nil, err
	}
	runtimeInput := normalizedRuntimeAttestationInput(material)
	if err := validateRuntimeReportedAttestationBinding(receipt, runtimeInput); err != nil {
		return nil, err
	}
	input := basePostHandshakeAttestationInput(receipt, runtimeInput)
	copyRuntimeAttestationDetails(input, runtimeInput)
	if err := validateRuntimeEvidenceClaims(input, expectedMeasurementDigests); err != nil {
		return nil, err
	}
	return input, nil
}

func normalizedRuntimeAttestationInput(material *launcherbackend.RuntimePostHandshakeMaterial) *launcherbackend.PostHandshakeRuntimeAttestationInput {
	if material == nil {
		return nil
	}
	return launcherbackend.NormalizePostHandshakeRuntimeAttestationInput(material.Attestation)
}

func basePostHandshakeAttestationInput(receipt *launcherbackend.BackendLaunchReceipt, runtimeInput *launcherbackend.PostHandshakeRuntimeAttestationInput) *launcherbackend.PostHandshakeRuntimeAttestationInput {
	return &launcherbackend.PostHandshakeRuntimeAttestationInput{
		RunID:                        receipt.RunID,
		IsolateID:                    receipt.IsolateID,
		SessionID:                    receipt.SessionID,
		SessionNonce:                 receipt.SessionNonce,
		RuntimeEvidenceCollected:     runtimeInput != nil && runtimeInput.RuntimeEvidenceCollected,
		LaunchContextDigest:          receipt.LaunchContextDigest,
		HandshakeTranscriptHash:      receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     receipt.IsolateSessionKeyIDValue,
		RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      receipt.RuntimeImageBootProfile,
		RuntimeImageVerifierRef:      receipt.RuntimeImageVerifierRef,
		AuthorityStateDigest:         receipt.AuthorityStateDigest,
		BootComponentDigestByName:    cloneMap(receipt.BootComponentDigestByName),
		BootComponentDigests:         componentDigestValues(receipt.BootComponentDigestByName),
		AttestationSourceKind:        launcherbackend.AttestationSourceKindUnknown,
		MeasurementProfile:           launcherbackend.MeasurementProfileUnknown,
		VerificationResult:           launcherbackend.AttestationVerificationResultUnknown,
		ReplayVerdict:                launcherbackend.AttestationReplayVerdictUnknown,
	}
}

func copyRuntimeAttestationDetails(input *launcherbackend.PostHandshakeRuntimeAttestationInput, runtimeInput *launcherbackend.PostHandshakeRuntimeAttestationInput) {
	if input == nil || runtimeInput == nil {
		return
	}
	input.AttestationSourceKind = runtimeInput.AttestationSourceKind
	input.MeasurementProfile = runtimeInput.MeasurementProfile
	input.FreshnessMaterial = append([]string{}, runtimeInput.FreshnessMaterial...)
	input.FreshnessBindingClaims = append([]string{}, runtimeInput.FreshnessBindingClaims...)
	input.EvidenceClaimsDigest = runtimeInput.EvidenceClaimsDigest
}

func validateRuntimeEvidenceClaims(input *launcherbackend.PostHandshakeRuntimeAttestationInput, expectedMeasurementDigests []string) error {
	if input == nil || !input.RuntimeEvidenceCollected {
		return nil
	}
	if input.AttestationSourceKind == launcherbackend.AttestationSourceKindUnknown || input.MeasurementProfile == launcherbackend.MeasurementProfileUnknown {
		return fmt.Errorf("runtime-reported attestation source and measurement profile are required when runtime evidence is collected")
	}
	if input.EvidenceClaimsDigest == "" {
		return fmt.Errorf("runtime-reported evidence_claims_digest is required when runtime evidence is collected")
	}
	if input.EvidenceClaimsDigest != expectedMeasurementDigests[0] {
		return fmt.Errorf("runtime-reported evidence_claims_digest must bind to admitted runtime identity")
	}
	return nil
}
