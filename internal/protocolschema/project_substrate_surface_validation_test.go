package protocolschema

import "testing"

func TestProjectSubstrateBrokerSurfaceSchemasValidateLifecycleContracts(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))

	postureGetRequestSchema := mustCompileObjectSchema(t, bundle, "objects/ProjectSubstratePostureGetRequest.schema.json")
	postureGetResponseSchema := mustCompileObjectSchema(t, bundle, "objects/ProjectSubstratePostureGetResponse.schema.json")
	upgradePreviewRequestSchema := mustCompileObjectSchema(t, bundle, "objects/ProjectSubstrateUpgradePreviewRequest.schema.json")
	upgradePreviewResponseSchema := mustCompileObjectSchema(t, bundle, "objects/ProjectSubstrateUpgradePreviewResponse.schema.json")
	upgradeApplyRequestSchema := mustCompileObjectSchema(t, bundle, "objects/ProjectSubstrateUpgradeApplyRequest.schema.json")
	upgradeApplyResponseSchema := mustCompileObjectSchema(t, bundle, "objects/ProjectSubstrateUpgradeApplyResponse.schema.json")

	assertSchemaValid(t, postureGetRequestSchema, validProjectSubstratePostureGetRequest(), "ProjectSubstratePostureGetRequest")
	assertSchemaValid(t, postureGetResponseSchema, validProjectSubstratePostureGetResponse(), "ProjectSubstratePostureGetResponse")
	assertSchemaValid(t, upgradePreviewRequestSchema, validProjectSubstrateUpgradePreviewRequest(), "ProjectSubstrateUpgradePreviewRequest")
	assertSchemaValid(t, upgradePreviewResponseSchema, validProjectSubstrateUpgradePreviewResponse(), "ProjectSubstrateUpgradePreviewResponse")
	assertSchemaValid(t, upgradeApplyRequestSchema, validProjectSubstrateUpgradeApplyRequest(), "ProjectSubstrateUpgradeApplyRequest")
	assertSchemaValid(t, upgradeApplyResponseSchema, validProjectSubstrateUpgradeApplyResponse(), "ProjectSubstrateUpgradeApplyResponse")
}

func validProjectSubstratePostureGetRequest() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.ProjectSubstratePostureGetRequest",
		"schema_version": "0.1.0",
		"request_id":     "req-project-substrate-posture-get",
	}
}

func validProjectSubstratePostureGetResponse() map[string]any {
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.ProjectSubstratePostureGetResponse",
		"schema_version":       "0.1.0",
		"request_id":           "req-project-substrate-posture-get",
		"repository_root":      "/repo",
		"contract":             validProjectSubstrateContractState(),
		"snapshot":             validProjectSubstrateValidationSnapshot(),
		"posture_summary":      validProjectSubstratePostureSummary(),
		"adoption":             validProjectSubstrateAdoptionResult(),
		"init_preview":         validProjectSubstrateInitPreview(),
		"upgrade_preview":      validProjectSubstrateUpgradePreview(),
		"remediation_guidance": []any{"upgrade_preview_available", "review_preview", "apply_reviewed_upgrade", "revalidate_project_substrate"},
	}
}

func validProjectSubstrateContractState() map[string]any {
	return map[string]any{
		"schema_id":                 "runecode.protocol.v0.ProjectSubstrateContractState",
		"schema_version":            "0.1.0",
		"contract_id":               "runecode.runecontext.project_substrate.v0",
		"contract_version":          "v0",
		"required_config_path":      "runecontext.yaml",
		"required_source_path":      "runecontext",
		"required_assurance_path":   "runecontext/assurance",
		"required_assurance_tier":   "verified",
		"repo_root_authority":       "explicit_config",
		"forbid_private_truth_copy": true,
	}
}

func validProjectSubstratePostureSummary() map[string]any {
	return map[string]any{
		"schema_id":                         "runecode.protocol.v0.ProjectSubstratePostureSummary",
		"schema_version":                    "0.1.0",
		"active_contract_id":                "runecode.runecontext.project_substrate.v0",
		"active_contract_version":           "v0",
		"active_runecontext_version":        "0.1.0-alpha.13",
		"contract_id":                       "runecode.runecontext.project_substrate.v0",
		"contract_version":                  "v0",
		"validation_state":                  "valid",
		"compatibility_posture":             "supported_with_upgrade_available",
		"normal_operation_allowed":          true,
		"supported_contract_version_min":    "v0",
		"supported_contract_version_max":    "v0",
		"recommended_contract_version":      "v0",
		"supported_runecontext_version_min": "0.1.0-alpha.13",
		"supported_runecontext_version_max": "0.1.0-alpha.16",
		"recommended_runecontext_version":   "0.1.0-alpha.14",
		"reason_codes":                      []any{"project_substrate_upgrade_available"},
		"validated_snapshot_digest":         "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"project_context_identity_digest":   "sha256:1111111111111111111111111111111111111111111111111111111111111111",
	}
}

