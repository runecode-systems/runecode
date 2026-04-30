package brokerapi

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestRunAndArtifactLocalTypedOperations(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	digest := putRunScopedArtifactForLocalOpsTest(t, s, "run-123", "step-1")
	_ = putTrustedPolicyContextForRun(t, s, "run-123", false)
	assertRunListAndDetailForLocalOps(t, s)
	assertArtifactListAndHeadForLocalOps(t, s, digest)
	assertArtifactReadStreamCompletes(t, s, digest)
}

func assertRunListAndDetailForLocalOps(t *testing.T, s *Service) {
	t.Helper()
	runList, errResp := s.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: "req-run-list", Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunList error response: %+v", errResp)
	}
	assertRunListSummaryForLocalOps(t, runList.Runs)
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-get", RunID: "run-123"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	assertRunDetailForLocalOps(t, runGet.Run)
}

func assertRunListSummaryForLocalOps(t *testing.T, runs []RunSummary) {
	t.Helper()
	assertRunListIdentityForLocalOps(t, runs)
	assertRunListWorkflowBindingForLocalOps(t, runs[0])
	assertRunListRuntimeDefaultsForLocalOps(t, runs[0])
}

func assertRunListIdentityForLocalOps(t *testing.T, runs []RunSummary) {
	t.Helper()
	if len(runs) != 1 || runs[0].RunID != "run-123" {
		t.Fatalf("run list = %+v, want run-123", runs)
	}
	if runs[0].WorkspaceID == "workspace-local" {
		t.Fatalf("workspace_id = %q, want project-context-bound workspace id", runs[0].WorkspaceID)
	}
	if runs[0].ProjectContextIdentity == "" {
		t.Fatal("project_context_identity_digest empty, want validated digest")
	}
	wantWorkspace := "workspace-" + strings.TrimPrefix(runs[0].ProjectContextIdentity, "sha256:")
	if runs[0].WorkspaceID != wantWorkspace {
		t.Fatalf("workspace_id = %q, want %q bound from project context digest", runs[0].WorkspaceID, wantWorkspace)
	}
}

func assertRunListWorkflowBindingForLocalOps(t *testing.T, summary RunSummary) {
	t.Helper()
	if summary.WorkflowKind != "" {
		t.Fatalf("workflow_kind = %q, want empty when broker has no trusted workflow kind", summary.WorkflowKind)
	}
	if summary.WorkflowDefinitionHash == "" {
		t.Fatal("workflow_definition_hash should use trusted manifest digest when unambiguous")
	}
}

func assertRunListRuntimeDefaultsForLocalOps(t *testing.T, summary RunSummary) {
	t.Helper()
	if summary.BackendKind != launcherbackend.BackendKindUnknown {
		t.Fatalf("backend_kind = %q, want %q", summary.BackendKind, launcherbackend.BackendKindUnknown)
	}
	if summary.IsolationAssuranceLevel != launcherbackend.IsolationAssuranceUnknown {
		t.Fatalf("isolation_assurance_level = %q, want %q", summary.IsolationAssuranceLevel, launcherbackend.IsolationAssuranceUnknown)
	}
	if summary.ProvisioningPosture != launcherbackend.ProvisioningPostureUnknown {
		t.Fatalf("provisioning_posture = %q, want %q", summary.ProvisioningPosture, launcherbackend.ProvisioningPostureUnknown)
	}
	if summary.AssuranceLevel != summary.IsolationAssuranceLevel {
		t.Fatalf("assurance_level alias = %q, want %q", summary.AssuranceLevel, summary.IsolationAssuranceLevel)
	}
	if summary.RuntimePostureDegraded {
		t.Fatalf("runtime_posture_degraded = %v, want false", summary.RuntimePostureDegraded)
	}
}

