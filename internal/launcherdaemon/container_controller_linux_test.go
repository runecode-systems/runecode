//go:build linux

package launcherdaemon

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestContainerControllerLaunchFailsClosedWhenAdmissionFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("container controller requires rootless launcher execution")
	}
	workRoot := t.TempDir()
	controller := NewContainerController(ContainerControllerConfig{WorkRoot: workRoot})

	_, err := controller.Launch(context.Background(), validContainerSpecForTests())
	if err == nil {
		t.Fatal("Launch expected admission failure")
	}
	if !strings.Contains(err.Error(), launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch) {
		t.Fatalf("Launch error = %v, want image descriptor signature mismatch", err)
	}
}

func TestContainerControllerLaunchUsesAdmittedRuntimeIdentityInReceipt(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("container controller requires rootless launcher execution")
	}
	workRoot, spec := admittedContainerSpecForReceiptTest(t)
	controller := NewContainerController(ContainerControllerConfig{WorkRoot: workRoot})
	updates, err := controller.Launch(context.Background(), spec)
	if err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	assertContainerLaunchReceiptUsesAdmittedRuntimeIdentity(t, updates, spec)
}

func TestContainerControllerLaunchUsesInjectedClockForAttestationTimestamp(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("container controller requires rootless launcher execution")
	}
	workRoot, spec := admittedContainerSpecForReceiptTest(t)
	attestedAt := time.Date(2026, time.February, 3, 4, 5, 6, 0, time.UTC)
	controller := NewContainerController(ContainerControllerConfig{WorkRoot: workRoot, Now: func() time.Time { return attestedAt }})
	updates, err := controller.Launch(context.Background(), spec)
	if err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	first, ok := <-updates
	if !ok || first.Facts == nil {
		t.Fatal("first runtime update must include facts")
	}
	if got, want := first.Facts.LaunchReceipt.AttestationVerificationTimestamp, ""; got != want {
		t.Fatalf("attestation verification timestamp = %q, want %q", got, want)
	}
	if first.Facts.PostHandshakeAttestationInput == nil {
		t.Fatal("post-handshake attestation input missing")
	}
	if got, want := first.Facts.PostHandshakeAttestationInput.VerificationTimestamp, attestedAt.Format(time.RFC3339); got != want {
		t.Fatalf("post-handshake verification timestamp = %q, want %q", got, want)
	}
}

func TestLaunchReceiptBuildersRequireSecureSessionValidationBeforeAttestedPosture(t *testing.T) {
	attestedAt := time.Date(2026, time.March, 4, 5, 6, 7, 0, time.UTC)
	for _, tc := range launchReceiptBuilderTests(attestedAt) {
		t.Run(tc.name, func(t *testing.T) {
			assertLaunchReceiptRequiresSecureSessionValidationBeforeAttestedPosture(t, tc.spec, tc.build)
		})
	}
}

func TestLaunchReceiptBuildersRequirePostHandshakeEvidenceForAttestationSuccess(t *testing.T) {
	attestedAt := time.Date(2026, time.March, 5, 6, 7, 8, 0, time.UTC)
	for _, tc := range launchReceiptBuilderTests(attestedAt) {
		t.Run(tc.name, func(t *testing.T) {
			evidence := buildLaunchReceiptEvidenceForTest(t, tc.spec, tc.build, func(facts *launcherbackend.RuntimeFactsSnapshot) {
				facts.PostHandshakeAttestationInput = nil
			})
			assertInvalidAttestationEvidence(t, evidence, "attestation_post_handshake_input_required")
		})
	}
}

func TestLaunchReceiptBuildersRequireSecureSessionValidationForAttestationSuccess(t *testing.T) {
	attestedAt := time.Date(2026, time.March, 6, 7, 8, 9, 0, time.UTC)
	for _, tc := range launchReceiptBuilderTests(attestedAt) {
		t.Run(tc.name, func(t *testing.T) {
			evidence := buildLaunchReceiptEvidenceForTest(t, tc.spec, tc.build, func(facts *launcherbackend.RuntimeFactsSnapshot) {
				facts.LaunchReceipt.SessionSecurity = nil
			})
			assertInvalidAttestationEvidence(t, evidence, "attestation_session_validation_required")
		})
	}
}

