package policyengine

import (
	"encoding/json"
	"fmt"
)

func decodeAllowlist(input ManifestInput) (PolicyAllowlist, string, error) {
	allowlist := PolicyAllowlist{}
	if err := unmarshalAndValidateAllowlistEnvelope(input, &allowlist); err != nil {
		return PolicyAllowlist{}, "", err
	}
	if err := validateAllowlistEntries(allowlist); err != nil {
		return PolicyAllowlist{}, "", err
	}
	digest, err := canonicalHashBytes(input.Payload)
	if err != nil {
		return PolicyAllowlist{}, "", err
	}
	if err := verifyExpectedHash(input.ExpectedHash, digest); err != nil {
		return PolicyAllowlist{}, "", err
	}
	return allowlist, digest, nil
}

func unmarshalAndValidateAllowlistEnvelope(input ManifestInput, allowlist *PolicyAllowlist) error {
	if err := decodeManifestPayload(input.Payload, allowlist); err != nil {
		return err
	}
	if err := validateAllowlistEnvelope(*allowlist, input.Payload); err != nil {
		return err
	}
	return nil
}

func decodeManifestPayload(payload []byte, target any) error {
	return json.Unmarshal(payload, target)
}

func validateAllowlistEnvelope(allowlist PolicyAllowlist, payload []byte) error {
	if err := requireSchemaIdentity(allowlist.SchemaID, policyAllowlistSchemaID); err != nil {
		return err
	}
	if err := requireSchemaVersion(allowlist.SchemaID, allowlist.SchemaVersion, policyAllowlistSchemaVersion); err != nil {
		return err
	}
	if err := validateAllowlistKinds(allowlist.AllowlistKind, allowlist.EntrySchemaID); err != nil {
		return err
	}
	if err := validateObjectPayloadAgainstSchema(payload, allowlistSchemaPath); err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	return nil
}

func validateAllowlistKinds(allowlistKind string, entrySchemaID string) error {
	if allowlistKind != "gateway_scope_rule" {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown allowlist_kind %q (fail-closed)", allowlistKind)}
	}
	if entrySchemaID != gatewayScopeRuleSchemaID {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("entry_schema_id %q does not match required %q", entrySchemaID, gatewayScopeRuleSchemaID)}
	}
	return nil
}

func validateAllowlistEntries(allowlist PolicyAllowlist) error {
	for i := range allowlist.Entries {
		if err := validateGatewayScopeRule(allowlist.Entries[i]); err != nil {
			return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("entries[%d]: %v", i, err)}
		}
	}
	return nil
}

func requireSchemaIdentity(got, want string) error {
	if got != want {
		return schemaIDError(got, want)
	}
	return nil
}

func requireSchemaVersion(schemaID, got, want string) error {
	if got != want {
		return schemaVersionError(schemaID, got, want)
	}
	return nil
}
