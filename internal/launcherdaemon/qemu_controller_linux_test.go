//go:build linux

package launcherdaemon

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestQEMUPrepareLaunchAssetsVerifiesToolchainArtifactBeforeLaunch(t *testing.T) {
	workRoot, qemuBinary, spec := qemuToolchainVerificationLaunchSpecForTests(t)
	controller := &qemuController{cfg: QEMUControllerConfig{WorkRoot: workRoot, QEMUBinary: qemuBinary, Now: time.Now}, instances: map[string]*qemuInstance{}}

	if _, _, _, _, err := controller.prepareLaunchAssets(context.Background(), qemuBinary, spec); err != nil {
		t.Fatalf("prepareLaunchAssets returned error: %v", err)
	}
}

func TestQEMUPrepareLaunchAssetsSurfacesToolchainVerificationFailure(t *testing.T) {
	workRoot, qemuBinary, spec := qemuToolchainVerificationLaunchSpecForTests(t)
	tamperedBinary := filepath.Join(workRoot, "fixtures", "qemu-system-x86_64-tampered")
	if err := os.WriteFile(tamperedBinary, []byte("#!/bin/sh\necho tampered\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(tampered qemu fixture) returned error: %v", err)
	}
	controller := &qemuController{cfg: QEMUControllerConfig{WorkRoot: workRoot, QEMUBinary: qemuBinary, Now: time.Now}, instances: map[string]*qemuInstance{}}

	_, _, _, _, err := controller.prepareLaunchAssets(context.Background(), tamperedBinary, spec)
	if err == nil {
		t.Fatal("prepareLaunchAssets expected toolchain verification failure")
	}
	if !strings.Contains(err.Error(), launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch) {
		t.Fatalf("prepareLaunchAssets error = %v, want image descriptor signature mismatch", err)
	}
}

func TestQEMULaunchReceiptCarriesTrustedRuntimeAttestation(t *testing.T) {
	_, _, spec := qemuToolchainVerificationLaunchSpecForTests(t)
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	attestedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	receipt, err := buildLaunchReceipt(spec, admission, "isolate-1", "session-1", strings.Repeat("a", 32), "9.0.0", "qemu-system-x86_64 9.0.0", nil, attestedAt)
	if err != nil {
		t.Fatalf("buildLaunchReceipt returned error: %v", err)
	}
	facts := launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: receipt, HardeningPosture: buildHardeningPosture()}
	evidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if receipt.ProvisioningPosture != launcherbackend.ProvisioningPostureAttested {
		t.Fatalf("provisioning posture = %q, want %q", receipt.ProvisioningPosture, launcherbackend.ProvisioningPostureAttested)
	}
	if evidence.Attestation == nil {
		t.Fatal("attestation evidence missing")
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != launcherbackend.AttestationVerificationResultValid {
		t.Fatalf("verification result = %q, want %q", evidence.AttestationVerification.VerificationResult, launcherbackend.AttestationVerificationResultValid)
	}
	if evidence.AttestationVerification.ReplayVerdict != launcherbackend.AttestationReplayVerdictOriginal {
		t.Fatalf("replay verdict = %q, want %q", evidence.AttestationVerification.ReplayVerdict, launcherbackend.AttestationReplayVerdictOriginal)
	}
	if got, want := receipt.AttestationVerificationTimestamp, attestedAt.Format(time.RFC3339); got != want {
		t.Fatalf("attestation verification timestamp = %q, want %q", got, want)
	}
}

func TestApplyTrustedRuntimeAttestationFailsClosedWithoutLaunchContextDigest(t *testing.T) {
	_, _, spec := qemuToolchainVerificationLaunchSpecForTests(t)
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	attestedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	receipt, err := buildLaunchReceipt(spec, admission, "isolate-1", "session-1", strings.Repeat("a", 32), "9.0.0", "qemu-system-x86_64 9.0.0", nil, attestedAt)
	if err != nil {
		t.Fatalf("buildLaunchReceipt returned error: %v", err)
	}
	receipt.LaunchContextDigest = ""

	err = applyTrustedRuntimeAttestation(&receipt, admission, attestedAt)
	if err == nil {
		t.Fatal("applyTrustedRuntimeAttestation expected missing launch context digest error")
	}
	if !strings.Contains(err.Error(), "session binding is required before attestation") {
		t.Fatalf("applyTrustedRuntimeAttestation error = %q, want session binding failure", err.Error())
	}
}

func TestQEMUPrepareLaunchDirUsesUniquePathWithFixedClock(t *testing.T) {
	workRoot := t.TempDir()
	controller := &qemuController{cfg: QEMUControllerConfig{WorkRoot: workRoot, Now: func() time.Time { return time.Unix(123, 0).UTC() }}, instances: map[string]*qemuInstance{}}
	spec := validSpecForTests()

	first, err := controller.prepareLaunchDir(spec)
	if err != nil {
		t.Fatalf("prepareLaunchDir(first) returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(first) })
	second, err := controller.prepareLaunchDir(spec)
	if err != nil {
		t.Fatalf("prepareLaunchDir(second) returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(second) })

	if first == second {
		t.Fatalf("prepareLaunchDir returned identical paths %q with fixed clock", first)
	}
	for _, dir := range []string{first, second} {
		manifest := filepath.Join(dir, "attachments")
		if _, err := os.Stat(manifest); err != nil {
			t.Fatalf("os.Stat(%q) returned error: %v", manifest, err)
		}
	}
	firstParts := strings.Split(filepath.Clean(first), string(os.PathSeparator))
	secondParts := strings.Split(filepath.Clean(second), string(os.PathSeparator))
	if len(firstParts) != len(secondParts) {
		t.Fatalf("launch dir depth mismatch: %q vs %q", first, second)
	}
	if !reflect.DeepEqual(firstParts[:len(firstParts)-1], secondParts[:len(secondParts)-1]) {
		t.Fatalf("launch dir parents differ: %q vs %q", first, second)
	}
}

func qemuToolchainVerificationLaunchSpecForTests(t *testing.T) (string, string, launcherbackend.BackendLaunchSpec) {
	t.Helper()
	workRoot := t.TempDir()
	qemuBinary := seedVerticalSliceQEMUFixture(t, workRoot)
	spec := validSpecForTests()
	materializeComponentDigests(t, workRoot, &spec.Image)
	seedRuntimeImageVerificationAssets(t, workRoot, qemuBinary, &spec)
	return workRoot, qemuBinary, spec
}