func TestMakeRuntimeIdentityUsesDistinctFullSessionIdentifier(t *testing.T) {
	isolateID, sessionID, nonce, err := makeRuntimeIdentity("run-1")
	if err != nil {
		t.Fatalf("makeRuntimeIdentity returned error: %v", err)
	}
	if !strings.HasPrefix(isolateID, "isolate-run-1-") {
		t.Fatalf("isolate id = %q, want isolate-run-1-*", isolateID)
	}
	if !strings.HasPrefix(sessionID, "session-") {
		t.Fatalf("session id = %q, want session-*", sessionID)
	}
	if len(nonce) != 32 {
		t.Fatalf("nonce length = %d, want 32 hex chars", len(nonce))
	}
	if got, want := len(strings.TrimPrefix(sessionID, "session-")), 16; got != want {
		t.Fatalf("session id suffix length = %d, want %d hex chars", got, want)
	}
	if strings.TrimPrefix(sessionID, "session-") == nonce[8:16] {
		t.Fatal("session id should no longer reuse only a 32-bit nonce slice")
	}
}

func assertLaunchReceiptRequiresSecureSessionValidationBeforeAttestedPosture(t *testing.T, spec launcherbackend.BackendLaunchSpec, build func(launcherbackend.BackendLaunchSpec, launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, *launcherbackend.PostHandshakeRuntimeAttestationInput, error)) {
	t.Helper()
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	receipt, attestationInput, err := build(spec, admission)
	if err != nil {
		t.Fatalf("build receipt returned error: %v", err)
	}
	if receipt.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("provisioning posture = %q, want %q", receipt.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
	}
	if receipt.IsolateID != "isolate-shared" || receipt.SessionID != "session-shared" || receipt.SessionNonce != strings.Repeat("a", 32) {
		t.Fatalf("receipt session tuple = (%q, %q, %q), want (%q, %q, %q)", receipt.IsolateID, receipt.SessionID, receipt.SessionNonce, "isolate-shared", "session-shared", strings.Repeat("a", 32))
	}
	if receipt.LaunchContextDigest == "" || receipt.HandshakeTranscriptHash == "" || receipt.IsolateSessionKeyIDValue == "" {
		t.Fatal("secure-session validated binding fields must be populated")
	}
	if receipt.SessionSecurity == nil {
		t.Fatal("session_security must be populated after secure-session validation")
	}
	facts := launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: receipt, PostHandshakeAttestationInput: attestationInput, HardeningPosture: launcherbackend.AppliedHardeningPosture{Requested: launcherbackend.HardeningRequestedHardened, Effective: launcherbackend.HardeningEffectiveHardened}}
	evidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.Launch.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("evidence launch provisioning posture = %q, want %q", evidence.Launch.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
	}
}

func buildLaunchReceiptEvidenceForTest(t *testing.T, spec launcherbackend.BackendLaunchSpec, build func(launcherbackend.BackendLaunchSpec, launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, *launcherbackend.PostHandshakeRuntimeAttestationInput, error), mutate func(*launcherbackend.RuntimeFactsSnapshot)) launcherbackend.RuntimeEvidenceSnapshot {
	t.Helper()
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	receipt, attestationInput, err := build(spec, admission)
	if err != nil {
		t.Fatalf("build receipt returned error: %v", err)
	}
	facts := launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt:                 receipt,
		PostHandshakeAttestationInput: attestationInput,
		HardeningPosture:              launcherbackend.AppliedHardeningPosture{Requested: launcherbackend.HardeningRequestedHardened, Effective: launcherbackend.HardeningEffectiveHardened},
	}
	if mutate != nil {
		mutate(&facts)
	}
	evidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	return evidence
}

