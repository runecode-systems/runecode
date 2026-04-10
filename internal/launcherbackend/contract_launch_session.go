package launcherbackend

import (
	"fmt"
	"sort"
	"strings"
)

func (c LaunchContext) Validate() error {
	if strings.TrimSpace(c.RunID) == "" || strings.TrimSpace(c.StageID) == "" || strings.TrimSpace(c.RoleInstanceID) == "" {
		return fmt.Errorf("run_id, stage_id, and role_instance_id are required")
	}
	if strings.TrimSpace(c.SessionID) == "" || strings.TrimSpace(c.SessionNonce) == "" {
		return fmt.Errorf("session_id and session_nonce are required")
	}
	if strings.TrimSpace(c.LaunchContextDigest) == "" {
		return fmt.Errorf("launch_context_digest is required")
	}
	if !looksLikeDigest(c.LaunchContextDigest) {
		return fmt.Errorf("launch_context_digest must be sha256:<64 lowercase hex>")
	}
	expected, err := c.CanonicalDigest()
	if err != nil {
		return err
	}
	if expected != c.LaunchContextDigest {
		return fmt.Errorf("launch_context_digest does not match canonical launch context content")
	}
	return nil
}

func (c LaunchContext) CanonicalDigest() (string, error) {
	normalized := c
	normalized.LaunchContextDigest = ""
	if len(normalized.ActiveManifestHashes) > 1 {
		sort.Strings(normalized.ActiveManifestHashes)
	}
	if len(normalized.PolicyDecisionRefs) > 1 {
		sort.Strings(normalized.PolicyDecisionRefs)
	}
	if len(normalized.ApprovedArtifactRefs) > 1 {
		sort.Strings(normalized.ApprovedArtifactRefs)
	}
	return canonicalSHA256Digest(normalized, "canonical launch context")
}

func (h HostHello) Validate() error {
	if err := validateHostHelloIdentity(h); err != nil {
		return err
	}
	if err := validateHostHelloTransport(h); err != nil {
		return err
	}
	return validateHostHelloFraming(h)
}

func validateHostHelloIdentity(h HostHello) error {
	if strings.TrimSpace(h.RunID) == "" || strings.TrimSpace(h.IsolateID) == "" || strings.TrimSpace(h.SessionID) == "" {
		return fmt.Errorf("run_id, isolate_id, and session_id are required")
	}
	if strings.TrimSpace(h.SessionNonce) == "" || len(strings.TrimSpace(h.SessionNonce)) < 16 {
		return fmt.Errorf("session_nonce must be present and at least 16 characters")
	}
	if !looksLikeDigest(h.LaunchContextDigest) {
		return fmt.Errorf("launch_context_digest must be sha256:<64 lowercase hex>")
	}
	return nil
}

func validateHostHelloTransport(h HostHello) error {
	if normalizeTransportKind(h.TransportKind) == TransportKindUnknown {
		return fmt.Errorf("transport_kind must be one of %q or %q", TransportKindVSock, TransportKindVirtioSerial)
	}
	if !h.TransportRequirements.MutualAuthenticationRequired || !h.TransportRequirements.EncryptionRequired || !h.TransportRequirements.ReplayProtectionRequired {
		return fmt.Errorf("transport requirements must require mutual authentication, encryption, and replay protection")
	}
	return nil
}

func validateHostHelloFraming(h HostHello) error {
	if h.Framing.FrameFormat != SessionFramingLengthPrefixedV1 {
		return fmt.Errorf("frame_format must be %q", SessionFramingLengthPrefixedV1)
	}
	if h.Framing.MaxFrameBytes <= 0 || h.Framing.MaxFrameBytes > SessionMaxFrameBytesHardLimit {
		return fmt.Errorf("max_frame_bytes must be between 1 and %d", SessionMaxFrameBytesHardLimit)
	}
	if h.Framing.MaxHandshakeMessageBytes <= 0 || h.Framing.MaxHandshakeMessageBytes > SessionMaxHandshakeMessageBytesHardLimit {
		return fmt.Errorf("max_handshake_message_bytes must be between 1 and %d", SessionMaxHandshakeMessageBytesHardLimit)
	}
	return nil
}

func (h IsolateHello) Validate() error {
	if err := validateIsolateHelloSessionIdentity(h); err != nil {
		return err
	}
	if err := validateIsolateHelloDigestAndSessionKey(h); err != nil {
		return err
	}
	return validateIsolateHelloProofOfPossession(h)
}

