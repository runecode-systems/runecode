package brokerapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestHandleRunnerTransportLineRejectsCheckpointRunIDMismatchOrEmpty(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-payload")

	tests := []struct {
		name         string
		requestRunID string
		wantErr      string
	}{
		{name: "empty", requestRunID: "", wantErr: "run_id is required"},
		{name: "mismatch", requestRunID: "run-payload", wantErr: "run_id must match bridged run_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := mustMarshalRunnerTransportLine(t, "runner_checkpoint_report_request", RunnerCheckpointReportRequest{
				SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
				SchemaVersion: "0.1.0",
				RequestID:     "req-checkpoint-transport",
				RunID:         tt.requestRunID,
				Report: RunnerCheckpointReport{
					SchemaID:       "runecode.protocol.v0.RunnerCheckpointReport",
					SchemaVersion:  "0.1.0",
					LifecycleState: "active",
					CheckpointCode: "step_attempt_started",
					OccurredAt:     now.Format(time.RFC3339),
					IdempotencyKey: "idem-checkpoint-transport",
				},
			})

			_, err := s.handleRunnerTransportLine(context.Background(), "bridge-request", "run-bridge", line, 1)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("handleRunnerTransportLine error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestHandleRunnerTransportLineRejectsResultRunIDMismatchOrEmpty(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-payload", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}

	tests := []struct {
		name         string
		requestRunID string
		wantErr      string
	}{
		{name: "empty", requestRunID: "", wantErr: "run_id is required"},
		{name: "mismatch", requestRunID: "run-payload", wantErr: "run_id must match bridged run_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := runnerResultTransportLineForTests(t, tt.requestRunID, now)

			_, err := s.handleRunnerTransportLine(context.Background(), "bridge-request", "run-bridge", line, 1)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("handleRunnerTransportLine error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestHandleRunnerTransportLineAcceptsMatchingRunID(t *testing.T) {
	t.Run("checkpoint", testHandleRunnerTransportLineAcceptsMatchingCheckpointRunID)
	t.Run("result", testHandleRunnerTransportLineAcceptsMatchingResultRunID)
}

func testHandleRunnerTransportLineAcceptsMatchingCheckpointRunID(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	putRunnerSeedArtifact(t, s, "run-bridge")

	line := mustMarshalRunnerTransportLine(t, "runner_checkpoint_report_request", RunnerCheckpointReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-checkpoint-transport",
		RunID:         "run-bridge",
		Report: RunnerCheckpointReport{
			SchemaID:       "runecode.protocol.v0.RunnerCheckpointReport",
			SchemaVersion:  "0.1.0",
			LifecycleState: "active",
			CheckpointCode: "step_attempt_started",
			OccurredAt:     now.Format(time.RFC3339),
			IdempotencyKey: "idem-checkpoint-transport",
		},
	})

	resp, err := s.handleRunnerTransportLine(context.Background(), "bridge-request", "run-bridge", line, 2)
	if err != nil {
		t.Fatalf("handleRunnerTransportLine returned error: %v", err)
	}
	payload, ok := resp.Payload.(RunnerCheckpointReportResponse)
	if !ok {
		t.Fatalf("response payload type = %T, want RunnerCheckpointReportResponse", resp.Payload)
	}
	if resp.MessageType != "runner_checkpoint_report_response" || payload.RequestID != "req-checkpoint-transport" || payload.RunID != "run-bridge" || !payload.Accepted {
		t.Fatalf("unexpected checkpoint transport response: %+v payload=%+v", resp, payload)
	}
}

func testHandleRunnerTransportLineAcceptsMatchingResultRunID(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 12, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-bridge", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}

	line := mustMarshalRunnerTransportLine(t, "runner_result_report_request", RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-result-transport",
		RunID:         "run-bridge",
		Report: RunnerResultReport{
			SchemaID:          "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:     "0.1.0",
			LifecycleState:    "failed",
			ResultCode:        "run_failed",
			OccurredAt:        now.Format(time.RFC3339),
			IdempotencyKey:    "idem-result-transport",
			FailureReasonCode: "policy_denied",
		},
	})

	resp, err := s.handleRunnerTransportLine(context.Background(), "bridge-request", "run-bridge", line, 3)
	if err != nil {
		t.Fatalf("handleRunnerTransportLine returned error: %v", err)
	}
	payload, ok := resp.Payload.(RunnerResultReportResponse)
	if !ok {
		t.Fatalf("response payload type = %T, want RunnerResultReportResponse", resp.Payload)
	}
	if resp.MessageType != "runner_result_report_response" || payload.RequestID != "req-result-transport" || payload.RunID != "run-bridge" || !payload.Accepted {
		t.Fatalf("unexpected result transport response: %+v payload=%+v", resp, payload)
	}
}

func runnerResultTransportLineForTests(t *testing.T, runID string, occurredAt time.Time) []byte {
	t.Helper()
	return mustMarshalRunnerTransportLine(t, "runner_result_report_request", RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-result-transport",
		RunID:         runID,
		Report: RunnerResultReport{
			SchemaID:          "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:     "0.1.0",
			LifecycleState:    "failed",
			ResultCode:        "run_failed",
			OccurredAt:        occurredAt.Format(time.RFC3339),
			IdempotencyKey:    "idem-result-transport",
			FailureReasonCode: "policy_denied",
		},
	})
}

func mustMarshalRunnerTransportLine(t *testing.T, messageType string, payload any) []byte {
	t.Helper()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal payload returned error: %v", err)
	}
	line, err := json.Marshal(stdioRunnerTransportRequest{MessageType: messageType, Payload: payloadBytes})
	if err != nil {
		t.Fatalf("json.Marshal transport request returned error: %v", err)
	}
	return line
}
