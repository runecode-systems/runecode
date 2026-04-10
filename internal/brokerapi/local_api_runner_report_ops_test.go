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
	s.now = func() time.Time { return now }
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: "run-checkpoint", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	resp, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-checkpoint",
		RunID:         "run-checkpoint",
		Report: RunnerCheckpointReport{
			SchemaID:             "runecode.protocol.v0.RunnerCheckpointReport",
			SchemaVersion:        "0.1.0",
			LifecycleState:       "blocked",
			CheckpointCode:       "approval_wait_entered",
			OccurredAt:           now.Format(time.RFC3339),
			IdempotencyKey:       "idem-1",
			StageAttemptID:       "stage-attempt-1",
			StepAttemptID:        "step-attempt-1",
			GateAttemptID:        "gate-attempt-1",
			PendingApprovalCount: 2,
		},
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunnerCheckpointReport error response: %+v", errResp)
	}
	if !resp.Accepted {
		t.Fatal("accepted = false, want true")
	}
	if resp.CanonicalLifecycleState != "blocked" {
		t.Fatalf("canonical_lifecycle_state = %q, want blocked", resp.CanonicalLifecycleState)
	}
	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run", RunID: "run-checkpoint"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	if runResp.Run.Summary.LifecycleState != "blocked" {
		t.Fatalf("summary.lifecycle_state = %q, want blocked", runResp.Run.Summary.LifecycleState)
	}
	if runResp.Run.AdvisoryState["provenance"] != "runner_reported" {
		t.Fatalf("advisory_state.provenance = %v, want runner_reported", runResp.Run.AdvisoryState["provenance"])
	}
	lastCheckpoint, ok := runResp.Run.AdvisoryState["last_checkpoint"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.last_checkpoint = %T, want map", runResp.Run.AdvisoryState["last_checkpoint"])
	}
	if lastCheckpoint["step_attempt_id"] != "step-attempt-1" {
		t.Fatalf("advisory_state.last_checkpoint.step_attempt_id = %v, want step-attempt-1", lastCheckpoint["step_attempt_id"])
	}
}

func TestRunnerCheckpointReportRejectsInvalidTransitionFailClosed(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-bad-transition", "starting"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}

	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-bad",
		RunID:         "run-bad-transition",
		Report: RunnerCheckpointReport{
			SchemaID:       "runecode.protocol.v0.RunnerCheckpointReport",
			SchemaVersion:  "0.1.0",
			LifecycleState: "pending",
			CheckpointCode: "run_started",
			OccurredAt:     now.Format(time.RFC3339),
			IdempotencyKey: "idem-bad",
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerCheckpointReport error = nil, want transition validation failure")
	}
	if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-bad", RunID: "run-bad-transition"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	if runResp.Run.Summary.LifecycleState != "starting" {
		t.Fatalf("summary.lifecycle_state = %q, want starting", runResp.Run.Summary.LifecycleState)
	}
}

func TestRunnerResultReportAcceptsTerminalTransition(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 12, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-terminal", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}

	resp, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-result",
		RunID:         "run-terminal",
		Report: RunnerResultReport{
			SchemaID:          "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:     "0.1.0",
			LifecycleState:    "failed",
			ResultCode:        "run_failed",
			OccurredAt:        now.Format(time.RFC3339),
			IdempotencyKey:    "idem-result",
			FailureReasonCode: "policy_denied",
		},
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunnerResultReport error response: %+v", errResp)
	}
	if !resp.Accepted {
		t.Fatal("accepted = false, want true")
	}
	if resp.CanonicalLifecycleState != "failed" {
		t.Fatalf("canonical_lifecycle_state = %q, want failed", resp.CanonicalLifecycleState)
	}
}

func TestRunnerResultReportRejectsUnknownRunFailClosed(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 12, 30, 0, 0, time.UTC)

	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-result-missing",
		RunID:         "run-missing",
		Report: RunnerResultReport{
			SchemaID:       "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:  "0.1.0",
			LifecycleState: "completed",
			ResultCode:     "run_completed",
			OccurredAt:     now.Format(time.RFC3339),
			IdempotencyKey: "idem-missing",
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error = nil, want transition validation failure")
	}
	if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}

