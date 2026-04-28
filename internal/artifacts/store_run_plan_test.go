package artifacts

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const storeRunPlanProjectContextDigest = "sha256:9999999999999999999999999999999999999999999999999999999999999999"

func TestRecordRunPlanAuthorityAndActiveLookupWithSupersession(t *testing.T) {
	store := newTestStore(t)
	runID := "run-run-plan-active"
	plan1 := putRunPlanArtifactForStoreTest(t, store, runID, "plan-1", "")
	if err := store.RecordRunPlanAuthority(
		newRunPlanAuthorityRecordForStoreTest(runID, "plan-1", "", plan1.Digest),
		newRunPlanCompilationRecordForStoreTest(runID, "plan-1", "", plan1.Digest),
	); err != nil {
		t.Fatalf("RecordRunPlanAuthority(plan-1) returned error: %v", err)
	}
	plan2 := putRunPlanArtifactForStoreTest(t, store, runID, "plan-2", "plan-1")
	if err := store.RecordRunPlanAuthority(
		newRunPlanAuthorityRecordForStoreTest(runID, "plan-2", "plan-1", plan2.Digest),
		newRunPlanCompilationRecordForStoreTest(runID, "plan-2", "plan-1", plan2.Digest),
	); err != nil {
		t.Fatalf("RecordRunPlanAuthority(plan-2) returned error: %v", err)
	}
	active, ok, err := store.ActiveRunPlanAuthority(runID)
	if err != nil {
		t.Fatalf("ActiveRunPlanAuthority returned error: %v", err)
	}
	if !ok {
		t.Fatal("ActiveRunPlanAuthority ok=false, want true")
	}
	if active.PlanID != "plan-2" {
		t.Fatalf("active.PlanID = %q, want plan-2", active.PlanID)
	}
}

