//go:build linux

package launcherdaemon

import (
	"context"
	"os"
	"strings"
	"testing"

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
