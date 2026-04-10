package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
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
			SchemaID:               "runecode.protocol.v0.RunnerCheckpointReport",
			SchemaVersion:          "0.1.0",
			LifecycleState:         "blocked",
			CheckpointCode:         "approval_wait_entered",
			OccurredAt:             now.Format(time.RFC3339),
			IdempotencyKey:         "idem-1",
			GateID:                 "policy_gate",
			GateKind:               "policy",
			GateVersion:            "1.0.0",
			GateLifecycleState:     "running",
			StageAttemptID:         "stage-attempt-1",
			StepAttemptID:          "step-attempt-1",
			GateAttemptID:          "gate-attempt-1",
			NormalizedInputDigests: []string{"sha256:" + strings.Repeat("a", 64)},
			PendingApprovalCount:   2,
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
	if lastCheckpoint["gate_id"] != "policy_gate" {
		t.Fatalf("advisory_state.last_checkpoint.gate_id = %v, want policy_gate", lastCheckpoint["gate_id"])
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

func TestRunnerCheckpointReportRejectsPartialGateBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC)
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("1", 64), CreatedByRole: "workspace", RunID: "run-gate-partial", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerCheckpointReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-gate-partial",
		RunID:         "run-gate-partial",
		Report: RunnerCheckpointReport{
			SchemaID:           "runecode.protocol.v0.RunnerCheckpointReport",
			SchemaVersion:      "0.1.0",
			LifecycleState:     "active",
			CheckpointCode:     "gate_started",
			OccurredAt:         now.Format(time.RFC3339),
			IdempotencyKey:     "idem-gate-partial",
			GateID:             "policy_gate",
			GateKind:           "policy",
			GateLifecycleState: "running",
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerCheckpointReport error = nil, want gate binding validation failure")
	}
	if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}

func TestRunnerResultReportRejectsOverriddenWithoutReference(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-overridden", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-gate-overridden",
		RunID:         "run-gate-overridden",
		Report: RunnerResultReport{
			SchemaID:               "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:          "0.1.0",
			LifecycleState:         "failed",
			ResultCode:             "gate_overridden",
			OccurredAt:             now.Format(time.RFC3339),
			IdempotencyKey:         "idem-gate-overridden",
			GateID:                 "policy_gate",
			GateKind:               "policy",
			GateVersion:            "1.0.0",
			GateLifecycleState:     "overridden",
			GateAttemptID:          "gate-attempt-2",
			NormalizedInputDigests: []string{"sha256:" + strings.Repeat("b", 64)},
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error = nil, want overridden reference validation failure")
	}
	if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}

func TestRunnerResultReportRejectsOverrideWithoutPolicyContextDigestBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 35, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-override-no-context", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	failed := buildFailedGateResult(now, "idem-failed-no-context", "gate-attempt-1", "sha256:"+strings.Repeat("a", 64))
	if _, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-failed-no-context", RunID: "run-gate-override-no-context", Report: failed}, RequestContext{}); errResp != nil {
		t.Fatalf("failed HandleRunnerResultReport error response: %+v", errResp)
	}
	failedRef, err := canonicalGateResultRef("run-gate-override-no-context", failed, "")
	if err != nil {
		t.Fatalf("canonicalGateResultRef returned error: %v", err)
	}
	override := buildOverriddenGateResult(now, failedRef)
	delete(override.Details, "policy_context_hash")
	if _, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-override-no-context", RunID: "run-gate-override-no-context", Report: override}, RequestContext{}); errResp == nil {
		t.Fatal("HandleRunnerResultReport error=nil, want missing policy_context_hash rejection")
	}
}

func TestRunnerResultReportRejectsReuseOfTerminalGateAttemptID(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 45, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-attempt-reuse", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	first := RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-gate-attempt-first",
		RunID:         "run-gate-attempt-reuse",
		Report: RunnerResultReport{
			SchemaID:           "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:      "0.1.0",
			LifecycleState:     "failed",
			ResultCode:         "gate_failed",
			OccurredAt:         now.Format(time.RFC3339),
			IdempotencyKey:     "idem-gate-attempt-first",
			GateID:             "policy_gate",
			GateKind:           "policy",
			GateVersion:        "1.0.0",
			GateLifecycleState: "failed",
			GateAttemptID:      "gate-attempt-same",
		},
	}
	if _, errResp := s.HandleRunnerResultReport(context.Background(), first, RequestContext{}); errResp != nil {
		t.Fatalf("first HandleRunnerResultReport error response: %+v", errResp)
	}
	second := first
	second.RequestID = "req-gate-attempt-second"
	second.Report.IdempotencyKey = "idem-gate-attempt-second"
	second.Report.OccurredAt = now.Add(time.Minute).Format(time.RFC3339)
	if _, errResp := s.HandleRunnerResultReport(context.Background(), second, RequestContext{}); errResp == nil {
		t.Fatal("second HandleRunnerResultReport error = nil, want attempt reuse rejection")
	} else if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}

