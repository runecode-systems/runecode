package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestProjectSupportedRuntimeRequirementsStateReducedAssuranceApprovalStatusSemantics(t *testing.T) {
	tests := []struct {
		name           string
		approvalStatus string
		prefillBacked  bool
		wantBacked     bool
		wantSatisfied  bool
	}{
		{name: "consumed supports reduced assurance", approvalStatus: "consumed", wantBacked: true, wantSatisfied: true},
		{name: "approved supports reduced assurance", approvalStatus: "approved", wantBacked: true, wantSatisfied: true},
		{name: "pending does not support reduced assurance", approvalStatus: "pending", wantBacked: false, wantSatisfied: false},
		{name: "superseded does not support reduced assurance", approvalStatus: "superseded", wantBacked: false, wantSatisfied: false},
		{name: "denied does not support reduced assurance", approvalStatus: "denied", wantBacked: false, wantSatisfied: false},
		{name: "expired does not support reduced assurance", approvalStatus: "expired", wantBacked: false, wantSatisfied: false},
		{name: "cancelled does not support reduced assurance", approvalStatus: "cancelled", wantBacked: false, wantSatisfied: false},
		{name: "prefilled true does not bypass pending evidence", approvalStatus: "pending", prefillBacked: true, wantBacked: false, wantSatisfied: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := map[string]any{
				"attestation_posture":        launcherbackend.AttestationPostureValid,
				"attestation_verifier_class": launcherbackend.AttestationVerifierClassTrustedDomainLocal,
				"runtime_posture_degraded":   true,
				"backend_posture_selection_evidence": map[string]any{
					"approval": map[string]any{"status": tt.approvalStatus},
				},
			}
			if tt.prefillBacked {
				state["reduced_assurance_approval_backed"] = true
			}

			projectSupportedRuntimeRequirementsState(state)
			assertReducedAssuranceApprovalProjection(t, state, tt.approvalStatus, tt.wantBacked, tt.wantSatisfied)
		})
	}
}

func assertReducedAssuranceApprovalProjection(t *testing.T, state map[string]any, approvalStatus string, wantBacked, wantSatisfied bool) {
	t.Helper()
	if state["reduced_assurance_approval_backed"] != wantBacked {
		t.Fatalf("reduced_assurance_approval_backed = %v, want %v", state["reduced_assurance_approval_backed"], wantBacked)
	}
	if state["reduced_assurance_approval_status"] != approvalStatus {
		t.Fatalf("reduced_assurance_approval_status = %v, want %q", state["reduced_assurance_approval_status"], approvalStatus)
	}
	if state["supported_runtime_requirements_satisfied"] != wantSatisfied {
		t.Fatalf("supported_runtime_requirements_satisfied = %v, want %v", state["supported_runtime_requirements_satisfied"], wantSatisfied)
	}
	if wantSatisfied {
		if _, ok := state["supported_runtime_requirement_reason_codes"]; ok {
			t.Fatalf("supported_runtime_requirement_reason_codes present for satisfied state: %v", state["supported_runtime_requirement_reason_codes"])
		}
		return
	}
	reasons, ok := state["supported_runtime_requirement_reason_codes"].([]string)
	if !ok {
		t.Fatalf("supported_runtime_requirement_reason_codes = %T, want []string", state["supported_runtime_requirement_reason_codes"])
	}
	if !containsStringInSlice(reasons, "reduced_assurance_selection_evidence_missing") {
		t.Fatalf("supported_runtime_requirement_reason_codes = %v, want include reduced_assurance_selection_evidence_missing", reasons)
	}
}

