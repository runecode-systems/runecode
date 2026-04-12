package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestRunnerResultReportAcceptsTerminalTransition(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 12, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-terminal", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	req := RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-result", RunID: "run-terminal", Report: RunnerResultReport{SchemaID: "runecode.protocol.v0.RunnerResultReport", SchemaVersion: "0.1.0", LifecycleState: "failed", ResultCode: "run_failed", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-result", FailureReasonCode: "policy_denied"}}
	resp, errResp := s.HandleRunnerResultReport(context.Background(), req, RequestContext{})
	if errResp != nil || !resp.Accepted || resp.CanonicalLifecycleState != "failed" {
		t.Fatalf("unexpected result response: resp=%+v err=%+v", resp, errResp)
	}
}

func TestRunnerResultReportRejectsUnknownRunFailClosed(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 12, 30, 0, 0, time.UTC)
	req := RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-result-missing", RunID: "run-missing", Report: RunnerResultReport{SchemaID: "runecode.protocol.v0.RunnerResultReport", SchemaVersion: "0.1.0", LifecycleState: "completed", ResultCode: "run_completed", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-missing"}}
	_, errResp := s.HandleRunnerResultReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerResultReportRejectsOverriddenWithoutReference(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-overridden", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	report := RunnerResultReport{SchemaID: "runecode.protocol.v0.RunnerResultReport", SchemaVersion: "0.1.0", LifecycleState: "failed", ResultCode: "gate_overridden", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-gate-overridden", GateID: "policy_gate", GateKind: "policy", GateVersion: "1.0.0", GateLifecycleState: "overridden", GateAttemptID: "gate-attempt-2", NormalizedInputDigests: []string{"sha256:" + strings.Repeat("b", 64)}}
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-overridden", RunID: "run-gate-overridden", Report: report}, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerResultReportRejectsOverrideWithoutPolicyContextDigestBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 35, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-override-no-context", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	failed := buildFailedGateResult(now, "idem-failed-no-context", "gate-attempt-1", "sha256:"+strings.Repeat("a", 64))
	mustAcceptRunnerResult(t, s, "run-gate-override-no-context", "req-failed-no-context", failed)
	failedRef, err := canonicalGateResultRef("run-gate-override-no-context", failed, "")
	if err != nil {
		t.Fatalf("canonicalGateResultRef returned error: %v", err)
	}
	override := buildOverriddenGateResult(now, failedRef)
	delete(override.Details, "policy_context_hash")
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-override-no-context", RunID: "run-gate-override-no-context", Report: override}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error=nil, want missing policy_context_hash rejection")
	}
}

func TestRunnerResultReportRejectsReuseOfTerminalGateAttemptID(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 18, 45, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-gate-attempt-reuse", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	first := RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-attempt-first", RunID: "run-gate-attempt-reuse", Report: buildFailedGateResult(now, "idem-gate-attempt-first", "gate-attempt-same", "sha256:"+strings.Repeat("a", 64))}
	mustAcceptRunnerResultReq(t, s, first)
	second := first
	second.RequestID, second.Report.IdempotencyKey, second.Report.OccurredAt = "req-gate-attempt-second", "idem-gate-attempt-second", now.Add(time.Minute).Format(time.RFC3339)
	_, errResp := s.HandleRunnerResultReport(context.Background(), second, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerResultReportOverrideRequiresApprovedGateOverrideBinding(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 19, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	override := prepareOverrideScenario(t, s, now, "run-gate-override-approval")
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-reject", RunID: "run-gate-override-approval", Report: override}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerResultReport error=nil, want missing override approval rejection")
	}
	recordOverrideApproval(t, s, now, "run-gate-override-approval", override)
	mustAcceptRunnerResultReq(t, s, RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-accept", RunID: "run-gate-override-approval", Report: override})
	assertRunLastResultHasOverrideBindings(t, s, "run-gate-override-approval")
}