func TestRunnerResultReportOverrideRequiresApprovedGateOverrideBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 19, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	if err := s.SetRunStatus("run-gate-override-approval", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	failed := buildFailedGateResult(now, "idem-gate-failed", "gate-attempt-1", "sha256:"+strings.Repeat("a", 64))
	if _, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-failed", RunID: "run-gate-override-approval", Report: failed}, RequestContext{}); errResp != nil {
		t.Fatalf("failed HandleRunnerResultReport error response: %+v", errResp)
	}
	failedRef, err := canonicalGateResultRef("run-gate-override-approval", failed, "")
	if err != nil {
		t.Fatalf("canonicalGateResultRef returned error: %v", err)
	}
	override := buildOverriddenGateResult(now, failedRef)
	if _, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-reject", RunID: "run-gate-override-approval", Report: override}, RequestContext{}); errResp == nil {
		t.Fatal("HandleRunnerResultReport error=nil, want missing override approval rejection")
	}
	action, err := overrideActionForResult(override, override.Details)
	if err != nil {
		t.Fatalf("overrideActionForResult returned error: %v", err)
	}
	actionHash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		t.Fatalf("CanonicalActionRequestHash returned error: %v", err)
	}
	decision := buildGateOverridePolicyDecision("run-gate-override-approval", actionHash, failedRef)
	if err := s.RecordPolicyDecision("run-gate-override-approval", "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	approvePendingGateOverride(t, s, now, "run-gate-override-approval")
	resp, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-accept", RunID: "run-gate-override-approval", Report: override}, RequestContext{})
	if errResp != nil {
		t.Fatalf("override HandleRunnerResultReport error response: %+v", errResp)
	}
	if !resp.Accepted {
		t.Fatal("override accepted=false, want true")
	}
	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-override", RunID: "run-gate-override-approval"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	lastResult, ok := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.last_result = %T, want map", runResp.Run.AdvisoryState["last_result"])
	}
	if _, ok := lastResult["override_action_request_hash"].(string); !ok {
		t.Fatalf("last_result.override_action_request_hash missing: %#v", lastResult)
	}
	if _, ok := lastResult["override_policy_decision_ref"].(string); !ok {
		t.Fatalf("last_result.override_policy_decision_ref missing: %#v", lastResult)
	}
}

func TestRunnerResultReportPersistsGateEvidenceArtifactAndLinksRunAdvisory(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-evidence", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	resp, errResp := s.HandleRunnerResultReport(context.Background(), buildGateEvidenceResultRequest(now), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunnerResultReport error response: %+v", errResp)
	}
	if !resp.Accepted {
		t.Fatal("accepted = false, want true")
	}

	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-evidence", RunID: "run-gate-evidence"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	lastResult, ok := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.last_result = %T, want map", runResp.Run.AdvisoryState["last_result"])
	}
	gateEvidenceRef, ok := lastResult["gate_evidence_ref"].(string)
	if !ok || !strings.HasPrefix(gateEvidenceRef, "sha256:") {
		t.Fatalf("gate_evidence_ref = %v, want digest identity", lastResult["gate_evidence_ref"])
	}

	record, err := s.Head(gateEvidenceRef)
	if err != nil {
		t.Fatalf("Head(gateEvidenceRef) returned error: %v", err)
	}
	if record.Reference.DataClass != artifacts.DataClassGateEvidence {
		t.Fatalf("gate evidence data_class = %q, want gate_evidence", record.Reference.DataClass)
	}
}

