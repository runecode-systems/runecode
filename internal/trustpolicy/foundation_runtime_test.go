package trustpolicy

import "testing"

func TestValidateIsolateSessionBindingRequiresTOFUAndHashes(t *testing.T) {
	binding := IsolateSessionBinding{
		RunID:                   "run-1",
		IsolateID:               "isolate-1",
		SessionID:               "session-1",
		SessionNonce:            "nonce-1",
		ProvisioningMode:        "tofu",
		ImageDigest:             Digest{HashAlg: "sha256", Hash: repeatedHex("a")},
		ActiveManifestHash:      Digest{HashAlg: "sha256", Hash: repeatedHex("b")},
		HandshakeTranscriptHash: Digest{HashAlg: "sha256", Hash: repeatedHex("c")},
		KeyID:                   KeyIDProfile,
		KeyIDValue:              repeatedHex("d"),
		IdentityBindingPosture:  "tofu",
	}
	if err := ValidateIsolateSessionBinding(binding); err != nil {
		t.Fatalf("ValidateIsolateSessionBinding returned error: %v", err)
	}
	binding.ProvisioningMode = "attested"
	if err := ValidateIsolateSessionBinding(binding); err == nil {
		t.Fatal("ValidateIsolateSessionBinding expected fail-closed unsupported provisioning mode")
	}
	binding.ProvisioningMode = "tofu"
	binding.KeyIDValue = repeatedHex("g")
	if err := ValidateIsolateSessionBinding(binding); err == nil {
		t.Fatal("ValidateIsolateSessionBinding expected fail-closed invalid hex key_id_value")
	}
}

func TestValidateAuditSignerEvidenceRequiresSessionScopeForIsolateIdentity(t *testing.T) {
	evidence := AuditSignerEvidence{
		SignerPurpose: "isolate_session_identity",
		SignerScope:   "node",
		SignerKey: SignatureBlock{
			Alg:        "ed25519",
			KeyID:      KeyIDProfile,
			KeyIDValue: repeatedHex("d"),
			Signature:  "c2ln",
		},
		IsolateBinding: &IsolateSessionBinding{
			RunID:                   "run-1",
			IsolateID:               "isolate-1",
			SessionID:               "session-1",
			SessionNonce:            "nonce-1",
			ProvisioningMode:        "tofu",
			ImageDigest:             Digest{HashAlg: "sha256", Hash: repeatedHex("a")},
			ActiveManifestHash:      Digest{HashAlg: "sha256", Hash: repeatedHex("b")},
			HandshakeTranscriptHash: Digest{HashAlg: "sha256", Hash: repeatedHex("c")},
			KeyID:                   KeyIDProfile,
			KeyIDValue:              repeatedHex("d"),
			IdentityBindingPosture:  "tofu",
		},
	}
	if err := ValidateAuditSignerEvidence(evidence); err == nil {
		t.Fatal("ValidateAuditSignerEvidence expected fail-closed isolate scope error")
	}
}

func TestValidateSignRequestPreconditionsRejectsHardwareBackedWithoutPresence(t *testing.T) {
	req := SignRequestPreconditions{
		LogicalPurpose:         "approval_authority",
		LogicalScope:           "user",
		KeyProtectionPosture:   "hardware_backed",
		IdentityBindingPosture: "attested",
		PresenceMode:           "none",
	}
	if err := ValidateSignRequestPreconditions(req); err == nil {
		t.Fatal("ValidateSignRequestPreconditions expected fail-closed presence requirement")
	}
}

func repeatedHex(nibble string) string {
	out := ""
	for len(out) < 64 {
		out += nibble
	}
	return out
}