func assertRunDetailForLocalOps(t *testing.T, detail RunDetail) {
	t.Helper()
	assertRunDetailCoreForLocalOps(t, detail)
	assertRunDetailAuthoritativeStateForLocalOps(t, detail.AuthoritativeState)
	assertRunDetailRoleCoverageForLocalOps(t, detail.RoleSummaries)
}

func assertRunDetailCoreForLocalOps(t *testing.T, detail RunDetail) {
	t.Helper()
	if detail.Summary.RunID != "run-123" {
		t.Fatalf("run detail run_id = %q, want run-123", detail.Summary.RunID)
	}
	if len(detail.ActiveManifestHashes) == 0 {
		t.Fatal("run detail active_manifest_hashes should not be empty")
	}
	if detail.AdvisoryState["provenance"] != "none_reported" {
		t.Fatalf("advisory_state.provenance = %v, want none_reported", detail.AdvisoryState["provenance"])
	}
	if _, ok := detail.AuthoritativeState["lifecycle_hint"]; ok {
		t.Fatal("authoritative_state must not contain runner lifecycle_hint")
	}
}

func assertRunDetailAuthoritativeStateForLocalOps(t *testing.T, state map[string]any) {
	t.Helper()
	assertRunDetailAuthoritativeBaseState(t, state)
	hardening, ok := state["applied_hardening_posture"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.applied_hardening_posture = %T, want map", state["applied_hardening_posture"])
	}
	if hardening["degraded"] != true {
		t.Fatalf("applied_hardening_posture.degraded = %v, want true", hardening["degraded"])
	}
	if hardening["degraded_reasons"] == nil {
		t.Fatal("applied_hardening_posture.degraded_reasons should be present for default unknown posture")
	}
}

func assertRunDetailAuthoritativeBaseState(t *testing.T, state map[string]any) {
	t.Helper()
	if state["source"] != "broker_store" {
		t.Fatalf("authoritative_state.source = %v, want broker_store", state["source"])
	}
	if state["provenance"] != "trusted_derived" {
		t.Fatalf("authoritative_state.provenance = %v, want trusted_derived", state["provenance"])
	}
	if state["runtime_facts_source"] != "launcher_backend_receipt" {
		t.Fatalf("authoritative_state.runtime_facts_source = %v, want launcher_backend_receipt", state["runtime_facts_source"])
	}
	if state["backend_kind"] != launcherbackend.BackendKindUnknown {
		t.Fatalf("authoritative_state.backend_kind = %v, want %q", state["backend_kind"], launcherbackend.BackendKindUnknown)
	}
	if state["isolation_assurance_level"] != launcherbackend.IsolationAssuranceUnknown {
		t.Fatalf("authoritative_state.isolation_assurance_level = %v, want %q", state["isolation_assurance_level"], launcherbackend.IsolationAssuranceUnknown)
	}
	if state["runtime_posture_degraded"] != false {
		t.Fatalf("authoritative_state.runtime_posture_degraded = %v, want false", state["runtime_posture_degraded"])
	}
	if state["provisioning_posture"] != launcherbackend.ProvisioningPostureUnknown {
		t.Fatalf("authoritative_state.provisioning_posture = %v, want %q", state["provisioning_posture"], launcherbackend.ProvisioningPostureUnknown)
	}
	if state["attestation_posture"] != launcherbackend.AttestationPostureUnknown {
		t.Fatalf("authoritative_state.attestation_posture = %v, want %q", state["attestation_posture"], launcherbackend.AttestationPostureUnknown)
	}
}

func assertRunDetailRoleCoverageForLocalOps(t *testing.T, roles []RunRoleSummary) {
	t.Helper()
	hasWorkspaceEdit := false
	for _, role := range roles {
		if role.RoleKind == "workspace-edit" {
			hasWorkspaceEdit = true
			break
		}
	}
	if !hasWorkspaceEdit {
		t.Fatalf("role_summaries = %+v, want workspace-edit role", roles)
	}
}