func buildFailedGateResult(now time.Time, idempotencyKey, gateAttemptID, normalizedInput string) RunnerResultReport {
	return RunnerResultReport{
		SchemaID:               "runecode.protocol.v0.RunnerResultReport",
		SchemaVersion:          "0.1.0",
		LifecycleState:         "failed",
		ResultCode:             "gate_failed",
		OccurredAt:             now.Format(time.RFC3339),
		IdempotencyKey:         idempotencyKey,
		GateID:                 "policy_gate",
		GateKind:               "policy",
		GateVersion:            "1.0.0",
		GateLifecycleState:     "failed",
		GateAttemptID:          gateAttemptID,
		NormalizedInputDigests: []string{normalizedInput},
	}
}

func buildOverriddenGateResult(now time.Time, failedRef string) RunnerResultReport {
	return RunnerResultReport{
		SchemaID:                  "runecode.protocol.v0.RunnerResultReport",
		SchemaVersion:             "0.1.0",
		LifecycleState:            "failed",
		ResultCode:                "gate_overridden",
		OccurredAt:                now.Add(2 * time.Minute).Format(time.RFC3339),
		IdempotencyKey:            "idem-gate-override",
		GateID:                    "policy_gate",
		GateKind:                  "policy",
		GateVersion:               "1.0.0",
		GateLifecycleState:        "overridden",
		GateAttemptID:             "gate-attempt-2",
		NormalizedInputDigests:    []string{"sha256:" + strings.Repeat("b", 64)},
		OverriddenFailedResultRef: failedRef,
		Details: map[string]any{
			"policy_context_hash": "sha256:" + strings.Repeat("b", 64),
			"override_reason":     "incident mitigation",
			"ticket_ref":          "INC-99",
			"override_expires_at": now.Add(20 * time.Minute).Format(time.RFC3339),
		},
	}
}

func buildGateOverridePolicyDecision(runID, actionHash, failedRef string) policyengine.PolicyDecision {
	return policyengine.PolicyDecision{
		SchemaID:                 "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:            "0.3.0",
		DecisionOutcome:          policyengine.DecisionRequireHumanApproval,
		PolicyReasonCode:         "approval_required",
		ManifestHash:             "sha256:" + strings.Repeat("c", 64),
		PolicyInputHashes:        []string{"sha256:" + strings.Repeat("d", 64)},
		ActionRequestHash:        actionHash,
		RelevantArtifactHashes:   []string{failedRef},
		DetailsSchemaID:          "runecode.protocol.details.policy.evaluation.v0",
		Details:                  map[string]any{"precedence": "invariants_hard_floor"},
		RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.hard_floor.v0",
		RequiredApproval: map[string]any{
			"approval_trigger_code":    "gate_override",
			"approval_assurance_level": "reauthenticated",
			"presence_mode":            "hardware_touch",
			"scope": map[string]any{
				"schema_id":      "runecode.protocol.v0.ApprovalBoundScope",
				"schema_version": "0.1.0",
				"workspace_id":   workspaceIDForRun(runID),
				"run_id":         runID,
				"action_kind":    policyengine.ActionKindGateOverride,
			},
			"changes_if_approved":  "gate override continuation",
			"approval_ttl_seconds": 1200,
		},
	}
}

func approvePendingGateOverride(t *testing.T, s *Service, now time.Time, runID string) {
	t.Helper()
	for _, ap := range s.ApprovalList() {
		if ap.RunID != runID || ap.ActionKind != policyengine.ActionKindGateOverride || ap.Status != "pending" {
			continue
		}
		ap.Status = "approved"
		decided := now.Add(3 * time.Minute)
		ap.DecidedAt = &decided
		if ap.ExpiresAt == nil {
			ex := now.Add(10 * time.Minute)
			ap.ExpiresAt = &ex
		}
		if err := s.RecordApproval(ap); err != nil {
			t.Fatalf("RecordApproval(approved) returned error: %v", err)
		}
		return
	}
	t.Fatal("missing pending gate override approval to approve")
}