func validProjectSubstrateAdoptionResult() map[string]any {
	return map[string]any{
		"schema_id":       "runecode.protocol.v0.ProjectSubstrateAdoptionResult",
		"schema_version":  "0.1.0",
		"repository_root": "/repo",
		"status":          "adopted",
		"snapshot":        validProjectSubstrateValidationSnapshot(),
	}
}

func validProjectSubstrateInitPreview() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.ProjectSubstrateInitPreview",
		"schema_version":    "0.1.0",
		"repository_root":   "/repo",
		"status":            "noop",
		"current_snapshot":  validProjectSubstrateValidationSnapshot(),
		"expected_snapshot": validProjectSubstrateValidationSnapshot(),
		"preview_token":     "sha256:2222222222222222222222222222222222222222222222222222222222222222",
	}
}

func validProjectSubstrateUpgradePreviewRequest() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.ProjectSubstrateUpgradePreviewRequest",
		"schema_version": "0.1.0",
		"request_id":     "req-project-substrate-upgrade-preview",
	}
}

func validProjectSubstrateUpgradePreviewResponse() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.ProjectSubstrateUpgradePreviewResponse",
		"schema_version": "0.1.0",
		"request_id":     "req-project-substrate-upgrade-preview",
		"preview":        validProjectSubstrateUpgradePreview(),
	}
}

func validProjectSubstrateUpgradeApplyRequest() map[string]any {
	return map[string]any{
		"schema_id":               "runecode.protocol.v0.ProjectSubstrateUpgradeApplyRequest",
		"schema_version":          "0.1.0",
		"request_id":              "req-project-substrate-upgrade-apply",
		"expected_preview_digest": "sha256:3333333333333333333333333333333333333333333333333333333333333333",
	}
}

func validProjectSubstrateUpgradeApplyResponse() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.ProjectSubstrateUpgradeApplyResponse",
		"schema_version": "0.1.0",
		"request_id":     "req-project-substrate-upgrade-apply",
		"apply_result": map[string]any{
			"schema_id":       "runecode.protocol.v0.ProjectSubstrateUpgradeApplyResult",
			"schema_version":  "0.1.0",
			"repository_root": "/repo",
			"status":          "applied",
			"applied_changes": []any{
				map[string]any{"path": "runecontext.yaml", "action": "update", "before_content_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "after_content_sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
			},
			"current_snapshot":   validProjectSubstrateValidationSnapshot(),
			"resulting_snapshot": validProjectSubstrateValidationSnapshot(),
			"preview_digest":     "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		},
	}
}

func validProjectSubstrateUpgradePreview() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.ProjectSubstrateUpgradePreview",
		"schema_version":    "0.1.0",
		"repository_root":   "/repo",
		"status":            "ready_for_apply",
		"reason_codes":      []any{"upgrade_apply_explicit_required"},
		"current_snapshot":  validProjectSubstrateValidationSnapshot(),
		"expected_snapshot": validProjectSubstrateValidationSnapshot(),
		"file_changes": []any{
			map[string]any{"path": "runecontext.yaml", "action": "update", "before_content_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "after_content_sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		},
		"required_follow_up": []any{"review_preview", "apply_reviewed_upgrade", "revalidate_project_substrate"},
		"preview_digest":     "sha256:3333333333333333333333333333333333333333333333333333333333333333",
	}
}

func validProjectSubstrateValidationSnapshot() map[string]any {
	return map[string]any{
		"schema_id":                       "runecode.protocol.v0.ProjectSubstrateValidationSnapshot",
		"schema_version":                  "0.1.0",
		"contract":                        validProjectSubstrateContractState(),
		"validation_state":                "valid",
		"runecontext_version":             "0.1.0-alpha.13",
		"declared_assurance_tier":         "verified",
		"declared_source_type":            "embedded",
		"declared_source_path":            "runecontext",
		"snapshot_digest":                 "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"validated_snapshot_digest":       "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"project_context_identity_digest": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"anchors": map[string]any{
			"has_config_anchor":      true,
			"has_source_anchor":      true,
			"has_assurance_anchor":   true,
			"has_assurance_baseline": true,
			"has_verified_posture":   true,
			"has_canonical_source":   true,
			"has_private_truth_copy": false,
		},
	}
}