func TestRunGetMarksAdvisoryAvailableForLifecycleHintOnly(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	if _, err := s.RecordRunnerCheckpoint("run-advisory-lifecycle-only", artifacts.RunnerCheckpointAdvisory{
		LifecycleState: "active",
		CheckpointCode: "run_started",
		OccurredAt:     now,
		IdempotencyKey: "idem-advisory-lifecycle",
	}); err != nil {
		t.Fatalf("RecordRunnerCheckpoint returned error: %v", err)
	}
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-advisory-lifecycle", RunID: "run-advisory-lifecycle-only"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if runGet.Run.AdvisoryState["available"] != true {
		t.Fatalf("advisory_state.available = %v, want true", runGet.Run.AdvisoryState["available"])
	}
	if runGet.Run.AdvisoryState["provenance"] != "runner_reported" {
		t.Fatalf("advisory_state.provenance = %v, want runner_reported", runGet.Run.AdvisoryState["provenance"])
	}
	bounded, ok := runGet.Run.AdvisoryState["bounded_keys"].([]string)
	if !ok {
		t.Fatalf("advisory_state.bounded_keys = %T, want []string", runGet.Run.AdvisoryState["bounded_keys"])
	}
	if len(bounded) == 0 {
		t.Fatal("advisory_state.bounded_keys empty, want lifecycle_hint included")
	}
}

func TestRunSummaryWorkflowDefinitionHashRequiresSingleTrustedManifest(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	if _, putErr := s.Put(artifacts.PutRequest{Payload: []byte("artifact-a"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("a", 64), CreatedByRole: "workspace", RunID: "run-ambiguous", StepID: "step-1"}); putErr != nil {
		t.Fatalf("Put artifact-a returned error: %v", putErr)
	}
	if _, putErr := s.Put(artifacts.PutRequest{Payload: []byte("artifact-b"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64), CreatedByRole: "workspace", RunID: "run-ambiguous", StepID: "step-2"}); putErr != nil {
		t.Fatalf("Put artifact-b returned error: %v", putErr)
	}

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-ambiguous", RunID: "run-ambiguous"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if runGet.Run.Summary.WorkflowDefinitionHash != "" {
		t.Fatalf("workflow_definition_hash = %q, want empty when no single trusted manifest identity", runGet.Run.Summary.WorkflowDefinitionHash)
	}
	if got := len(runGet.Run.ActiveManifestHashes); got != 2 {
		t.Fatalf("active_manifest_hashes len = %d, want 2", got)
	}
}

func TestRecordRuntimeFactsFailClosesInvalidTerminalReport(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-terminal-invalid", "step-1")
	if err := s.RecordRuntimeFacts("run-terminal-invalid", launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-terminal-invalid"},
		TerminalReport: &launcherbackend.BackendTerminalReport{
			TerminationKind: "unknown_state",
			FailClosed:      false,
		},
	}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-terminal-invalid", RunID: "run-terminal-invalid"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	terminal, ok := runGet.Run.AuthoritativeState["backend_terminal"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.backend_terminal = %T, want map", runGet.Run.AuthoritativeState["backend_terminal"])
	}
	if terminal["termination_kind"] != launcherbackend.BackendTerminationKindFailed {
		t.Fatalf("backend_terminal.termination_kind = %v, want %q", terminal["termination_kind"], launcherbackend.BackendTerminationKindFailed)
	}
	if terminal["failure_reason_code"] != launcherbackend.BackendErrorCodeTerminalReportInvalid {
		t.Fatalf("backend_terminal.failure_reason_code = %v, want %q", terminal["failure_reason_code"], launcherbackend.BackendErrorCodeTerminalReportInvalid)
	}
	if terminal["fail_closed"] != true {
		t.Fatalf("backend_terminal.fail_closed = %v, want true", terminal["fail_closed"])
	}
	if terminal["fallback_posture"] != launcherbackend.BackendFallbackPostureNoAutomaticFallback {
		t.Fatalf("backend_terminal.fallback_posture = %v, want %q", terminal["fallback_posture"], launcherbackend.BackendFallbackPostureNoAutomaticFallback)
	}
}

