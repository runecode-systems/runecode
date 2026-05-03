package trustpolicy

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func validateMetaAuditActionCore(receiptKind string, payload metaAuditActionReceiptPayload) error {
	if strings.TrimSpace(payload.ActionCode) != receiptKind {
		return fmt.Errorf("action_code=%q does not match audit_receipt_kind=%q", payload.ActionCode, receiptKind)
	}
	if strings.TrimSpace(payload.ActionFamily) != "meta_audit" {
		return fmt.Errorf("action_family must be meta_audit")
	}
	if strings.TrimSpace(payload.ScopeKind) == "" {
		return fmt.Errorf("scope_kind is required")
	}
	if strings.TrimSpace(payload.Result) == "" {
		return fmt.Errorf("result is required")
	}
	return nil
}

func validateMetaAuditActionOptionalDigests(payload metaAuditActionReceiptPayload) error {
	for _, field := range []struct {
		name   string
		digest *Digest
	}{
		{name: "scope_ref_digest", digest: payload.ScopeRefDigest},
		{name: "manifest_digest", digest: payload.ManifestDigest},
		{name: "object_digest", digest: payload.ObjectDigest},
	} {
		if err := validateOptionalReceiptDigestField(field.digest, field.name); err != nil {
			return err
		}
	}
	return nil
}

func validateMetaAuditActionOperator(operator *PrincipalIdentity) error {
	if operator == nil {
		return nil
	}
	raw, err := json.Marshal(operator)
	if err != nil {
		return fmt.Errorf("operator marshal failed: %w", err)
	}
	if err := validateReceiptRecorder(raw); err != nil {
		return fmt.Errorf("operator: %w", err)
	}
	return nil
}

func validatePrincipalIdentityOptional(identity *PrincipalIdentity, field string) error {
	if identity == nil {
		return nil
	}
	raw, err := json.Marshal(identity)
	if err != nil {
		return fmt.Errorf("%s marshal failed: %w", field, err)
	}
	if err := validateReceiptRecorder(raw); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func validateOptionalReceiptDigestField(d *Digest, field string) error {
	if d == nil {
		return nil
	}
	if _, err := d.Identity(); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func marshalJSONOrNull(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		return []byte("null")
	}
	return b
}

func validateOptionalDigestIdentityString(value string, field string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if !isDigestIdentity(trimmed) {
		return fmt.Errorf("%s must be digest identity", field)
	}
	return nil
}

func isDigestIdentity(value string) bool {
	alg, hash, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok || alg != "sha256" || len(hash) != 64 || strings.ToLower(hash) != hash {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil
}