func assertInvalidAttestationEvidence(t *testing.T, evidence launcherbackend.RuntimeEvidenceSnapshot, expectedReason string) {
	t.Helper()
	if evidence.Attestation != nil {
		t.Fatalf("attestation evidence = %#v, want nil for invalid attestation", evidence.Attestation)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if got, want := evidence.AttestationVerification.VerificationResult, launcherbackend.AttestationVerificationResultInvalid; got != want {
		t.Fatalf("verification result = %q, want %q", got, want)
	}
	if !slices.Contains(evidence.AttestationVerification.ReasonCodes, expectedReason) {
		t.Fatalf("reason codes = %v, want %s", evidence.AttestationVerification.ReasonCodes, expectedReason)
	}
	attestationPosture, _ := launcherbackend.DeriveAttestationPostureFromEvidence(evidence)
	if attestationPosture == launcherbackend.AttestationPostureValid {
		t.Fatalf("attestation posture = %q, want not valid", attestationPosture)
	}
}

func launchReceiptBuilderTests(attestedAt time.Time) []struct {
	name  string
	spec  launcherbackend.BackendLaunchSpec
	build func(launcherbackend.BackendLaunchSpec, launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, *launcherbackend.PostHandshakeRuntimeAttestationInput, error)
} {
	return []struct {
		name  string
		spec  launcherbackend.BackendLaunchSpec
		build func(launcherbackend.BackendLaunchSpec, launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, *launcherbackend.PostHandshakeRuntimeAttestationInput, error)
	}{
		{
			name: "microvm",
			spec: validSpecForTests(),
			build: func(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, *launcherbackend.PostHandshakeRuntimeAttestationInput, error) {
				return buildLaunchReceipt(spec, admission, "isolate-shared", "session-shared", strings.Repeat("a", 32), "9.0.0", "qemu-system-x86_64 9.0.0", nil, attestedAt)
			},
		},
		{
			name: "container",
			spec: validContainerSpecForTests(),
			build: func(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, *launcherbackend.PostHandshakeRuntimeAttestationInput, error) {
				return containerLaunchReceipt(spec, admission, "isolate-shared", "session-shared", strings.Repeat("a", 32), attestedAt)
			},
		},
	}
}

func admittedContainerSpecForReceiptTest(t *testing.T) (string, launcherbackend.BackendLaunchSpec) {
	t.Helper()
	workRoot := t.TempDir()
	spec := validContainerSpecForTests()
	materializeComponentDigests(t, workRoot, &spec.Image)
	seedRuntimeImageSignatureAssets(t, workRoot, &spec.Image)
	spec.Image.Signing.Toolchain = &launcherbackend.RuntimeToolchainSigningHooks{
		DescriptorSchemaID:      launcherbackend.RuntimeToolchainDescriptorSchemaID,
		DescriptorSchemaVersion: launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
		DescriptorDigest:        "sha256:" + repeatHex('4'),
		SignerRef:               "signer:runtime-toolchain",
		SignatureDigest:         "sha256:" + repeatHex('5'),
		VerifierSetRef:          "sha256:" + repeatHex('8'),
		BundleDigest:            "sha256:" + repeatHex('6'),
	}
	seedRuntimeToolchainSignatureAssets(t, workRoot, &spec.Image, false)
	return workRoot, spec
}

func assertContainerLaunchReceiptUsesAdmittedRuntimeIdentity(t *testing.T, updates <-chan RuntimeUpdate, spec launcherbackend.BackendLaunchSpec) {
	t.Helper()
	first, ok := <-updates
	if !ok || first.Facts == nil {
		t.Fatal("first runtime update must include facts")
	}
	receipt := first.Facts.LaunchReceipt
	if receipt.RuntimeImageDescriptorDigest != spec.Image.DescriptorDigest {
		t.Fatalf("receipt runtime image digest = %q, want %q", receipt.RuntimeImageDescriptorDigest, spec.Image.DescriptorDigest)
	}
	if receipt.RuntimeToolchainDescriptorDigest == "" || receipt.RuntimeToolchainSignatureDigest == "" {
		t.Fatal("receipt should include admitted runtime toolchain identity")
	}
	if receipt.AuthorityStateDigest == "" || receipt.AuthorityStateRevision == 0 {
		t.Fatal("receipt should include authority state identity used for admission")
	}
	if got, want := receipt.BootComponentDigestByName["image"], spec.Image.ComponentDigests["image"]; got != want {
		t.Fatalf("receipt boot component image digest = %q, want %q", got, want)
	}
}
