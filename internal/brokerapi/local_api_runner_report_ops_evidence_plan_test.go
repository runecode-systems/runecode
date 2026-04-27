package brokerapi

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestRunnerResultReportPersistsGateEvidenceArtifactAndLinksRunAdvisory(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-evidence", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	resp, errResp := s.HandleRunnerResultReport(context.Background(), buildGateEvidenceResultRequest(now), RequestContext{})
	if errResp != nil || !resp.Accepted {
		t.Fatalf("unexpected result response: resp=%+v err=%+v", resp, errResp)
	}
	runResp := mustRunGet(t, s, "run-gate-evidence", "req-run-evidence")
	lastResult := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	gateEvidenceRef, ok := lastResult["gate_evidence_ref"].(string)
	if !ok || !strings.HasPrefix(gateEvidenceRef, "sha256:") {
		t.Fatalf("gate_evidence_ref = %v, want digest identity", lastResult["gate_evidence_ref"])
	}
	evidence := mustLoadGateEvidenceArtifact(t, s, gateEvidenceRef)
	if got, _ := evidence["project_context_identity_digest"].(string); got == "" {
		t.Fatal("gate evidence project_context_identity_digest missing")
	}
}

func mustLoadGateEvidenceArtifact(t *testing.T, s *Service, gateEvidenceRef string) map[string]any {
	t.Helper()
	rec, err := s.Head(gateEvidenceRef)
	if err != nil || rec.Reference.DataClass != artifacts.DataClassGateEvidence {
		t.Fatalf("unexpected gate evidence head: rec=%+v err=%v", rec, err)
	}
	r, err := s.Get(gateEvidenceRef)
	if err != nil {
		t.Fatalf("Get(gate_evidence_ref) returned error: %v", err)
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(gate evidence) returned error: %v", err)
	}
	var evidence map[string]any
	if err := json.Unmarshal(body, &evidence); err != nil {
		t.Fatalf("Unmarshal(gate evidence) returned error: %v", err)
	}
	return evidence
}

func TestRunnerResultReportRejectsMalformedGateEvidenceFailClosed(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 12, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-evidence-invalid", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	req := buildGateEvidenceResultRequest(now)
	req.RunID, req.RequestID = "run-gate-evidence-invalid", "req-gate-evidence-invalid"
	req.Report.GateAttemptID = "gate-attempt-10"
	req.Report.GateEvidence = buildGateEvidencePayload(now, req.RunID, req.Report.GateAttemptID)
	req.Report.GateEvidence.Outcome = map[string]any{}
	_, errResp := s.HandleRunnerResultReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerResultReportRejectsMismatchedProvidedGateEvidenceRef(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 12, 45, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-evidence-ref-mismatch", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	req := buildGateEvidenceResultRequest(now)
	req.RunID, req.RequestID = "run-gate-evidence-ref-mismatch", "req-gate-evidence-ref-mismatch"
	req.Report.GateAttemptID = "gate-attempt-11"
	req.Report.GateEvidenceRef = "sha256:" + strings.Repeat("a", 64)
	req.Report.GateEvidence = buildGateEvidencePayload(now, req.RunID, req.Report.GateAttemptID)
	_, errResp := s.HandleRunnerResultReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerGateReportsFailClosedWhenPlanPlacementMissing(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-missing")
	req := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-plan-missing", RunID: "run-plan-missing", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "gate_started", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-plan-missing", PlanCheckpointCode: "step_validation_started", PlanOrderIndex: 0, GateID: "policy_gate", GateKind: "policy", GateVersion: "1.0.0", GateLifecycleState: "running", GateAttemptID: "gate-attempt-1", NormalizedInputDigests: []string{"sha256:" + strings.Repeat("a", 64)}}}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerGateReportsAcceptTrustedPlanPlacementAndRetryPosture(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-ok")
	putTrustedRunPlanForGatePlan(t, s, "run-plan-ok", "plan_run_plan_ok_0001", 2)
	mustAcceptPlannedFailure(t, s, "run-plan-ok", now, "idem-plan-ok-1", "gate-attempt-1")
	mustAcceptPlannedFailure(t, s, "run-plan-ok", now.Add(time.Minute), "idem-plan-ok-2", "gate-attempt-2")
	third := plannedFailedGateResult(now.Add(2*time.Minute), "idem-plan-ok-3", "gate-attempt-3")
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-plan-ok-3", RunID: "run-plan-ok", Report: third}, RequestContext{})
	if errResp == nil {
		t.Fatal("third HandleRunnerResultReport error=nil, want max_attempts fail-closed rejection")
	}
	runResp := mustRunGet(t, s, "run-plan-ok", "req-plan-ok-get")
	lastResult, _ := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	details, _ := lastResult["details"].(map[string]any)
	if got, _ := details["run_plan_id"].(string); got != "plan_run_plan_ok_0001" {
		t.Fatalf("last_result.run_plan_id = %q, want plan_run_plan_ok_0001", got)
	}
	if got, _ := details["workflow_definition_hash"].(string); !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("last_result.workflow_definition_hash = %q, want sha256:*", got)
	}
}