func TestRunDetailIncludesPersistedPolicyDecisionRefs(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_ = putRunScopedArtifactForLocalOpsTest(t, s, "run-policy-refs", "step-1")

	err := s.RecordPolicyDecision("run-policy-refs", "", policyDecisionFixtureForRunRefs())
	if err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-policy-refs", RunID: "run-policy-refs"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if len(runGet.Run.LatestPolicyDecisionRefs) == 0 {
		t.Fatal("latest_policy_decision_refs should include persisted policy decision digest")
	}
	if !strings.HasPrefix(runGet.Run.LatestPolicyDecisionRefs[0], "sha256:") {
		t.Fatalf("latest_policy_decision_refs[0] = %q, want sha256 digest", runGet.Run.LatestPolicyDecisionRefs[0])
	}
}

func TestRecordRuntimeLifecycleStateUpdatesAuthoritativeProjection(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-lifecycle-update", "step-1")
	if err := s.RecordRuntimeFacts("run-lifecycle-update", launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-lifecycle-update"}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	err := s.RecordRuntimeLifecycleState("run-lifecycle-update", launcherbackend.RuntimeLifecycleState{
		BackendLifecycle:            &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateActive, PreviousState: launcherbackend.BackendLifecycleStateBinding, TransitionCount: 3},
		ProvisioningPosture:         launcherbackend.ProvisioningPostureTOFU,
		ProvisioningPostureDegraded: true,
		ProvisioningDegradedReasons: []string{"key_material_transient"},
		LaunchFailureReasonCode:     launcherbackend.BackendErrorCodeAccelerationUnavailable,
	})
	if err != nil {
		t.Fatalf("RecordRuntimeLifecycleState returned error: %v", err)
	}

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-runtime-lifecycle", RunID: "run-lifecycle-update"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	state := runGet.Run.AuthoritativeState
	if state["provisioning_posture"] != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("authoritative_state.provisioning_posture = %v, want %q", state["provisioning_posture"], launcherbackend.ProvisioningPostureTOFU)
	}
	if state["launch_failure_reason_code"] != launcherbackend.BackendErrorCodeAccelerationUnavailable {
		t.Fatalf("authoritative_state.launch_failure_reason_code = %v, want %q", state["launch_failure_reason_code"], launcherbackend.BackendErrorCodeAccelerationUnavailable)
	}
}

func TestRecordRuntimeFactsFailClosesInvalidHardeningPosture(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-hardening-invalid", "step-1")
	if err := s.RecordRuntimeFacts("run-hardening-invalid", launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-hardening-invalid"},
		HardeningPosture: launcherbackend.AppliedHardeningPosture{
			Requested:           launcherbackend.HardeningRequestedHardened,
			Effective:           launcherbackend.HardeningEffectiveDegraded,
			DegradedReasons:     []string{"seccomp_unavailable"},
			BackendEvidenceRefs: []string{"/usr/bin/qemu-system-x86_64"},
		},
	}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-hardening-invalid", RunID: "run-hardening-invalid"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	hardening, ok := runGet.Run.AuthoritativeState["applied_hardening_posture"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.applied_hardening_posture = %T, want map", runGet.Run.AuthoritativeState["applied_hardening_posture"])
	}
	reasons, ok := hardening["degraded_reasons"].([]string)
	if !ok {
		t.Fatalf("applied_hardening_posture.degraded_reasons = %T, want []string", hardening["degraded_reasons"])
	}
	if !slices.Contains(reasons, "hardening_posture_invalid") {
		t.Fatalf("degraded_reasons = %v, want include hardening_posture_invalid", reasons)
	}
}

