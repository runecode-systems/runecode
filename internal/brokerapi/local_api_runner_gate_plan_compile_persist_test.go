package brokerapi

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestCompileAndPersistRunPlanBuildsDurableAuthorityAndCompilationBinding(t *testing.T) {
	s := newTrustedRunPlanBrokerService(t)
	runID := "run-compile-persist"
	if err := s.SetRunStatus(runID, "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	workflowRef, processRef := putTrustedWorkflowAndProcessDefinitions(t, s, runID)
	result, err := s.CompileAndPersistRunPlan(CompileAndPersistRunPlanRequest{
		RunID:                 runID,
		PlanID:                "plan-compile-persist-0001",
		WorkflowDefinitionRef: workflowRef.Digest,
		ProcessDefinitionRef:  processRef.Digest,
		PolicyContextHash:     "sha256:" + strings.Repeat("5", 64),
	})
	if err != nil {
		t.Fatalf("CompileAndPersistRunPlan returned error: %v", err)
	}
	assertActiveAuthorityMatchesResult(t, s, runID, result)
	assertCompilationRecordMatchesRefs(t, s, runID, result.PlanID, workflowRef.Digest, processRef.Digest)
}

func newTrustedRunPlanBrokerService(t *testing.T) *Service {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	if strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest) == "" {
		s.projectSubstrate.Snapshot.ProjectContextIdentityDigest = trustedRunPlanProjectContextDigest
	}
	s.projectSubstrate.Snapshot.ValidatedSnapshotDigest = s.projectSubstrate.Snapshot.ProjectContextIdentityDigest
	return s
}

func putTrustedWorkflowAndProcessDefinitions(t *testing.T, s *Service, runID string) (artifacts.ArtifactReference, artifacts.ArtifactReference) {
	t.Helper()
	workflowPayload, processPayload := buildTrustedPlanCompileInputs(t, 2, trustedRunPlanOptions{})
	workflowRef, err := s.Put(putTrustedDefinitionRequest(runID, "workflow_definition", workflowPayload))
	if err != nil {
		t.Fatalf("Put(workflow) returned error: %v", err)
	}
	processRef, err := s.Put(putTrustedDefinitionRequest(runID, "process_definition", processPayload))
	if err != nil {
		t.Fatalf("Put(process) returned error: %v", err)
	}
	return workflowRef, processRef
}

func assertActiveAuthorityMatchesResult(t *testing.T, s *Service, runID string, result CompileAndPersistRunPlanResult) {
	t.Helper()
	active, ok, err := s.ActiveRunPlanAuthority(runID)
	if err != nil {
		t.Fatalf("ActiveRunPlanAuthority returned error: %v", err)
	}
	if !ok {
		t.Fatal("ActiveRunPlanAuthority ok=false, want true")
	}
	if active.PlanID != result.PlanID || active.RunPlanDigest != result.RunPlanDigest {
		t.Fatalf("active authority mismatch: active=%+v result=%+v", active, result)
	}
}

func assertCompilationRecordMatchesRefs(t *testing.T, s *Service, runID, planID, workflowRef, processRef string) {
	t.Helper()
	compilation, ok := s.RunPlanCompilationRecord(runID, planID)
	if !ok {
		t.Fatal("RunPlanCompilationRecord ok=false, want true")
	}
	if compilation.WorkflowDefinitionRef != workflowRef || compilation.ProcessDefinitionRef != processRef {
		t.Fatalf("compilation refs = (%q,%q), want (%q,%q)", compilation.WorkflowDefinitionRef, compilation.ProcessDefinitionRef, workflowRef, processRef)
	}
	if strings.TrimSpace(compilation.BindingDigest) == "" || strings.TrimSpace(compilation.RecordDigest) == "" {
		t.Fatalf("compilation digests missing: %+v", compilation)
	}
	if strings.TrimSpace(compilation.CompileCacheKey) == "" {
		t.Fatalf("compilation compile_cache_key missing: %+v", compilation)
	}
}

func TestCompileRunGatePlanUsesIndexedAuthorityWithoutArtifactRescan(t *testing.T) {
	s := newTrustedRunPlanBrokerService(t)
	runID := "run-compile-indexed-only"
	if err := s.SetRunStatus(runID, "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	workflowRef, processRef := putTrustedWorkflowAndProcessDefinitions(t, s, runID)
	if _, err := s.CompileAndPersistRunPlan(CompileAndPersistRunPlanRequest{RunID: runID, PlanID: "plan-indexed-0001", WorkflowDefinitionRef: workflowRef.Digest, ProcessDefinitionRef: processRef.Digest, PolicyContextHash: "sha256:" + strings.Repeat("5", 64)}); err != nil {
		t.Fatalf("CompileAndPersistRunPlan returned error: %v", err)
	}
	malicious := []byte(`{"schema_id":"runecode.protocol.v0.RunPlan","schema_version":"0.4.0","plan_id":"plan-indexed-evil","run_id":"` + runID + `","workflow_definition_hash":"invalid"}`)
	if _, err := s.Put(putTrustedDefinitionRequest(runID, runPlanAuthorityStepID("plan-indexed-evil"), malicious)); err != nil {
		t.Fatalf("Put(malicious run plan blob) returned error: %v", err)
	}
	planned, err := s.compileRunGatePlan(runID)
	if err != nil {
		t.Fatalf("compileRunGatePlan returned error: %v", err)
	}
	if planned.planID != "plan-indexed-0001" {
		t.Fatalf("planned.planID = %q, want plan-indexed-0001", planned.planID)
	}
	if !planned.hasEntries() {
		t.Fatal("planned.hasEntries() = false, want true")
	}
}

func TestCompileRunGatePlanDoesNotDiscoverAuthorityFromRawArtifacts(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-compile-no-raw-discovery"
	if err := s.SetRunStatus(runID, "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	rawRunPlan := []byte(`{
		"schema_id":"runecode.protocol.v0.RunPlan",
		"schema_version":"0.4.0",
		"plan_id":"plan-raw-only",
		"run_id":"` + runID + `",
		"workflow_definition_hash":"sha256:` + strings.Repeat("a", 64) + `",
		"process_definition_hash":"sha256:` + strings.Repeat("b", 64) + `",
		"policy_context_hash":"sha256:` + strings.Repeat("c", 64) + `",
		"gate_definitions":[]
	}`)
	if _, err := s.Put(putTrustedDefinitionRequest(runID, runPlanAuthorityStepID("plan-raw-only"), rawRunPlan)); err != nil {
		t.Fatalf("Put(raw run plan artifact) returned error: %v", err)
	}

	planned, err := s.compileRunGatePlan(runID)
	if err != nil {
		t.Fatalf("compileRunGatePlan returned error: %v", err)
	}
	if planned.planID != "" {
		t.Fatalf("planned.planID = %q, want empty plan without indexed authority", planned.planID)
	}
	if planned.hasEntries() {
		t.Fatal("planned.hasEntries() = true, want false without indexed authority")
	}
}

func putTrustedDefinitionRequest(runID, stepID string, payload []byte) artifacts.PutRequest {
	return artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: "sha256:" + strings.Repeat("2", 64),
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 runID,
		StepID:                stepID,
	}
}
