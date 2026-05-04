package launcherdaemon

import (
	"strings"
	"testing"
	"time"

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
	if receipt.AttestationVerificationResult == launcherbackend.AttestationVerificationResultValid {
		t.Fatal("populateRuntimeSessionBinding must not set attestation verification success")
	}
}

func TestUpgradeReceiptAfterSecureSessionValidationKeepsReceiptPostureAtTOFU(t *testing.T) {
	spec, admission, receipt := runtimeAttestationReceiptFixtureForValidation(t)
	if receipt.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("pre-validation provisioning posture = %q, want %q", receipt.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
	}

	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	postHandshakeInput, err := upgradeReceiptAfterSecureSessionValidation(&receipt, spec, admission, now)
	if err != nil {
		t.Fatalf("upgradeReceiptAfterSecureSessionValidation returned error: %v", err)
	}
	assertValidatedReceiptStillTOFU(t, receipt)
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
}

func TestUpgradeReceiptAfterSecureSessionValidationFailsWithoutSessionBinding(t *testing.T) {
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

	_, err = upgradeReceiptAfterSecureSessionValidation(&receipt, spec, admission, time.Now())
	if err == nil {
		t.Fatal("upgradeReceiptAfterSecureSessionValidation expected error")
	}
	if !strings.Contains(err.Error(), "session binding is required before secure session validation") {
		t.Fatalf("error = %q, want missing secure-session binding", err.Error())
	}
}