func policyDecisionFixtureForRunRefs() policyengine.PolicyDecision {
	return policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           "sha256:" + strings.Repeat("1", 64),
		PolicyInputHashes:      []string{"sha256:" + strings.Repeat("2", 64)},
		ActionRequestHash:      "sha256:" + strings.Repeat("3", 64),
		RelevantArtifactHashes: []string{"sha256:" + strings.Repeat("4", 64)},
		DetailsSchemaID:        "runecode.protocol.details.policy.decision.v0",
		Details:                map[string]any{"rule": "deny_by_default"},
	}
}

func TestRunGetNotFoundUsesRunSpecificCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleRunGet(context.Background(), RunGetRequest{
		SchemaID:      "runecode.protocol.v0.RunGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-missing",
		RunID:         "run-missing",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunGet expected not-found error")
	}
	if errResp.Error.Code != "broker_not_found_run" {
		t.Fatalf("error code = %q, want broker_not_found_run", errResp.Error.Code)
	}
}

func TestRunGetFallsBackToDegradedAuditSummaryWhenRawAuditExists(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, service, "run-fallback", "step-1")
	service.auditLedger = nil

	resp, errResp := service.HandleRunGet(context.Background(), RunGetRequest{
		SchemaID:      "runecode.protocol.v0.RunGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-fallback",
		RunID:         "run-fallback",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if !resp.Run.AuditSummary.CurrentlyDegraded {
		t.Fatal("audit summary should be degraded when verification surface is unavailable")
	}
	if resp.Run.AuditSummary.IntegrityStatus != "degraded" {
		t.Fatalf("integrity_status = %q, want degraded", resp.Run.AuditSummary.IntegrityStatus)
	}
	if resp.Run.AuditSummary.StoragePostureStatus != "ok" {
		t.Fatalf("storage_posture_status = %q, want ok", resp.Run.AuditSummary.StoragePostureStatus)
	}
}

func TestRunPendingApprovalsUseCanonicalPendingRecords(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref, putErr := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat("c", 64), CreatedByRole: "workspace", RunID: "run-pending-canonical", StepID: "step-1"})
	if putErr != nil {
		t.Fatalf("Put returned error: %v", putErr)
	}
	expectedID := createPendingApprovalFromPolicyDecision(t, s, "run-pending-canonical", "step-1", ref.Digest)

	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-get-pending", RunID: "run-pending-canonical"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	if runResp.Run.Summary.PendingApprovalCount != 1 {
		t.Fatalf("pending_approval_count = %d, want 1", runResp.Run.Summary.PendingApprovalCount)
	}
	if len(runResp.Run.PendingApprovalIDs) != 1 {
		t.Fatalf("pending_approval_ids len = %d, want 1", len(runResp.Run.PendingApprovalIDs))
	}
	if runResp.Run.PendingApprovalIDs[0] != expectedID {
		t.Fatalf("pending_approval_ids[0] = %q, want %q", runResp.Run.PendingApprovalIDs[0], expectedID)
	}

	approvalResp, approvalErr := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-pending", ApprovalID: runResp.Run.PendingApprovalIDs[0]}, RequestContext{})
	if approvalErr != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", approvalErr)
	}
	if approvalResp.SignedApprovalRequest == nil {
		t.Fatal("signed_approval_request = nil, want pending request envelope")
	}
}

