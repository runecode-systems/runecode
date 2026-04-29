package protocolschema

import "testing"

func TestRuneContextDraftArtifactSchemasValidateMinimalAndRejectInvalid(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	assertDraftChangeArtifactSchema(t, bundle)
	assertDraftSpecArtifactSchema(t, bundle)
}

func assertDraftChangeArtifactSchema(t *testing.T, bundle compiledBundle) {
	t.Helper()
	schema := mustCompileObjectSchema(t, bundle, "objects/RuneContextChangeDraftArtifact.schema.json")
	valid := map[string]any{"schema_id": "runecode.protocol.v0.RuneContextChangeDraftArtifact", "schema_version": "0.1.0", "data_class": "spec_text", "change_id": "CHG-2026-049-gap-closure", "artifact_digest": testDigestValue("1"), "source_prompt_identity_digest": testDigestValue("2")}
	if err := schema.Validate(valid); err != nil {
		t.Fatalf("change draft minimal fixture should be valid: %v", err)
	}
	invalid := map[string]any{"schema_id": "runecode.protocol.v0.RuneContextChangeDraftArtifact", "schema_version": "0.1.0", "data_class": "spec_text", "artifact_digest": testDigestValue("1"), "source_prompt_identity_digest": testDigestValue("2")}
	if err := schema.Validate(invalid); err == nil {
		t.Fatal("change draft missing change_id unexpectedly validated")
	}
}

func assertDraftSpecArtifactSchema(t *testing.T, bundle compiledBundle) {
	t.Helper()
	schema := mustCompileObjectSchema(t, bundle, "objects/RuneContextSpecDraftArtifact.schema.json")
	valid := map[string]any{"schema_id": "runecode.protocol.v0.RuneContextSpecDraftArtifact", "schema_version": "0.1.0", "data_class": "spec_text", "spec_id": "spec-typing-contract-v0", "artifact_digest": testDigestValue("3"), "source_prompt_identity_digest": testDigestValue("4")}
	if err := schema.Validate(valid); err != nil {
		t.Fatalf("spec draft minimal fixture should be valid: %v", err)
	}
	invalid := map[string]any{"schema_id": "runecode.protocol.v0.RuneContextSpecDraftArtifact", "schema_version": "0.1.0", "data_class": "diffs", "spec_id": "spec-typing-contract-v0", "artifact_digest": testDigestValue("3"), "source_prompt_identity_digest": testDigestValue("4")}
	if err := schema.Validate(invalid); err == nil {
		t.Fatal("spec draft wrong data_class unexpectedly validated")
	}
}

func TestRuneContextApprovedImplementationInputSetSchemaValidateMinimalAndRejectInvalid(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	schema := mustCompileObjectSchema(t, bundle, "objects/RuneContextApprovedImplementationInputSet.schema.json")

	valid := map[string]any{
		"schema_id":                          "runecode.protocol.v0.RuneContextApprovedImplementationInputSet",
		"schema_version":                     "0.1.0",
		"input_set_digest":                   testDigestValue("a"),
		"approved_input_digests":             []any{testDigestValue("b")},
		"workflow_definition_hash":           testDigestValue("c"),
		"process_definition_hash":            testDigestValue("d"),
		"approval_profile":                   "moderate",
		"autonomy_posture":                   "operator_guided",
		"validated_project_substrate_digest": testDigestValue("e"),
		"project_substrate_snapshot_digest":  testDigestValue("f"),
		"control_input_digest":               testDigestValue("1"),
		"repo_identity_digest":               testDigestValue("2"),
		"repo_state_identity_digest":         testDigestValue("3"),
	}

	if err := schema.Validate(valid); err != nil {
		t.Fatalf("approved implementation input set minimal fixture should be valid: %v", err)
	}

	invalid := cloneFixtureMap(t, valid)
	invalid["approved_input_digests"] = []any{}
	if err := schema.Validate(invalid); err == nil {
		t.Fatal("approved implementation input set with empty approved_input_digests unexpectedly validated")
	}
}
