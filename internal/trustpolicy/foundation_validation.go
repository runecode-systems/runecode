package trustpolicy

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	ApprovalDecisionSchemaID      = "runecode.protocol.v0.ApprovalDecision"
	ApprovalDecisionSchemaVersion = "0.3.0"
)

var (
	allowedAssuranceLevels = map[string]struct{}{
		"none":                  {},
		"session_authenticated": {},
		"reauthenticated":       {},
		"hardware_backed":       {},
	}
	allowedPresenceModes = map[string]struct{}{
		"none":            {},
		"os_confirmation": {},
		"passphrase":      {},
		"hardware_touch":  {},
	}
	allowedKeyProtectionPostures = map[string]struct{}{
		"hardware_backed":    {},
		"os_keystore":        {},
		"passphrase_wrapped": {},
		"ephemeral_memory":   {},
	}
	allowedIdentityBindingPostures = map[string]struct{}{
		"attested": {},
		"tofu":     {},
	}
	allowedVerifierStatuses = map[string]struct{}{
		"active":      {},
		"retired":     {},
		"revoked":     {},
		"compromised": {},
	}
	allowedDecisionOutcomes = map[string]struct{}{
		"approve":   {},
		"deny":      {},
		"expired":   {},
		"cancelled": {},
	}
)

func ValidateApprovalDecisionEvidence(decision ApprovalDecision) error {
	if err := validateApprovalDecisionSchema(decision); err != nil {
		return err
	}
	if err := validateApprovalDecisionEnums(decision); err != nil {
		return err
	}
	if err := validateApprovalDecisionHashes(decision); err != nil {
		return err
	}
	if err := validateApprovalDecisionAssurance(decision); err != nil {
		return err
	}
	return nil
}

func validateApprovalDecisionSchema(decision ApprovalDecision) error {
	if decision.SchemaID != ApprovalDecisionSchemaID {
		return fmt.Errorf("unexpected approval decision schema_id %q", decision.SchemaID)
	}
	if decision.SchemaVersion != ApprovalDecisionSchemaVersion {
		return fmt.Errorf("unexpected approval decision schema_version %q", decision.SchemaVersion)
	}
	return nil
}

func validateApprovalDecisionEnums(decision ApprovalDecision) error {
	if _, ok := allowedDecisionOutcomes[decision.DecisionOutcome]; !ok {
		return fmt.Errorf("unsupported decision_outcome %q", decision.DecisionOutcome)
	}
	if _, ok := allowedAssuranceLevels[decision.ApprovalAssuranceLevel]; !ok {
		return fmt.Errorf("unsupported approval_assurance_level %q", decision.ApprovalAssuranceLevel)
	}
	if _, ok := allowedPresenceModes[decision.PresenceMode]; !ok {
		return fmt.Errorf("unsupported presence_mode %q", decision.PresenceMode)
	}
	if _, ok := allowedKeyProtectionPostures[decision.KeyProtectionPosture]; !ok {
		return fmt.Errorf("unsupported key_protection_posture %q", decision.KeyProtectionPosture)
	}
	if _, ok := allowedIdentityBindingPostures[decision.IdentityBindingPosture]; !ok {
		return fmt.Errorf("unsupported identity_binding_posture %q", decision.IdentityBindingPosture)
	}
	if decision.ConsumptionPosture != "single_use" {
		return fmt.Errorf("unsupported consumption_posture %q", decision.ConsumptionPosture)
	}
	return nil
}

func validateApprovalDecisionHashes(decision ApprovalDecision) error {
	if _, err := decision.ApprovalRequestHash.Identity(); err != nil {
		return fmt.Errorf("approval_request_hash: %w", err)
	}
	if decision.ApprovalAssertionHash != nil {
		if _, err := decision.ApprovalAssertionHash.Identity(); err != nil {
			return fmt.Errorf("approval_assertion_hash: %w", err)
		}
	}
	return nil
}

func validateApprovalDecisionAssurance(decision ApprovalDecision) error {
	if decision.ApprovalAssuranceLevel == "hardware_backed" {
		if decision.PresenceMode == "none" {
			return fmt.Errorf("hardware_backed assurance requires a non-none presence mode")
		}
		if decision.ApprovalAssertionHash == nil {
			return fmt.Errorf("hardware_backed assurance requires approval_assertion_hash")
		}
	}
	return nil
}

func normalizeVerifierRecord(record VerifierRecord) (string, error) {
	if err := validateVerifierRecordSchema(record); err != nil {
		return "", err
	}
	decodedPublicKey, err := validateVerifierRecordKeyMaterial(record)
	if err != nil {
		return "", err
	}
	if err := validateVerifierRecordPosture(record); err != nil {
		return "", err
	}
	if err := validateVerifierRecordTimestamps(record); err != nil {
		return "", err
	}
	return canonicalVerifierIdentity(record.KeyIDValue, decodedPublicKey)
}

