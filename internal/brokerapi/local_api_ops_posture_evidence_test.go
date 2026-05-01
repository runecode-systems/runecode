package brokerapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestRunDetailAuthoritativeStateIncludesBackendPostureSelectionEvidenceRefs(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-backend-evidence"
	const instanceID = "launcher-instance-1"
	const selectorRunID = "instance-control:launcher-instance-1"
	const manifestHash = "sha256:" + "1111111111111111111111111111111111111111111111111111111111111111"
	const actionHash = "sha256:" + "3333333333333333333333333333333333333333333333333333333333333333"
	const requestDigest = "sha256:" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const decisionDigest = "sha256:" + "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	policyRef := recordBackendPosturePolicyDecisionForRun(t, s, selectorRunID, manifestHash, actionHash, instanceID)
	approvalID := recordBackendPostureApprovalForRun(t, s, runID, selectorRunID, policyRef, manifestHash, actionHash, requestDigest, decisionDigest, instanceID)
	recordContainerRuntimeFactsForBackendEvidence(t, s, runID)

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-backend-evidence", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	evidence := backendPostureSelectionEvidenceForState(t, runGet.Run.AuthoritativeState)
	policyEvidence := backendPosturePolicyRefsFromEvidence(t, evidence)
	if len(policyEvidence) == 0 || policyEvidence[0] != policyRef {
		t.Fatalf("backend_posture_selection_evidence.policy_decision_refs = %v, want include %q", policyEvidence, policyRef)
	}
	if runGet.Run.AuthoritativeState["attestation_verifier_class"] != launcherbackend.AttestationVerifierClassUnknown {
		t.Fatalf("authoritative_state.attestation_verifier_class = %v, want %q without persisted attestation evidence", runGet.Run.AuthoritativeState["attestation_verifier_class"], launcherbackend.AttestationVerifierClassUnknown)
	}
	if runGet.Run.AuthoritativeState["supported_runtime_requirements_satisfied"] != false {
		t.Fatalf("authoritative_state.supported_runtime_requirements_satisfied = %v, want false without attestation evidence", runGet.Run.AuthoritativeState["supported_runtime_requirements_satisfied"])
	}
	approvalEvidence := backendPostureApprovalEvidenceFromEvidence(t, evidence)
	assertBackendPostureApprovalEvidence(t, approvalEvidence, approvalID, requestDigest, decisionDigest, policyRef)
}

func TestRunDetailAuthoritativeStateBackendPostureSelectionEvidenceUsesBackendScopedPolicyRefs(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-backend-evidence-scoped-refs"
	const instanceID = "launcher-instance-1"
	const selectorRunID = "instance-control:launcher-instance-1"
	const manifestHash = "sha256:" + "1111111111111111111111111111111111111111111111111111111111111111"
	const actionHash = "sha256:" + "3333333333333333333333333333333333333333333333333333333333333333"
	const backendActionHash = "sha256:" + "4444444444444444444444444444444444444444444444444444444444444444"
	const requestDigest = "sha256:" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const decisionDigest = "sha256:" + "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	genericRunPolicyRef := recordBackendPosturePolicyDecisionForRun(t, s, runID, manifestHash, actionHash, instanceID)
	backendPolicyRef := recordBackendPosturePolicyDecisionForRun(t, s, selectorRunID, manifestHash, backendActionHash, instanceID)
	recordBackendPostureApprovalForRun(t, s, runID, selectorRunID, backendPolicyRef, manifestHash, backendActionHash, requestDigest, decisionDigest, instanceID)
	recordContainerRuntimeFactsForBackendEvidence(t, s, runID)

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-backend-evidence-scoped-refs", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	evidence := backendPostureSelectionEvidenceForState(t, runGet.Run.AuthoritativeState)
	policyEvidence := backendPosturePolicyRefsFromEvidence(t, evidence)
	if !containsStringInSlice(policyEvidence, backendPolicyRef) {
		t.Fatalf("backend_posture_selection_evidence.policy_decision_refs = %v, want include backend policy ref %q", policyEvidence, backendPolicyRef)
	}
	if containsStringInSlice(policyEvidence, genericRunPolicyRef) {
		t.Fatalf("backend_posture_selection_evidence.policy_decision_refs = %v, should omit generic run policy ref %q", policyEvidence, genericRunPolicyRef)
	}
}

