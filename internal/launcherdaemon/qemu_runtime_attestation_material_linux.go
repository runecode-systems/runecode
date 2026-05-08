//go:build linux

package launcherdaemon

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

const (
	qemuRuntimeMaterialToken = "RUNE_POST_HANDSHAKE_MATERIAL="
)

type qemuRuntimeMaterialEnvelope struct {
	SecureSession *launcherbackend.RuntimeSecureSessionMaterial         `json:"secure_session,omitempty"`
	Attestation   *launcherbackend.PostHandshakeRuntimeAttestationInput `json:"attestation,omitempty"`
}

func defaultQEMURuntimePostHandshakeMaterialProvider(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt) (*launcherbackend.RuntimePostHandshakeMaterial, error) {
	return defaultRuntimePostHandshakeMaterialProvider(spec, receipt)
}

func qemuGuestRuntimeMaterialKernelArg(runtimeMaterialLine string) string {
	trimmed := strings.TrimSpace(runtimeMaterialLine)
	if trimmed == "" {
		return ""
	}
	return "RUNE_POST_HANDSHAKE_MATERIAL_LINE=" + trimmed
}

func encodeQEMURuntimePostHandshakeMaterialPayload(material *launcherbackend.RuntimePostHandshakeMaterial) (string, error) {
	if material == nil {
		return "", nil
	}
	if material.SecureSession == nil && material.Attestation == nil {
		return "", nil
	}
	envelope := qemuRuntimeMaterialEnvelope{SecureSession: material.SecureSession, Attestation: material.Attestation}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("qemu runtime post-handshake material encode failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func encodeQEMURuntimePostHandshakeMaterialLine(material *launcherbackend.RuntimePostHandshakeMaterial) (string, error) {
	payload, err := encodeQEMURuntimePostHandshakeMaterialPayload(material)
	if err != nil {
		return "", err
	}
	if payload == "" {
		return "", nil
	}
	return qemuRuntimeMaterialToken + payload, nil
}

func parseQEMURuntimeMaterialLine(line string) (*launcherbackend.RuntimePostHandshakeMaterial, error) {
	payload := strings.TrimSpace(line)
	if !strings.HasPrefix(payload, qemuRuntimeMaterialToken) {
		return nil, nil
	}
	payload = strings.TrimSpace(strings.TrimPrefix(payload, qemuRuntimeMaterialToken))
	if payload == "" {
		return nil, fmt.Errorf("qemu runtime post-handshake material payload is required")
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("qemu runtime post-handshake material decode failed: %w", err)
	}
	decoded = bytes.TrimSpace(decoded)
	if len(decoded) == 0 {
		return nil, fmt.Errorf("qemu runtime post-handshake material payload is required")
	}
	var envelope qemuRuntimeMaterialEnvelope
	if err := json.Unmarshal(decoded, &envelope); err != nil {
		return nil, fmt.Errorf("qemu runtime post-handshake material parse failed: %w", err)
	}
	material := &launcherbackend.RuntimePostHandshakeMaterial{SecureSession: envelope.SecureSession, Attestation: envelope.Attestation}
	if material.SecureSession == nil && material.Attestation == nil {
		return nil, fmt.Errorf("qemu runtime post-handshake material is empty")
	}
	return material, nil
}

func mergeQEMURuntimePostHandshakeMaterial(seed, runtime *launcherbackend.RuntimePostHandshakeMaterial) *launcherbackend.RuntimePostHandshakeMaterial {
	if seed == nil {
		return runtime
	}
	if runtime == nil {
		return seed
	}
	merged := &launcherbackend.RuntimePostHandshakeMaterial{}
	if runtime.SecureSession != nil {
		merged.SecureSession = runtime.SecureSession
	} else {
		merged.SecureSession = seed.SecureSession
	}
	merged.Attestation = mergeQEMURuntimeAttestation(seed.Attestation, runtime.Attestation)
	return merged
}

func mergeQEMURuntimeAttestation(seed, runtime *launcherbackend.PostHandshakeRuntimeAttestationInput) *launcherbackend.PostHandshakeRuntimeAttestationInput {
	if seed == nil {
		return runtime
	}
	if runtime == nil {
		return seed
	}
	merged := *runtime
	mergeQEMUAuthoritativeAttestationFields(&merged, seed)
	mergeQEMURuntimeAttestationFields(&merged, seed)
	if len(merged.BootComponentDigestByName) == 0 {
		merged.BootComponentDigestByName = cloneMap(seed.BootComponentDigestByName)
	}
	if len(merged.BootComponentDigests) == 0 {
		merged.BootComponentDigests = append([]string{}, seed.BootComponentDigests...)
	}
	return &merged
}

func mergeQEMUAuthoritativeAttestationFields(merged *launcherbackend.PostHandshakeRuntimeAttestationInput, seed *launcherbackend.PostHandshakeRuntimeAttestationInput) {
	if merged == nil || seed == nil {
		return
	}
	merged.RunID = seed.RunID
	merged.IsolateID = seed.IsolateID
	merged.SessionID = seed.SessionID
	merged.SessionNonce = seed.SessionNonce
	merged.LaunchContextDigest = seed.LaunchContextDigest
	merged.HandshakeTranscriptHash = seed.HandshakeTranscriptHash
	merged.IsolateSessionKeyIDValue = seed.IsolateSessionKeyIDValue
	merged.RuntimeImageDescriptorDigest = seed.RuntimeImageDescriptorDigest
	merged.RuntimeImageBootProfile = seed.RuntimeImageBootProfile
	merged.RuntimeImageVerifierRef = seed.RuntimeImageVerifierRef
	merged.AuthorityStateDigest = seed.AuthorityStateDigest
	merged.BootComponentDigestByName = cloneMap(seed.BootComponentDigestByName)
	merged.BootComponentDigests = append([]string{}, seed.BootComponentDigests...)
}

func mergeQEMURuntimeAttestationFields(merged *launcherbackend.PostHandshakeRuntimeAttestationInput, seed *launcherbackend.PostHandshakeRuntimeAttestationInput) {
	if merged == nil || seed == nil {
		return
	}
	if merged.AttestationSourceKind == "" || merged.AttestationSourceKind == launcherbackend.AttestationSourceKindUnknown {
		merged.AttestationSourceKind = seed.AttestationSourceKind
	}
	if merged.MeasurementProfile == "" || merged.MeasurementProfile == launcherbackend.MeasurementProfileUnknown {
		merged.MeasurementProfile = seed.MeasurementProfile
	}
	if !merged.RuntimeEvidenceCollected {
		merged.RuntimeEvidenceCollected = seed.RuntimeEvidenceCollected
	}
	if merged.EvidenceClaimsDigest == "" {
		merged.EvidenceClaimsDigest = seed.EvidenceClaimsDigest
	}
	if len(merged.FreshnessMaterial) == 0 {
		merged.FreshnessMaterial = append([]string{}, seed.FreshnessMaterial...)
	}
	if len(merged.FreshnessBindingClaims) == 0 {
		merged.FreshnessBindingClaims = append([]string{}, seed.FreshnessBindingClaims...)
	}
}
