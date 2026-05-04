package brokerapi

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestProjectAttestationIdentityStateSeparatesSessionEvidenceAndVerificationSignals(t *testing.T) {
	baseEvidence := launcherbackend.RuntimeEvidenceSnapshot{
		Launch:  launcherbackend.LaunchRuntimeEvidence{ProvisioningPosture: launcherbackend.ProvisioningPostureAttested},
		Session: &launcherbackend.SessionRuntimeEvidence{EvidenceDigest: "sha256:" + strings.Repeat("1", 64)},
	}

	state := map[string]any{}
	projectAttestationIdentityState(state, baseEvidence)
	assertAttestationSignals(t, state, true, false, false, false)

	withAttestation := baseEvidence
	withAttestation.Attestation = &launcherbackend.IsolateAttestationEvidence{EvidenceDigest: "sha256:" + strings.Repeat("2", 64)}
	state = map[string]any{}
	projectAttestationIdentityState(state, withAttestation)
	assertAttestationSignals(t, state, true, true, false, false)

	withValidVerification := withAttestation
	withValidVerification.AttestationVerification = &launcherbackend.IsolateAttestationVerificationRecord{
		VerificationResult: launcherbackend.AttestationVerificationResultValid,
		ReplayVerdict:      launcherbackend.AttestationReplayVerdictOriginal,
	}
	state = map[string]any{}
	projectAttestationIdentityState(state, withValidVerification)
	assertAttestationSignals(t, state, true, true, true, false)

	withInvalidVerification := withAttestation
	withInvalidVerification.AttestationVerification = &launcherbackend.IsolateAttestationVerificationRecord{
		VerificationResult: launcherbackend.AttestationVerificationResultInvalid,
		ReplayVerdict:      launcherbackend.AttestationReplayVerdictUnknown,
	}
	state = map[string]any{}
	projectAttestationIdentityState(state, withInvalidVerification)
	assertAttestationSignals(t, state, true, true, false, true)
}

func assertAttestationSignals(t *testing.T, state map[string]any, wantSessionBinding, wantEvidence, wantVerificationSucceeded, wantVerificationFailed bool) {
	t.Helper()
	if got, _ := state["session_binding_present"].(bool); got != wantSessionBinding {
		t.Fatalf("session_binding_present = %v, want %v", got, wantSessionBinding)
	}
	if got, _ := state["attestation_evidence_present"].(bool); got != wantEvidence {
		t.Fatalf("attestation_evidence_present = %v, want %v", got, wantEvidence)
	}
	if got, _ := state["attestation_verification_succeeded"].(bool); got != wantVerificationSucceeded {
		t.Fatalf("attestation_verification_succeeded = %v, want %v", got, wantVerificationSucceeded)
	}
	if got, _ := state["attestation_verification_failed"].(bool); got != wantVerificationFailed {
		t.Fatalf("attestation_verification_failed = %v, want %v", got, wantVerificationFailed)
	}
}
