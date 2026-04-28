package runplan

import (
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestCompileBuildsDeterministicRunPlan(t *testing.T) {
	processPayload := processPayloadForTest(t, []any{gateDef("build_gate", "step_validation_started", 0), gateDef("lint_gate", "step_validation_finished", 1)}, []string{"workspace-edit", "workspace-test"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	plan, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	assertDeterministicRunPlanShape(t, plan)
	if err := ValidateRunPlan(plan); err != nil {
		t.Fatalf("ValidateRunPlan returned error: %v", err)
	}
}

func deterministicCompileInput(workflowPayload []byte, processPayload []byte) CompileInput {
	return CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		SupersedesPlanID:        "plan_run_123_0000",
		CompiledAt:              time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	}
}

func assertDeterministicRunPlanShape(t *testing.T, plan RunPlan) {
	t.Helper()
	assertRunPlanMetadata(t, plan)
	assertRunPlanExecutorBindings(t, plan)
	assertRunPlanGateDefinitions(t, plan)
	assertRunPlanEntries(t, plan)
}

func assertRunPlanMetadata(t *testing.T, plan RunPlan) {
	if plan.SchemaID != runPlanSchemaID {
		t.Fatalf("schema_id = %q, want %q", plan.SchemaID, runPlanSchemaID)
	}
	if plan.PlanID != "plan_run_123_0001" {
		t.Fatalf("plan_id = %q", plan.PlanID)
	}
	if plan.SupersedesPlanID != "plan_run_123_0000" {
		t.Fatalf("supersedes_plan_id = %q", plan.SupersedesPlanID)
	}
	if plan.WorkflowVersion != "1.0.0" {
		t.Fatalf("workflow_version = %q, want 1.0.0", plan.WorkflowVersion)
	}
	if plan.ApprovalProfile != "moderate" {
		t.Fatalf("approval_profile = %q, want moderate", plan.ApprovalProfile)
	}
	if plan.AutonomyPosture != "balanced" {
		t.Fatalf("autonomy_posture = %q, want balanced", plan.AutonomyPosture)
	}
	if plan.PolicyBindingID != "policy_binding_default" {
		t.Fatalf("policy_binding_id = %q, want policy_binding_default", plan.PolicyBindingID)
	}
}

func assertRunPlanExecutorBindings(t *testing.T, plan RunPlan) {
	if len(plan.ExecutorBindings) != 1 {
		t.Fatalf("executor_bindings len = %d, want 1", len(plan.ExecutorBindings))
	}
}

func assertRunPlanGateDefinitions(t *testing.T, plan RunPlan) {
	if len(plan.GateDefinitions) != 2 {
		t.Fatalf("gate_definitions len = %d, want 2", len(plan.GateDefinitions))
	}
	if got := plan.GateDefinitions[0].Gate.GateID; got != "build_gate" {
		t.Fatalf("first gate_id = %q, want build_gate", got)
	}
	if got := plan.GateDefinitions[1].Gate.GateID; got != "lint_gate" {
		t.Fatalf("second gate_id = %q, want lint_gate", got)
	}
	if got := plan.GateDefinitions[0].StageID; got != "validation" {
		t.Fatalf("first stage_id = %q, want validation", got)
	}
	if got := plan.GateDefinitions[0].StepID; got == "" {
		t.Fatal("first step_id is empty, want stable logical step identity")
	}
}

func assertRunPlanEntries(t *testing.T, plan RunPlan) {
	if len(plan.Entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(plan.Entries))
	}
	if got := plan.Entries[0].EntryID; got != "build_gate_build_step" {
		t.Fatalf("entries[0].entry_id = %q, want build_gate_build_step", got)
	}
	if got := plan.Entries[0].EntryKind; got != "gate" {
		t.Fatalf("entries[0].entry_kind = %q, want gate", got)
	}
	if len(plan.Entries[0].SupportedWaitKinds) != 2 {
		t.Fatalf("entries[0].supported_wait_kinds len = %d, want 2", len(plan.Entries[0].SupportedWaitKinds))
	}
}

func TestCompileFailsClosedOnUnknownExecutorBinding(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_unknown", "unknown-executor", []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	_, err := Compile(CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	})
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
}

func TestCompileCarriesDistinctApprovalProfileAndAutonomyPostureBinding(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	processHash := mustCanonicalHash(t, processPayload)
	workflowPayload := mustJSON(t, map[string]any{
		"schema_id":                        workflowDefinitionSchemaID,
		"schema_version":                   workflowDefinitionVersion,
		"workflow_id":                      "workflow_main",
		"workflow_version":                 "1.0.0",
		"selected_process_id":              "process_default",
		"selected_process_definition_hash": processHash,
		"reviewed_process_artifacts":       []any{map[string]any{"process_id": "process_default", "process_definition_hash": processHash}},
		"approval_profile":                 "moderate",
		"autonomy_posture":                 "operator_guided",
	})

	plan, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if plan.ApprovalProfile != "moderate" {
		t.Fatalf("approval_profile = %q, want moderate", plan.ApprovalProfile)
	}
	if plan.AutonomyPosture != "operator_guided" {
		t.Fatalf("autonomy_posture = %q, want operator_guided", plan.AutonomyPosture)
	}
}

func TestCompileHashBindsWorkflowSelectionControls(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	processHash := mustCanonicalHash(t, processPayload)

	balancedWorkflow := mustJSON(t, map[string]any{
		"schema_id":                        workflowDefinitionSchemaID,
		"schema_version":                   workflowDefinitionVersion,
		"workflow_id":                      "workflow_main",
		"workflow_version":                 "1.0.0",
		"selected_process_id":              "process_default",
		"selected_process_definition_hash": processHash,
		"reviewed_process_artifacts":       []any{map[string]any{"process_id": "process_default", "process_definition_hash": processHash}},
		"approval_profile":                 "moderate",
		"autonomy_posture":                 "balanced",
	})
	operatorGuidedWorkflow := mustJSON(t, map[string]any{
		"schema_id":                        workflowDefinitionSchemaID,
		"schema_version":                   workflowDefinitionVersion,
		"workflow_id":                      "workflow_main",
		"workflow_version":                 "1.0.0",
		"selected_process_id":              "process_default",
		"selected_process_definition_hash": processHash,
		"reviewed_process_artifacts":       []any{map[string]any{"process_id": "process_default", "process_definition_hash": processHash}},
		"approval_profile":                 "moderate",
		"autonomy_posture":                 "operator_guided",
	})

	balancedPlan, err := Compile(deterministicCompileInput(balancedWorkflow, processPayload))
	if err != nil {
		t.Fatalf("Compile balanced workflow returned error: %v", err)
	}
	operatorGuidedPlan, err := Compile(deterministicCompileInput(operatorGuidedWorkflow, processPayload))
	if err != nil {
		t.Fatalf("Compile operator-guided workflow returned error: %v", err)
	}

	if balancedPlan.WorkflowDefinitionHash == operatorGuidedPlan.WorkflowDefinitionHash {
		t.Fatal("workflow_definition_hash did not change when autonomy_posture changed")
	}
}

func TestCompileHashesEquivalentDefinitionJSONToSameCanonicalDigest(t *testing.T) {
	processCanonical := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	processHash := mustCanonicalHash(t, processCanonical)
	workflowCanonical := mustJSON(t, map[string]any{
		"schema_id":                        workflowDefinitionSchemaID,
		"schema_version":                   workflowDefinitionVersion,
		"workflow_id":                      "workflow_main",
		"workflow_version":                 "1.0.0",
		"selected_process_id":              "process_default",
		"selected_process_definition_hash": processHash,
		"reviewed_process_artifacts":       []any{map[string]any{"process_id": "process_default", "process_definition_hash": processHash}},
		"approval_profile":                 "moderate",
		"autonomy_posture":                 "balanced",
	})
	workflowAdapterOrder := []byte(`{"workflow_version":"1.0.0","reviewed_process_artifacts":[{"process_definition_hash":"` + processHash + `","process_id":"process_default"}],"autonomy_posture":"balanced","approval_profile":"moderate","selected_process_definition_hash":"` + processHash + `","selected_process_id":"process_default","schema_version":"0.5.0","workflow_id":"workflow_main","schema_id":"runecode.protocol.v0.WorkflowDefinition"}`)

	processAdapterOrder := []byte(`{"dependency_edges":[],"gate_definitions":[{"executor_binding_id":"binding_workspace_runner","gate":{"schema_id":"runecode.protocol.v0.GateContract","schema_version":"0.1.0","gate_id":"build_gate","gate_kind":"build","gate_version":"1.0.0","normalized_inputs":[{"input_id":"source_tree","input_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"}],"plan_binding":{"checkpoint_code":"step_validation_started","order_index":0},"retry_semantics":{"retry_mode":"new_attempt_required","max_attempts":3},"override_semantics":{"override_mode":"policy_action_required","action_kind":"action_gate_override","approval_trigger_code":"gate_override"}},"schema_id":"runecode.protocol.v0.GateDefinition","schema_version":"0.2.0","checkpoint_code":"step_validation_started","order_index":0,"stage_id":"validation","step_id":"build_gate_build_step","role_instance_id":"workspace_editor_1"}],"schema_id":"runecode.protocol.v0.ProcessDefinition","schema_version":"0.4.0","process_id":"process_default","executor_bindings":[{"executor_class":"workspace_ordinary","allowed_role_kinds":["workspace-edit"],"executor_id":"workspace-runner","binding_id":"binding_workspace_runner"}]}`)

	canonicalPlan, err := Compile(deterministicCompileInput(workflowCanonical, processCanonical))
	if err != nil {
		t.Fatalf("Compile with canonical payloads returned error: %v", err)
	}
	adapterPlan, err := Compile(deterministicCompileInput(workflowAdapterOrder, processAdapterOrder))
	if err != nil {
		t.Fatalf("Compile with adapter-ordered payloads returned error: %v", err)
	}
	if canonicalPlan.WorkflowDefinitionHash != adapterPlan.WorkflowDefinitionHash {
		t.Fatalf("workflow_definition_hash mismatch: canonical=%q adapter=%q", canonicalPlan.WorkflowDefinitionHash, adapterPlan.WorkflowDefinitionHash)
	}
	if canonicalPlan.ProcessDefinitionHash != adapterPlan.ProcessDefinitionHash {
		t.Fatalf("process_definition_hash mismatch: canonical=%q adapter=%q", canonicalPlan.ProcessDefinitionHash, adapterPlan.ProcessDefinitionHash)
	}
}

func TestCompileRejectsDuplicateWorkflowObjectKeysDuringCanonicalization(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	processHash := mustCanonicalHash(t, processPayload)
	workflowWithDuplicateKey := []byte(`{"schema_id":"runecode.protocol.v0.WorkflowDefinition","schema_version":"0.5.0","workflow_id":"workflow_main","workflow_version":"1.0.0","selected_process_id":"process_default","selected_process_definition_hash":"` + processHash + `","reviewed_process_artifacts":[],"reviewed_process_artifacts":[{"process_id":"process_default","process_definition_hash":"` + processHash + `"}],"approval_profile":"moderate","autonomy_posture":"balanced"}`)

	_, err := Compile(deterministicCompileInput(workflowWithDuplicateKey, processPayload))
	if err == nil {
		t.Fatal("Compile error = nil, want canonicalization rejection for duplicate workflow key")
	}
	if !strings.Contains(err.Error(), "canonicalize workflow definition payload before validation/hash") {
		t.Fatalf("Compile error = %v, want canonicalization failure prefix", err)
	}
}

func workflowPayloadForTest(t *testing.T, processPayload []byte, selectedProcessID string, reviewedProcessIDs []string) []byte {
	t.Helper()
	selectedHash := mustCanonicalHash(t, processPayload)
	reviewed := make([]any, 0, len(reviewedProcessIDs))
	for _, id := range reviewedProcessIDs {
		hash := "sha256:" + strings.Repeat("f", 64)
		if id == selectedProcessID {
			hash = selectedHash
		}
		reviewed = append(reviewed, map[string]any{"process_id": id, "process_definition_hash": hash})
	}
	return mustJSON(t, map[string]any{
		"schema_id":                        workflowDefinitionSchemaID,
		"schema_version":                   workflowDefinitionVersion,
		"workflow_id":                      "workflow_main",
		"workflow_version":                 "1.0.0",
		"selected_process_id":              selectedProcessID,
		"selected_process_definition_hash": selectedHash,
		"reviewed_process_artifacts":       reviewed,
		"policy_binding_id":                "policy_binding_default",
		"approval_profile":                 "moderate",
		"autonomy_posture":                 "balanced",
	})
}

func mustCanonicalHash(t *testing.T, payload []byte) string {
	t.Helper()
	canonicalPayload, err := policyengine.CanonicalizeJSONBytes(payload)
	if err != nil {
		t.Fatalf("CanonicalizeJSONBytes returned error: %v", err)
	}
	return policyengine.HashCanonicalJSONBytes(canonicalPayload)
}

func processPayloadForTest(t *testing.T, gates []any, roles []string) []byte {
	t.Helper()
	return mustJSON(t, map[string]any{
		"schema_id":         processDefinitionSchemaID,
		"schema_version":    processDefinitionVersion,
		"process_id":        "process_default",
		"executor_bindings": []any{executorBindingFixture("binding_workspace_runner", "workspace-runner", roles)},
		"gate_definitions":  gates,
		"dependency_edges":  []any{},
	})
}

func executorBindingFixture(bindingID, executorID string, roles []string) map[string]any {
	roleItems := make([]any, 0, len(roles))
	for _, role := range roles {
		roleItems = append(roleItems, role)
	}
	return map[string]any{
		"binding_id":         bindingID,
		"executor_id":        executorID,
		"executor_class":     "workspace_ordinary",
		"allowed_role_kinds": roleItems,
	}
}

func TestCompileRejectsSupersedesSameAsPlanID(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	_, err := Compile(CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		SupersedesPlanID:        "plan_run_123_0001",
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	})
	if err == nil {
		t.Fatal("Compile returned nil error, want failure")
	}
}

