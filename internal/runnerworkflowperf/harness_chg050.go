package runnerworkflowperf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/perffixtures"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/runplan"
)

func measureCHG050CompileAndLoad(repoRoot string, runner func(repoRoot string, timeout time.Duration, args ...string) (float64, error), timeout time.Duration) ([]perfcontracts.MeasurementRecord, error) {
	tmpRoot, err := os.MkdirTemp("", "runecode-runnerworkflowperf-chg050-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	fixture, err := perffixtures.BuildWorkflowFixture(filepath.Join(tmpRoot, "fixture"), perffixtures.FixtureWorkflowCHG050Compile)
	if err != nil {
		return nil, fmt.Errorf("build CHG-050 fixture: %w", err)
	}
	compileMS, validationCanonicalizationMS, persistLoadMS, err := buildPersistAndLoadCHG050(tmpRoot)
	if err != nil {
		return nil, err
	}
	startupMS, err := runner(repoRoot, timeout, "node", "--experimental-strip-types", "scripts/perf-runner-workflow.js", "--mode", "immutable-startup", "--runplan", fixture.RunPlan)
	if err != nil {
		return nil, fmt.Errorf("measure immutable runplan startup: %w", err)
	}
	return []perfcontracts.MeasurementRecord{
		{MetricID: "metric.workflow.chg050.compile.wall_ms", Value: compileMS, Unit: "ms"},
		{MetricID: "metric.workflow.chg050.validation_canonicalization.wall_ms", Value: validationCanonicalizationMS, Unit: "ms"},
		{MetricID: "metric.workflow.chg050.runplan_persist_load.wall_ms", Value: persistLoadMS, Unit: "ms"},
		{MetricID: "metric.workflow.chg050.runner_start_immutable_runplan.wall_ms", Value: startupMS, Unit: "ms"},
	}, nil
}

func buildPersistAndLoadCHG050(tmpRoot string) (float64, float64, float64, error) {
	compileInput, validationCanonicalizationMS, err := buildCHG050CompileInput()
	if err != nil {
		return 0, 0, 0, err
	}
	compileStart := time.Now()
	plan, err := runplan.Compile(compileInput)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("compile CHG-050 runplan: %w", err)
	}
	compileMS := float64(time.Since(compileStart).Milliseconds())
	persistLoadMS, err := persistAndLoadCHG050RunPlan(tmpRoot, plan)
	if err != nil {
		return 0, 0, 0, err
	}
	return compileMS, validationCanonicalizationMS, persistLoadMS, nil
}

func buildCHG050CompileInput() (runplan.CompileInput, float64, error) {
	validationCanonicalizationStart := time.Now()
	processBytes, processHash, err := marshalCHG050ProcessDefinition()
	if err != nil {
		return runplan.CompileInput{}, 0, err
	}
	workflowBytes, err := marshalCHG050WorkflowDefinition(processHash)
	if err != nil {
		return runplan.CompileInput{}, 0, err
	}
	validationCanonicalizationMS := float64(time.Since(validationCanonicalizationStart).Milliseconds())
	return runplan.CompileInput{RunID: "run-chg050", PlanID: "plan-chg050-v1", CompiledAt: time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC), WorkflowDefinitionBytes: workflowBytes, ProcessDefinitionBytes: processBytes, ProjectContextIdentityDigest: "sha256:" + strings.Repeat("3", 64), PolicyContextHash: "sha256:" + strings.Repeat("4", 64), ExecutorRegistry: policyengine.BuildExecutorRegistryProjection()}, validationCanonicalizationMS, nil
}

func marshalCHG050ProcessDefinition() ([]byte, string, error) {
	processDefinition := map[string]any{"schema_id": "runecode.protocol.v0.ProcessDefinition", "schema_version": "0.4.0", "process_id": "process_chg050", "executor_bindings": []map[string]any{{"binding_id": "binding_workspace_runner", "executor_id": "workspace-runner", "executor_class": "workspace_ordinary", "allowed_role_kinds": []string{"workspace-edit"}}}, "gate_definitions": []map[string]any{{"schema_id": "runecode.protocol.v0.GateDefinition", "schema_version": "0.2.0", "checkpoint_code": "step_validation_started", "order_index": 0, "stage_id": "validation", "step_id": "validate_step", "role_instance_id": "workspace_editor_1", "executor_binding_id": "binding_workspace_runner", "gate": map[string]any{"schema_id": "runecode.protocol.v0.GateContract", "schema_version": "0.1.0", "gate_id": "lint_gate", "gate_kind": "lint", "gate_version": "1.0.0", "normalized_inputs": []map[string]any{{"input_id": "source_tree", "input_digest": "sha256:" + strings.Repeat("2", 64)}}, "plan_binding": map[string]any{"checkpoint_code": "step_validation_started", "order_index": 0}, "retry_semantics": map[string]any{"retry_mode": "new_attempt_required", "max_attempts": 2}, "override_semantics": map[string]any{"override_mode": "policy_action_required", "action_kind": "action_gate_override", "approval_trigger_code": "gate_override"}}}}, "dependency_edges": []map[string]any{}}
	processBytes, err := json.Marshal(processDefinition)
	if err != nil {
		return nil, "", err
	}
	processCanonical, err := policyengine.CanonicalizeJSONBytes(processBytes)
	if err != nil {
		return nil, "", err
	}
	return processBytes, policyengine.HashCanonicalJSONBytes(processCanonical), nil
}

