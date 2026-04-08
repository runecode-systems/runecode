package policyengine

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func Compile(input CompileInput) (*CompiledContext, error) {
	registry, err := compileVerifierRegistry(input)
	if err != nil {
		return nil, err
	}

	roleManifest, roleHash, roleSignerIDs, err := decodeRoleManifest(input.RoleManifest, registry, input.RequireSignedContextVerify)
	if err != nil {
		return nil, err
	}
	runManifest, runHash, runSignerIDs, err := decodeCapabilityManifest(input.RunManifest, "run", registry, input.RequireSignedContextVerify)
	if err != nil {
		return nil, err
	}
	var stageManifest *CapabilityManifest
	var stageHash *string
	stageSignerIDs := []string{}
	if input.StageManifest != nil {
		decoded, digest, signerIDs, decodeErr := decodeCapabilityManifest(*input.StageManifest, "stage", registry, input.RequireSignedContextVerify)
		if decodeErr != nil {
			return nil, decodeErr
		}
		stageManifest = &decoded
		stageHash = &digest
		stageSignerIDs = signerIDs
	}
	allowlistsByHash := map[string]PolicyAllowlist{}
	for i := range input.Allowlists {
		allowlist, digest, decodeErr := decodeAllowlist(input.Allowlists[i])
		if decodeErr != nil {
			return nil, fmt.Errorf("allowlist[%d]: %w", i, decodeErr)
		}
		allowlistsByHash[digest] = allowlist
	}
	ruleSet, ruleSetHash, err := decodeRuleSet(input.RuleSet)
	if err != nil {
		return nil, err
	}
	if roleManifest.ApprovalProfile != runManifest.ApprovalProfile {
		return nil, &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("role/run approval_profile mismatch: %q != %q", roleManifest.ApprovalProfile, runManifest.ApprovalProfile)}
	}
	if stageManifest != nil && runManifest.ApprovalProfile != stageManifest.ApprovalProfile {
		return nil, &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("run/stage approval_profile mismatch: %q != %q", runManifest.ApprovalProfile, stageManifest.ApprovalProfile)}
	}

	roleCaps := sortedUnique(roleManifest.CapabilityOptIns)
	runCaps := sortedUnique(runManifest.CapabilityOptIns)
	stageCaps := []string{}
	if stageManifest != nil {
		stageCaps = sortedUnique(stageManifest.CapabilityOptIns)
	}

	effectiveCaps := intersect(roleCaps, runCaps)
	if stageManifest != nil {
		effectiveCaps = intersect(effectiveCaps, stageCaps)
	}
	effectiveCaps = without(effectiveCaps, input.FixedInvariants.DeniedCapabilities)

	activeAllowlistRefs, err := activeAllowlistRefs(roleManifest, runManifest, stageManifest, allowlistsByHash)
	if err != nil {
		return nil, err
	}

	context := EffectivePolicyContext{
		SchemaID:              "runecode.policyengine.v0.EffectivePolicyContext",
		SchemaVersion:         "0.1.0",
		FixedInvariants:       normalizeInvariants(input.FixedInvariants),
		ActiveRoleFamily:      roleManifest.RoleFamily,
		ActiveRoleKind:        roleManifest.RoleKind,
		ApprovalProfile:       ApprovalProfile(roleManifest.ApprovalProfile),
		RoleManifestHash:      digestToStruct(roleHash),
		RunManifestHash:       digestToStruct(runHash),
		RoleManifestSignerIDs: roleSignerIDs,
		RunManifestSignerIDs:  runSignerIDs,
		RoleCapabilities:      roleCaps,
		RunCapabilities:       runCaps,
		StageCapabilities:     stageCaps,
		EffectiveCapabilities: effectiveCaps,
		ActiveAllowlistRefs:   activeAllowlistRefs,
	}
	if stageHash != nil {
		d := digestToStruct(*stageHash)
		context.StageManifestHash = &d
		context.StageManifestSignerIDs = stageSignerIDs
	}
	if ruleSetHash != "" {
		context.EvaluationRuleSetHash = ruleSetHash
		context.EvaluationRuleSetSchema = policyRuleSetSchemaID
	}

	policyInputHashes := []string{roleHash, runHash}
	if stageHash != nil {
		policyInputHashes = append(policyInputHashes, *stageHash)
	}
	policyInputHashes = append(policyInputHashes, activeAllowlistRefs...)
	if ruleSetHash != "" {
		policyInputHashes = append(policyInputHashes, ruleSetHash)
	}
	policyInputHashes = sortedUnique(policyInputHashes)
	context.PolicyInputHashes = policyInputHashes

	manifestHash, err := canonicalHashValue(context)
	if err != nil {
		return nil, err
	}

	compiled := &CompiledContext{
		Context:           context,
		ManifestHash:      manifestHash,
		PolicyInputHashes: policyInputHashes,
		AllowlistsByHash:  map[string]PolicyAllowlist{},
		RuleSet:           ruleSet,
	}
	for _, ref := range activeAllowlistRefs {
		compiled.AllowlistsByHash[ref] = allowlistsByHash[ref]
	}
	return compiled, nil
}

