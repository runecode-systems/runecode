package policyengine

import (
	"encoding/json"
	"testing"
)

func mustCompile(t *testing.T, input CompileInput) *CompiledContext {
	t.Helper()
	compiled, err := Compile(input)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	return compiled
}

func compileInputWithOneCapability(capability string) CompileInput {
	role := validRoleManifestPayload()
	role["capability_opt_ins"] = []any{capability}
	run := validRunCapabilityManifestPayload()
	run["capability_opt_ins"] = []any{capability}
	stage := validStageCapabilityManifestPayload()
	stage["capability_opt_ins"] = []any{capability}
	return CompileInput{
		FixedInvariants: FixedInvariants{},
		RoleManifest:    mustManifestInput(role),
		RunManifest:     mustManifestInput(run),
		StageManifest:   ptr(mustManifestInput(stage)),
		Allowlists: []ManifestInput{
			mustManifestInput(validAllowlistPayload("allowlist-a")),
			mustManifestInput(validAllowlistPayload("allowlist-b")),
			mustManifestInput(validAllowlistPayload("allowlist-c")),
		},
	}
}

func mustManifestInput(value map[string]any) ManifestInput {
	b, _ := json.Marshal(value)
	h, _ := canonicalHashBytes(b)
	return ManifestInput{Payload: b, ExpectedHash: h}
}

func testManifestInput(t *testing.T, value map[string]any, expectedHash string) ManifestInput {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if expectedHash == "" {
		expectedHash, err = canonicalHashBytes(b)
		if err != nil {
			t.Fatalf("canonicalHashBytes returned error: %v", err)
		}
	}
	return ManifestInput{Payload: b, ExpectedHash: expectedHash}
}

func mustAllowlistHash(value map[string]any) string {
	b, _ := json.Marshal(value)
	h, _ := canonicalHashBytes(b)
	return h
}

func mustDigestObject(identity string) map[string]any {
	return map[string]any{"hash_alg": "sha256", "hash": identity[len("sha256:"):]}
}

func toAnySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out
}

func ptr[T any](v T) *T { return &v }
