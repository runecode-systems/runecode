package launcherbackend

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

func verifyIsolateSessionProofOfPossession(host HostHello, isolate IsolateHello, transcriptHash string) error {
	if isolate.IsolateSessionKey.Alg != "ed25519" || isolate.ProofOfPossession.Alg != "ed25519" {
		return fmt.Errorf("proof-of-possession currently supports only ed25519 isolate identity keys")
	}
	publicKeyBytes, err := isolateSessionPublicKeyBytes(isolate.IsolateSessionKey)
	if err != nil {
		return err
	}
	expectedKeyIDValue := sha256Hex(publicKeyBytes)
	if err := verifySessionProofKeyDigests(isolate, expectedKeyIDValue); err != nil {
		return err
	}
	signature, err := proofSignatureBytes(isolate.ProofOfPossession)
	if err != nil {
		return err
	}
	payload, err := sessionProofPayload(host, isolate, transcriptHash)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKeyBytes), payload, signature) {
		return fmt.Errorf("proof_of_possession.signature failed verification")
	}
	return nil
}

func sessionProofPayload(host HostHello, isolate IsolateHello, transcriptHash string) ([]byte, error) {
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
		ChannelKeyMode:        SessionChannelKeyModeDistinct,
		IdentityKeySeparation: true,
	}
	return canonicalJSONBytes(payload, "session proof payload")
}

func BuildSecureSessionSummary(host HostHello, isolate IsolateHello, ready SessionReady, binding SessionBindingRecord) (SecureSessionSummary, error) {
	publicKeyBytes, err := isolateSessionPublicKeyBytes(isolate.IsolateSessionKey)
	if err != nil {
		return SecureSessionSummary{}, err
	}
	identity := buildSecureSessionIdentity(isolate.IsolateSessionKey, publicKeyBytes)
	channel := buildSecureSessionChannel(host, ready)
	security := buildSecureSessionSecurity(channel)
	return SecureSessionSummary{
		BindingRecord:     binding,
		Identity:          identity,
		Channel:           channel,
		SecurityPosture:   security,
		TranscriptBinding: binding.HandshakeTranscriptHash,
	}, nil
}

func isolateSessionPublicKeyBytes(key IsolateSessionKey) ([]byte, error) {
	if key.PublicKeyEncoding != "base64" {
		return nil, fmt.Errorf("isolate_session_key.public_key_encoding must be base64")
	}
	publicKeyBytes, err := base64.StdEncoding.DecodeString(key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("isolate_session_key.public_key must be valid base64: %w", err)
	}
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("isolate_session_key.public_key must decode to %d bytes for ed25519", ed25519.PublicKeySize)
	}
	return publicKeyBytes, nil
}

func verifySessionProofKeyDigests(isolate IsolateHello, expectedKeyIDValue string) error {
	if isolate.IsolateSessionKey.KeyIDValue != expectedKeyIDValue {
		return fmt.Errorf("isolate_session_key.key_id_value does not match public key digest")
	}
	if isolate.ProofOfPossession.KeyIDValue != expectedKeyIDValue {
		return fmt.Errorf("proof_of_possession.key_id_value does not match public key digest")
	}
	return nil
}

func proofSignatureBytes(proof SessionKeyProof) ([]byte, error) {
	signature, err := base64.StdEncoding.DecodeString(proof.Signature)
	if err != nil {
		return nil, fmt.Errorf("proof_of_possession.signature must be valid base64: %w", err)
	}
	if len(signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("proof_of_possession.signature must decode to %d bytes for ed25519", ed25519.SignatureSize)
	}
	return signature, nil
}

func buildSecureSessionIdentity(key IsolateSessionKey, publicKeyBytes []byte) SecureSessionIdentity {
	return SecureSessionIdentity{
		Algorithm:         key.Alg,
		KeyID:             key.KeyID,
		KeyIDValue:        key.KeyIDValue,
		KeyOrigin:         key.KeyOrigin,
		PublicKeyEncoding: key.PublicKeyEncoding,
		PublicKeyDigest:   "sha256:" + sha256Hex(publicKeyBytes),
	}
}

func buildSecureSessionChannel(host HostHello, ready SessionReady) SecureSessionChannel {
	return SecureSessionChannel{
		TransportKind:             host.TransportKind,
		ChannelKeyMode:            ready.ChannelKeyMode,
		FrameFormat:               host.Framing.FrameFormat,
		MaxFrameBytes:             host.Framing.MaxFrameBytes,
		MaxHandshakeMessageBytes:  host.Framing.MaxHandshakeMessageBytes,
		MutualAuthentication:      ready.MutuallyAuthenticated,
		Encryption:                ready.Encrypted,
		ReplayProtection:          host.TransportRequirements.ReplayProtectionRequired,
		ProofOfPossessionVerified: ready.ProofOfPossessionVerified,
	}
}

func buildSecureSessionSecurity(channel SecureSessionChannel) SessionSecurityPosture {
	return SessionSecurityPosture{
		MutuallyAuthenticated:     channel.MutualAuthentication,
		Encrypted:                 channel.Encryption,
		ProofOfPossessionVerified: channel.ProofOfPossessionVerified,
		ReplayProtected:           channel.ReplayProtection,
		FrameFormat:               channel.FrameFormat,
		MaxFrameBytes:             channel.MaxFrameBytes,
		MaxHandshakeMessageBytes:  channel.MaxHandshakeMessageBytes,
	}
}

func sha256Hex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