func compileVerifierRegistry(input CompileInput) (*trustpolicy.VerifierRegistry, error) {
	if !input.RequireSignedContextVerify {
		return nil, nil
	}
	if len(input.VerifierRecords) == 0 {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: "signed context verification requires at least one verifier record"}
	}
	registry, err := trustpolicy.NewVerifierRegistry(input.VerifierRecords)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("invalid verifier records for signed context verification: %v", err)}
	}
	return registry, nil
}

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

func verifyContextSignatures(payload []byte, schemaID, schemaVersion string, registry *trustpolicy.VerifierRegistry, required bool) ([]string, error) {
	signatures, err := extractSignatureBlocks(payload)
	if err != nil {
		return nil, err
	}
	if len(signatures) == 0 {
		if required {
			return nil, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s payload missing signatures for required signed context verification", schemaID)}
		}
		return []string{}, nil
	}
	if registry == nil {
		if required {
			return nil, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: "signed context verification requested but verifier registry unavailable"}
		}
		return []string{}, nil
	}
	canonicalPayload, err := canonicalPayloadWithoutSignatures(payload)
	if err != nil {
		return nil, err
	}
	signerIDs := make([]string, 0, len(signatures))
	for idx := range signatures {
		signature := signatures[idx]
		verifier, resolveErr := registry.Resolve(signature)
		if resolveErr != nil {
			return nil, &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("signature[%d] verifier resolution failed: %v", idx, resolveErr)}
		}
		sigBytes, decodeErr := signature.SignatureBytes()
		if decodeErr != nil {
			return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("signature[%d] decode failed: %v", idx, decodeErr)}
		}
		pub, pubErr := verifier.PublicKey.DecodedBytes()
		if pubErr != nil {
			return nil, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("signature[%d] verifier key decode failed: %v", idx, pubErr)}
		}
		if len(pub) != ed25519.PublicKeySize {
			return nil, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("signature[%d] verifier key length invalid", idx)}
		}
		if !ed25519.Verify(pub, canonicalPayload, sigBytes) {
			return nil, &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("signature[%d] verification failed for %s@%s", idx, schemaID, schemaVersion)}
		}
		signerIDs = append(signerIDs, signature.KeyID+":"+signature.KeyIDValue)
	}
	return sortedUnique(signerIDs), nil
}