func buildGateEvidenceResultRequest(now time.Time) RunnerResultReportRequest {
	return RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-gate-evidence",
		RunID:         "run-gate-evidence",
		Report: RunnerResultReport{
			SchemaID:           "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:      "0.1.0",
			LifecycleState:     "failed",
			ResultCode:         "gate_failed",
			OccurredAt:         now.Format(time.RFC3339),
			IdempotencyKey:     "idem-gate-evidence-1",
			GateID:             "policy_gate",
			GateKind:           "policy",
			GateVersion:        "1.0.0",
			GateLifecycleState: "failed",
			StageID:            "stage-1",
			StepID:             "step-1",
			RoleInstanceID:     "role-1",
			GateAttemptID:      "gate-attempt-9",
			GateEvidence: &GateEvidence{
				SchemaID:              "runecode.protocol.v0.GateEvidence",
				SchemaVersion:         "0.1.0",
				GateID:                "policy_gate",
				GateKind:              "policy",
				GateVersion:           "1.0.0",
				RunID:                 "run-gate-evidence",
				StageID:               "stage-1",
				StepID:                "step-1",
				RoleInstanceID:        "role-1",
				GateAttemptID:         "gate-attempt-9",
				StartedAt:             now.Add(-2 * time.Minute).Format(time.RFC3339),
				FinishedAt:            now.Format(time.RFC3339),
				Runtime:               map[string]any{"tool": "policyengine"},
				Outcome:               map[string]any{"deterministic_outcome": "failed"},
				OutputArtifactDigests: []string{"sha256:" + strings.Repeat("d", 64)},
			},
		},
	}
}

func TestRunnerResultReportRejectsMalformedGateEvidenceFailClosed(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 12, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-evidence-invalid", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}

	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-gate-evidence-invalid",
		RunID:         "run-gate-evidence-invalid",
		Report: RunnerResultReport{
			SchemaID:           "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:      "0.1.0",
			LifecycleState:     "failed",
			ResultCode:         "gate_failed",
			OccurredAt:         now.Format(time.RFC3339),
			IdempotencyKey:     "idem-gate-evidence-invalid",
			GateID:             "policy_gate",
			GateKind:           "policy",
			GateVersion:        "1.0.0",
			GateLifecycleState: "failed",
			GateAttemptID:      "gate-attempt-10",
			GateEvidence: &GateEvidence{
				SchemaID:      "runecode.protocol.v0.GateEvidence",
				SchemaVersion: "0.1.0",
				GateID:        "policy_gate",
				GateKind:      "policy",
				GateVersion:   "1.0.0",
				RunID:         "run-gate-evidence-invalid",
				GateAttemptID: "gate-attempt-10",
				StartedAt:     now.Add(-time.Minute).Format(time.RFC3339),
				FinishedAt:    now.Format(time.RFC3339),
				Runtime:       map[string]any{"tool": "policyengine"},
				Outcome:       map[string]any{},
			},
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error = nil, want malformed gate evidence validation failure")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestRunnerResultReportRejectsMismatchedProvidedGateEvidenceRef(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 12, 45, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-evidence-ref-mismatch", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}

	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-gate-evidence-ref-mismatch",
		RunID:         "run-gate-evidence-ref-mismatch",
		Report: RunnerResultReport{
			SchemaID:           "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:      "0.1.0",
			LifecycleState:     "failed",
			ResultCode:         "gate_failed",
			OccurredAt:         now.Format(time.RFC3339),
			IdempotencyKey:     "idem-gate-evidence-ref-mismatch",
			GateID:             "policy_gate",
			GateKind:           "policy",
			GateVersion:        "1.0.0",
			GateLifecycleState: "failed",
			GateAttemptID:      "gate-attempt-11",
			GateEvidenceRef:    "sha256:" + strings.Repeat("a", 64),
			GateEvidence: &GateEvidence{
				SchemaID:      "runecode.protocol.v0.GateEvidence",
				SchemaVersion: "0.1.0",
				GateID:        "policy_gate",
				GateKind:      "policy",
				GateVersion:   "1.0.0",
				RunID:         "run-gate-evidence-ref-mismatch",
				GateAttemptID: "gate-attempt-11",
				StartedAt:     now.Add(-time.Minute).Format(time.RFC3339),
				FinishedAt:    now.Format(time.RFC3339),
				Runtime:       map[string]any{"tool": "policyengine"},
				Outcome:       map[string]any{"deterministic_outcome": "failed"},
			},
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error = nil, want gate_evidence_ref mismatch rejection")
	}
	if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}