func TestRunnerGateReportsUseSupersedingTrustedRunPlanAfterCacheWarm(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 11, 10, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-supersede-cache")
	putTrustedRunPlanForGatePlan(t, s, "run-plan-supersede-cache", "plan_run_plan_supersede_cache_0001", 1)
	mustAcceptPlannedFailure(t, s, "run-plan-supersede-cache", now, "idem-plan-supersede-cache-1", "gate-attempt-1")

	rejected := plannedFailedGateResult(now.Add(time.Minute), "idem-plan-supersede-cache-2", "gate-attempt-2")
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-plan-supersede-cache-2", RunID: "run-plan-supersede-cache", Report: rejected}, RequestContext{})
	if errResp == nil {
		t.Fatal("second HandleRunnerResultReport error=nil, want max_attempts fail-closed rejection before superseding plan")
	}

	putTrustedRunPlanForGatePlanWithOptions(t, s, "run-plan-supersede-cache", "plan_run_plan_supersede_cache_0002", 2, trustedRunPlanOptions{SupersedesPlanID: "plan_run_plan_supersede_cache_0001"})
	mustAcceptPlannedFailure(t, s, "run-plan-supersede-cache", now.Add(2*time.Minute), "idem-plan-supersede-cache-3", "gate-attempt-2")

	runResp := mustRunGet(t, s, "run-plan-supersede-cache", "req-plan-supersede-cache-get")
	lastResult, _ := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	details, _ := lastResult["details"].(map[string]any)
	if got, _ := details["run_plan_id"].(string); got != "plan_run_plan_supersede_cache_0002" {
		t.Fatalf("last_result.run_plan_id = %q, want plan_run_plan_supersede_cache_0002", got)
	}
}

