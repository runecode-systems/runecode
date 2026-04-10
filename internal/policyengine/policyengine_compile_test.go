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

func TestCompileFailsClosedWhenExpectedHashMissing(t *testing.T) {
	roleInput := testManifestInput(t, validRoleManifestPayload(), "")
	runInput := testManifestInput(t, validRunCapabilityManifestPayload(), "")
	allowlistInput := testManifestInput(t, validAllowlistPayload("allowlist-a"), "")
	roleInput.ExpectedHash = ""
	input := CompileInput{
		RoleManifest: roleInput,
		RunManifest:  runInput,
		Allowlists:   []ManifestInput{allowlistInput},
	}
	_, err := Compile(input)
	if err == nil {
		t.Fatal("Compile returned nil error, want failure when expected_hash is empty")
	}
	evalErr, ok := err.(*EvaluationError)
	if !ok {
		t.Fatalf("error type = %T, want *EvaluationError", err)
	}
	if evalErr.Code != ErrCodeBrokerValidationOperation {
		t.Fatalf("error code = %q, want %q", evalErr.Code, ErrCodeBrokerValidationOperation)
	}
}

func TestCompileWorkspaceRoleCapabilityManifestPolicyByRoleKind(t *testing.T) {
	tests := []struct {
		name       string
		roleKind   string
		capability string
	}{
		{name: "workspace-read allows explicit read capability", roleKind: "workspace-read", capability: "cap_artifact_read"},
		{name: "workspace-edit allows explicit edit capability", roleKind: "workspace-edit", capability: "cap_stage"},
		{name: "workspace-test allows explicit test capability", roleKind: "workspace-test", capability: "cap_exec"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			role := validRoleManifestPayload()
			role["role_kind"] = tc.roleKind
			role["capability_opt_ins"] = []any{tc.capability}
			rolePrincipal := role["principal"].(map[string]any)
			rolePrincipal["role_kind"] = tc.roleKind

			run := validRunCapabilityManifestPayload()
			run["capability_opt_ins"] = []any{tc.capability}
			runPrincipal := run["principal"].(map[string]any)
			runPrincipal["role_kind"] = tc.roleKind

			stage := validStageCapabilityManifestPayload()
			stage["capability_opt_ins"] = []any{tc.capability}
			stagePrincipal := stage["principal"].(map[string]any)
			stagePrincipal["role_kind"] = tc.roleKind

			_, err := Compile(CompileInput{
				RoleManifest:  testManifestInput(t, role, ""),
				RunManifest:   testManifestInput(t, run, ""),
				StageManifest: ptr(testManifestInput(t, stage, "")),
				Allowlists: []ManifestInput{
					testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
					testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
					testManifestInput(t, validAllowlistPayload("allowlist-c"), ""),
				},
			})
			if err != nil {
				t.Fatalf("Compile returned error: %v", err)
			}
		})
	}
}

func TestCompileFailsClosedOnCapabilityManifestRoleKindMismatch(t *testing.T) {
	role := validRoleManifestPayload()
	role["role_kind"] = "workspace-edit"
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_kind"] = "workspace-edit"

	run := validRunCapabilityManifestPayload()
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_kind"] = "workspace-test"

	_, err := Compile(CompileInput{
		RoleManifest: testManifestInput(t, role, ""),
		RunManifest:  testManifestInput(t, run, ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	})
	if err == nil {
		t.Fatal("Compile returned nil error, want fail-closed role binding mismatch")
	}
}

func TestCompileFailsClosedWhenRunCapabilityManifestMissingRoleBinding(t *testing.T) {
	run := validRunCapabilityManifestPayload()
	runPrincipal := run["principal"].(map[string]any)
	delete(runPrincipal, "role_kind")

	_, err := Compile(CompileInput{
		RoleManifest: testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:  testManifestInput(t, run, ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	})
	if err == nil {
		t.Fatal("Compile returned nil error, want fail-closed explicit capability-manifest role binding enforcement")
	}
}

func TestCompileFailsClosedOnWorkspaceReadCapabilityEscalation(t *testing.T) {
	role := validRoleManifestPayload()
	role["role_kind"] = "workspace-read"
	role["capability_opt_ins"] = []any{"cap_stage"}
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_kind"] = "workspace-read"

	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{"cap_stage"}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_kind"] = "workspace-read"

	_, err := Compile(CompileInput{
		RoleManifest: testManifestInput(t, role, ""),
		RunManifest:  testManifestInput(t, run, ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	})
	if err == nil {
		t.Fatal("Compile returned nil error, want fail-closed workspace-read capability policy rejection")
	}
}