func TestProjectSupportedRuntimeRequirementsStateClearsStaleFields(t *testing.T) {
	state := map[string]any{
		"attestation_posture":                        launcherbackend.AttestationPostureValid,
		"attestation_verifier_class":                 launcherbackend.AttestationVerifierClassTrustedDomainLocal,
		"runtime_posture_degraded":                   false,
		"supported_runtime_requirement_reason_codes": []string{"stale_reason"},
		"reduced_assurance_approval_backed":          true,
		"reduced_assurance_approval_status":          "consumed",
		"supported_runtime_requirements_satisfied":   false,
	}

	projectSupportedRuntimeRequirementsState(state)

	if state["supported_runtime_requirements_satisfied"] != true {
		t.Fatalf("supported_runtime_requirements_satisfied = %v, want true", state["supported_runtime_requirements_satisfied"])
	}
	if _, ok := state["supported_runtime_requirement_reason_codes"]; ok {
		t.Fatalf("supported_runtime_requirement_reason_codes present after satisfied projection: %v", state["supported_runtime_requirement_reason_codes"])
	}
	if _, ok := state["reduced_assurance_approval_backed"]; ok {
		t.Fatalf("reduced_assurance_approval_backed present for non-degraded runtime: %v", state["reduced_assurance_approval_backed"])
	}
	if _, ok := state["reduced_assurance_approval_status"]; ok {
		t.Fatalf("reduced_assurance_approval_status present for non-degraded runtime: %v", state["reduced_assurance_approval_status"])
	}
}

func TestRunDetailAuthoritativeStateBackendPostureSelectionEvidenceFallbackDoesNotCrossInstances(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-backend-evidence-cross-instance"
	const otherInstanceID = "launcher-instance-2"
	const otherSelectorRunID = "instance-control:launcher-instance-2"
	const manifestHash = "sha256:" + "1111111111111111111111111111111111111111111111111111111111111111"
	const actionHash = "sha256:" + "3333333333333333333333333333333333333333333333333333333333333333"
	const requestDigest = "sha256:" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const decisionDigest = "sha256:" + "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	otherPolicyRef := recordBackendPosturePolicyDecisionForRun(t, s, otherSelectorRunID, manifestHash, actionHash, otherInstanceID)
	recordBackendPostureApprovalForRun(t, s, runID, otherSelectorRunID, otherPolicyRef, manifestHash, actionHash, requestDigest, decisionDigest, otherInstanceID)
	recordContainerRuntimeFactsForBackendEvidence(t, s, runID)

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-backend-evidence-cross-instance", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	if _, ok := runGet.Run.AuthoritativeState["backend_posture_selection_evidence"]; ok {
		t.Fatalf("backend_posture_selection_evidence present for unrelated instance approval: %v", runGet.Run.AuthoritativeState["backend_posture_selection_evidence"])
	}
	if runGet.Run.AuthoritativeState["reduced_assurance_approval_backed"] != false {
		t.Fatalf("authoritative_state.reduced_assurance_approval_backed = %v, want false without matching instance approval", runGet.Run.AuthoritativeState["reduced_assurance_approval_backed"])
	}
	if runGet.Run.AuthoritativeState["supported_runtime_requirements_satisfied"] != false {
		t.Fatalf("authoritative_state.supported_runtime_requirements_satisfied = %v, want false without matching instance approval", runGet.Run.AuthoritativeState["supported_runtime_requirements_satisfied"])
	}
	reasons, ok := runGet.Run.AuthoritativeState["supported_runtime_requirement_reason_codes"].([]string)
	if !ok {
		t.Fatalf("supported_runtime_requirement_reason_codes = %T, want []string", runGet.Run.AuthoritativeState["supported_runtime_requirement_reason_codes"])
	}
	if !containsStringInSlice(reasons, "reduced_assurance_selection_evidence_missing") {
		t.Fatalf("supported_runtime_requirement_reason_codes = %v, want include reduced_assurance_selection_evidence_missing", reasons)
	}
}