func TestDeleteDigestRemovesRunPlanAuthorityAndCompilation(t *testing.T) {
	store := newTestStore(t)
	runID := "run-run-plan-delete"
	plan := putRunPlanArtifactForStoreTest(t, store, runID, "plan-delete", "")
	if err := store.RecordRunPlanAuthority(
		newRunPlanAuthorityRecordForStoreTest(runID, "plan-delete", "", plan.Digest),
		newRunPlanCompilationRecordForStoreTest(runID, "plan-delete", "", plan.Digest),
	); err != nil {
		t.Fatalf("RecordRunPlanAuthority returned error: %v", err)
	}
	if err := store.DeleteDigest(plan.Digest); err != nil {
		t.Fatalf("DeleteDigest returned error: %v", err)
	}
	if _, ok := store.RunPlanAuthority(runID, "plan-delete"); ok {
		t.Fatal("RunPlanAuthority still present after DeleteDigest")
	}
	if _, ok := store.RunPlanCompilationRecord(runID, "plan-delete"); ok {
		t.Fatal("RunPlanCompilationRecord still present after DeleteDigest")
	}
	if _, ok, err := store.ActiveRunPlanAuthority(runID); err != nil || ok {
		t.Fatalf("ActiveRunPlanAuthority after delete = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestRecordRunPlanAuthorityRejectsMissingRunPlanArtifact(t *testing.T) {
	store := newTestStore(t)
	runID := "run-run-plan-missing-artifact"
	digest := "sha256:" + strings.Repeat("a", 64)
	err := store.RecordRunPlanAuthority(
		newRunPlanAuthorityRecordForStoreTest(runID, "plan-missing", "", digest),
		newRunPlanCompilationRecordForStoreTest(runID, "plan-missing", "", digest),
	)
	if err == nil {
		t.Fatal("RecordRunPlanAuthority error = nil, want missing artifact rejection")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("RecordRunPlanAuthority error = %v, want not found", err)
	}
}

func TestRecordRunPlanAuthorityRejectsAuthorityCompilationMismatch(t *testing.T) {
	store := newTestStore(t)
	runID := "run-run-plan-mismatch"
	plan := putRunPlanArtifactForStoreTest(t, store, runID, "plan-mismatch", "")
	authority := newRunPlanAuthorityRecordForStoreTest(runID, "plan-mismatch", "", plan.Digest)
	compilation := newRunPlanCompilationRecordForStoreTest(runID, "plan-mismatch", "", plan.Digest)
	compilation.ProcessDefinitionHash = "sha256:" + strings.Repeat("6", 64)

	err := store.RecordRunPlanAuthority(authority, compilation)
	if err == nil {
		t.Fatal("RecordRunPlanAuthority error = nil, want mismatch rejection")
	}
	if !strings.Contains(err.Error(), "process_definition_hash mismatch") {
		t.Fatalf("RecordRunPlanAuthority error = %v, want process_definition_hash mismatch", err)
	}
}

func TestActiveRunPlanAuthorityRejectsAmbiguousActivePlans(t *testing.T) {
	store := newTestStore(t)
	runID := "run-run-plan-ambiguous"
	planA := putRunPlanArtifactForStoreTest(t, store, runID, "plan-a", "")
	if err := store.RecordRunPlanAuthority(
		newRunPlanAuthorityRecordForStoreTest(runID, "plan-a", "", planA.Digest),
		newRunPlanCompilationRecordForStoreTest(runID, "plan-a", "", planA.Digest),
	); err != nil {
		t.Fatalf("RecordRunPlanAuthority(plan-a) returned error: %v", err)
	}
	planB := putRunPlanArtifactForStoreTest(t, store, runID, "plan-b", "")
	if err := store.RecordRunPlanAuthority(
		newRunPlanAuthorityRecordForStoreTest(runID, "plan-b", "", planB.Digest),
		newRunPlanCompilationRecordForStoreTest(runID, "plan-b", "", planB.Digest),
	); err != nil {
		t.Fatalf("RecordRunPlanAuthority(plan-b) returned error: %v", err)
	}

	_, ok, err := store.ActiveRunPlanAuthority(runID)
	if err == nil {
		t.Fatal("ActiveRunPlanAuthority error = nil, want ambiguity failure")
	}
	if ok {
		t.Fatal("ActiveRunPlanAuthority ok=true, want false on ambiguity")
	}
	if !strings.Contains(err.Error(), "ambiguous active trusted run plan authority") {
		t.Fatalf("ActiveRunPlanAuthority error = %v, want ambiguity message", err)
	}
}

func putRunPlanArtifactForStoreTest(t *testing.T, store *Store, runID, planID, supersedesPlanID string) ArtifactReference {
	t.Helper()
	payloadObj := runPlanArtifactPayloadForStoreTest(runID, planID)
	if strings.TrimSpace(supersedesPlanID) != "" {
		payloadObj["supersedes_plan_id"] = strings.TrimSpace(supersedesPlanID)
	}
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		t.Fatalf("Marshal run plan artifact returned error: %v", err)
	}
	ref, err := store.Put(PutRequest{Payload: payload, ContentType: "application/json", DataClass: DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("2", 64), CreatedByRole: "brokerapi", TrustedSource: true, RunID: runID, StepID: "compiled_run_plan/" + planID})
	if err != nil {
		t.Fatalf("Put run plan artifact returned error: %v", err)
	}
	return ref
}

func runPlanArtifactPayloadForStoreTest(runID, planID string) map[string]any {
	return map[string]any{
		"schema_id":                       "runecode.protocol.v0.RunPlan",
		"schema_version":                  "0.4.0",
		"plan_id":                         planID,
		"run_id":                          runID,
		"project_context_identity_digest": storeRunPlanProjectContextDigest,
		"workflow_id":                     "workflow_main",
		"workflow_version":                "1.0.0",
		"process_id":                      "process_default",
		"approval_profile":                "moderate",
		"autonomy_posture":                "balanced",
		"workflow_definition_hash":        "sha256:" + strings.Repeat("3", 64),
		"process_definition_hash":         "sha256:" + strings.Repeat("4", 64),
		"policy_context_hash":             "sha256:" + strings.Repeat("5", 64),
		"compiled_at":                     "2026-04-05T10:00:00Z",
		"role_instance_ids":               []any{"workspace_editor_1"},
		"executor_bindings": []any{map[string]any{
			"binding_id":         "binding_workspace_runner",
			"executor_id":        "workspace-runner",
			"executor_class":     "workspace_ordinary",
			"allowed_role_kinds": []any{"workspace-edit", "workspace-test"},
		}},
		"gate_definitions": []any{gateDefinitionPayloadForStoreTest()},
		"dependency_edges": []any{},
		"entries":          []any{runPlanEntryPayloadForStoreTest()},
	}
}

func gateDefinitionPayloadForStoreTest() map[string]any {
	return map[string]any{
		"schema_id":           "runecode.protocol.v0.GateDefinition",
		"schema_version":      "0.2.0",
		"checkpoint_code":     "step_validation_started",
		"order_index":         0,
		"stage_id":            "validation",
		"step_id":             "validation_policy",
		"role_instance_id":    "workspace_editor_1",
		"executor_binding_id": "binding_workspace_runner",
		"gate":                gateContractPayloadForStoreTest(),
	}
}

func runPlanEntryPayloadForStoreTest() map[string]any {
	return map[string]any{
		"entry_id":             "validation_policy",
		"entry_kind":           "gate",
		"order_index":          0,
		"stage_id":             "validation",
		"step_id":              "validation_policy",
		"role_instance_id":     "workspace_editor_1",
		"executor_binding_id":  "binding_workspace_runner",
		"checkpoint_code":      "step_validation_started",
		"gate":                 gateContractPayloadForStoreTest(),
		"depends_on_entry_ids": []any{},
		"blocks_entry_ids":     []any{},
		"supported_wait_kinds": []any{"waiting_operator_input", "waiting_approval"},
	}
}

func gateContractPayloadForStoreTest() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GateContract",
		"schema_version": "0.1.0",
		"gate_id":        "policy_gate",
		"gate_kind":      "policy",
		"gate_version":   "1.0.0",
		"normalized_inputs": []any{map[string]any{
			"input_id":     "policy_context",
			"input_digest": "sha256:" + strings.Repeat("a", 64),
		}},
		"plan_binding":       map[string]any{"checkpoint_code": "step_validation_started", "order_index": 0},
		"retry_semantics":    map[string]any{"retry_mode": "new_attempt_required", "max_attempts": 2},
		"override_semantics": map[string]any{"override_mode": "policy_action_required", "action_kind": "action_gate_override", "approval_trigger_code": "gate_override"},
	}
}