func TestCompileUsesCurrentTimeWhenCompiledAtZero(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})
	plan, err := Compile(CompileInput{
		RunID:                   "run_123",
		PlanID:                  "plan_run_123_0001",
		WorkflowDefinitionBytes: workflowPayload,
		ProcessDefinitionBytes:  processPayload,
		PolicyContextHash:       "sha256:" + strings.Repeat("a", 64),
		ExecutorRegistry:        policyengine.BuildExecutorRegistryProjection(),
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if plan.CompiledAt == "0001-01-01T00:00:00Z" {
		t.Fatal("compiled_at used zero time, want current timestamp fallback")
	}
	if _, err := time.Parse(time.RFC3339, plan.CompiledAt); err != nil {
		t.Fatalf("compiled_at parse error: %v", err)
	}
}

func TestCompileBindsProjectContextIdentityDigestWhenProvided(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})
	input := deterministicCompileInput(workflowPayload, processPayload)
	input.ProjectContextIdentityDigest = "sha256:" + strings.Repeat("9", 64)

	plan, err := Compile(input)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if plan.ProjectContextIdentityDigest != input.ProjectContextIdentityDigest {
		t.Fatalf("project_context_identity_digest = %q, want %q", plan.ProjectContextIdentityDigest, input.ProjectContextIdentityDigest)
	}
}