func validateIsolateHelloSessionIdentity(hello IsolateHello) error {
	if strings.TrimSpace(hello.RunID) == "" || strings.TrimSpace(hello.IsolateID) == "" || strings.TrimSpace(hello.SessionID) == "" {
		return fmt.Errorf("run_id, isolate_id, and session_id are required")
	}
	if strings.TrimSpace(hello.SessionNonce) == "" {
		return fmt.Errorf("session_nonce is required")
	}
	if len(strings.TrimSpace(hello.SessionNonce)) < 16 {
		return fmt.Errorf("session_nonce must be present and at least 16 characters")
	}
	return nil
}

func validateIsolateHelloDigestAndSessionKey(hello IsolateHello) error {
	if !looksLikeDigest(hello.LaunchContextDigest) {
		return fmt.Errorf("launch_context_digest must be sha256:<64 lowercase hex>")
	}
	if !looksLikeDigest(hello.HandshakeTranscriptHash) {
		return fmt.Errorf("handshake_transcript_hash must be sha256:<64 lowercase hex>")
	}
	if hello.IsolateSessionKey.Alg != "ed25519" {
		return fmt.Errorf("isolate_session_key.alg must be ed25519")
	}
	if hello.IsolateSessionKey.KeyID == "" || hello.IsolateSessionKey.KeyIDValue == "" {
		return fmt.Errorf("isolate_session_key key identity is required")
	}
	if !looksLikeHexKeyIDValue(hello.IsolateSessionKey.KeyIDValue) {
		return fmt.Errorf("isolate_session_key.key_id_value must be 64 lowercase hex characters")
	}
	if hello.IsolateSessionKey.KeyOrigin != SessionKeyOriginIsolateBoundaryEphemeral {
		return fmt.Errorf("isolate_session_key.key_origin must be %q", SessionKeyOriginIsolateBoundaryEphemeral)
	}
	if strings.TrimSpace(hello.IsolateSessionKey.PublicKey) == "" || strings.TrimSpace(hello.IsolateSessionKey.PublicKeyEncoding) == "" {
		return fmt.Errorf("isolate_session_key public key material is required")
	}
	return nil
}

func validateIsolateHelloProofOfPossession(hello IsolateHello) error {
	if hello.ProofOfPossession.Alg != "ed25519" {
		return fmt.Errorf("proof_of_possession.alg must be ed25519")
	}
	if strings.TrimSpace(hello.ProofOfPossession.Signature) == "" {
		return fmt.Errorf("proof_of_possession.signature is required")
	}
	if hello.ProofOfPossession.KeyID != hello.IsolateSessionKey.KeyID || hello.ProofOfPossession.KeyIDValue != hello.IsolateSessionKey.KeyIDValue {
		return fmt.Errorf("proof_of_possession key identity must match isolate_session_key")
	}
	return nil
}

func (r SessionReady) Validate() error {
	if err := validateSessionReadyIdentity(r); err != nil {
		return err
	}
	if err := validateSessionReadyModeAndBinding(r); err != nil {
		return err
	}
	return validateSessionReadySecurity(r)
}

func validateSessionReadyIdentity(r SessionReady) error {
	if strings.TrimSpace(r.RunID) == "" || strings.TrimSpace(r.IsolateID) == "" || strings.TrimSpace(r.SessionID) == "" {
		return fmt.Errorf("run_id, isolate_id, and session_id are required")
	}
	if strings.TrimSpace(r.SessionNonce) == "" {
		return fmt.Errorf("session_nonce is required")
	}
	if !looksLikeHexKeyIDValue(r.IsolateKeyIDValue) {
		return fmt.Errorf("isolate_key_id_value must be 64 lowercase hex characters")
	}
	if !looksLikeDigest(r.HandshakeTranscriptHash) {
		return fmt.Errorf("handshake_transcript_hash must be sha256:<64 lowercase hex>")
	}
	return nil
}

func validateSessionReadyModeAndBinding(r SessionReady) error {
	if r.ProvisioningMode != ProvisioningPostureTOFU {
		return fmt.Errorf("unsupported provisioning_mode %q", r.ProvisioningMode)
	}
	if r.IdentityBindingPosture != ProvisioningPostureTOFU {
		return fmt.Errorf("identity_binding_posture %q is incompatible with provisioning_mode tofu", r.IdentityBindingPosture)
	}
	if r.ChannelKeyMode != SessionChannelKeyModeDistinct {
		return fmt.Errorf("channel_key_mode must be %q", SessionChannelKeyModeDistinct)
	}
	return nil
}

func validateSessionReadySecurity(r SessionReady) error {
	if !r.MutuallyAuthenticated || !r.Encrypted || !r.ProofOfPossessionVerified {
		return fmt.Errorf("session_ready requires mutually authenticated+encrypted channel with proof-of-possession verification")
	}
	return nil
}
