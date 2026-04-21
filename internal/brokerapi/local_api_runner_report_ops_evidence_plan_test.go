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
	putTrustedWorkflowDefinitionForGatePlan(t, s, "run-plan-ok", 2)
	mustAcceptPlannedFailure(t, s, "run-plan-ok", now, "idem-plan-ok-1", "gate-attempt-1")
	mustAcceptPlannedFailure(t, s, "run-plan-ok", now.Add(time.Minute), "idem-plan-ok-2", "gate-attempt-2")
	third := plannedFailedGateResult(now.Add(2*time.Minute), "idem-plan-ok-3", "gate-attempt-3")
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-plan-ok-3", RunID: "run-plan-ok", Report: third}, RequestContext{})
	if errResp == nil {
		t.Fatal("third HandleRunnerResultReport error=nil, want max_attempts fail-closed rejection")
	}
}

func TestRunnerResultReportRejectsGateScopedRunResultCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-plan-invalid-code")
	putTrustedWorkflowDefinitionForGatePlan(t, s, "run-plan-invalid-code", 2)
	report := plannedFailedGateResult(now, "idem-plan-invalid-code", "gate-attempt-1")
	report.ResultCode = "run_failed"
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-plan-invalid-code", RunID: "run-plan-invalid-code", Report: report}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error=nil, want gate-scoped run_* result_code rejection")
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
