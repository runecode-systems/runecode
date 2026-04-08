package policyengine

import (
	"errors"
	"testing"
)

func TestCompileBuildsEffectiveContextWithFrozenPrecedenceAndHashes(t *testing.T) {
	input := CompileInput{
		FixedInvariants: FixedInvariants{DeniedCapabilities: []string{"always_denied"}, DeniedActionKinds: []string{"backend_posture_change"}},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		StageManifest:   ptr(testManifestInput(t, validStageCapabilityManifestPayload(), "")),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-c"), ""),
		},
	}

	compiled, err := Compile(input)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if compiled.ManifestHash == "" {
		t.Fatal("ManifestHash must be set")
	}
	if len(compiled.PolicyInputHashes) != 6 {
		t.Fatalf("PolicyInputHashes len = %d, want 6", len(compiled.PolicyInputHashes))
	}
	if got := compiled.Context.EffectiveCapabilities; len(got) != 1 || got[0] != "cap_stage" {
		t.Fatalf("EffectiveCapabilities = %v, want [cap_stage]", got)
	}
	if got := compiled.Context.ActiveAllowlistRefs; len(got) != 3 {
		t.Fatalf("ActiveAllowlistRefs len = %d, want 3", len(got))
	}
}

func TestCompileFailsClosedWhenActiveAllowlistMissing(t *testing.T) {
	input := CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists:      []ManifestInput{},
	}

	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	var evalErr *EvaluationError
	ok := errors.As(err, &evalErr)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerLimitPolicyReject {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerLimitPolicyReject)
	}
}

func TestCompileRejectsSchemaVersionMismatch(t *testing.T) {
	role := validRoleManifestPayload()
	role["schema_version"] = "9.9.9"
	input := CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, role, ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}

	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	var evalErr *EvaluationError
	ok := errors.As(err, &evalErr)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeUnsupportedSchemaVersion {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeUnsupportedSchemaVersion)
	}
}

func TestCompileFailsClosedOnUnknownGatewayScopeKind(t *testing.T) {
	allowlist := validAllowlistPayload("allowlist-a")
	entries := allowlist["entries"].([]any)
	rule := entries[0].(map[string]any)
	rule["scope_kind"] = "gateway_destination_legacy"

	input := CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, allowlist, ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}

	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	var evalErr *EvaluationError
	ok := errors.As(err, &evalErr)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema && evalErr.Code != ErrCodeBrokerValidationOperation {
		t.Fatalf("error code = %q, want schema/operation validation fail-closed", evalErr.Code)
	}
}

func TestCompileFailsClosedOnUnknownDestinationDescriptorKind(t *testing.T) {
	allowlist := validAllowlistPayload("allowlist-a")
	entries := allowlist["entries"].([]any)
	rule := entries[0].(map[string]any)
	destination := rule["destination"].(map[string]any)
	destination["descriptor_kind"] = "raw_url"

	input := CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, allowlist, ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}

	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	var evalErr *EvaluationError
	ok := errors.As(err, &evalErr)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationSchema && evalErr.Code != ErrCodeBrokerValidationOperation {
		t.Fatalf("error code = %q, want schema/operation validation fail-closed", evalErr.Code)
	}
}

func TestCompileFailsClosedOnUnknownApprovalProfile(t *testing.T) {
	role := validRoleManifestPayload()
	role["approval_profile"] = "legacy"
	input := CompileInput{
		RoleManifest: testManifestInput(t, role, ""),
		RunManifest:  testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}
	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
	evalErr, ok := err.(*EvaluationError)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationOperation {
		if evalErr.Code != ErrCodeBrokerValidationSchema {
			t.Fatalf("error code = %q, want fail-closed validation rejection", evalErr.Code)
		}
	}
}