func TestBuildAuthoritativeRunStateUsesLaunchEvidenceAsReceiptSourceWhenPersisted(t *testing.T) {
	runtimeFacts, runtimeEvidence := authoritativeRunStateEvidenceFixtures()
	state := buildAuthoritativeRunState(
		authoritativeRunStateSummaryFixture(),
		nil,
		nil,
		nil,
		nil,
		nil,
		runtimeFacts,
		runtimeEvidence,
		"",
	)
	assertAuthoritativeStateUsesLaunchEvidence(t, state)
}

func authoritativeRunStateEvidenceFixtures() (launcherbackend.RuntimeFactsSnapshot, launcherbackend.RuntimeEvidenceSnapshot) {
	return launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{
			RunID:                   "run-evidence-authoritative",
			StageID:                 "artifact_flow",
			RoleInstanceID:          "workspace-1",
			RoleFamily:              "workspace",
			BackendKind:             launcherbackend.BackendKindContainer,
			IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded,
			ProvisioningPosture:     launcherbackend.ProvisioningPostureTOFU,
			IsolateID:               "isolate-from-stale-receipt",
		}}, launcherbackend.RuntimeEvidenceSnapshot{Launch: launcherbackend.LaunchRuntimeEvidence{
			RunID:                   "run-evidence-authoritative",
			StageID:                 "artifact_flow",
			RoleInstanceID:          "workspace-1",
			RoleFamily:              "workspace",
			RoleKind:                "workspace-edit",
			BackendKind:             launcherbackend.BackendKindMicroVM,
			IsolationAssuranceLevel: launcherbackend.IsolationAssuranceIsolated,
			ProvisioningPosture:     launcherbackend.ProvisioningPostureAttested,
			IsolateID:               "isolate-from-evidence",
			EvidenceDigest:          "sha256:" + strings.Repeat("1", 64),
		}}
}

func authoritativeRunStateSummaryFixture() RunSummary {
	return RunSummary{RunID: "run-evidence-authoritative", WorkspaceID: "workspace-run-evidence-authoritative", LifecycleState: "active"}
}

func assertAuthoritativeStateUsesLaunchEvidence(t *testing.T, state map[string]any) {
	t.Helper()
	if state["backend_kind"] != launcherbackend.BackendKindMicroVM {
		t.Fatalf("authoritative_state.backend_kind = %v, want %q from launch evidence", state["backend_kind"], launcherbackend.BackendKindMicroVM)
	}
	if state["provisioning_posture"] != launcherbackend.ProvisioningPostureAttested {
		t.Fatalf("authoritative_state.provisioning_posture = %v, want %q from launch evidence", state["provisioning_posture"], launcherbackend.ProvisioningPostureAttested)
	}
	if state["isolate_id"] != "isolate-from-evidence" {
		t.Fatalf("authoritative_state.isolate_id = %v, want isolate-from-evidence from launch evidence", state["isolate_id"])
	}
	if state["runtime_posture_degraded"] != false {
		t.Fatalf("authoritative_state.runtime_posture_degraded = %v, want false from launch evidence posture", state["runtime_posture_degraded"])
	}
	if state["attestation_verifier_class"] != launcherbackend.AttestationVerifierClassUnknown {
		t.Fatalf("authoritative_state.attestation_verifier_class = %v, want %q when no attestation evidence is present", state["attestation_verifier_class"], launcherbackend.AttestationVerifierClassUnknown)
	}
}

