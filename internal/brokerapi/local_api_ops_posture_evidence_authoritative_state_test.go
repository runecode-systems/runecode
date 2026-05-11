package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-systems/runecode/internal/launcherbackend"
	"github.com/runecode-systems/runecode/internal/policyengine"
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