func validateVerifierRecordSchema(record VerifierRecord) error {
	if record.SchemaID != VerifierSchemaID {
		return fmt.Errorf("unexpected verifier schema_id %q", record.SchemaID)
	}
	if record.SchemaVersion != VerifierSchemaVersion {
		return fmt.Errorf("unexpected verifier schema_version %q", record.SchemaVersion)
	}
	if record.KeyID != KeyIDProfile {
		return fmt.Errorf("unsupported key_id profile %q", record.KeyID)
	}
	if record.KeyIDValue == "" {
		return fmt.Errorf("key_id_value is required")
	}
	if len(record.KeyIDValue) != 64 {
		return fmt.Errorf("key_id_value must be 64 lowercase hex characters")
	}
	return nil
}

func validateVerifierRecordKeyMaterial(record VerifierRecord) ([]byte, error) {
	decodedPublicKey, err := record.PublicKey.DecodedBytes()
	if err != nil {
		return nil, err
	}
	if len(decodedPublicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("ed25519 public key must be %d bytes", ed25519.PublicKeySize)
	}
	if record.Alg != "ed25519" {
		return nil, fmt.Errorf("unsupported verifier algorithm %q", record.Alg)
	}
	return decodedPublicKey, nil
}

func validateVerifierRecordPosture(record VerifierRecord) error {
	if _, ok := allowedKeyProtectionPostures[record.KeyProtectionPosture]; !ok {
		return fmt.Errorf("unsupported key_protection_posture %q", record.KeyProtectionPosture)
	}
	if _, ok := allowedIdentityBindingPostures[record.IdentityBindingPosture]; !ok {
		return fmt.Errorf("unsupported identity_binding_posture %q", record.IdentityBindingPosture)
	}
	if _, ok := allowedPresenceModes[record.PresenceMode]; !ok {
		return fmt.Errorf("unsupported presence_mode %q", record.PresenceMode)
	}
	if _, ok := allowedVerifierStatuses[record.Status]; !ok {
		return fmt.Errorf("unsupported verifier status %q", record.Status)
	}
	return nil
}

func validateVerifierRecordTimestamps(record VerifierRecord) error {
	if record.CreatedAt == "" {
		return fmt.Errorf("created_at is required")
	}
	if _, err := time.Parse(time.RFC3339, record.CreatedAt); err != nil {
		return fmt.Errorf("invalid created_at: %w", err)
	}
	if record.StatusChangedAt != "" {
		if _, err := time.Parse(time.RFC3339, record.StatusChangedAt); err != nil {
			return fmt.Errorf("invalid status_changed_at: %w", err)
		}
	}
	if len(record.StatusReason) > 512 {
		return fmt.Errorf("status_reason exceeds max length")
	}
	return nil
}

func canonicalVerifierIdentity(keyIDValue string, decodedPublicKey []byte) (string, error) {
	keyID, err := keyIdentityFromPublicKey(decodedPublicKey)
	if err != nil {
		return "", err
	}
	if keyID != keyIDValue {
		return "", fmt.Errorf("key_id_value does not match canonical public-key digest")
	}
	return KeyIDProfile + ":" + keyIDValue, nil
}

func signatureVerifierIdentity(signature SignatureBlock) (string, error) {
	if signature.Alg != "ed25519" {
		return "", fmt.Errorf("unsupported signature algorithm %q", signature.Alg)
	}
	if signature.KeyID != KeyIDProfile {
		return "", fmt.Errorf("unsupported key_id profile %q", signature.KeyID)
	}
	if signature.KeyIDValue == "" {
		return "", fmt.Errorf("key_id_value is required")
	}
	if len(signature.KeyIDValue) != 64 {
		return "", fmt.Errorf("key_id_value must be 64 lowercase hex characters")
	}
	if _, err := hex.DecodeString(signature.KeyIDValue); err != nil {
		return "", fmt.Errorf("invalid key_id_value: %w", err)
	}
	if signature.Signature == "" {
		return "", fmt.Errorf("signature is required")
	}
	if _, err := base64.StdEncoding.DecodeString(signature.Signature); err != nil {
		return "", fmt.Errorf("signature is not valid base64: %w", err)
	}
	return KeyIDProfile + ":" + signature.KeyIDValue, nil
}

func keyIdentityFromPublicKey(publicKey []byte) (string, error) {
	if len(publicKey) == 0 {
		return "", fmt.Errorf("public key is required")
	}
	digest := sha256.Sum256(publicKey)
	return hex.EncodeToString(digest[:]), nil
}
