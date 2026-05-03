package trustpolicy

import (
	"fmt"
	"strings"
)

func validateSecretLeaseReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaSecretLeaseV0 {
		return fmt.Errorf("%s receipts require secret lease payload schema", receipt.AuditReceiptKind)
	}
	payload := secretLeaseReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode secret lease payload: %w", err)
	}
	if err := validateSecretLeaseAction(receipt.AuditReceiptKind, payload.LeaseAction); err != nil {
		return err
	}
	if err := validateSecretLeaseRequiredDigests(payload); err != nil {
		return err
	}
	if err := validateSecretLeaseRequiredFields(receipt.AuditReceiptKind, payload); err != nil {
		return err
	}
	return validateSecretLeaseOptionalDigests(payload)
}

func validateSecretLeaseRequiredDigests(payload secretLeaseReceiptPayload) error {
	for _, field := range []struct {
		name   string
		digest Digest
	}{
		{name: "lease_id_digest", digest: payload.LeaseIDDigest},
		{name: "secret_ref_digest", digest: payload.SecretRefDigest},
		{name: "consumer_id_digest", digest: payload.ConsumerIDDigest},
		{name: "scope_digest", digest: payload.ScopeDigest},
	} {
		if _, err := field.digest.Identity(); err != nil {
			return fmt.Errorf("%s: %w", field.name, err)
		}
	}
	return nil
}

func validateSecretLeaseRequiredFields(kind string, payload secretLeaseReceiptPayload) error {
	if strings.TrimSpace(payload.RoleKind) == "" {
		return fmt.Errorf("role_kind is required")
	}
	if kind == auditReceiptKindSecretLeaseRevoked && strings.TrimSpace(payload.RevokedAt) == "" {
		return fmt.Errorf("revoked_at is required for %s", kind)
	}
	return nil
}

func validateSecretLeaseOptionalDigests(payload secretLeaseReceiptPayload) error {
	for _, field := range []struct {
		name   string
		digest *Digest
	}{
		{name: "reason_digest", digest: payload.ReasonDigest},
		{name: "repository_identity_digest", digest: payload.RepositoryIDDigest},
		{name: "action_request_digest", digest: payload.ActionRequestDigest},
		{name: "policy_context_digest", digest: payload.PolicyContextDigest},
		{name: "run_id_digest", digest: payload.RunIDDigest},
	} {
		if err := validateOptionalReceiptDigestField(field.digest, field.name); err != nil {
			return err
		}
	}
	return nil
}

func validateSecretLeaseAction(kind, action string) error {
	action = strings.TrimSpace(action)
	if action != "issued" && action != "revoked" {
		return fmt.Errorf("lease_action must be issued or revoked")
	}
	if kind == auditReceiptKindSecretLeaseIssued && action != "issued" {
		return fmt.Errorf("lease_action must be issued for %s", kind)
	}
	if kind == auditReceiptKindSecretLeaseRevoked && action != "revoked" {
		return fmt.Errorf("lease_action must be revoked for %s", kind)
	}
	return nil
}
