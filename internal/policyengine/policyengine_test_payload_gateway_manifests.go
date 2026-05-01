package policyengine

func compileGatewayInputWithOneCapability(roleKind string, capability string, allowlist map[string]any) CompileInput {
	role := gatewayCapabilityRoleManifest(roleKind, capability, allowlist)
	run := gatewayCapabilityRunManifest(roleKind, capability, allowlist)

	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    testManifestInput(nil, role, ""),
		RunManifest:     testManifestInput(nil, run, ""),
		Allowlists:      []ManifestInput{testManifestInput(nil, allowlist, "")},
	}
}

func gatewayCapabilityRoleManifest(roleKind string, capability string, allowlist map[string]any) map[string]any {
	role := validRoleManifestPayload()
	role["role_family"] = "gateway"
	role["role_kind"] = roleKind
	role["capability_opt_ins"] = []any{capability}
	rolePrincipal := role["principal"].(map[string]any)
	rolePrincipal["role_family"] = "gateway"
	rolePrincipal["role_kind"] = roleKind
	role["allowlist_refs"] = []any{mustDigestObject(testAllowlistHash(nil, allowlist))}
	return role
}

func gatewayCapabilityRunManifest(roleKind string, capability string, allowlist map[string]any) map[string]any {
	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{capability}
	runPrincipal := run["principal"].(map[string]any)
	runPrincipal["role_family"] = "gateway"
	runPrincipal["role_kind"] = roleKind
	run["allowlist_refs"] = []any{mustDigestObject(testAllowlistHash(nil, allowlist))}
	return run
}