func TestRunDetailAuthoritativeStateBackendPostureSelectionEvidenceAllowsLegacyRunScopedFallback(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-backend-evidence-legacy-fallback"
	const instanceID = "launcher-instance-1"
	const manifestHash = "sha256:" + "1111111111111111111111111111111111111111111111111111111111111111"
	const actionHash = "sha256:" + "3333333333333333333333333333333333333333333333333333333333333333"
	const requestDigest = "sha256:" + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const decisionDigest = "sha256:" + "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	legacyPolicyRef := recordBackendPosturePolicyDecisionForRun(t, s, runID, manifestHash, actionHash, instanceID)
	recordLegacyRunScopedBackendPostureApprovalForRun(t, s, runID, legacyPolicyRef, manifestHash, actionHash, requestDigest, decisionDigest)
	recordContainerRuntimeFactsForBackendEvidence(t, s, runID)

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-backend-evidence-legacy-fallback", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	evidence := backendPostureSelectionEvidenceForState(t, runGet.Run.AuthoritativeState)
	approvalEvidence := backendPostureApprovalEvidenceFromEvidence(t, evidence)
	if approvalEvidence["status"] != "consumed" {
		t.Fatalf("backend_posture_selection_evidence.approval.status = %v, want consumed", approvalEvidence["status"])
	}
	if runGet.Run.AuthoritativeState["reduced_assurance_approval_backed"] != true {
		t.Fatalf("authoritative_state.reduced_assurance_approval_backed = %v, want true for legacy run-scoped approval", runGet.Run.AuthoritativeState["reduced_assurance_approval_backed"])
	}
}

func TestRunDetailAuthoritativeStatePrefersSelectorApprovalOverLegacyRunScopedApproval(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-backend-evidence-selector-preferred"
	const instanceID = "launcher-instance-1"
	const selectorRunID = "instance-control:launcher-instance-1"
	const manifestHash = "sha256:" + "1111111111111111111111111111111111111111111111111111111111111111"
	const legacyActionHash = "sha256:" + "3333333333333333333333333333333333333333333333333333333333333333"
	const selectorActionHash = "sha256:" + "4444444444444444444444444444444444444444444444444444444444444444"
	legacyRequestDigest := "sha256:" + strings.Repeat("b", 64)
	legacyDecisionDigest := "sha256:" + strings.Repeat("c", 64)
	selectorRequestDigest := "sha256:" + strings.Repeat("d", 64)
	selectorDecisionDigest := "sha256:" + strings.Repeat("e", 64)

	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	_ = putRunScopedArtifactForLocalOpsTest(t, s, selectorRunID, "step-1")
	legacyPolicyRef := recordBackendPosturePolicyDecisionForRun(t, s, runID, manifestHash, legacyActionHash, instanceID)
	recordLegacyRunScopedBackendPostureApprovalForRun(t, s, runID, legacyPolicyRef, manifestHash, legacyActionHash, legacyRequestDigest, legacyDecisionDigest)
	selectorPolicyRef := recordBackendPosturePolicyDecisionForRun(t, s, selectorRunID, manifestHash, selectorActionHash, instanceID)
	selectorApprovalID := recordBackendPostureApprovalForRun(t, s, runID, selectorRunID, selectorPolicyRef, manifestHash, selectorActionHash, selectorRequestDigest, selectorDecisionDigest, instanceID)
	recordContainerRuntimeFactsForBackendEvidence(t, s, runID)

	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-backend-evidence-selector-preferred", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	evidence := backendPostureSelectionEvidenceForState(t, runGet.Run.AuthoritativeState)
	approvalEvidence := backendPostureApprovalEvidenceFromEvidence(t, evidence)
	assertBackendPostureApprovalEvidence(t, approvalEvidence, selectorApprovalID, selectorRequestDigest, selectorDecisionDigest, selectorPolicyRef)
}

func recordLegacyRunScopedBackendPostureApprovalForRun(t *testing.T, s *Service, runID, policyRef, manifestHash, actionHash, requestDigest, decisionDigest string) string {
	t.Helper()
	approvalID := "sha256:" + strings.Repeat("e", 64)
	now := time.Now().UTC().Round(0)
	if err := s.RecordApproval(artifacts.ApprovalRecord{
		ApprovalID:             approvalID,
		Status:                 "consumed",
		WorkspaceID:            workspaceIDForRun(runID),
		RunID:                  runID,
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