func newRunPlanAuthorityRecordForStoreTest(runID, planID, supersedesPlanID, digest string) RunPlanAuthorityRecord {
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	return RunPlanAuthorityRecord{
		RunID:                        runID,
		PlanID:                       planID,
		SupersedesPlanID:             supersedesPlanID,
		RunPlanDigest:                digest,
		WorkflowDefinitionHash:       "sha256:" + strings.Repeat("3", 64),
		ProcessDefinitionHash:        "sha256:" + strings.Repeat("4", 64),
		PolicyContextHash:            "sha256:" + strings.Repeat("5", 64),
		ProjectContextIdentityDigest: storeRunPlanProjectContextDigest,
		CompiledAt:                   now,
		RecordedAt:                   now,
		Entries: []RunPlanGateEntryRecord{{
			EntryID:                 "validation_policy",
			EntryKind:               "gate",
			PlanCheckpointCode:      "step_validation_started",
			PlanOrderIndex:          0,
			GateID:                  "policy_gate",
			GateKind:                "policy",
			GateVersion:             "1.0.0",
			StageID:                 "validation",
			StepID:                  "validation_policy",
			RoleInstanceID:          "workspace_editor_1",
			MaxAttempts:             2,
			ExpectedInputDigests:    []string{"sha256:" + strings.Repeat("a", 64)},
			DependencyCacheHandoffs: nil,
		}},
	}
}

func newRunPlanCompilationRecordForStoreTest(runID, planID, supersedesPlanID, digest string) RunPlanCompilationRecord {
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	return RunPlanCompilationRecord{
		RunID:                        runID,
		PlanID:                       planID,
		SupersedesPlanID:             supersedesPlanID,
		RunPlanDigest:                digest,
		WorkflowDefinitionRef:        "sha256:" + strings.Repeat("3", 64),
		ProcessDefinitionRef:         "sha256:" + strings.Repeat("4", 64),
		WorkflowDefinitionHash:       "sha256:" + strings.Repeat("3", 64),
		ProcessDefinitionHash:        "sha256:" + strings.Repeat("4", 64),
		PolicyContextHash:            "sha256:" + strings.Repeat("5", 64),
		ProjectContextIdentityDigest: storeRunPlanProjectContextDigest,
		CompiledAt:                   now,
		RecordedAt:                   now,
	}
}
