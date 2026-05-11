package brokerapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/runecode-systems/runecode/internal/launcherbackend"
)

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

func TestRunDetailAuthoritativeStateKeepsSyntheticReceiptAttestationUnsupportedAcrossBackends(t *testing.T) {
	tests := []struct {
		name      string
		backend   string
		isolation string
	}{
		{name: "microvm", backend: launcherbackend.BackendKindMicroVM, isolation: launcherbackend.IsolationAssuranceIsolated},
		{name: "container", backend: launcherbackend.BackendKindContainer, isolation: launcherbackend.IsolationAssuranceDegraded},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			state, evidence := recordAndFetchSyntheticReceiptOnlyAttestation(t, tc.backend, tc.isolation)
			assertSyntheticReceiptOnlyAuthoritativeState(t, state)
			assertSyntheticReceiptOnlyRuntimeEvidence(t, evidence)
		})
	}
}

func recordAndFetchSyntheticReceiptOnlyAttestation(t *testing.T, backend string, isolation string) (map[string]any, launcherbackend.RuntimeEvidenceSnapshot) {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-synthetic-receipt-only-" + backend
	_ = putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	if err := s.RecordRuntimeFacts(runID, syntheticReceiptOnlyAttestationFacts(runID, backend, isolation)); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	runGet, errResp := s.HandleRunGet(context.Background(), RunGetRequest{SchemaID: "runecode.protocol.v0.RunGetRequest", SchemaVersion: "0.1.0", RequestID: "req-run-synthetic-receipt", RunID: runID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleRunGet error response: %+v", errResp)
	}
	return runGet.Run.AuthoritativeState, s.RuntimeEvidence(runID)
}

func assertSyntheticReceiptOnlyAuthoritativeState(t *testing.T, state map[string]any) {
	t.Helper()
	if state["provisioning_posture"] != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("authoritative_state.provisioning_posture = %v, want %q", state["provisioning_posture"], launcherbackend.ProvisioningPostureTOFU)
	}
	if state["supported_runtime_requirements_satisfied"] != false {
		t.Fatalf("authoritative_state.supported_runtime_requirements_satisfied = %v, want false for synthetic receipt-only attestation", state["supported_runtime_requirements_satisfied"])
	}
	if state["attestation_posture"] == launcherbackend.AttestationPostureValid {
		t.Fatalf("authoritative_state.attestation_posture = %v, want not %q", state["attestation_posture"], launcherbackend.AttestationPostureValid)
	}
	if state["attestation_evidence_present"] != false {
		t.Fatalf("authoritative_state.attestation_evidence_present = %v, want false", state["attestation_evidence_present"])
	}
	if got := renderTruthfulnessShapeFromAuthoritativeState(state); got != "secure session bound without verified attestation; beta attested story still gated by post-handshake verification" {
		t.Fatalf("truthfulness cue = %q, want secure-session-bound truthful wording", got)
	}
}

func assertSyntheticReceiptOnlyRuntimeEvidence(t *testing.T, evidence launcherbackend.RuntimeEvidenceSnapshot) {
	t.Helper()
	if evidence.Attestation != nil {
		t.Fatalf("runtime evidence attestation = %#v, want nil without post-handshake evidence", evidence.Attestation)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("runtime evidence attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != launcherbackend.AttestationVerificationResultInvalid {
		t.Fatalf("runtime evidence verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, launcherbackend.AttestationVerificationResultInvalid)
	}
	if !containsStringInSlice(evidence.AttestationVerification.ReasonCodes, "attestation_post_handshake_input_required") {
		t.Fatalf("runtime evidence reason_codes = %v, want include attestation_post_handshake_input_required", evidence.AttestationVerification.ReasonCodes)
	}
}

func renderTruthfulnessShapeFromAuthoritativeState(state map[string]any) string {
	posture, reasons := attestationTruthfulnessStateForTest(state)
	verificationSucceeded, _ := state["attestation_verification_succeeded"].(bool)
	sessionBindingPresent, _ := state["session_binding_present"].(bool)
	attestationEvidencePresent, _ := state["attestation_evidence_present"].(bool)
	supportedRuntimeSatisfied, _ := state["supported_runtime_requirements_satisfied"].(bool)
	currentEvidence := "launch-only evidence"
	switch {
	case verificationSucceeded:
		currentEvidence = "post-handshake verification succeeded"
	case attestationEvidencePresent:
		currentEvidence = "post-handshake evidence collected but not yet supportable"
	case sessionBindingPresent:
		currentEvidence = "secure session bound without verified attestation"
	}
	if supportedRuntimeSatisfied && posture == launcherbackend.AttestationPostureValid {
		return currentEvidence + "; supported attested posture earned from verified post-handshake evidence"
	}
	if len(reasons) > 0 {
		return currentEvidence + "; beta attested story still gated by post-handshake verification; reasons=" + strings.Join(reasons, ",")
	}
	return currentEvidence + "; beta attested story still gated by post-handshake verification"
}

func attestationTruthfulnessStateForTest(state map[string]any) (string, []string) {
	posture, _ := state["attestation_posture"].(string)
	if reasons, ok := state["attestation_reason_codes"].([]string); ok {
		return posture, append([]string{}, reasons...)
	}
	reasonsAny, _ := state["attestation_reason_codes"].([]any)
	reasons := make([]string, 0, len(reasonsAny))
	for _, value := range reasonsAny {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			reasons = append(reasons, s)
		}
	}
	return posture, reasons
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
	if run.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("summary.provisioning_posture = %q, want %q", run.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
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
