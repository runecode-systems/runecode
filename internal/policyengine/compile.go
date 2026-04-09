package policyengine

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func decodeRoleManifest(input ManifestInput, registry *trustpolicy.VerifierRegistry, requireSignedContextVerify bool) (RoleManifest, string, []string, error) {
	manifest := RoleManifest{}
	if err := json.Unmarshal(input.Payload, &manifest); err != nil {
		return RoleManifest{}, "", nil, err
	}
	if manifest.SchemaID != roleManifestSchemaID {
		return RoleManifest{}, "", nil, schemaIDError(manifest.SchemaID, roleManifestSchemaID)
	}
	if manifest.SchemaVersion != roleManifestSchemaVersion {
		return RoleManifest{}, "", nil, schemaVersionError(manifest.SchemaID, manifest.SchemaVersion, roleManifestSchemaVersion)
	}
	if err := validateObjectPayloadAgainstSchema(input.Payload, roleManifestSchemaPath); err != nil {
		return RoleManifest{}, "", nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	if err := validateApprovalProfile(manifest.ApprovalProfile); err != nil {
		return RoleManifest{}, "", nil, err
	}
	signerIDs, err := verifyContextSignatures(input.Payload, manifest.SchemaID, manifest.SchemaVersion, registry, requireSignedContextVerify)
	if err != nil {
		return RoleManifest{}, "", nil, err
	}
	digest, err := canonicalHashBytes(input.Payload)
	if err != nil {
		return RoleManifest{}, "", nil, err
	}
	if err := verifyExpectedHash(input.ExpectedHash, digest); err != nil {
		return RoleManifest{}, "", nil, err
	}
	return manifest, digest, signerIDs, nil
}

func decodeCapabilityManifest(input ManifestInput, expectedScope string, registry *trustpolicy.VerifierRegistry, requireSignedContextVerify bool) (CapabilityManifest, string, []string, error) {
	schemaPath := runCapabilitySchemaPath
	if expectedScope == "stage" {
		schemaPath = stageCapabilitySchemaPath
	}
	manifest := CapabilityManifest{}
	if err := json.Unmarshal(input.Payload, &manifest); err != nil {
		return CapabilityManifest{}, "", nil, err
	}
	if manifest.SchemaID != capabilityManifestSchemaID {
		return CapabilityManifest{}, "", nil, schemaIDError(manifest.SchemaID, capabilityManifestSchemaID)
	}
	if manifest.SchemaVersion != capabilityManifestVersion {
		return CapabilityManifest{}, "", nil, schemaVersionError(manifest.SchemaID, manifest.SchemaVersion, capabilityManifestVersion)
	}
	if manifest.ManifestScope != expectedScope {
		return CapabilityManifest{}, "", nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("manifest_scope %q does not match expected %q", manifest.ManifestScope, expectedScope)}
	}
	if err := validateObjectPayloadAgainstSchema(input.Payload, schemaPath); err != nil {
		return CapabilityManifest{}, "", nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	if err := validateApprovalProfile(manifest.ApprovalProfile); err != nil {
		return CapabilityManifest{}, "", nil, err
	}
	signerIDs, err := verifyContextSignatures(input.Payload, manifest.SchemaID, manifest.SchemaVersion, registry, requireSignedContextVerify)
	if err != nil {
		return CapabilityManifest{}, "", nil, err
	}
	digest, err := canonicalHashBytes(input.Payload)
	if err != nil {
		return CapabilityManifest{}, "", nil, err
	}
	if err := verifyExpectedHash(input.ExpectedHash, digest); err != nil {
		return CapabilityManifest{}, "", nil, err
	}
	return manifest, digest, signerIDs, nil
}

func validateApprovalProfile(profile string) error {
	if ApprovalProfile(profile) != ApprovalProfileModerate {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown approval_profile %q (fail-closed)", profile)}
	}
	return nil
}

func validateGatewayScopeRule(rule GatewayScopeRule) error {
	if rule.SchemaID != gatewayScopeRuleSchemaID {
		return schemaIDError(rule.SchemaID, gatewayScopeRuleSchemaID)
	}
	if rule.SchemaVersion != gatewayScopeRuleVersion {
		return schemaVersionError(rule.SchemaID, rule.SchemaVersion, gatewayScopeRuleVersion)
	}
	if rule.ScopeKind != "gateway_destination" {
		return fmt.Errorf("unknown scope_kind %q (fail-closed)", rule.ScopeKind)
	}
	if err := validateDestinationDescriptor(rule.Destination); err != nil {
		return fmt.Errorf("destination: %w", err)
	}
	return nil
}

func validateDestinationDescriptor(descriptor DestinationDescriptor) error {
	if descriptor.SchemaID != destinationDescriptorSchemaID {
		return schemaIDError(descriptor.SchemaID, destinationDescriptorSchemaID)
	}
	if descriptor.SchemaVersion != destinationDescriptorVersion {
		return schemaVersionError(descriptor.SchemaID, descriptor.SchemaVersion, destinationDescriptorVersion)
	}
	return nil
}

func activeAllowlistRefs(role RoleManifest, run CapabilityManifest, stage *CapabilityManifest, allowlists map[string]PolicyAllowlist) ([]string, error) {
	roleRefs, err := digestIdentities(role.AllowlistRefs)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("role allowlist digest identity invalid: %v", err)}
	}
	runRefs, err := digestIdentities(run.AllowlistRefs)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("run allowlist digest identity invalid: %v", err)}
	}
	refs := append(roleRefs, runRefs...)
	if stage != nil {
		stageRefs, stageErr := digestIdentities(stage.AllowlistRefs)
		if stageErr != nil {
			return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("stage allowlist digest identity invalid: %v", stageErr)}
		}
		refs = append(refs, stageRefs...)
	}
	refs = sortedUnique(refs)
	for _, ref := range refs {
		if _, err := normalizeHashIdentity(ref); err != nil {
			return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("allowlist ref %q is not a sha256 identity: %v", ref, err)}
		}
		if _, ok := allowlists[ref]; !ok {
			return nil, &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("allowlist %q referenced by active manifests but missing from active signed context", ref)}
		}
	}
	return refs, nil
}