func TestRunLifecycleUsesRunnerAdvisoryActiveForPartialApprovalBlocking(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC)
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("2", 64), CreatedByRole: "workspace", RunID: "run-partial", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if err := s.RecordRunnerApprovalWait(artifacts.RunnerApproval{ApprovalID: "sha256:" + strings.Repeat("a", 64), RunID: "run-partial", StageID: "stage-a", StepID: "step-a", RoleInstanceID: "role-a", Status: "pending", ApprovalType: "exact_action", BoundActionHash: "sha256:" + strings.Repeat("b", 64), OccurredAt: now}); err != nil {
		t.Fatalf("RecordRunnerApprovalWait returned error: %v", err)
	}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-partial-active", RunID: "run-partial", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "step_attempt_started", OccurredAt: now.Add(time.Minute).Format(time.RFC3339), IdempotencyKey: "idem-partial-active", StageID: "stage-b", StepID: "step-b", StepAttemptID: "attempt-b"}}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunnerCheckpointReport error response: %+v", errResp)
	}

	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-partial", RunID: "run-partial"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	if runResp.Run.Summary.LifecycleState != "active" {
		t.Fatalf("summary.lifecycle_state = %q, want active for partial blocking", runResp.Run.Summary.LifecycleState)
	}
	if runResp.Run.Coordination.Blocked {
		t.Fatal("coordination.blocked = true, want false for partial blocking detail")
	}
	if runResp.Run.Coordination.WaitReasonCode != "partial_pending_approval" {
		t.Fatalf("coordination.wait_reason_code = %q, want partial_pending_approval", runResp.Run.Coordination.WaitReasonCode)
	}
	stepAttempts, ok := runResp.Run.AdvisoryState["step_attempts"].(map[string]any)
	if !ok {
		t.Fatalf("advisory_state.step_attempts = %T, want map[string]any", runResp.Run.AdvisoryState["step_attempts"])
	}
	attemptB, ok := stepAttempts["attempt-b"].(map[string]any)
	if !ok {
		t.Fatalf("step_attempts[attempt-b] = %T, want map[string]any", stepAttempts["attempt-b"])
	}
	if blockedOnScope, _ := attemptB["blocked_on_scope_pending_approval"].(bool); blockedOnScope {
		t.Fatal("attempt-b blocked_on_scope_pending_approval = true, want false")
	}
}

func TestRunLifecycleRemainsBlockedWhenCanonicalPendingApprovalExists(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("3", 64), CreatedByRole: "workspace", RunID: "run-blocked", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	_ = createPendingApprovalFromPolicyDecision(t, s, "run-blocked", "step-1", "sha256:"+strings.Repeat("9", 64))
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-blocked-active", RunID: "run-blocked", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "step_attempt_started", OccurredAt: now.Add(time.Minute).Format(time.RFC3339), IdempotencyKey: "idem-blocked-active", StageID: "stage-1", StepID: "step-1", StepAttemptID: "attempt-1"}}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunnerCheckpointReport error response: %+v", errResp)
	}
	runResp, runErr := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-blocked", RunID: "run-blocked"}, RequestContext{})
	if runErr != nil {
		t.Fatalf("HandleRunGet error response: %+v", runErr)
	}
	if runResp.Run.Summary.LifecycleState != "blocked" {
		t.Fatalf("summary.lifecycle_state = %q, want blocked", runResp.Run.Summary.LifecycleState)
	}
}

func TestRunnerCheckpointRejectsOversizedDetailsMap(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC)
	if _, err := s.Put(artifacts.PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("4", 64), CreatedByRole: "workspace", RunID: "run-details", StepID: "step-1"}); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	details := map[string]any{}
	for i := 0; i < 65; i++ {
		details["k"+time.Unix(int64(i), 0).UTC().Format("150405")+"-"+strings.Repeat("x", i%3)] = i
	}
	_, errResp := s.HandleRunnerCheckpointReport(context.Background(), RunnerCheckpointReportRequest{SchemaID: "runecode.protocol.v0.RunnerCheckpointReportRequest", SchemaVersion: "0.1.0", RequestID: "req-details-oversized", RunID: "run-details", Report: RunnerCheckpointReport{SchemaID: "runecode.protocol.v0.RunnerCheckpointReport", SchemaVersion: "0.1.0", LifecycleState: "active", CheckpointCode: "step_attempt_started", OccurredAt: now.Format(time.RFC3339), IdempotencyKey: "idem-details-oversized", StageID: "stage-1", StepID: "step-1", StepAttemptID: "attempt-1", Details: details}}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunnerCheckpointReport error = nil, want details validation failure")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}
