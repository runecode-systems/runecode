package brokerapi

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

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
	return policyengine.PolicyDecision{SchemaID: "runecode.protocol.v0.PolicyDecision", SchemaVersion: "0.3.0", DecisionOutcome: policyengine.DecisionRequireHumanApproval, PolicyReasonCode: "approval_required", ManifestHash: "sha256:" + strings.Repeat("c", 64), PolicyInputHashes: []string{"sha256:" + strings.Repeat("d", 64)}, ActionRequestHash: actionHash, RelevantArtifactHashes: []string{failedRef}, DetailsSchemaID: "runecode.protocol.details.policy.evaluation.v0", Details: map[string]any{"precedence": "invariants_hard_floor"}, RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.hard_floor.v0", RequiredApproval: map[string]any{"approval_trigger_code": "gate_override", "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "scope": map[string]any{"schema_id": "runecode.protocol.v0.ApprovalBoundScope", "schema_version": "0.1.0", "workspace_id": workspaceIDForRun(runID), "run_id": runID, "action_kind": policyengine.ActionKindGateOverride}, "changes_if_approved": "gate override continuation", "approval_ttl_seconds": 1200}}
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
	req := RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-evidence", RunID: "run-gate-evidence"}
	req.Report = RunnerResultReport{SchemaID: "runecode.protocol.v0.RunnerResultReport", SchemaVersion: "0.1.0", LifecycleState: "failed", ResultCode: "gate_failed", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-gate-evidence-1", GateID: "policy_gate", GateKind: "policy", GateVersion: "1.0.0", GateLifecycleState: "failed", StageID: "stage-1", StepID: "step-1", RoleInstanceID: "role-1", GateAttemptID: "gate-attempt-9", GateEvidence: buildGateEvidencePayload(now, req.RunID, "gate-attempt-9")}
	return req
}

func buildGateEvidencePayload(now time.Time, runID, attemptID string) *GateEvidence {
	return &GateEvidence{SchemaID: "runecode.protocol.v0.GateEvidence", SchemaVersion: "0.1.0", GateID: "policy_gate", GateKind: "policy", GateVersion: "1.0.0", RunID: runID, StageID: "stage-1", StepID: "step-1", RoleInstanceID: "role-1", GateAttemptID: attemptID, StartedAt: now.Add(-2 * time.Minute).Format(time.RFC3339), FinishedAt: now.Format(time.RFC3339), Runtime: map[string]any{"tool": "policyengine"}, Outcome: map[string]any{"deterministic_outcome": "failed"}, OutputArtifactDigests: []string{"sha256:" + strings.Repeat("d", 64)}}
}

func prepareConsumableGateOverride(t *testing.T, s *Service, now time.Time, runID string) RunnerResultReport {
	t.Helper()
	if err := s.SetRunStatus(runID, "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	failed := buildFailedGateResult(now, "idem-gate-failed-consume", "gate-attempt-1", "sha256:"+strings.Repeat("a", 64))
	if _, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-failed-consume", RunID: runID, Report: failed}, RequestContext{}); errResp != nil {
		t.Fatalf("failed HandleRunnerResultReport error response: %+v", errResp)
	}
	failedRef, err := canonicalGateResultRef(runID, failed, "")
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
	if err := s.RecordPolicyDecision(runID, "", buildGateOverridePolicyDecision(runID, actionHash, failedRef)); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	approvePendingGateOverride(t, s, now, runID)
	return override
}

func assertConsumedGateOverrideApproval(t *testing.T, s *Service, runID string) {
	t.Helper()
	for _, ap := range s.ApprovalList() {
		if ap.RunID == runID && ap.ActionKind == policyengine.ActionKindGateOverride && ap.Status == "consumed" {
			if ap.ConsumedAt == nil {
				t.Fatal("consumed approval missing consumed_at")
			}
			return
		}
	}
	t.Fatal("expected gate override approval to be consumed")
}

func putTrustedWorkflowDefinitionForGatePlan(t *testing.T, s *Service, runID string, maxAttempts int) {
	t.Helper()
	payload := `{"schema_id":"runecode.protocol.v0.WorkflowDefinition","schema_version":"0.2.0","workflow_id":"workflow_main","executor_bindings":[{"binding_id":"binding_workspace_runner","executor_id":"workspace-runner","executor_class":"workspace_ordinary","allowed_role_kinds":["workspace-edit","workspace-test"]}],"gate_definitions":[{"schema_id":"runecode.protocol.v0.GateDefinition","schema_version":"0.1.0","checkpoint_code":"step_validation_started","order_index":0,"role_instance_id":"workspace_editor_1","executor_binding_id":"binding_workspace_runner","gate":{"schema_id":"runecode.protocol.v0.GateContract","schema_version":"0.1.0","gate_id":"policy_gate","gate_kind":"policy","gate_version":"1.0.0","normalized_inputs":[{"input_id":"policy_context","input_digest":"sha256:` + strings.Repeat("a", 64) + `"}],"plan_binding":{"checkpoint_code":"step_validation_started","order_index":0},"retry_semantics":{"retry_mode":"new_attempt_required","max_attempts":` + strconv.Itoa(maxAttempts) + `},"override_semantics":{"override_mode":"policy_action_required","action_kind":"action_gate_override","approval_trigger_code":"gate_override"}}}]}`
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte(payload), ContentType: "application/json", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("2", 64), CreatedByRole: "brokerapi", TrustedSource: true, RunID: runID, StepID: "plan"}); err != nil {
		t.Fatalf("Put trusted workflow definition returned error: %v", err)
	}
}