func TestBuildAuthoritativeRunStateProjectsVerifierClassAndSupportedRuntimeRequirements(t *testing.T) {
	runtimeFacts := launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: "run-container-attested", StageID: "artifact_flow", RoleInstanceID: "workspace-1", RoleFamily: "workspace", BackendKind: launcherbackend.BackendKindContainer, IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded, ProvisioningPosture: launcherbackend.ProvisioningPostureAttested}}
	runtimeEvidence := launcherbackend.RuntimeEvidenceSnapshot{
		Launch:                  launcherbackend.LaunchRuntimeEvidence{RunID: "run-container-attested", StageID: "artifact_flow", RoleInstanceID: "workspace-1", RoleFamily: "workspace", RoleKind: "workspace-edit", BackendKind: launcherbackend.BackendKindContainer, IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded, ProvisioningPosture: launcherbackend.ProvisioningPostureAttested, EvidenceDigest: "sha256:" + strings.Repeat("1", 64)},
		Attestation:             &launcherbackend.IsolateAttestationEvidence{AttestationSourceKind: launcherbackend.AttestationSourceKindTrustedRuntime, MeasurementProfile: launcherbackend.MeasurementProfileContainerImageV1, EvidenceDigest: "sha256:" + strings.Repeat("2", 64)},
		AttestationVerification: &launcherbackend.IsolateAttestationVerificationRecord{VerificationResult: launcherbackend.AttestationVerificationResultValid, ReplayVerdict: launcherbackend.AttestationReplayVerdictOriginal, VerificationDigest: "sha256:" + strings.Repeat("3", 64)},
	}
	approvals := []ApprovalSummary{{ApprovalID: "ap-1", Status: "consumed", PolicyDecisionHash: "sha256:" + strings.Repeat("4", 64), BoundScope: ApprovalBoundScope{ActionKind: policyengine.ActionKindBackendPosture, InstanceID: "launcher-instance-1", RunID: "instance-control:launcher-instance-1"}}}
	state := buildAuthoritativeRunState(RunSummary{RunID: "run-container-attested", WorkspaceID: "workspace-run-container-attested", LifecycleState: "active"}, nil, nil, nil, nil, approvals, runtimeFacts, runtimeEvidence, "launcher-instance-1")
	if state["attestation_verifier_class"] != launcherbackend.AttestationVerifierClassTrustedDomainLocal {
		t.Fatalf("authoritative_state.attestation_verifier_class = %v, want %q", state["attestation_verifier_class"], launcherbackend.AttestationVerifierClassTrustedDomainLocal)
	}
	if state["reduced_assurance_approval_backed"] != true {
		t.Fatalf("authoritative_state.reduced_assurance_approval_backed = %v, want true", state["reduced_assurance_approval_backed"])
	}
	if state["supported_runtime_requirements_satisfied"] != true {
		t.Fatalf("authoritative_state.supported_runtime_requirements_satisfied = %v, want true", state["supported_runtime_requirements_satisfied"])
	}
}

func TestRunIdentityOmitsBackendSpecificProvenanceForContainerRunSummary(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-container-identity"
	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	recordContainerIdentityRuntimeFacts(t, s, runID)

	run := fetchSingleRunSummary(t, s, "req-run-container-identity")
	assertContainerSummaryIdentityFields(t, run)
	assertSummaryOmitsBackendSpecificProvenance(t, run)
}

func TestRunSummaryKeepsAuditPostureDistinctFromBackendAndRuntimePosture(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-posture-separation"
	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	if err := s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{
		RunID:                   runID,
		StageID:                 "artifact_flow",
		RoleInstanceID:          "workspace-1",
		RoleFamily:              "workspace",
		BackendKind:             launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded,
		ProvisioningPosture:     launcherbackend.ProvisioningPostureAttested,
	}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	s.auditLedger = nil

	run := fetchSingleRunSummary(t, s, "req-run-posture-separation")
	if run.BackendKind != launcherbackend.BackendKindContainer || run.IsolationAssuranceLevel != launcherbackend.IsolationAssuranceDegraded || !run.RuntimePostureDegraded {
		t.Fatalf("runtime posture projection changed unexpectedly: %+v", run)
	}
	if !run.AuditCurrentlyDegraded || run.AuditIntegrityStatus != "degraded" || run.AuditAnchoringStatus != "degraded" {
		t.Fatalf("audit posture should degrade independently when verification unavailable: %+v", run)
	}
}

