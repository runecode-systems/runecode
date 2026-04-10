package policyengine

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func Compile(input CompileInput) (*CompiledContext, error) {
	decoded, err := decodeCompileInputs(input)
	if err != nil {
		return nil, err
	}
	if err := ensureApprovalProfileCompatibility(decoded); err != nil {
		return nil, err
	}
	if err := ensureRoleBindingAndCapabilityManifestCompatibility(decoded); err != nil {
		return nil, err
	}
	roleCaps, runCaps, stageCaps, effectiveCaps := computeCapabilitySets(decoded, input.FixedInvariants)
	activeRefs, err := activeAllowlistRefs(decoded.roleManifest, decoded.runManifest, decoded.stageManifest, decoded.allowlistsByHash)
	if err != nil {
		return nil, err
	}
	context := buildEffectivePolicyContext(decoded, input.FixedInvariants, roleCaps, runCaps, stageCaps, effectiveCaps, activeRefs)
	policyInputHashes := buildPolicyInputHashes(decoded, activeRefs)
	context.PolicyInputHashes = policyInputHashes
	manifestHash, err := canonicalHashValue(context)
	if err != nil {
		return nil, err
	}
	return buildCompiledContext(context, manifestHash, policyInputHashes, decoded.allowlistsByHash, activeRefs, decoded.ruleSet), nil
}

type decodedCompileInputs struct {
	roleManifest  RoleManifest
	runManifest   CapabilityManifest
	stageManifest *CapabilityManifest

	roleHash  string
	runHash   string
	stageHash *string

	roleSignerIDs  []string
	runSignerIDs   []string
	stageSignerIDs []string

	allowlistsByHash map[string]PolicyAllowlist
	ruleSet          *PolicyRuleSet
	ruleSetHash      string
}

func decodeCompileInputs(input CompileInput) (*decodedCompileInputs, error) {
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
	stageManifest, stageHash, stageSignerIDs, err := decodeOptionalStageManifest(input.StageManifest, registry, input.RequireSignedContextVerify)
	if err != nil {
		return nil, err
	}
	allowlistsByHash, err := decodeAllowlists(input.Allowlists)
	if err != nil {
		return nil, err
	}
	ruleSet, ruleSetHash, err := decodeRuleSet(input.RuleSet)
	if err != nil {
		return nil, err
	}
	return &decodedCompileInputs{
		roleManifest:     roleManifest,
		runManifest:      runManifest,
		stageManifest:    stageManifest,
		roleHash:         roleHash,
		runHash:          runHash,
		stageHash:        stageHash,
		roleSignerIDs:    roleSignerIDs,
		runSignerIDs:     runSignerIDs,
		stageSignerIDs:   stageSignerIDs,
		allowlistsByHash: allowlistsByHash,
		ruleSet:          ruleSet,
		ruleSetHash:      ruleSetHash,
	}, nil
}

func decodeOptionalStageManifest(input *ManifestInput, registry *trustpolicy.VerifierRegistry, requireSignedContextVerify bool) (*CapabilityManifest, *string, []string, error) {
	if input == nil {
		return nil, nil, []string{}, nil
	}
	decoded, digest, signerIDs, err := decodeCapabilityManifest(*input, "stage", registry, requireSignedContextVerify)
	if err != nil {
		return nil, nil, nil, err
	}
	return &decoded, &digest, signerIDs, nil
}

func decodeAllowlists(inputs []ManifestInput) (map[string]PolicyAllowlist, error) {
	allowlistsByHash := map[string]PolicyAllowlist{}
	for i := range inputs {
		allowlist, digest, err := decodeAllowlist(inputs[i])
		if err != nil {
			return nil, fmt.Errorf("allowlist[%d]: %w", i, err)
		}
		allowlistsByHash[digest] = allowlist
	}
	return allowlistsByHash, nil
}

func ensureApprovalProfileCompatibility(decoded *decodedCompileInputs) error {
	if decoded.roleManifest.ApprovalProfile != decoded.runManifest.ApprovalProfile {
		return &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("role/run approval_profile mismatch: %q != %q", decoded.roleManifest.ApprovalProfile, decoded.runManifest.ApprovalProfile)}
	}
	if decoded.stageManifest != nil && decoded.runManifest.ApprovalProfile != decoded.stageManifest.ApprovalProfile {
		return &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("run/stage approval_profile mismatch: %q != %q", decoded.runManifest.ApprovalProfile, decoded.stageManifest.ApprovalProfile)}
	}
	return nil
}

func ensureRoleBindingAndCapabilityManifestCompatibility(decoded *decodedCompileInputs) error {
	if decoded.roleManifest.RoleFamily == "workspace" {
		if err := validateWorkspaceRoleCapabilityManifest(decoded.roleManifest.RoleKind, decoded.roleManifest.CapabilityOptIns, "role manifest"); err != nil {
			return err
		}
	}
	if err := ensureCapabilityManifestRoleBinding(decoded.roleManifest.RoleFamily, decoded.roleManifest.RoleKind, decoded.runManifest, "run manifest"); err != nil {
		return err
	}
	if decoded.stageManifest != nil {
		if err := ensureCapabilityManifestRoleBinding(decoded.roleManifest.RoleFamily, decoded.roleManifest.RoleKind, *decoded.stageManifest, "stage manifest"); err != nil {
			return err
		}
	}
	return nil
}