func TestRunnerResultReportConsumesGateOverrideApprovalSingleUse(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 13, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	override := prepareConsumableGateOverride(t, s, now, "run-gate-override-consume")
	mustAcceptRunnerResultReq(t, s, RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-consume-1", RunID: "run-gate-override-consume", Report: override})
	assertConsumedGateOverrideApproval(t, s, "run-gate-override-consume")
	override.GateAttemptID, override.IdempotencyKey, override.OccurredAt = "gate-attempt-3", "idem-gate-override-second-consume", now.Add(3*time.Minute).Format(time.RFC3339)
	_, errResp := s.HandleRunnerResultReport(context.Background(), RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-consume-2", RunID: "run-gate-override-consume", Report: override}, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestRunnerResultReportOverrideUsesMostRecentMatchingPolicyDecision(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	override := prepareOverrideScenario(t, s, now, "run-gate-override-recency")
	actionHash := mustOverrideActionHash(t, override)
	firstDecision := seedApprovedGateOverrideDecision(t, s, now, "run-gate-override-recency", actionHash, override.OverriddenFailedResultRef, "first")
	now = now.Add(5 * time.Minute)
	secondDecision := seedApprovedGateOverrideDecision(t, s, now, "run-gate-override-recency", actionHash, override.OverriddenFailedResultRef, "second")
	if secondDecision == firstDecision {
		t.Fatal("expected distinct second policy decision digest")
	}
	mustAcceptRunnerResultReq(t, s, RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-recency", RunID: "run-gate-override-recency", Report: override})
	assertOverridePolicyDecisionBinding(t, s, "run-gate-override-recency", secondDecision)
}

func mustOverrideActionHash(t *testing.T, override RunnerResultReport) string {
	t.Helper()
	action, err := overrideActionForResult(override, override.Details)
	if err != nil {
		t.Fatalf("overrideActionForResult returned error: %v", err)
	}
	actionHash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		t.Fatalf("CanonicalActionRequestHash returned error: %v", err)
	}
	return actionHash
}

func seedApprovedGateOverrideDecision(t *testing.T, s *Service, now time.Time, runID, actionHash, failedRef, nonce string) string {
	t.Helper()
	decisionPayload := buildGateOverridePolicyDecision(runID, actionHash, failedRef)
	decisionPayload.Details["decision_nonce"] = nonce
	if err := s.RecordPolicyDecision(runID, "", decisionPayload); err != nil {
		t.Fatalf("RecordPolicyDecision(%s) returned error: %v", nonce, err)
	}
	decisionRef := latestPolicyDecisionRefForAction(t, s, runID, actionHash)
	approvePendingGateOverrideByPolicyDecision(t, s, now, runID, decisionRef)
	return decisionRef
}

func assertOverridePolicyDecisionBinding(t *testing.T, s *Service, runID, wantRef string) {
	t.Helper()
	runResp := mustRunGet(t, s, runID, "req-run-override-recency")
	lastResult, ok := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.last_result = %T, want map", runResp.Run.AdvisoryState["last_result"])
	}
	boundRef, _ := lastResult["override_policy_decision_ref"].(string)
	if boundRef != wantRef {
		t.Fatalf("override_policy_decision_ref = %q, want most recent %q", boundRef, wantRef)
	}
}

func TestRunnerResultReportOverrideAllowsAnyValidMatchingApprovalExpiry(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC)
	s.SetNowFuncForTests(func() time.Time { return now })
	override := prepareOverrideScenario(t, s, now, "run-gate-override-expiry-any-valid")
	actionHash := mustOverrideActionHash(t, override)

	decisionRef := seedApprovedGateOverrideDecision(t, s, now, "run-gate-override-expiry-any-valid", actionHash, override.OverriddenFailedResultRef, "nonce-expiry")
	seedAdditionalApprovedGateOverrideApproval(t, s, now, "run-gate-override-expiry-any-valid", decisionRef, "sha256:"+strings.Repeat("f", 64))
	markApprovedGateOverrideExpired(t, s, now, "run-gate-override-expiry-any-valid", decisionRef)
	now = now.Add(time.Minute)

	mustAcceptRunnerResultReq(t, s, RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-gate-override-expiry-any-valid", RunID: "run-gate-override-expiry-any-valid", Report: override})
	assertOverridePolicyDecisionBinding(t, s, "run-gate-override-expiry-any-valid", decisionRef)
}

func seedAdditionalApprovedGateOverrideApproval(t *testing.T, s *Service, now time.Time, runID, policyDecisionRef, approvalID string) {
	t.Helper()
	base := mustFindApprovedGateOverrideApproval(t, s, runID, policyDecisionRef)
	duplicate := base
	duplicate.ApprovalID = approvalID
	duplicate.RequestedAt = now.Add(-10 * time.Minute)
	expiresAt := now.Add(20 * time.Minute)
	duplicate.ExpiresAt = &expiresAt
	if err := s.RecordApproval(duplicate); err != nil {
		t.Fatalf("RecordApproval(duplicate approved) returned error: %v", err)
	}
}

func markApprovedGateOverrideExpired(t *testing.T, s *Service, now time.Time, runID, policyDecisionRef string) {
	t.Helper()
	approval := mustFindApprovedGateOverrideApproval(t, s, runID, policyDecisionRef)
	approval.RequestedAt = now.Add(-1 * time.Minute)
	expiresAt := now.Add(-30 * time.Second)
	approval.ExpiresAt = &expiresAt
	if err := s.RecordApproval(approval); err != nil {
		t.Fatalf("RecordApproval(expired) returned error: %v", err)
	}
}

func mustFindApprovedGateOverrideApproval(t *testing.T, s *Service, runID, policyDecisionRef string) artifacts.ApprovalRecord {
	t.Helper()
	for _, ap := range s.ApprovalList() {
		if ap.RunID == runID && ap.PolicyDecisionHash == policyDecisionRef && ap.ActionKind == policyengine.ActionKindGateOverride && ap.Status == "approved" {
			return ap
		}
	}
	t.Fatalf("missing approved gate override approval for policy decision %q", policyDecisionRef)
	return artifacts.ApprovalRecord{}
}

func latestPolicyDecisionRefForAction(t *testing.T, s *Service, runID, actionHash string) string {
	t.Helper()
	ref, ok := s.latestGateOverridePolicyDecisionRef(runID, actionHash)
	if !ok || ref == "" {
		t.Fatalf("latestGateOverridePolicyDecisionRef(%q) missing", runID)
	}
	return ref
}

func approvePendingGateOverrideByPolicyDecision(t *testing.T, s *Service, now time.Time, runID, policyDecisionRef string) {
	t.Helper()
	for _, ap := range s.ApprovalList() {
		if ap.RunID != runID || ap.ActionKind != policyengine.ActionKindGateOverride || ap.Status != "pending" {
			continue
		}
		if ap.PolicyDecisionHash != policyDecisionRef {
			continue
		}
		ap.Status = "approved"
		decided := now.Add(time.Minute)
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
	t.Fatalf("missing pending gate override approval for policy decision %q", policyDecisionRef)
}

func TestRunnerResultReportSanitizesAndDeepCopiesDetails(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 14, 0, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-details-deepcopy", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	details := map[string]any{"nested": map[string]any{"value": "original"}, "arr": []any{map[string]any{"k": "v"}}}
	mustAcceptRunnerResultReq(t, s, RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-details-deepcopy", RunID: "run-details-deepcopy", Report: RunnerResultReport{SchemaID: "runecode.protocol.v0.RunnerResultReport", SchemaVersion: "0.1.0", LifecycleState: "failed", ResultCode: "run_failed", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-details-deepcopy", Details: details}})
	details["nested"].(map[string]any)["value"] = "mutated"
	details["arr"].([]any)[0].(map[string]any)["k"] = "mutated"
	assertResultDetailsStoredDeepCopy(t, mustRunGet(t, s, "run-details-deepcopy", "req-details-deepcopy-run"))
}