func TestRunnerResultReportConsumesGateOverrideApprovalSingleUse(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 13, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	if err := s.SetRunStatus("run-gate-override-consume", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	failed := buildFailedGateResult(now, "idem-gate-failed-consume", "gate-attempt-1", "sha256:"+strings.Repeat("a", 64))
	if _, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-failed-consume", RunID: "run-gate-override-consume", Report: failed}, RequestContext{}); errResp != nil {
		t.Fatalf("failed HandleRunnerResultReport error response: %+v", errResp)
	}
	failedRef, err := canonicalGateResultRef("run-gate-override-consume", failed, "")
	if err != nil {
		t.Fatalf("canonicalGateResultRef returned error: %v", err)
	}
	override := buildOverriddenGateResult(now, failedRef)
	action, err := overrideActionForResult(override, override.Details)
	if err != nil {
		t.Fatalf("overrideActionForResult returned error: %v", err)
	}
	actionHash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		t.Fatalf("CanonicalActionRequestHash returned error: %v", err)
	}
	decision := buildGateOverridePolicyDecision("run-gate-override-consume", actionHash, failedRef)
	if err := s.RecordPolicyDecision("run-gate-override-consume", "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	approvePendingGateOverride(t, s, now, "run-gate-override-consume")

	if _, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-consume-1", RunID: "run-gate-override-consume", Report: override}, RequestContext{}); errResp != nil {
		t.Fatalf("first override HandleRunnerResultReport error response: %+v", errResp)
	}

	consumed := false
	for _, ap := range s.ApprovalList() {
		if ap.RunID == "run-gate-override-consume" && ap.ActionKind == policyengine.ActionKindGateOverride && ap.Status == "consumed" {
			consumed = true
			if ap.ConsumedAt == nil {
				t.Fatal("consumed approval missing consumed_at")
			}
		}
	}
	if !consumed {
		t.Fatal("expected gate override approval to be consumed")
	}
	override.GateAttemptID = "gate-attempt-3"
	override.IdempotencyKey = "idem-gate-override-second-consume"
	override.OccurredAt = now.Add(3 * time.Minute).Format(time.RFC3339)
	if _, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-consume-2", RunID: "run-gate-override-consume", Report: override}, RequestContext{}); errResp == nil {
		t.Fatal("second override should fail after approval is consumed")
	} else if errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("error code = %q, want broker_validation_runner_transition_invalid", errResp.Error.Code)
	}
}

func TestRunnerResultReportSanitizesAndDeepCopiesDetails(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 14, 0, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-details-deepcopy", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	details := map[string]any{
		"nested": map[string]any{"value": "original"},
		"arr":    []any{map[string]any{"k": "v"}},
	}
	request := RunnerResultReportRequest{
		SchemaID:      "runecode.protocol.v0.RunnerResultReportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-details-deepcopy",
		RunID:         "run-details-deepcopy",
		Report: RunnerResultReport{
			SchemaID:       "runecode.protocol.v0.RunnerResultReport",
			SchemaVersion:  "0.1.0",
			LifecycleState: "failed",
			ResultCode:     "run_failed",
			OccurredAt:     now.Format(time.RFC3339),
			IdempotencyKey: "idem-details-deepcopy",
			Details:        details,
		},
	}
	if _, errResp := s.HandleRunnerResultReport(context.Background(), request, RequestContext{}); errResp != nil {
		t.Fatalf("HandleRunnerResultReport error response: %+v", errResp)
	}
	details["nested"].(map[string]any)["value"] = "mutated"
	details["arr"].([]any)[0].(map[string]any)["k"] = "mutated"

	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-details-deepcopy-run", RunID: "run-details-deepcopy"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	lastResult, ok := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.last_result = %T, want map", runResp.Run.AdvisoryState["last_result"])
	}
	storedDetails, ok := lastResult["details"].(map[string]any)
	if !ok {
		t.Fatalf("last_result.details = %T, want map", lastResult["details"])
	}
	nested, ok := storedDetails["nested"].(map[string]any)
	if !ok {
		t.Fatalf("stored nested details = %T, want map", storedDetails["nested"])
	}
	if nested["value"] != "original" {
		t.Fatalf("nested.value = %v, want original", nested["value"])
	}
	arr, ok := storedDetails["arr"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("stored arr = %#v, want single-element array", storedDetails["arr"])
	}
	arrMap, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("stored arr[0] = %T, want map", arr[0])
	}
	if arrMap["k"] != "v" {
		t.Fatalf("arr[0].k = %v, want v", arrMap["k"])
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
