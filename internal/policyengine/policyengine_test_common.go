package policyengine

import (
	"encoding/json"
	"fmt"
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
		RoleManifest:    testManifestInput(nil, role, ""),
		RunManifest:     testManifestInput(nil, run, ""),
		StageManifest:   ptr(testManifestInput(nil, stage, "")),
		Allowlists: []ManifestInput{
			testManifestInput(nil, validAllowlistPayload("allowlist-a"), ""),
			testManifestInput(nil, validAllowlistPayload("allowlist-b"), ""),
			testManifestInput(nil, validAllowlistPayload("allowlist-c"), ""),
		},
	}
}

func testManifestInput(t *testing.T, value map[string]any, expectedHash string) ManifestInput {
	if t != nil {
		t.Helper()
	}
	b, err := json.Marshal(value)
	if err != nil {
		failTestOrPanic(t, "Marshal returned error: %v", err)
	}
	if expectedHash == "" {
		expectedHash, err = canonicalHashBytes(b)
		if err != nil {
			failTestOrPanic(t, "canonicalHashBytes returned error: %v", err)
		}
	}
	return ManifestInput{Payload: b, ExpectedHash: expectedHash}
}

func testAllowlistHash(t *testing.T, value map[string]any) string {
	if t != nil {
		t.Helper()
	}
	b, err := json.Marshal(value)
	if err != nil {
		failTestOrPanic(t, "Marshal returned error: %v", err)
	}
	h, err := canonicalHashBytes(b)
	if err != nil {
		failTestOrPanic(t, "canonicalHashBytes returned error: %v", err)
	}
	return h
}

func failTestOrPanic(t *testing.T, format string, args ...any) {
	if t != nil {
		t.Fatalf(format, args...)
	}
	panic(fmt.Sprintf(format, args...))
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
