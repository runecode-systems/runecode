package brokerapi

import (
	"context"
	"slices"
	"strings"
	"testing"

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
	if len(runs) != 1 || runs[0].RunID != "run-123" {
		t.Fatalf("run list = %+v, want run-123", runs)
	}
	if runs[0].WorkspaceID != "workspace-local" {
		t.Fatalf("workspace_id = %q, want workspace-local", runs[0].WorkspaceID)
	}
	if runs[0].WorkflowKind != "" {
		t.Fatalf("workflow_kind = %q, want empty when broker has no trusted workflow kind", runs[0].WorkflowKind)
	}
	if runs[0].WorkflowDefinitionHash == "" {
		t.Fatal("workflow_definition_hash should use trusted manifest digest when unambiguous")
	}
	if runs[0].BackendKind != launcherbackend.BackendKindUnknown {
		t.Fatalf("backend_kind = %q, want %q", runs[0].BackendKind, launcherbackend.BackendKindUnknown)
	}
	if runs[0].IsolationAssuranceLevel != launcherbackend.IsolationAssuranceUnknown {
		t.Fatalf("isolation_assurance_level = %q, want %q", runs[0].IsolationAssuranceLevel, launcherbackend.IsolationAssuranceUnknown)
	}
	if runs[0].ProvisioningPosture != launcherbackend.ProvisioningPostureUnknown {
		t.Fatalf("provisioning_posture = %q, want %q", runs[0].ProvisioningPosture, launcherbackend.ProvisioningPostureUnknown)
	}
	if runs[0].AssuranceLevel != runs[0].IsolationAssuranceLevel {
		t.Fatalf("assurance_level alias = %q, want %q", runs[0].AssuranceLevel, runs[0].IsolationAssuranceLevel)
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
}

func assertRunDetailAuthoritativeStateForLocalOps(t *testing.T, state map[string]any) {
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
	if state["provisioning_posture"] != launcherbackend.ProvisioningPostureUnknown {
		t.Fatalf("authoritative_state.provisioning_posture = %v, want %q", state["provisioning_posture"], launcherbackend.ProvisioningPostureUnknown)
	}
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
	s.RecordRuntimeFacts("run-terminal-invalid", launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-terminal-invalid"},
		TerminalReport: &launcherbackend.BackendTerminalReport{
			TerminationKind: "unknown_state",
			FailClosed:      false,
		},
	})

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

func TestRecordRuntimeFactsFailClosesInvalidHardeningPosture(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putRunScopedArtifactForLocalOpsTest(t, s, "run-hardening-invalid", "step-1")
	s.RecordRuntimeFacts("run-hardening-invalid", launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-hardening-invalid"},
		HardeningPosture: launcherbackend.AppliedHardeningPosture{
			Requested:           launcherbackend.HardeningRequestedHardened,
			Effective:           launcherbackend.HardeningEffectiveDegraded,
			DegradedReasons:     []string{"seccomp_unavailable"},
			BackendEvidenceRefs: []string{"/usr/bin/qemu-system-x86_64"},
		},
	})

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

func TestRunGetFallsBackWhenAuditVerificationUnavailable(t *testing.T) {
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
	if resp.Run.AuditSummary.IntegrityStatus != "failed" {
		t.Fatalf("integrity_status = %q, want failed", resp.Run.AuditSummary.IntegrityStatus)
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