func recordBackendPosturePolicyDecisionForRun(t *testing.T, s *Service, runID, manifestHash, actionHash, instanceID string) string {
	t.Helper()
	decision := policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           manifestHash,
		PolicyInputHashes:      []string{"sha256:" + strings.Repeat("2", 64)},
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: []string{"sha256:" + strings.Repeat("4", 64)},
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details:                map[string]any{"precedence": "approval_profile_moderate", "instance_id": instanceID},
	}
	if err := s.RecordPolicyDecision(runID, "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	refs := s.PolicyDecisionRefsForRun(runID)
	if len(refs) == 0 {
		t.Fatal("PolicyDecisionRefsForRun returned empty refs")
	}
	return refs[0]
}

func recordBackendPostureApprovalForRun(t *testing.T, s *Service, runID, selectorRunID, policyRef, manifestHash, actionHash, requestDigest, decisionDigest, instanceID string) string {
	t.Helper()
	approvalID := "sha256:" + strings.Repeat("a", 64)
	now := time.Now().UTC().Round(0)
	if err := s.RecordApproval(artifacts.ApprovalRecord{
		ApprovalID:             approvalID,
		Status:                 "consumed",
		WorkspaceID:            workspaceIDForRun(runID),
		InstanceID:             instanceID,
		RunID:                  selectorRunID,
		ActionKind:             policyengine.ActionKindBackendPosture,
		RequestedAt:            now.Add(-2 * time.Minute),
		DecidedAt:              func() *time.Time { t := now.Add(-1 * time.Minute); return &t }(),
		ConsumedAt:             func() *time.Time { t := now; return &t }(),
		ApprovalTriggerCode:    "reduced_assurance_backend",
		ChangesIfApproved:      "Reduced-assurance backend posture change may be applied.",
		ApprovalAssuranceLevel: "reauthenticated",
		PresenceMode:           "hardware_touch",
		PolicyDecisionHash:     policyRef,
		ManifestHash:           manifestHash,
		ActionRequestHash:      actionHash,
		RequestDigest:          requestDigest,
		DecisionDigest:         decisionDigest,
	}); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return approvalID
}

func recordContainerRuntimeFactsForBackendEvidence(t *testing.T, s *Service, runID string) {
	t.Helper()
	if err := s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{
		RunID:                   runID,
		StageID:                 "artifact_flow",
		RoleInstanceID:          "workspace-1",
		RoleFamily:              "workspace",
		BackendKind:             launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded,
		ProvisioningPosture:     launcherbackend.ProvisioningPostureAttested,
	}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
}

func backendPostureSelectionEvidenceForState(t *testing.T, state map[string]any) map[string]any {
	t.Helper()
	evidence, ok := state["backend_posture_selection_evidence"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.backend_posture_selection_evidence = %T, want map", state["backend_posture_selection_evidence"])
	}
	return evidence
}

func backendPosturePolicyRefsFromEvidence(t *testing.T, evidence map[string]any) []string {
	t.Helper()
	if refs, ok := evidence["policy_decision_refs"].([]string); ok {
		return refs
	}
	refsAny, ok := evidence["policy_decision_refs"].([]any)
	if !ok {
		t.Fatalf("backend_posture_selection_evidence.policy_decision_refs = %T, want []string", evidence["policy_decision_refs"])
	}
	refs := make([]string, 0, len(refsAny))
	for _, item := range refsAny {
		value, ok := item.(string)
		if !ok {
			t.Fatalf("policy_decision_refs entry = %T, want string", item)
		}
		refs = append(refs, value)
	}
	return refs
}

func backendPostureApprovalEvidenceFromEvidence(t *testing.T, evidence map[string]any) map[string]any {
	t.Helper()
	approvalEvidence, ok := evidence["approval"].(map[string]any)
	if !ok {
		t.Fatalf("backend_posture_selection_evidence.approval = %T, want map", evidence["approval"])
	}
	return approvalEvidence
}

