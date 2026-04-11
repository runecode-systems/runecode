package policyengine

import "testing"

func compileInputWithThreeAllowlistsAndStage(t *testing.T) CompileInput {
	t.Helper()
	return CompileInput{
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
}

func compileInputWithNoAllowlists(t *testing.T) CompileInput {
	t.Helper()
	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists:      []ManifestInput{},
	}
}

func compileInputWithRoleAndTwoAllowlists(t *testing.T, role map[string]any) CompileInput {
	t.Helper()
	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, role, ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}
}

func compileInputWithAllowlistOverrideAndSecondary(t *testing.T, allowlist map[string]any) CompileInput {
	t.Helper()
	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(t, validRoleManifestPayload(), ""),
		RunManifest:     testManifestInput(t, validRunCapabilityManifestPayload(), ""),
		Allowlists: []ManifestInput{
			testManifestInput(t, allowlist, ""),
			testManifestInput(t, validAllowlistPayload("allowlist-b"), ""),
		},
	}
}

func compileWorkspaceRoleCapabilityManifestByRoleKind(t *testing.T, roleKind, capability string) error {
	t.Helper()
	role := validRoleManifestPayload()
	role["role_kind"] = roleKind
	role["capability_opt_ins"] = []any{capability}
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_kind"] = roleKind

	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{capability}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_kind"] = roleKind

	stage := validStageCapabilityManifestPayload()
	stage["capability_opt_ins"] = []any{capability}
	stagePrincipal := stage["principal"].(map[string]any)
	stagePrincipal["role_kind"] = roleKind

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
	return err
}
