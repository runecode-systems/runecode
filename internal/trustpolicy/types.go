package trustpolicy

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type Digest struct {
	HashAlg string `json:"hash_alg"`
	Hash    string `json:"hash"`
}

func (d Digest) Identity() (string, error) {
	if d.HashAlg != "sha256" {
		return "", fmt.Errorf("unsupported hash algorithm %q", d.HashAlg)
	}
	if len(d.Hash) != 64 {
		return "", fmt.Errorf("sha256 hash must be 64 lowercase hex characters")
	}
	if _, err := hex.DecodeString(d.Hash); err != nil {
		return "", fmt.Errorf("invalid sha256 hash value: %w", err)
	}
	if strings.ToLower(d.Hash) != d.Hash {
		return "", fmt.Errorf("sha256 hash must be lowercase hex")
	}
	return "sha256:" + d.Hash, nil
}

type SignatureBlock struct {
	Alg        string `json:"alg"`
	KeyID      string `json:"key_id"`
	KeyIDValue string `json:"key_id_value"`
	Signature  string `json:"signature"`
}

func (s SignatureBlock) SignatureBytes() ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 signature: %w", err)
	}
	return b, nil
}

type SignedObjectEnvelope struct {
	SchemaID             string          `json:"schema_id"`
	SchemaVersion        string          `json:"schema_version"`
	PayloadSchemaID      string          `json:"payload_schema_id"`
	PayloadSchemaVersion string          `json:"payload_schema_version"`
	Payload              json.RawMessage `json:"payload"`
	SignatureInput       string          `json:"signature_input"`
	Signature            SignatureBlock  `json:"signature"`
}

type PublicKey struct {
	Encoding string `json:"encoding"`
	Value    string `json:"value"`
}

func (p PublicKey) DecodedBytes() ([]byte, error) {
	if p.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported public key encoding %q", p.Encoding)
	}
	b, err := base64.StdEncoding.DecodeString(p.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 public key: %w", err)
	}
	return b, nil
}

type PrincipalIdentity struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	ActorKind     string `json:"actor_kind"`
	PrincipalID   string `json:"principal_id"`
	InstanceID    string `json:"instance_id"`
}

type VerifierRecord struct {
	SchemaID               string             `json:"schema_id"`
	SchemaVersion          string             `json:"schema_version"`
	KeyID                  string             `json:"key_id"`
	KeyIDValue             string             `json:"key_id_value"`
	Alg                    string             `json:"alg"`
	PublicKey              PublicKey          `json:"public_key"`
	LogicalPurpose         string             `json:"logical_purpose"`
	LogicalScope           string             `json:"logical_scope"`
	OwnerPrincipal         PrincipalIdentity  `json:"owner_principal"`
	KeyProtectionPosture   string             `json:"key_protection_posture"`
	IdentityBindingPosture string             `json:"identity_binding_posture"`
	PresenceMode           string             `json:"presence_mode"`
	CreatedAt              string             `json:"created_at"`
	Status                 string             `json:"status"`
	CreatedBy              *PrincipalIdentity `json:"created_by,omitempty"`
	StatusChangedAt        string             `json:"status_changed_at,omitempty"`
	StatusReason           string             `json:"status_reason,omitempty"`
}

type ApprovalDecision struct {
	SchemaID               string            `json:"schema_id"`
	SchemaVersion          string            `json:"schema_version"`
	ApprovalRequestHash    Digest            `json:"approval_request_hash"`
	Approver               PrincipalIdentity `json:"approver"`
	DecisionOutcome        string            `json:"decision_outcome"`
	ApprovalAssuranceLevel string            `json:"approval_assurance_level"`
	PresenceMode           string            `json:"presence_mode"`
	KeyProtectionPosture   string            `json:"key_protection_posture"`
	IdentityBindingPosture string            `json:"identity_binding_posture"`
	ApprovalAssertionHash  *Digest           `json:"approval_assertion_hash,omitempty"`
	DecidedAt              string            `json:"decided_at"`
	ConsumptionPosture     string            `json:"consumption_posture"`
}