func containsStringInSlice(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func assertBackendPostureApprovalEvidence(t *testing.T, approvalEvidence map[string]any, approvalID, requestDigest, decisionDigest, policyRef string) {
	t.Helper()
	if approvalEvidence["approval_id"] != approvalID {
		t.Fatalf("backend_posture_selection_evidence.approval.approval_id = %v, want %q", approvalEvidence["approval_id"], approvalID)
	}
	if approvalEvidence["approval_request_digest"] != requestDigest {
		t.Fatalf("backend_posture_selection_evidence.approval.approval_request_digest = %v, want %q", approvalEvidence["approval_request_digest"], requestDigest)
	}
	if approvalEvidence["approval_decision_digest"] != decisionDigest {
		t.Fatalf("backend_posture_selection_evidence.approval.approval_decision_digest = %v, want %q", approvalEvidence["approval_decision_digest"], decisionDigest)
	}
	if approvalEvidence["policy_decision_hash"] != policyRef {
		t.Fatalf("backend_posture_selection_evidence.approval.policy_decision_hash = %v, want %q", approvalEvidence["policy_decision_hash"], policyRef)
	}
	if approvalEvidence["status"] != "consumed" {
		t.Fatalf("backend_posture_selection_evidence.approval.status = %v, want consumed", approvalEvidence["status"])
	}
}

func recordContainerIdentityRuntimeFacts(t *testing.T, s *Service, runID string) {
	t.Helper()
	if err := s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{
		RunID:                        runID,
		StageID:                      "artifact_flow",
		RoleInstanceID:               "workspace-1",
		RoleFamily:                   "workspace",
		BackendKind:                  launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel:      launcherbackend.IsolationAssuranceDegraded,
		ProvisioningPosture:          launcherbackend.ProvisioningPostureAttested,
		HypervisorImplementation:     launcherbackend.HypervisorImplementationNotApplicable,
		AccelerationKind:             launcherbackend.AccelerationKindNotApplicable,
		TransportKind:                launcherbackend.TransportKindNotApplicable,
		QEMUProvenance:               &launcherbackend.QEMUProvenance{Version: "9.1.0", BuildIdentity: "qemu-system-x86_64"},
		RuntimeImageDescriptorDigest: "sha256:" + strings.Repeat("d", 64),
	}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
}

func fetchSingleRunSummary(t *testing.T, s *Service, requestID string) RunSummary {
	t.Helper()
	runList, errResp := s.HandleRunList(context.Background(), RunListRequest{SchemaID: "runecode.protocol.v0.RunListRequest", SchemaVersion: "0.1.0", RequestID: requestID, Limit: 10}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunList error response: %+v", errResp)
	}
	if len(runList.Runs) != 1 {
		t.Fatalf("run count = %d, want 1", len(runList.Runs))
	}
	return runList.Runs[0]
}

func assertContainerSummaryIdentityFields(t *testing.T, run RunSummary) {
	t.Helper()
	if run.BackendKind != launcherbackend.BackendKindContainer {
		t.Fatalf("summary.backend_kind = %q, want %q", run.BackendKind, launcherbackend.BackendKindContainer)
	}
	if run.IsolationAssuranceLevel != launcherbackend.IsolationAssuranceDegraded {
		t.Fatalf("summary.isolation_assurance_level = %q, want %q", run.IsolationAssuranceLevel, launcherbackend.IsolationAssuranceDegraded)
	}
	if run.ProvisioningPosture != launcherbackend.ProvisioningPostureAttested {
		t.Fatalf("summary.provisioning_posture = %q, want %q", run.ProvisioningPosture, launcherbackend.ProvisioningPostureAttested)
	}
}

func assertSummaryOmitsBackendSpecificProvenance(t *testing.T, run RunSummary) {
	t.Helper()
	payload, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	serialized := string(payload)
	for _, forbidden := range []string{"qemu_provenance", "hypervisor_implementation", "transport_kind", "runtime_image_descriptor_digest"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("run summary identity contains backend-specific provenance field %q: %s", forbidden, serialized)
		}
	}
}
