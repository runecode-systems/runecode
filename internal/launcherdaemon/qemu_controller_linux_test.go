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

func qemuToolchainVerificationLaunchSpecForTests(t *testing.T) (string, string, launcherbackend.BackendLaunchSpec) {
	t.Helper()
	workRoot := t.TempDir()
	qemuBinary := seedVerticalSliceQEMUFixture(t, workRoot)
	spec := validSpecForTests()
	materializeComponentDigests(t, workRoot, &spec.Image)
	seedRuntimeImageVerificationAssets(t, workRoot, qemuBinary, &spec)
	return workRoot, qemuBinary, spec
}