func TestRunnerCheckpointReportIdempotencyReturnsAcceptedFalseWithoutDuplicateSideEffects(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 14, 0, 0, 0, time.UTC)
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: "run-idem", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	request := RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-idem", RunID: "run-idem", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "step_attempt_started", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-checkpoint"}}
	resp1, errResp := s.HandleRunnerCheckpointReport(context.Background(), request, RequestContext{})
	if errResp != nil {
		t.Fatalf("first HandleRunnerCheckpointReport error response: %+v", errResp)
	}
	if !resp1.Accepted {
		t.Fatal("first response accepted=false, want true")
	}
	resp2, errResp := s.HandleRunnerCheckpointReport(context.Background(), request, RequestContext{})
	if errResp != nil {
		t.Fatalf("second HandleRunnerCheckpointReport error response: %+v", errResp)
	}
	if resp2.Accepted {
		t.Fatal("second response accepted=true, want false for idempotent replay")
	}
}

func TestRunnerCheckpointReportRejectsInvalidExecutionPhaseOrder(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 15, 0, 0, 0, time.UTC)
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: "run-phase", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-phase-invalid",
		RunID:         "run-phase",
		Report: RunnerCheckpointReport{
			SchemaID:       "runecode.protocol.v0.RunnerCheckpointReport",
			SchemaVersion:  "0.1.0",
			LifecycleState: "active",
			CheckpointCode: "step_execution_started",
			OccurredAt:     now.Format(time.RFC3339),
			IdempotencyKey: "idem-phase-invalid",
			StageID:        "stage-1",
			StepID:         "step-1",
			StepAttemptID:  "attempt-1",
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerCheckpointReport error = nil, want invalid phase transition")
	}
	if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}

func TestRunnerCheckpointReportEnforcesProposeValidateAuthorizeExecuteAttestSequence(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 16, 0, 0, 0, time.UTC)
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: "run-seq", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	sequence := []string{"step_attempt_started", "step_validation_started", "approval_wait_entered", "approval_wait_cleared", "step_execution_started", "step_attest_started", "step_attempt_finished"}
	for i, code := range sequence {
		_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{
			SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
			SchemaVersion: "0.1.0",
			RequestID:     "req-seq-" + code,
			RunID:         "run-seq",
			Report: RunnerCheckpointReport{
				SchemaID:       "runecode.protocol.v0.RunnerCheckpointReport",
				SchemaVersion:  "0.1.0",
				LifecycleState: "active",
				CheckpointCode: code,
				OccurredAt:     now.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
				IdempotencyKey: "idem-seq-" + code,
				StageID:        "stage-1",
				StepID:         "step-1",
				StepAttemptID:  "attempt-1",
			},
		}, RequestContext{})
		if errResp != nil {
			t.Fatalf("checkpoint %s rejected: %+v", code, errResp)
		}
	}

	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-seq", RunID: "run-seq"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	stepAttempts, ok := runResp.Run.AdvisoryState["step_attempts"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.step_attempts = %T, want map[string]any", runResp.Run.AdvisoryState["step_attempts"])
	}
	attempt, ok := stepAttempts["attempt-1"].(map[string]any)
	if !ok {
		t.Fatalf("step_attempts[attempt-1] = %T, want map[string]any", stepAttempts["attempt-1"])
	}
	if attempt["current_phase"] != "attest" {
		t.Fatalf("current_phase = %v, want attest", attempt["current_phase"])
	}
	if attempt["phase_status"] != "finished" {
		t.Fatalf("phase_status = %v, want finished", attempt["phase_status"])
	}
}

func TestRunnerCheckpointReportRejectsUnknownCheckpointCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 17, 0, 0, 0, time.UTC)
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: "run-code", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-code", RunID: "run-code", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "unknown_code", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-code"}}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerCheckpointReport error = nil, want invalid checkpoint code")
	}
	if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}

func TestRunnerResultReportRejectsUnknownResultCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 17, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-result-code", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-result-code", RunID: "run-result-code", Report: RunnerResultReport{SchemaID: "runecode.protocol.v0.RunnerResultReport", SchemaVersion: "0.1.0", LifecycleState: "failed", ResultCode: "bad_code", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-result-code"}}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error = nil, want invalid result code")
	}
	if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}
