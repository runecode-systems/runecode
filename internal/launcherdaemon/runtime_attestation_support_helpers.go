package launcherdaemon

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func deriveSyntheticSecureSessionKeyPair(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt) (string, ed25519.PublicKey, ed25519.PrivateKey) {
	seedHash := sha256.Sum256(syntheticHashInput("secure-session-key", spec.RunID, spec.StageID, spec.RoleInstanceID, receipt.IsolateID, receipt.SessionID, receipt.SessionNonce, receipt.RuntimeImageDescriptorDigest))
	privateKey := ed25519.NewKeyFromSeed(seedHash[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)
	publicKeyHash := sha256.Sum256(publicKey)
	return hex.EncodeToString(publicKeyHash[:]), publicKey, privateKey
}

func secureSessionProofPayload(host launcherbackend.HostHello, isolate launcherbackend.IsolateHello, transcriptHash string) ([]byte, error) {
	payload := struct {
		Schema                string `json:"schema"`
		RunID                 string `json:"run_id"`
		IsolateID             string `json:"isolate_id"`
		SessionID             string `json:"session_id"`
		SessionNonce          string `json:"session_nonce"`
		LaunchContextDigest   string `json:"launch_context_digest"`
		HandshakeTranscript   string `json:"handshake_transcript_hash"`
		TransportKind         string `json:"transport_kind"`
		KeyID                 string `json:"key_id"`
		KeyIDValue            string `json:"key_id_value"`
		ChannelKeyMode        string `json:"channel_key_mode"`
		IdentityKeySeparation bool   `json:"identity_key_separation"`
	}{
		Schema:                "runecode.secure_session_proof_payload.v1",
		RunID:                 host.RunID,
		IsolateID:             host.IsolateID,
		SessionID:             host.SessionID,
		SessionNonce:          host.SessionNonce,
		LaunchContextDigest:   host.LaunchContextDigest,
		HandshakeTranscript:   transcriptHash,
		TransportKind:         host.TransportKind,
		KeyID:                 isolate.IsolateSessionKey.KeyID,
		KeyIDValue:            isolate.IsolateSessionKey.KeyIDValue,
		ChannelKeyMode:        launcherbackend.SessionChannelKeyModeDistinct,
		IdentityKeySeparation: true,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(raw)
}

func canonicalTrustedRuntimeMeasurementDigests(receipt *launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord) ([]string, error) {
	if receipt == nil {
		return nil, nil
	}
	if receipt.RuntimeImageDescriptorDigest == "" || receipt.RuntimeImageBootProfile == "" {
		return nil, fmt.Errorf("runtime identity is required before attestation")
	}
	if receipt.IsolateID == "" || receipt.SessionID == "" || receipt.SessionNonce == "" || receipt.LaunchContextDigest == "" || receipt.HandshakeTranscriptHash == "" || receipt.IsolateSessionKeyIDValue == "" {
		return nil, fmt.Errorf("session binding is required before attestation")
	}
	if admission.AttestationMeasurementProfile == "" || len(admission.AttestationExpectedMeasurementDigests) == 0 {
		return nil, fmt.Errorf("admitted attestation expectations are required")
	}
	expectedMeasurementDigests, err := launcherbackend.DeriveExpectedMeasurementDigests(admission.AttestationMeasurementProfile, admission.BootContractVersion, admission.ComponentDigests)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(expectedMeasurementDigests, admission.AttestationExpectedMeasurementDigests) {
		return nil, fmt.Errorf("admitted attestation expectations do not match canonical runtime identity")
	}
	return expectedMeasurementDigests, nil
}

func componentDigestValues(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	unique := out[:0]
	for _, value := range out {
		if value == "" {
			continue
		}
		if len(unique) == 0 || unique[len(unique)-1] != value {
			unique = append(unique, value)
		}
	}
	return append([]string{}, unique...)
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func canonicalJSONBytes(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(raw)
}