func TestCompileKeepsDistinctGateKindVersionVariantsWithinProcessDefinition(t *testing.T) {
	processPayload := processPayloadForTest(t, []any{
		gateDefWithKindVersion("same_gate", "step_validation_started", 0, "build", "1.0.0"),
		gateDefWithKindVersion("same_gate", "step_validation_started", 0, "test", "2.0.0"),
	}, []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	plan, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if len(plan.GateDefinitions) != 2 {
		t.Fatalf("gate_definitions len = %d, want 2 distinct kind/version variants", len(plan.GateDefinitions))
	}
	seen := map[string]bool{}
	for _, gate := range plan.GateDefinitions {
		seen[gate.Gate.GateKind+"@"+gate.Gate.GateVersion] = true
	}
	if !seen["build@1.0.0"] || !seen["test@2.0.0"] {
		t.Fatalf("compiled gate variants = %+v, want build@1.0.0 and test@2.0.0", seen)
	}
}

func TestCompileRejectsConflictingGateDefinitionForSameDedupeKeyWithinProcessDefinition(t *testing.T) {
	processGateA := gateDef("build_gate", "step_validation_started", 0)
	processGateB := gateDef("build_gate", "step_validation_started", 0)
	processGateB["executor_binding_id"] = "binding_workspace_runner_alt"
	processPayload := mustJSON(t, map[string]any{
		"schema_id":      processDefinitionSchemaID,
		"schema_version": processDefinitionVersion,
		"process_id":     "process_default",
		"executor_bindings": []any{
			executorBindingFixture("binding_workspace_runner", "workspace-runner", []string{"workspace-edit"}),
			executorBindingFixture("binding_workspace_runner_alt", "workspace-runner", []string{"workspace-edit"}),
		},
		"gate_definitions": []any{processGateA, processGateB},
		"dependency_edges": []any{},
	})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_default"})

	_, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err == nil {
		t.Fatal("Compile error = nil, want conflict rejection for same dedupe key")
	}
	if !strings.Contains(err.Error(), "conflicts within process definition") {
		t.Fatalf("Compile error = %v, want conflict error", err)
	}
}

func TestCompileRejectsWorkflowSelectionMismatch(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_other", []string{"process_default", "process_other"})

	_, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err == nil {
		t.Fatal("Compile error = nil, want selection mismatch rejection")
	}
	if !strings.Contains(err.Error(), "selected_process_id") {
		t.Fatalf("Compile error = %v, want selected_process_id error", err)
	}
}

func TestCompileRejectsWorkflowWithoutReviewedProcessForSelection(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	workflowPayload := workflowPayloadForTest(t, processPayload, "process_default", []string{"process_other"})

	_, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err == nil {
		t.Fatal("Compile error = nil, want reviewed process selection rejection")
	}
	if !strings.Contains(err.Error(), "reviewed_process_artifacts") {
		t.Fatalf("Compile error = %v, want reviewed_process_artifacts error", err)
	}
}

func TestCompileRejectsDuplicateReviewedProcessArtifacts(t *testing.T) {
	processPayload := processPayloadWithSingleBinding(t, "binding_workspace_runner", "workspace-runner", []string{"workspace-edit"})
	selectedHash := mustCanonicalHash(t, processPayload)
	workflowPayload := mustJSON(t, map[string]any{
		"schema_id":                        workflowDefinitionSchemaID,
		"schema_version":                   workflowDefinitionVersion,
		"workflow_id":                      "workflow_main",
		"workflow_version":                 "1.0.0",
		"selected_process_id":              "process_default",
		"selected_process_definition_hash": selectedHash,
		"reviewed_process_artifacts": []any{
			map[string]any{"process_id": "process_default", "process_definition_hash": selectedHash},
			map[string]any{"process_id": "process_default", "process_definition_hash": selectedHash},
		},
		"approval_profile": "moderate",
		"autonomy_posture": "balanced",
	})

	_, err := Compile(deterministicCompileInput(workflowPayload, processPayload))
	if err == nil {
		t.Fatal("Compile error = nil, want duplicate reviewed process rejection")
	}
	if !strings.Contains(err.Error(), "duplicate process_id") {
		t.Fatalf("Compile error = %v, want duplicate process_id error", err)
	}
}

func gateDef(gateID, checkpoint string, order int) map[string]any {
	return gateDefWithKindVersion(gateID, checkpoint, order, "build", "1.0.0")
}

func gateDefWithKindVersion(gateID, checkpoint string, order int, gateKind, gateVersion string) map[string]any {
	return map[string]any{
		"schema_id":           gateDefinitionSchemaID,
		"schema_version":      gateDefinitionVersion,
		"checkpoint_code":     checkpoint,
		"order_index":         order,
		"stage_id":            "validation",
		"step_id":             gateID + "_" + gateKind + "_step",
		"role_instance_id":    "workspace_editor_1",
		"executor_binding_id": "binding_workspace_runner",
		"gate": map[string]any{
			"schema_id":      "runecode.protocol.v0.GateContract",
			"schema_version": "0.1.0",
			"gate_id":        gateID,
			"gate_kind":      gateKind,
			"gate_version":   gateVersion,
			"normalized_inputs": []any{
				map[string]any{"input_id": "source_tree", "input_digest": "sha256:" + strings.Repeat("1", 64)},
			},
			"plan_binding":       map[string]any{"checkpoint_code": checkpoint, "order_index": order},
			"retry_semantics":    map[string]any{"retry_mode": "new_attempt_required", "max_attempts": 3},
			"override_semantics": map[string]any{"override_mode": "policy_action_required", "action_kind": "action_gate_override", "approval_trigger_code": "gate_override"},
		},
	}
}

func processPayloadWithSingleBinding(t *testing.T, bindingID, executorID string, roles []string) []byte {
	t.Helper()
	return mustJSON(t, map[string]any{
		"schema_id":         processDefinitionSchemaID,
		"schema_version":    processDefinitionVersion,
		"process_id":        "process_default",
		"executor_bindings": []any{executorBindingFixture(bindingID, executorID, roles)},
		"gate_definitions":  []any{gateDef("build_gate", "step_validation_started", 0)},
		"dependency_edges":  []any{},
	})
}
