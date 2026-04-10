//go:build linux

package launcherdaemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestQEMUVerticalSliceHelloWorld(t *testing.T) {
	skipIfVerticalSliceUnavailable(t)

	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	brokerSvc, err := brokerapi.NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	svc, err := New(Config{Controller: NewQEMUController(QEMUControllerConfig{WorkRoot: t.TempDir()}), Reporter: brokerSvc})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	t.Cleanup(func() { _ = svc.Stop(context.Background()) })

	runID := "run-vertical-slice"
	spec := validSpecForTests()
	spec.RunID = runID
	spec.StageID = "stage-hello"
	spec.RoleInstanceID = "role-hello"
	spec.ResourceLimits.ActiveTimeoutSeconds = 20

	if _, err := svc.Launch(context.Background(), spec); err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	waitForCompletedTerminalReport(t, brokerSvc, runID, 45*time.Second)
}

func skipIfVerticalSliceUnavailable(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("launcher hardening requires unprivileged execution")
	}
	if _, err := os.Stat("/usr/bin/qemu-system-x86_64"); err != nil {
		t.Skip("qemu-system-x86_64 unavailable")
	}
	if _, err := os.Stat("/dev/kvm"); err != nil {
		t.Skip("/dev/kvm unavailable")
	}
	if kernels, _ := filepath.Glob("/boot/vmlinuz-*"); len(kernels) == 0 {
		t.Skip("no readable /boot/vmlinuz-* kernel")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable for initramfs build")
	}
}

func waitForCompletedTerminalReport(t *testing.T, brokerSvc *brokerapi.Service, runID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		facts := brokerSvc.RuntimeFacts(runID)
		if facts.TerminalReport != nil {
			if facts.TerminalReport.TerminationKind != launcherbackend.BackendTerminationKindCompleted {
				t.Fatalf("terminal kind = %q, failure=%q", facts.TerminalReport.TerminationKind, facts.TerminalReport.FailureReasonCode)
			}
			if facts.LaunchReceipt.BackendKind != launcherbackend.BackendKindMicroVM {
				t.Fatalf("backend_kind = %q, want %q", facts.LaunchReceipt.BackendKind, launcherbackend.BackendKindMicroVM)
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("timed out waiting for terminal report")
}