func ensureCapabilityManifestRoleBinding(expectedRoleFamily, expectedRoleKind string, manifest CapabilityManifest, manifestLabel string) error {
	if manifest.Principal.RoleFamily == "" {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s principal.role_family is required for explicit role capability manifests (fail-closed)", manifestLabel)}
	}
	if manifest.Principal.RoleKind == "" {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s principal.role_kind is required for explicit role capability manifests (fail-closed)", manifestLabel)}
	}
	if manifest.Principal.RoleFamily != "" && manifest.Principal.RoleFamily != expectedRoleFamily {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s principal.role_family %q does not match role manifest role_family %q", manifestLabel, manifest.Principal.RoleFamily, expectedRoleFamily)}
	}
	if manifest.Principal.RoleKind != "" && manifest.Principal.RoleKind != expectedRoleKind {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s principal.role_kind %q does not match role manifest role_kind %q", manifestLabel, manifest.Principal.RoleKind, expectedRoleKind)}
	}
	if expectedRoleFamily == "workspace" {
		if err := validateWorkspaceRoleCapabilityManifest(expectedRoleKind, manifest.CapabilityOptIns, manifestLabel); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkspaceRoleCapabilityManifest(roleKind string, capabilityOptIns []string, manifestLabel string) error {
	allowed, known := workspaceRoleAllowedCapabilities()[roleKind]
	if !known {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s role_kind %q missing explicit workspace capability manifest policy (fail-closed)", manifestLabel, roleKind)}
	}
	for _, cap := range capabilityOptIns {
		if _, ok := allowed[cap]; !ok {
			return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s capability %q is not allowed for workspace role_kind %q", manifestLabel, cap, roleKind)}
		}
	}
	return nil
}

func knownWorkspaceRoleKinds() map[string]struct{} {
	return map[string]struct{}{
		"workspace-read": {},
		"workspace-edit": {},
		"workspace-test": {},
	}
}

func workspaceRoleAllowedCapabilities() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		"workspace-read": {
			"cap_artifact_read": {},
		},
		"workspace-edit": {
			"cap_stage":         {},
			"cap_run":           {},
			"cap_exec":          {},
			"cap_artifact_read": {},
			"cap_backend":       {},
			"promotion":         {},
			"cap_other":         {},
			"always_denied":     {},
		},
		"workspace-test": {
			"cap_stage":         {},
			"cap_run":           {},
			"cap_exec":          {},
			"cap_artifact_read": {},
			"cap_backend":       {},
			"promotion":         {},
			"cap_other":         {},
			"always_denied":     {},
		},
	}
}

func computeCapabilitySets(decoded *decodedCompileInputs, invariants FixedInvariants) ([]string, []string, []string, []string) {
	roleCaps := sortedUnique(decoded.roleManifest.CapabilityOptIns)
	runCaps := sortedUnique(decoded.runManifest.CapabilityOptIns)
	stageCaps := []string{}
	if decoded.stageManifest != nil {
		stageCaps = sortedUnique(decoded.stageManifest.CapabilityOptIns)
	}
	effectiveCaps := intersect(roleCaps, runCaps)
	if decoded.stageManifest != nil {
		effectiveCaps = intersect(effectiveCaps, stageCaps)
	}
	effectiveCaps = without(effectiveCaps, invariants.DeniedCapabilities)
	return roleCaps, runCaps, stageCaps, effectiveCaps
}

func buildEffectivePolicyContext(decoded *decodedCompileInputs, invariants FixedInvariants, roleCaps, runCaps, stageCaps, effectiveCaps, activeAllowlistRefs []string) EffectivePolicyContext {
	context := EffectivePolicyContext{
		SchemaID:              "runecode.policyengine.v0.EffectivePolicyContext",
		SchemaVersion:         "0.1.0",
		FixedInvariants:       normalizeInvariants(invariants),
		ActiveRoleFamily:      decoded.roleManifest.RoleFamily,
		ActiveRoleKind:        decoded.roleManifest.RoleKind,
		ApprovalProfile:       ApprovalProfile(decoded.roleManifest.ApprovalProfile),
		RoleManifestHash:      digestToStruct(decoded.roleHash),
		RunManifestHash:       digestToStruct(decoded.runHash),
		RoleManifestSignerIDs: decoded.roleSignerIDs,
		RunManifestSignerIDs:  decoded.runSignerIDs,
		RoleCapabilities:      roleCaps,
		RunCapabilities:       runCaps,
		StageCapabilities:     stageCaps,
		EffectiveCapabilities: effectiveCaps,
		ActiveAllowlistRefs:   activeAllowlistRefs,
	}
	if decoded.stageHash != nil {
		d := digestToStruct(*decoded.stageHash)
		context.StageManifestHash = &d
		context.StageManifestSignerIDs = decoded.stageSignerIDs
	}
	if decoded.ruleSetHash != "" {
		context.EvaluationRuleSetHash = decoded.ruleSetHash
		context.EvaluationRuleSetSchema = policyRuleSetSchemaID
	}
	return context
}

func buildPolicyInputHashes(decoded *decodedCompileInputs, activeAllowlistRefs []string) []string {
	hashes := []string{decoded.roleHash, decoded.runHash}
	if decoded.stageHash != nil {
		hashes = append(hashes, *decoded.stageHash)
	}
	hashes = append(hashes, activeAllowlistRefs...)
	if decoded.ruleSetHash != "" {
		hashes = append(hashes, decoded.ruleSetHash)
	}
	return sortedUnique(hashes)
}

func buildCompiledContext(context EffectivePolicyContext, manifestHash string, policyInputHashes []string, allowlistsByHash map[string]PolicyAllowlist, activeAllowlistRefs []string, ruleSet *PolicyRuleSet) *CompiledContext {
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
	return compiled
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