func extractSignatureBlocks(payload []byte) ([]trustpolicy.SignatureBlock, error) {
	root := map[string]any{}
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("decode signed context payload: %v", err)}
	}
	raw, ok := root["signatures"]
	if !ok {
		return []trustpolicy.SignatureBlock{}, nil
	}
	rawSlice, ok := raw.([]any)
	if !ok {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "signatures field must be an array"}
	}
	if len(rawSlice) == 0 {
		return []trustpolicy.SignatureBlock{}, nil
	}
	b, err := json.Marshal(rawSlice)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("marshal signatures field: %v", err)}
	}
	out := []trustpolicy.SignatureBlock{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("decode signatures field: %v", err)}
	}
	return out, nil
}

func canonicalPayloadWithoutSignatures(payload []byte) ([]byte, error) {
	root := map[string]any{}
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("decode signed context payload: %v", err)}
	}
	delete(root, "signatures")
	b, err := json.Marshal(root)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("marshal payload without signatures: %v", err)}
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("canonicalize payload without signatures: %v", err)}
	}
	return canonical, nil
}

func validateApprovalProfile(profile string) error {
	if ApprovalProfile(profile) != ApprovalProfileModerate {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown approval_profile %q (fail-closed)", profile)}
	}
	return nil
}

func decodeAllowlist(input ManifestInput) (PolicyAllowlist, string, error) {
	allowlist := PolicyAllowlist{}
	if err := json.Unmarshal(input.Payload, &allowlist); err != nil {
		return PolicyAllowlist{}, "", err
	}
	if allowlist.SchemaID != policyAllowlistSchemaID {
		return PolicyAllowlist{}, "", schemaIDError(allowlist.SchemaID, policyAllowlistSchemaID)
	}
	if allowlist.SchemaVersion != policyAllowlistSchemaVersion {
		return PolicyAllowlist{}, "", schemaVersionError(allowlist.SchemaID, allowlist.SchemaVersion, policyAllowlistSchemaVersion)
	}
	if allowlist.AllowlistKind != "gateway_scope_rule" {
		return PolicyAllowlist{}, "", &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown allowlist_kind %q (fail-closed)", allowlist.AllowlistKind)}
	}
	if allowlist.EntrySchemaID != gatewayScopeRuleSchemaID {
		return PolicyAllowlist{}, "", &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("entry_schema_id %q does not match required %q", allowlist.EntrySchemaID, gatewayScopeRuleSchemaID)}
	}
	if err := validateObjectPayloadAgainstSchema(input.Payload, allowlistSchemaPath); err != nil {
		return PolicyAllowlist{}, "", &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	for i := range allowlist.Entries {
		if err := validateGatewayScopeRule(allowlist.Entries[i]); err != nil {
			return PolicyAllowlist{}, "", &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("entries[%d]: %v", i, err)}
		}
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

func decodeRuleSet(input *ManifestInput) (*PolicyRuleSet, string, error) {
	if input == nil {
		return nil, "", nil
	}
	ruleSet := PolicyRuleSet{}
	if err := json.Unmarshal(input.Payload, &ruleSet); err != nil {
		return nil, "", err
	}
	if ruleSet.SchemaID != policyRuleSetSchemaID {
		return nil, "", schemaIDError(ruleSet.SchemaID, policyRuleSetSchemaID)
	}
	if ruleSet.SchemaVersion != policyRuleSetSchemaVersion {
		return nil, "", schemaVersionError(ruleSet.SchemaID, ruleSet.SchemaVersion, policyRuleSetSchemaVersion)
	}
	if err := validateObjectPayloadAgainstSchema(input.Payload, ruleSetSchemaPath); err != nil {
		return nil, "", &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	for i := range ruleSet.Rules {
		if err := ensureKnownPolicyReasonCode(ruleSet.Rules[i].ReasonCode); err != nil {
			return nil, "", &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("rules[%d].reason_code: %v", i, err)}
		}
	}
	digest, err := canonicalHashBytes(input.Payload)
	if err != nil {
		return nil, "", err
	}
	if err := verifyExpectedHash(input.ExpectedHash, digest); err != nil {
		return nil, "", err
	}
	return &ruleSet, digest, nil
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
		return nil
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
