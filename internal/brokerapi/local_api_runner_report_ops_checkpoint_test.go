package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestRunnerCheckpointReportAcceptsValidTransitionAndProjectsAdvisoryState(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	putRunnerSeedArtifact(t, s, "run-checkpoint")

	report := RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "blocked", CheckpointCode: "approval_wait_entered", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-1", GateID: "policy_gate", GateKind: "policy", GateVersion: "1.0.0", GateLifecycleState: "running", StageAttemptID: "stage-attempt-1", StepAttemptID: "step-attempt-1", GateAttemptID: "gate-attempt-1", NormalizedInputDigests: []string{"sha256:" + strings.Repeat("a", 64)}, PendingApprovalCount: 2}
	resp, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-checkpoint", RunID: "run-checkpoint", Report: report}, RequestContext{})
	if errResp != nil || !resp.Accepted || resp.CanonicalLifecycleState != "blocked" {
		t.Fatalf("unexpected checkpoint response: resp=%+v err=%+v", resp, errResp)
	}
	runResp := mustRunGet(t, s, "run-checkpoint", "req-run")
	assertCheckpointProjection(t, runResp)
}

func TestRunnerCheckpointReportRejectsInvalidTransitionFailClosed(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-bad-transition", "starting"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-bad", RunID: "run-bad-transition", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "pending", CheckpointCode: "run_started", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-bad"}}, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
	if got := mustRunGet(t, s, "run-bad-transition", "req-run-bad").Run.Summary.LifecycleState; got != "starting" {
		t.Fatalf("summary.lifecycle_state = %q, want starting", got)
	}
}

func TestRunnerCheckpointReportRejectsPartialGateBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-gate-partial")
	req := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-partial", RunID: "run-gate-partial", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "gate_started", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-gate-partial", GateID: "policy_gate", GateKind: "policy", GateLifecycleState: "running"}}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerCheckpointReportIdempotencyReturnsAcceptedFalseWithoutDuplicateSideEffects(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 14, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-idem")
	req := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-idem", RunID: "run-idem", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "step_attempt_started", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-checkpoint"}}
	resp1, err1 := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{})
	resp2, err2 := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{})
	if err1 != nil || err2 != nil || !resp1.Accepted || resp2.Accepted {
		t.Fatalf("idempotency mismatch: resp1=%+v err1=%+v resp2=%+v err2=%+v", resp1, err1, resp2, err2)
	}
}

func TestRunnerCheckpointReportRejectsInvalidExecutionPhaseOrder(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 15, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-phase")
	req := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-phase-invalid", RunID: "run-phase", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "step_execution_started", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-phase-invalid", StageID: "stage-1", StepID: "step-1", StepAttemptID: "attempt-1"}}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerCheckpointReportEnforcesProposeValidateAuthorizeExecuteAttestSequence(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 16, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-seq")
	mustAcceptCheckpointSequence(t, s, "run-seq", now, []string{"step_attempt_started", "step_validation_started", "approval_wait_entered", "approval_wait_cleared", "step_execution_started", "step_attest_started", "step_attempt_finished"})
	runResp := mustRunGet(t, s, "run-seq", "req-run-seq")
	assertStepAttemptPhase(t, runResp, "attempt-1", "attest", "finished")
}

func TestRunnerCheckpointReportRejectsUnknownCheckpointCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 17, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-code")
	req := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-code", RunID: "run-code", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "unknown_code", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-code"}}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerCheckpointReportProjectsApprovalWaitIntoSessionExecution(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	seedSessionRuntimeFactsForOpsTest(t, s, "run-checkpoint-session", "sess-checkpoint-session")
	mustSessionExecutionTrigger(t, s, SessionExecutionTriggerRequest{SchemaID: "runecode.protocol.v0.SessionExecutionTriggerRequest", SchemaVersion: "0.1.0", RequestID: "req-checkpoint-session-trigger", SessionID: "sess-checkpoint-session", TriggerSource: "interactive_user", RequestedOperation: "start", UserMessageContentText: "run"})

	req := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-checkpoint-session", RunID: "run-checkpoint-session", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "blocked", CheckpointCode: "approval_wait_entered", OccurredAt: now.Add(time.Minute).Format(time.RFC3339), IdempotencyKey: "idem-checkpoint-session", StageID: "stage-1", StepID: "step-1", StepAttemptID: "attempt-1"}}
	if _, errResp := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{}); errResp != nil {
		t.Fatalf("HandleRunnerCheckpointReport error response: %+v", errResp)
	}

	get := mustSessionGet(t, s, "req-checkpoint-session-get", "sess-checkpoint-session")
	if get.Session.CurrentTurnExecution == nil {
		t.Fatal("current_turn_execution missing")
	}
	if get.Session.CurrentTurnExecution.ExecutionState != "waiting" {
		t.Fatalf("execution_state = %q, want waiting", get.Session.CurrentTurnExecution.ExecutionState)
	}
	if get.Session.CurrentTurnExecution.WaitKind != "approval" {
		t.Fatalf("wait_kind = %q, want approval", get.Session.CurrentTurnExecution.WaitKind)
	}
	if get.Session.CurrentTurnExecution.WaitState != "waiting_approval" {
		t.Fatalf("wait_state = %q, want waiting_approval", get.Session.CurrentTurnExecution.WaitState)
	}
}

func putRunnerSeedArtifact(t *testing.T, s *Service, runID string) {
	t.Helper()
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: runID, StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
}

func mustRunGet(t *testing.T, s *Service, runID, reqID string) RunGetResponse {
	t.Helper()
	resp, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: reqID, RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	return resp
}

func assertCheckpointProjection(t *testing.T, runResp RunGetResponse) {
	t.Helper()
	if runResp.Run.Summary.LifecycleState != "blocked" || runResp.Run.AdvisoryState["provenance"] != "runner_reported" {
		t.Fatalf("unexpected run summary/advisory: %+v", runResp.Run)
	}
	last, ok := runResp.Run.AdvisoryState["last_checkpoint"].(map[string]any)
	if !ok || last["step_attempt_id"] != "step-attempt-1" || last["gate_id"] != "policy_gate" {
		t.Fatalf("unexpected last checkpoint: %#v", runResp.Run.AdvisoryState["last_checkpoint"])
	}
}

func mustAcceptCheckpointSequence(t *testing.T, s *Service, runID string, start time.Time, sequence []string) {
	t.Helper()
	for i, code := range sequence {
		req := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-seq-" + code, RunID: runID, Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: code, OccurredAt: start.Add(time.Duration(i) * time.Minute).Format(time.RFC3339), IdempotencyKey: "idem-seq-" + code, StageID: "stage-1", StepID: "step-1", StepAttemptID: "attempt-1"}}
		if _, errResp := s.HandleRunnerCheckpointReport(context.Background(), req, RequestContext{}); errResp != nil {
			t.Fatalf("checkpoint %s rejected: %+v", code, errResp)
		}
	}
}

func assertStepAttemptPhase(t *testing.T, runResp RunGetResponse, attemptID, phase, status string) {
	t.Helper()
	stepAttempts, ok := runResp.Run.AdvisoryState["step_attempts"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.step_attempts = %T, want map[string]any", runResp.Run.AdvisoryState["step_attempts"])
	}
	attempt, ok := stepAttempts[attemptID].(map[string]any)
	if !ok || attempt["current_phase"] != phase || attempt["phase_status"] != status {
		t.Fatalf("unexpected step attempt payload: %#v", stepAttempts[attemptID])
	}
}