func marshalCHG050WorkflowDefinition(processHash string) ([]byte, error) {
	workflowDefinition := map[string]any{"schema_id": "runecode.protocol.v0.WorkflowDefinition", "schema_version": "0.5.0", "workflow_id": "workflow_chg050", "workflow_version": "1.0.0", "selected_process_id": "process_chg050", "selected_process_definition_hash": processHash, "reviewed_process_artifacts": []map[string]any{{"process_id": "process_chg050", "process_definition_hash": processHash}}, "approval_profile": "moderate", "autonomy_posture": "balanced"}
	workflowBytes, err := json.Marshal(workflowDefinition)
	if err != nil {
		return nil, err
	}
	_, err = policyengine.CanonicalizeJSONBytes(workflowBytes)
	if err != nil {
		return nil, err
	}
	return workflowBytes, nil
}

func persistAndLoadCHG050RunPlan(tmpRoot string, plan runplan.RunPlan) (float64, error) {
	planBytes, err := json.Marshal(plan)
	if err != nil {
		return 0, err
	}
	persistStart := time.Now()
	store, err := artifacts.NewStore(filepath.Join(tmpRoot, "store"))
	if err != nil {
		return 0, err
	}
	ref, err := store.Put(artifacts.PutRequest{Payload: planBytes, ContentType: "application/json", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("8", 64), CreatedByRole: "brokerapi", TrustedSource: true, RunID: "run-chg050", StepID: "compiled_run_plan/plan-chg050-v1"})
	if err != nil {
		return 0, err
	}
	authority := artifacts.RunPlanAuthorityRecord{RunID: "run-chg050", PlanID: "plan-chg050-v1", RunPlanDigest: ref.Digest, WorkflowDefinitionHash: plan.WorkflowDefinitionHash, ProcessDefinitionHash: plan.ProcessDefinitionHash, PolicyContextHash: plan.PolicyContextHash, ProjectContextIdentityDigest: plan.ProjectContextIdentityDigest, CompiledAt: time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC), Entries: authorityEntriesFromPlan(plan)}
	compilation := artifacts.RunPlanCompilationRecord{RunID: "run-chg050", PlanID: "plan-chg050-v1", RunPlanDigest: ref.Digest, CompileCacheKey: "cache-key-chg050-v1", WorkflowDefinitionRef: "sha256:" + strings.Repeat("5", 64), ProcessDefinitionRef: "sha256:" + strings.Repeat("6", 64), WorkflowDefinitionHash: plan.WorkflowDefinitionHash, ProcessDefinitionHash: plan.ProcessDefinitionHash, PolicyContextHash: plan.PolicyContextHash, ProjectContextIdentityDigest: plan.ProjectContextIdentityDigest, CompiledAt: time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC)}
	if err := store.RecordRunPlanAuthority(authority, compilation); err != nil {
		return 0, err
	}
	if err := verifyCHG050Persistence(store); err != nil {
		return 0, err
	}
	return float64(time.Since(persistStart).Milliseconds()), nil
}

func verifyCHG050Persistence(store *artifacts.Store) error {
	if _, ok, err := store.ActiveRunPlanAuthority("run-chg050"); err != nil || !ok {
		if err != nil {
			return err
		}
		return fmt.Errorf("active runplan authority missing")
	}
	if _, ok := store.RunPlanCompilationRecordByCacheKey("cache-key-chg050-v1"); !ok {
		return fmt.Errorf("runplan compilation cache lookup missing")
	}
	return nil
}

func authorityEntriesFromPlan(plan runplan.RunPlan) []artifacts.RunPlanGateEntryRecord {
	if len(plan.Entries) == 0 {
		return nil
	}
	out := make([]artifacts.RunPlanGateEntryRecord, 0, len(plan.Entries))
	for _, entry := range plan.Entries {
		out = append(out, artifacts.RunPlanGateEntryRecord{
			EntryID:              strings.TrimSpace(entry.EntryID),
			EntryKind:            strings.TrimSpace(entry.EntryKind),
			PlanCheckpointCode:   strings.TrimSpace(entry.CheckpointCode),
			PlanOrderIndex:       entry.OrderIndex,
			GateID:               strings.TrimSpace(entry.Gate.GateID),
			GateKind:             strings.TrimSpace(entry.Gate.GateKind),
			GateVersion:          strings.TrimSpace(entry.Gate.GateVersion),
			StageID:              strings.TrimSpace(entry.StageID),
			StepID:               strings.TrimSpace(entry.StepID),
			RoleInstanceID:       strings.TrimSpace(entry.RoleInstanceID),
			MaxAttempts:          maxAttemptsFromRetrySemantics(entry.Gate.RetrySemantics),
			ExpectedInputDigests: expectedInputDigests(entry.Gate.NormalizedInputs),
		})
	}
	return out
}

func expectedInputDigests(inputs []map[string]any) []string {
	if len(inputs) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(inputs))
	for _, input := range inputs {
		raw, _ := input["input_digest"].(string)
		digest := strings.TrimSpace(raw)
		if digest == "" {
			continue
		}
		if _, ok := seen[digest]; ok {
			continue
		}
		seen[digest] = struct{}{}
		out = append(out, digest)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func maxAttemptsFromRetrySemantics(retry map[string]any) int {
	if retry == nil {
		return 1
	}
	if value, ok := retry["max_attempts"].(int); ok && value > 0 {
		return value
	}
	if value, ok := retry["max_attempts"].(float64); ok && int(value) > 0 {
		return int(value)
	}
	return 1
}