func TestRunnerGateReportsRejectPlannedScopeMismatch(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 11, 30, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-scope-mismatch")
	putTrustedRunPlanForGatePlan(t, s, "run-plan-scope-mismatch", "plan_run_plan_scope_mismatch_0001", 2)
	report := plannedFailedGateResult(now, "idem-scope-mismatch", "gate-attempt-1")
	report.StageID = "wrong-stage"
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-scope-mismatch", RunID: "run-plan-scope-mismatch", Report: report}, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerGateReportsFailClosedOnPlannedProjectContextDrift(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 11, 45, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-context-drift")
	putTrustedRunPlanForGatePlanWithOptions(t, s, "run-plan-context-drift", "plan_run_plan_context_drift_0001", 2, trustedRunPlanOptions{ProjectContextIdentityDigest: "sha256:" + strings.Repeat("f", 64)})
	report := plannedFailedGateResult(now, "idem-context-drift", "gate-attempt-1")
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-context-drift", RunID: "run-plan-context-drift", Report: report}, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerGateReportsPropagateDependencyCacheHandoffsMetadata(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 12, 15, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-deps")
	putTrustedRunPlanForGatePlanWithOptions(t, s, "run-plan-deps", "plan_run_plan_deps_0001", 2, trustedRunPlanOptions{DependencyCacheHandoffs: []map[string]any{{"request_digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "consumer_role": "workspace-edit", "required": true}}})
	mustAcceptPlannedFailure(t, s, "run-plan-deps", now, "idem-plan-deps", "gate-attempt-1")
	runResp := mustRunGet(t, s, "run-plan-deps", "req-plan-deps-get")
	lastResult, _ := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	details, _ := lastResult["details"].(map[string]any)
	switch raw := details["dependency_cache_handoffs"].(type) {
	case []any:
		if len(raw) != 1 {
			t.Fatalf("last_result.details.dependency_cache_handoffs len = %d, want 1", len(raw))
		}
	case []map[string]any:
		if len(raw) != 1 {
			t.Fatalf("last_result.details.dependency_cache_handoffs len = %d, want 1", len(raw))
		}
	default:
		t.Fatalf("last_result.details.dependency_cache_handoffs = %#v, want single handoff", details["dependency_cache_handoffs"])
	}
}

func TestRunnerGateReportsRejectMismatchedPolicyContextHashAgainstPlanBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 12, 45, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-policy-mismatch")
	putTrustedRunPlanForGatePlan(t, s, "run-plan-policy-mismatch", "plan_run_plan_policy_mismatch_0001", 2)
	report := plannedFailedGateResult(now, "idem-policy-mismatch", "gate-attempt-1")
	report.Details = map[string]any{"policy_context_hash": "sha256:" + strings.Repeat("e", 64)}
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-policy-mismatch", RunID: "run-plan-policy-mismatch", Report: report}, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerResultReportRejectsGateScopedRunResultCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-invalid-code")
	putTrustedRunPlanForGatePlan(t, s, "run-plan-invalid-code", "plan_run_plan_invalid_code_0001", 2)
	report := plannedFailedGateResult(now, "idem-plan-invalid-code", "gate-attempt-1")
	report.ResultCode = "run_failed"
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-plan-invalid-code", RunID: "run-plan-invalid-code", Report: report}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error=nil, want gate-scoped run_* result_code rejection")
	}
}

func TestRunnerGateReportsRejectLegacyProcessDefinitionAuthority(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 10, 30, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-legacy-process")
	putTrustedProcessDefinitionForGatePlanLegacy(t, s, "run-plan-legacy-process", 2)
	req := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-plan-legacy-process", RunID: "run-plan-legacy-process", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "gate_started", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-plan-legacy-process", PlanCheckpointCode: "step_validation_started", PlanOrderIndex: 0, GateID: "policy_gate", GateKind: "policy", GateVersion: "1.0.0", GateLifecycleState: "running", GateAttemptID: "gate-attempt-1", NormalizedInputDigests: []string{"sha256:" + strings.Repeat("a", 64)}}}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerResultReportRejectsMismatchedGateEvidenceRefWithoutPersistingArtifact(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)
	req := buildGateEvidenceResultRequest(now)
	req.RunID, req.RequestID = "run-evidence-mismatch", "req-evidence-mismatch"
	req.Report.GateEvidenceRef = "sha256:" + strings.Repeat("f", 64)
	_, errResp := s.HandleRunnerResultReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
	for _, rec := range s.List() {
		if rec.RunID == "run-evidence-mismatch" && rec.Reference.DataClass == artifacts.DataClassGateEvidence {
			t.Fatalf("unexpected persisted gate evidence artifact %q for rejected request", rec.Reference.Digest)
		}
	}
}

func plannedFailedGateResult(now time.Time, idempotencyKey, attemptID string) RunnerResultReport {
	r := buildFailedGateResult(now, idempotencyKey, attemptID, "sha256:"+strings.Repeat("a", 64))
	r.PlanCheckpointCode, r.PlanOrderIndex = "step_validation_started", 0
	return r
}

func mustAcceptPlannedFailure(t *testing.T, s *Service, runID string, now time.Time, idem, attemptID string) {
	t.Helper()
	report := plannedFailedGateResult(now, idem, attemptID)
	mustAcceptRunnerResult(t, s, runID, "req-"+idem, report)
}
