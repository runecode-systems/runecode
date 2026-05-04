package launcherdaemon

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestPopulateRuntimeSessionBindingDoesNotAwardAttestedPosture(t *testing.T) {
	spec := validSpecForTests()
	binding := mustDeriveRuntimeSessionBinding(t, spec, spec.Image.DescriptorDigest, "isolate-1", "session-1", strings.Repeat("a", 32))
	receipt := launcherbackend.BackendLaunchReceipt{ProvisioningPosture: launcherbackend.ProvisioningPostureUnknown}

	populateRuntimeSessionBinding(&receipt, binding)

	if got, want := receipt.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU; got != want {
		t.Fatalf("provisioning posture = %q, want %q", got, want)
	}
	if receipt.LaunchContextDigest != "" || receipt.HandshakeTranscriptHash != "" || receipt.IsolateSessionKeyIDValue != "" {
		t.Fatal("launch-time session binding must not claim validated secure-session fields")
	}
	if receipt.AttestationVerificationResult == launcherbackend.AttestationVerificationResultValid {
		t.Fatal("populateRuntimeSessionBinding must not set attestation verification success")
	}
}

func TestRecordValidatedSecureSessionKeepsReceiptPostureAtTOFU(t *testing.T) {
	spec, admission, receipt := runtimeAttestationReceiptFixtureForValidation(t)
	if receipt.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("pre-validation provisioning posture = %q, want %q", receipt.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
	}
	if receipt.SessionSecurity != nil {
		t.Fatal("session_security must be empty before runtime secure-session update")
	}
	summary, launchContextDigest, err := validateSecureSessionAndBuildSummary(spec, receipt)
	if err != nil {
		t.Fatalf("validateSecureSessionAndBuildSummary returned error: %v", err)
	}
	if err := recordValidatedSecureSession(&receipt, summary, launchContextDigest); err != nil {
		t.Fatalf("recordValidatedSecureSession returned error: %v", err)
	}
	assertValidatedReceiptStillTOFU(t, receipt)

	postHandshakeInput, err := buildPostHandshakeAttestationProgress(receipt, admission)
	if err != nil {
		t.Fatalf("buildPostHandshakeAttestationProgress returned error: %v", err)
	}
	assertPostHandshakeInputUsesReceiptBinding(t, receipt, postHandshakeInput)
}

func runtimeAttestationReceiptFixtureForValidation(t *testing.T) (launcherbackend.BackendLaunchSpec, launcherbackend.RuntimeAdmissionRecord, launcherbackend.BackendLaunchReceipt) {
	t.Helper()
	spec := validSpecForTests()
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	binding := mustDeriveRuntimeSessionBinding(t, spec, admission.DescriptorDigest, "isolate-1", "session-1", strings.Repeat("a", 32))
	receipt := launcherbackend.BackendLaunchReceipt{
		RunID:                        spec.RunID,
		StageID:                      spec.StageID,
		RoleInstanceID:               spec.RoleInstanceID,
		BackendKind:                  launcherbackend.BackendKindMicroVM,
		IsolationAssuranceLevel:      launcherbackend.IsolationAssuranceIsolated,
		TransportKind:                launcherbackend.TransportKindVSock,
		RuntimeImageDescriptorDigest: admission.DescriptorDigest,
		RuntimeImageBootProfile:      admission.BootContractVersion,
		BootComponentDigestByName:    cloneMap(admission.ComponentDigests),
		AuthorityStateDigest:         admission.AuthorityStateDigest,
	}
	populateRuntimeSessionBinding(&receipt, binding)
	return spec, admission, receipt
}

func assertValidatedReceiptStillTOFU(t *testing.T, receipt launcherbackend.BackendLaunchReceipt) {
	t.Helper()
	if got, want := receipt.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU; got != want {
		t.Fatalf("provisioning posture = %q, want %q", got, want)
	}
	if receipt.SessionSecurity == nil {
		t.Fatal("session_security missing after secure-session validation")
	}
	if !receipt.SessionSecurity.MutuallyAuthenticated || !receipt.SessionSecurity.Encrypted || !receipt.SessionSecurity.ProofOfPossessionVerified {
		t.Fatal("session_security did not record validated secure-session posture")
	}
	if got, want := receipt.AttestationVerificationResult, launcherbackend.AttestationVerificationResultUnknown; got != want {
		t.Fatalf("attestation verification result = %q, want %q", got, want)
	}
}

func assertPostHandshakeInputUsesReceiptBinding(t *testing.T, receipt launcherbackend.BackendLaunchReceipt, postHandshakeInput *launcherbackend.PostHandshakeRuntimeAttestationInput) {
	t.Helper()
	if postHandshakeInput == nil {
		t.Fatal("post-handshake attestation input missing")
	}
	if got, want := postHandshakeInput.LaunchContextDigest, receipt.LaunchContextDigest; got != want {
		t.Fatalf("post-handshake launch context digest = %q, want %q", got, want)
	}
	if got, want := postHandshakeInput.VerificationResult, launcherbackend.AttestationVerificationResultUnknown; got != want {
		t.Fatalf("post-handshake verification result = %q, want %q", got, want)
	}
	if postHandshakeInput.VerificationTimestamp != "" {
		t.Fatalf("post-handshake verification timestamp = %q, want empty", postHandshakeInput.VerificationTimestamp)
	}
}

func TestBuildPostHandshakeAttestationProgressFailsWithoutValidatedSessionBinding(t *testing.T) {
	spec := validSpecForTests()
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	receipt := launcherbackend.BackendLaunchReceipt{
		RunID:                        spec.RunID,
		StageID:                      spec.StageID,
		RoleInstanceID:               spec.RoleInstanceID,
		TransportKind:                launcherbackend.TransportKindVSock,
		RuntimeImageDescriptorDigest: admission.DescriptorDigest,
		RuntimeImageBootProfile:      admission.BootContractVersion,
		BootComponentDigestByName:    cloneMap(admission.ComponentDigests),
	}

	_, err = buildPostHandshakeAttestationProgress(receipt, admission)
	if err == nil {
		t.Fatal("buildPostHandshakeAttestationProgress expected error")
	}
	if !strings.Contains(err.Error(), "session binding is required before attestation") {
		t.Fatalf("error = %q, want missing secure-session binding", err.Error())
	}
}
