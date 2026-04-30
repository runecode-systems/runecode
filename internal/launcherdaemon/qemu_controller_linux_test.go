//go:build linux

package launcherdaemon

import (
	"context"
	"os"
	"path/filepath"
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

	receipt, err := buildLaunchReceipt(spec, admission, "isolate-1", "session-1", strings.Repeat("a", 32), "9.0.0", "qemu-system-x86_64 9.0.0", nil)
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
