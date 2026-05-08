package launcherdaemon

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func defaultRuntimePostHandshakeMaterialProvider(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt) (*launcherbackend.RuntimePostHandshakeMaterial, error) {
	handshakeTuple, _, err := secureSessionHandshakeTuple(spec, receipt)
	if err != nil {
		return nil, err
	}
	measurementProfile, evidenceClaimsDigest, err := runtimeAttestationMeasurementInputs(receipt)
	if err != nil {
		return nil, err
	}
	return &launcherbackend.RuntimePostHandshakeMaterial{
		SecureSession: &launcherbackend.RuntimeSecureSessionMaterial{
			LaunchContext: handshakeTuple.launchContext,
			HostHello:     handshakeTuple.host,
			IsolateHello:  handshakeTuple.isolate,
			SessionReady:  handshakeTuple.ready,
		},
		Attestation: &launcherbackend.PostHandshakeRuntimeAttestationInput{
			RunID:                        receipt.RunID,
			IsolateID:                    receipt.IsolateID,
			SessionID:                    receipt.SessionID,
			SessionNonce:                 receipt.SessionNonce,
			RuntimeEvidenceCollected:     true,
			LaunchContextDigest:          handshakeTuple.launchContext.LaunchContextDigest,
			HandshakeTranscriptHash:      handshakeTuple.ready.HandshakeTranscriptHash,
			IsolateSessionKeyIDValue:     handshakeTuple.ready.IsolateKeyIDValue,
			RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
			RuntimeImageBootProfile:      receipt.RuntimeImageBootProfile,
			RuntimeImageVerifierRef:      receipt.RuntimeImageVerifierRef,
			AuthorityStateDigest:         receipt.AuthorityStateDigest,
			BootComponentDigestByName:    cloneMap(receipt.BootComponentDigestByName),
			BootComponentDigests:         componentDigestValues(receipt.BootComponentDigestByName),
			AttestationSourceKind:        launcherbackend.AttestationSourceKindTrustedRuntime,
			MeasurementProfile:           measurementProfile,
			FreshnessMaterial:            []string{"session_nonce"},
			FreshnessBindingClaims:       []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"},
			EvidenceClaimsDigest:         evidenceClaimsDigest,
		},
	}, nil
}

func runtimeAttestationMeasurementInputs(receipt launcherbackend.BackendLaunchReceipt) (string, string, error) {
	measurementProfile, err := measurementProfileForLaunchReceipt(receipt)
	if err != nil {
		return "", "", err
	}
	expectedMeasurementDigests, err := launcherbackend.DeriveExpectedMeasurementDigests(measurementProfile, receipt.RuntimeImageBootProfile, receipt.BootComponentDigestByName)
	if err != nil {
		return "", "", err
	}
	if len(expectedMeasurementDigests) == 0 {
		return "", "", fmt.Errorf("expected measurement digests are required for post-handshake attestation material")
	}
	return measurementProfile, expectedMeasurementDigests[0], nil
}

func measurementProfileForLaunchReceipt(receipt launcherbackend.BackendLaunchReceipt) (string, error) {
	switch strings.TrimSpace(receipt.RuntimeImageBootProfile) {
	case launcherbackend.BootProfileMicroVMLinuxKernelInitrdV1:
		return launcherbackend.MeasurementProfileMicroVMBootV1, nil
	case launcherbackend.BootProfileContainerOCIImageV1:
		return launcherbackend.MeasurementProfileContainerImageV1, nil
	default:
		return "", fmt.Errorf("unsupported runtime image boot profile %q for post-handshake attestation material", receipt.RuntimeImageBootProfile)
	}
}
