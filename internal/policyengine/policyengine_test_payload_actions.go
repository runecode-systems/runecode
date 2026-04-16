package policyengine

import (
	"encoding/json"
	"strings"
)

func newActionRequest(actionKind, capabilityID, payloadSchemaID string, payload map[string]any, roleFamily, roleKind string) ActionRequest {
	return ActionRequest{
		SchemaID:              actionRequestSchemaID,
		SchemaVersion:         actionRequestSchemaVersion,
		ActionKind:            actionKind,
		CapabilityID:          capabilityID,
		ActionPayloadSchemaID: payloadSchemaID,
		ActionPayload:         payload,
		ActorKind:             "daemon",
		RoleFamily:            roleFamily,
		RoleKind:              roleKind,
	}
}

func newSchemaPayload(schemaID string, fields map[string]any) map[string]any {
	payload := map[string]any{
		"schema_id":      schemaID,
		"schema_version": "0.1.0",
	}
	for key, value := range fields {
		payload[key] = value
	}
	return payload
}

func validWorkspaceWriteActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindWorkspaceWrite,
		capabilityID,
		actionPayloadWorkspaceSchemaID,
		newSchemaPayload(actionPayloadWorkspaceSchemaID, map[string]any{
			"target_path": "src/main.go",
			"write_mode":  "update",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validExecutorRunActionRequest(capabilityID string, executorClass string, argv []string) ActionRequest {
	return newActionRequest(
		ActionKindExecutorRun,
		capabilityID,
		actionPayloadExecutorSchemaID,
		newSchemaPayload(actionPayloadExecutorSchemaID, map[string]any{
			"executor_class": executorClass,
			"executor_id":    "workspace-runner",
			"argv":           toAnySlice(argv),
			"network_access": "none",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validArtifactReadActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindArtifactRead,
		capabilityID,
		actionPayloadArtifactSchemaID,
		newSchemaPayload(actionPayloadArtifactSchemaID, map[string]any{
			"artifact_hash": mustDigestObject("sha256:" + strings.Repeat("3", 64)),
			"read_mode":     "head",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validPromotionActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindPromotion,
		capabilityID,
		actionPayloadPromotionSchemaID,
		newSchemaPayload(actionPayloadPromotionSchemaID, map[string]any{
			"promotion_kind":       "excerpt",
			"source_artifact_hash": mustDigestObject("sha256:" + strings.Repeat("4", 64)),
			"target_data_class":    "approved_file_excerpts",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validBackendPostureActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindBackendPosture,
		capabilityID,
		actionPayloadBackendSchemaID,
		newSchemaPayload(actionPayloadBackendSchemaID, map[string]any{
			"target_instance_id":             "launcher-instance-1",
			"target_backend_kind":            "microvm",
			"selection_mode":                 "explicit_selection",
			"change_kind":                    "select_backend",
			"assurance_change_kind":          "maintain_assurance",
			"opt_in_kind":                    "none",
			"reduced_assurance_acknowledged": false,
		}),
		"workspace",
		"workspace-edit",
	)
}

func validGateOverrideActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindGateOverride,
		capabilityID,
		actionPayloadGateSchemaID,
		newSchemaPayload(actionPayloadGateSchemaID, map[string]any{
			"gate_id":                      "policy_engine_gate",
			"gate_kind":                    "policy",
			"gate_version":                 "1.0.0",
			"gate_attempt_id":              "gate-attempt-1",
			"overridden_failed_result_ref": "sha256:" + strings.Repeat("2", 64),
			"policy_context_hash":          "sha256:" + strings.Repeat("3", 64),
			"override_mode":                "break_glass",
			"justification":                "Emergency trust maintenance",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validStageSummarySignOffActionRequest(capabilityID, summaryHash string) ActionRequest {
	relevantDigest := mustDigestObject(summaryHash)
	stageSummary := map[string]any{
		"schema_id":                "runecode.protocol.v0.StageSummary",
		"schema_version":           "0.1.0",
		"run_id":                   "run-1",
		"plan_id":                  "plan-1",
		"stage_id":                 "stage-1",
		"summary_revision":         float64(1),
		"manifest_hash":            mustDigestObject("sha256:" + strings.Repeat("1", 64)),
		"stage_capability_context": map[string]any{},
		"requested_high_risk_capability_categories": []any{"stage_sign_off"},
		"requested_scope_change_types":              []any{},
		"relevant_artifact_hashes":                  []any{relevantDigest},
	}
	stageSummaryBytes, err := json.Marshal(stageSummary)
	if err != nil {
		panic(err)
	}
	canonicalSummaryHash, err := canonicalHashBytes(stageSummaryBytes)
	if err != nil {
		panic(err)
	}
	return newActionRequest(
		ActionKindStageSummarySign,
		capabilityID,
		actionPayloadStageSchemaID,
		newSchemaPayload(actionPayloadStageSchemaID, map[string]any{
			"run_id":             "run-1",
			"stage_id":           "stage-1",
			"stage_summary":      stageSummary,
			"stage_summary_hash": mustDigestObject(canonicalSummaryHash),
			"approval_profile":   "moderate",
			"summary_revision":   float64(1),
		}),
		"workspace",
		"workspace-edit",
	)
}
