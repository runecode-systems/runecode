package trustpolicy

import (
	"encoding/hex"
	"fmt"
)

type IsolateSessionBinding struct {
	RunID                   string `json:"run_id"`
	IsolateID               string `json:"isolate_id"`
	SessionID               string `json:"session_id"`
	SessionNonce            string `json:"session_nonce"`
	ProvisioningMode        string `json:"provisioning_mode"`
	ImageDigest             Digest `json:"image_digest"`
	ActiveManifestHash      Digest `json:"active_manifest_hash"`
	HandshakeTranscriptHash Digest `json:"handshake_transcript_hash"`
	KeyID                   string `json:"key_id"`
	KeyIDValue              string `json:"key_id_value"`
	IdentityBindingPosture  string `json:"identity_binding_posture"`
}

type AuditSignerEvidence struct {
	SignerPurpose   string                 `json:"signer_purpose"`
	SignerScope     string                 `json:"signer_scope"`
	SignerKey       SignatureBlock         `json:"signer_key"`
	IsolateBinding  *IsolateSessionBinding `json:"isolate_binding,omitempty"`
	ApprovalContext *ApprovalDecision      `json:"approval_context,omitempty"`
}

type SignRequestPreconditions struct {
	LogicalPurpose          string            `json:"logical_purpose"`
	LogicalScope            string            `json:"logical_scope"`
	KeyProtectionPosture    string            `json:"key_protection_posture"`
	IdentityBindingPosture  string            `json:"identity_binding_posture"`
	PresenceMode            string            `json:"presence_mode"`
	ApprovalDecisionContext *ApprovalDecision `json:"approval_decision_context,omitempty"`
}

func ValidateIsolateSessionBinding(binding IsolateSessionBinding) error {
	if err := validateIsolateBindingCore(binding); err != nil {
		return err
	}
	if err := validateIsolateBindingKeyIdentity(binding); err != nil {
		return err
	}
	if err := validateIsolateBindingDigests(binding); err != nil {
		return err
	}
	return validateIsolateBindingPosture(binding)
}

func ValidateAuditSignerEvidence(evidence AuditSignerEvidence) error {
	if err := validateAuditSignerCore(evidence); err != nil {
		return err
	}
	if err := validateAuditSignerIsolateEvidence(evidence); err != nil {
		return err
	}
	if evidence.ApprovalContext != nil {
		if err := ValidateApprovalDecisionEvidence(*evidence.ApprovalContext); err != nil {
			return err
		}
	}
	return nil
}

func ValidateSignRequestPreconditions(request SignRequestPreconditions) error {
	if request.LogicalPurpose == "" || request.LogicalScope == "" {
		return fmt.Errorf("logical_purpose and logical_scope are required")
	}
	if _, ok := allowedKeyProtectionPostures[request.KeyProtectionPosture]; !ok {
		return fmt.Errorf("unsupported key_protection_posture %q", request.KeyProtectionPosture)
	}
	if _, ok := allowedIdentityBindingPostures[request.IdentityBindingPosture]; !ok {
		return fmt.Errorf("unsupported identity_binding_posture %q", request.IdentityBindingPosture)
	}
	if _, ok := allowedPresenceModes[request.PresenceMode]; !ok {
		return fmt.Errorf("unsupported presence_mode %q", request.PresenceMode)
	}
	if request.PresenceMode == "none" && request.KeyProtectionPosture == "hardware_backed" {
		return fmt.Errorf("hardware-backed signing requires explicit user presence mode")
	}
	if request.ApprovalDecisionContext != nil {
		if err := ValidateApprovalDecisionEvidence(*request.ApprovalDecisionContext); err != nil {
			return err
		}
	}
	return nil
}

func validateIsolateBindingCore(binding IsolateSessionBinding) error {
	if binding.RunID == "" || binding.IsolateID == "" || binding.SessionID == "" {
		return fmt.Errorf("run_id, isolate_id, and session_id are required")
	}
	if binding.SessionNonce == "" {
		return fmt.Errorf("session_nonce is required")
	}
	if binding.ProvisioningMode != "tofu" {
		return fmt.Errorf("unsupported provisioning_mode %q", binding.ProvisioningMode)
	}
	return nil
}

func validateIsolateBindingKeyIdentity(binding IsolateSessionBinding) error {
	if binding.KeyID != KeyIDProfile {
		return fmt.Errorf("unsupported key_id profile %q", binding.KeyID)
	}
	if len(binding.KeyIDValue) != 64 {
		return fmt.Errorf("key_id_value must be 64 lowercase hex characters")
	}
	if _, err := hex.DecodeString(binding.KeyIDValue); err != nil {
		return fmt.Errorf("key_id_value must be lowercase hex: %w", err)
	}
	return nil
}

func validateIsolateBindingDigests(binding IsolateSessionBinding) error {
	if _, err := binding.ImageDigest.Identity(); err != nil {
		return fmt.Errorf("image_digest: %w", err)
	}
	if _, err := binding.ActiveManifestHash.Identity(); err != nil {
		return fmt.Errorf("active_manifest_hash: %w", err)
	}
	if _, err := binding.HandshakeTranscriptHash.Identity(); err != nil {
		return fmt.Errorf("handshake_transcript_hash: %w", err)
	}
	return nil
}

func validateIsolateBindingPosture(binding IsolateSessionBinding) error {
	if binding.IdentityBindingPosture != "tofu" {
		return fmt.Errorf("identity_binding_posture %q is incompatible with provisioning_mode tofu", binding.IdentityBindingPosture)
	}
	return nil
}

func validateAuditSignerCore(evidence AuditSignerEvidence) error {
	if evidence.SignerPurpose == "" || evidence.SignerScope == "" {
		return fmt.Errorf("signer_purpose and signer_scope are required")
	}
	if _, err := signatureVerifierIdentity(evidence.SignerKey); err != nil {
		return err
	}
	return nil
}

func validateAuditSignerIsolateEvidence(evidence AuditSignerEvidence) error {
	if evidence.SignerPurpose != "isolate_session_identity" {
		return nil
	}
	if evidence.SignerScope != "session" {
		return fmt.Errorf("isolate_session_identity signer must use session scope")
	}
	if evidence.IsolateBinding == nil {
		return fmt.Errorf("isolate signer evidence requires isolate_binding")
	}
	return ValidateIsolateSessionBinding(*evidence.IsolateBinding)
}
