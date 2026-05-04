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
	_ = spec
	_ = receipt
	return nil, nil
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
	if runtime.Attestation != nil {
		if seed.Attestation != nil && runtime.Attestation.RuntimeEvidenceCollected {
			attestation := *seed.Attestation
			attestation.RuntimeEvidenceCollected = true
			merged.Attestation = &attestation
		} else {
			merged.Attestation = runtime.Attestation
		}
	} else {
		merged.Attestation = seed.Attestation
	}
	return merged
}