func TestRunnerResultReportRejectsUnknownResultCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 1, 17, 30, 0, 0, time.UTC)
	if err := s.SetRunStatus("run-result-code", "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	req := RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: "req-result-code", RunID: "run-result-code", Report: RunnerResultReport{SchemaID: "runecode.protocol.v0.RunnerResultReport", SchemaVersion: "0.1.0", LifecycleState: "failed", ResultCode: "bad_code", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-result-code"}}
	_, errResp := s.HandleRunnerResultReport(context.Background(), req, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_validation_runner_transition_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func mustAcceptRunnerResult(t *testing.T, s *Service, runID, requestID string, report RunnerResultReport) {
	t.Helper()
	mustAcceptRunnerResultReq(t, s, RunnerResultReportRequest{SchemaID: "runecode.protocol.v0.RunnerResultReportRequest", SchemaVersion: "0.1.0", RequestID: requestID, RunID: runID, Report: report})
}

func mustAcceptRunnerResultReq(t *testing.T, s *Service, req RunnerResultReportRequest) {
	t.Helper()
	if _, errResp := s.HandleRunnerResultReport(context.Background(), req, RequestContext{}); errResp != nil {
		t.Fatalf("HandleRunnerResultReport error response: %+v", errResp)
	}
}

func prepareOverrideScenario(t *testing.T, s *Service, now time.Time, runID string) RunnerResultReport {
	t.Helper()
	if err := s.SetRunStatus(runID, "active"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	failed := buildFailedGateResult(now, "idem-gate-failed", "gate-attempt-1", "sha256:"+strings.Repeat("a", 64))
	mustAcceptRunnerResult(t, s, runID, "req-gate-failed", failed)
	failedRef, err := canonicalGateResultRef(runID, failed, "")
	if err != nil {
		t.Fatalf("canonicalGateResultRef returned error: %v", err)
	}
	return buildOverriddenGateResult(now, failedRef)
}

func recordOverrideApproval(t *testing.T, s *Service, now time.Time, runID string, override RunnerResultReport) {
	t.Helper()
	action, err := overrideActionForResult(override, override.Details)
	if err != nil {
		t.Fatalf("overrideActionForResult returned error: %v", err)
	}
	actionHash, err := policyengine.CanonicalActionRequestHash(action)
	if err != nil {
		t.Fatalf("CanonicalActionRequestHash returned error: %v", err)
	}
	if err := s.RecordPolicyDecision(runID, "", buildGateOverridePolicyDecision(runID, actionHash, override.OverriddenFailedResultRef)); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	approvePendingGateOverride(t, s, now, runID)
}

func assertRunLastResultHasOverrideBindings(t *testing.T, s *Service, runID string) {
	t.Helper()
	runResp := mustRunGet(t, s, runID, "req-run-override")
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

func assertResultDetailsStoredDeepCopy(t *testing.T, runResp RunGetResponse) {
	t.Helper()
	lastResult, ok := runResp.Run.AdvisoryState["last_result"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.last_result = %T, want map", runResp.Run.AdvisoryState["last_result"])
	}
	storedDetails := lastResult["details"].(map[string]any)
	if storedDetails["nested"].(map[string]any)["value"] != "original" {
		t.Fatalf("nested.value = %v, want original", storedDetails["nested"].(map[string]any)["value"])
	}
	if storedDetails["arr"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("arr[0].k = %v, want v", storedDetails["arr"].([]any)[0].(map[string]any)["k"])
	}
}
