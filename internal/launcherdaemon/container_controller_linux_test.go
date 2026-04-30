//go:build linux

package launcherdaemon

import (
	"context"
	"os"
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
	if got, want := first.Facts.LaunchReceipt.AttestationVerificationTimestamp, attestedAt.Format(time.RFC3339); got != want {
		t.Fatalf("attestation verification timestamp = %q, want %q", got, want)
	}
}

func TestLaunchReceiptBuildersUseDerivedRuntimeSessionBinding(t *testing.T) {
	attestedAt := time.Date(2026, time.March, 4, 5, 6, 7, 0, time.UTC)
	for _, tc := range launchReceiptBuilderTests(attestedAt) {
		t.Run(tc.name, func(t *testing.T) {
			assertLaunchReceiptUsesDerivedRuntimeSessionBinding(t, tc.spec, tc.build)
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

func assertLaunchReceiptUsesDerivedRuntimeSessionBinding(t *testing.T, spec launcherbackend.BackendLaunchSpec, build func(launcherbackend.BackendLaunchSpec, launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, error)) {
	t.Helper()
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	receipt, err := build(spec, admission)
	if err != nil {
		t.Fatalf("build receipt returned error: %v", err)
	}
	binding := mustDeriveRuntimeSessionBinding(t, spec, admission.DescriptorDigest, "isolate-shared", "session-shared", strings.Repeat("a", 32))
	if receipt.ProvisioningPosture != launcherbackend.ProvisioningPostureAttested {
		t.Fatalf("provisioning posture = %q, want %q", receipt.ProvisioningPosture, launcherbackend.ProvisioningPostureAttested)
	}
	assertReceiptSessionBindingMatches(t, receipt, binding)
}

func assertReceiptSessionBindingMatches(t *testing.T, receipt launcherbackend.BackendLaunchReceipt, binding runtimeSessionBinding) {
	t.Helper()
	actual := runtimeSessionBinding{
		IsolateID:                receipt.IsolateID,
		SessionID:                receipt.SessionID,
		SessionNonce:             receipt.SessionNonce,
		LaunchContextDigest:      receipt.LaunchContextDigest,
		HandshakeTranscriptHash:  receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue: receipt.IsolateSessionKeyIDValue,
	}
	if actual != binding {
		t.Fatalf("receipt session binding fields = %+v, want %+v", actual, binding)
	}
}

func launchReceiptBuilderTests(attestedAt time.Time) []struct {
	name  string
	spec  launcherbackend.BackendLaunchSpec
	build func(launcherbackend.BackendLaunchSpec, launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, error)
} {
	return []struct {
		name  string
		spec  launcherbackend.BackendLaunchSpec
		build func(launcherbackend.BackendLaunchSpec, launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, error)
	}{
		{
			name: "microvm",
			spec: validSpecForTests(),
			build: func(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, error) {
				return buildLaunchReceipt(spec, admission, "isolate-shared", "session-shared", strings.Repeat("a", 32), "9.0.0", "qemu-system-x86_64 9.0.0", nil, attestedAt)
			},
		},
		{
			name: "container",
			spec: validContainerSpecForTests(),
			build: func(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord) (launcherbackend.BackendLaunchReceipt, error) {
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
