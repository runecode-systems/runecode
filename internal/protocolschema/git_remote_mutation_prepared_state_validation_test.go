package protocolschema

import "testing"

func TestGitRemoteMutationPreparedStateLegacyExecutedCompatibility(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/GitRemoteMutationPreparedState.schema.json")

	legacyExecuted := validGitRemoteMutationPreparedState()
	legacyExecuted["lifecycle_state"] = "executed"
	legacyExecuted["execution_state"] = "completed"
	legacyExecuted["last_execute_request_id"] = "req-git-execute-legacy"
	delete(legacyExecuted, "last_execute_provider_auth_lease_id")
	delete(legacyExecuted, "last_execute_attempt_id")
	delete(legacyExecuted, "last_execute_attempt_typed_request_hash")
	assertSchemaValid(t, schema, legacyExecuted, "GitRemoteMutationPreparedState legacy executed compatibility")

	partialCurrent := validGitRemoteMutationPreparedState()
	partialCurrent["lifecycle_state"] = "executing"
	partialCurrent["execution_state"] = "not_started"
	partialCurrent["last_execute_request_id"] = "req-git-execute-current"
	partialCurrent["last_execute_provider_auth_lease_id"] = "lease-git-provider"
	delete(partialCurrent, "last_execute_attempt_id")
	delete(partialCurrent, "last_execute_attempt_typed_request_hash")
	assertSchemaInvalid(t, schema, partialCurrent, "GitRemoteMutationPreparedState accepted partial current execute bindings")

	partialExecuted := validGitRemoteMutationPreparedState()
	partialExecuted["lifecycle_state"] = "executed"
	partialExecuted["execution_state"] = "completed"
	partialExecuted["last_execute_request_id"] = "req-git-execute-current"
	partialExecuted["last_execute_attempt_id"] = "sha256:" + repeatHexNibble("e")
	delete(partialExecuted, "last_execute_provider_auth_lease_id")
	delete(partialExecuted, "last_execute_attempt_typed_request_hash")
	assertSchemaInvalid(t, schema, partialExecuted, "GitRemoteMutationPreparedState accepted partial executed execute bindings")

	fullCurrent := validGitRemoteMutationPreparedState()
	fullCurrent["last_execute_request_id"] = "req-git-execute-current"
	fullCurrent["last_execute_provider_auth_lease_id"] = "lease-git-provider"
	fullCurrent["last_execute_attempt_id"] = "sha256:" + repeatHexNibble("e")
	fullCurrent["last_execute_attempt_typed_request_hash"] = testDigestValue("f")
	assertSchemaValid(t, schema, fullCurrent, "GitRemoteMutationPreparedState full execute bindings")
}

func repeatHexNibble(nibble string) string {
	value := ""
	for len(value) < 64 {
		value += nibble
	}
	return value[:64]
}

func validGitRemoteMutationPreparedState() map[string]any {
	state := validGitRemoteMutationPreparedStateCore()
	state["typed_request"] = validGitRefUpdateRequest()
	state["derived_summary"] = validGitRemoteMutationDerivedSummary()
	return state
}

func validGitRemoteMutationPreparedStateCore() map[string]any {
	return map[string]any{
		"schema_id":                    "runecode.protocol.v0.GitRemoteMutationPreparedState",
		"schema_version":               "0.1.0",
		"prepared_mutation_id":         "prepared-git-mutation-1",
		"run_id":                       "run-1",
		"provider":                     "github",
		"destination_ref":              "github.com/runecode-systems/runecode/refs/heads/main",
		"request_kind":                 "git_ref_update",
		"typed_request_schema_id":      "runecode.protocol.v0.GitRefUpdateRequest",
		"typed_request_schema_version": "0.1.0",
		"typed_request_hash":           testDigestValue("a"),
		"action_request_hash":          testDigestValue("b"),
		"policy_decision_hash":         testDigestValue("c"),
		"lifecycle_state":              "prepared",
		"execution_state":              "not_started",
		"created_at":                   "2026-05-03T00:00:00Z",
		"updated_at":                   "2026-05-03T00:00:00Z",
		"last_prepare_request_id":      "req-git-prepare",
	}
}

func validGitRefUpdateRequest() map[string]any {
	return map[string]any{
		"schema_id":                         "runecode.protocol.v0.GitRefUpdateRequest",
		"schema_version":                    "0.1.0",
		"request_kind":                      "git_ref_update",
		"repository_identity":               validGitRemoteDestinationDescriptor(),
		"target_ref":                        "refs/heads/main",
		"expected_old_ref_hash":             testDigestValue("1"),
		"referenced_patch_artifact_digests": []any{testDigestValue("2")},
		"commit_intent":                     validGitCommitIntent(),
		"expected_result_tree_hash":         testDigestValue("3"),
		"allow_force_push":                  false,
		"allow_ref_deletion":                false,
	}
}

func validGitRemoteMutationDerivedSummary() map[string]any {
	return map[string]any{
		"schema_id":                         "runecode.protocol.v0.GitRemoteMutationDerivedSummary",
		"schema_version":                    "0.1.0",
		"repository_identity":               "github.com/runecode-systems/runecode",
		"target_refs":                       []any{"refs/heads/main"},
		"referenced_patch_artifact_digests": []any{testDigestValue("4")},
		"expected_result_tree_hash":         testDigestValue("d"),
	}
}

func validGitRemoteDestinationDescriptor() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.DestinationDescriptor",
		"schema_version":           "0.1.0",
		"descriptor_kind":          "git_remote",
		"canonical_host":           "github.com",
		"git_repository_identity":  "github.com/runecode-systems/runecode",
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}

func validGitCommitIntent() map[string]any {
	identity := map[string]any{
		"display_name": "Rune Code",
		"email":        "zeb@runecode.org",
	}
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GitCommitIntent",
		"schema_version": "0.1.0",
		"message": map[string]any{
			"subject": "Apply verification-plane update",
		},
		"trailers": []any{
			map[string]any{"key": "Change-Id", "value": "CHG-2026-059"},
		},
		"author":    identity,
		"committer": identity,
		"signoff":   identity,
	}
}