func digestIdentities(digests []trustpolicy.Digest) ([]string, error) {
	out := make([]string, 0, len(digests))
	for _, d := range digests {
		id, err := d.Identity()
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func verifyExpectedHash(expected, actual string) error {
	if expected == "" {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: "expected_hash is required for trusted manifest inputs (fail-closed)"}
	}
	normalized, err := normalizeHashIdentity(expected)
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	if normalized != actual {
		return &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("content hash mismatch: expected %q, got %q", normalized, actual)}
	}
	return nil
}

func normalizeInvariants(invariants FixedInvariants) FixedInvariants {
	return FixedInvariants{
		DeniedCapabilities: sortedUnique(invariants.DeniedCapabilities),
		DeniedActionKinds:  sortedUnique(invariants.DeniedActionKinds),
	}
}

func digestToStruct(identity string) trustpolicy.Digest {
	return trustpolicy.Digest{HashAlg: "sha256", Hash: identity[len("sha256:"):]}
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	clone := append([]string{}, values...)
	sort.Strings(clone)
	out := make([]string, 0, len(clone))
	for _, v := range clone {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}

func intersect(left []string, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, v := range right {
		set[v] = struct{}{}
	}
	out := make([]string, 0, len(left))
	for _, v := range left {
		if _, ok := set[v]; ok {
			out = append(out, v)
		}
	}
	return sortedUnique(out)
}

func without(values []string, denied []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	if len(denied) == 0 {
		return sortedUnique(values)
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if !slices.Contains(denied, v) {
			out = append(out, v)
		}
	}
	return sortedUnique(out)
}
