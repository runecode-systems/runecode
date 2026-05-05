package launcherdaemon

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestServiceLaunchFailureRecordsDeniedRuntimeFacts(t *testing.T) {
	reporter := &fakeReporter{}
	svc, err := New(Config{Controller: &fakeController{startErr: errors.New("backend_error_code=image_descriptor_signature_mismatch: denied")}, Reporter: reporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := svc.SetInstanceBackendKind(launcherbackend.BackendKindContainer); err != nil {
		t.Fatalf("SetInstanceBackendKind(container) returned error: %v", err)
	}
	if _, err := svc.Launch(context.Background(), validContainerSpecForTests()); err == nil {
		t.Fatal("Launch expected error")
	}
	facts := reporter.factsSnapshot()
	if len(facts) != 1 {
		t.Fatalf("runtime facts count = %d, want 1 denied-launch record", len(facts))
	}
	assertDeniedLaunchReceipt(t, facts[0].LaunchReceipt, validContainerSpecForTests())
}

func TestServiceLaunchFailurePreservesBackendErrorWhenDeniedFactsReportingFails(t *testing.T) {
	launchErr := errors.New("backend launch denied")
	reportErr := errors.New("runtime-facts persist failed")
	reporter := &fakeReporter{factsErr: reportErr}
	svc, err := New(Config{Controller: &fakeController{startErr: launchErr}, Reporter: reporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := svc.SetInstanceBackendKind(launcherbackend.BackendKindContainer); err != nil {
		t.Fatalf("SetInstanceBackendKind(container) returned error: %v", err)
	}
	_, err = svc.Launch(context.Background(), validContainerSpecForTests())
	if err == nil {
		t.Fatal("Launch expected error")
	}
	if !errors.Is(err, launchErr) {
		t.Fatalf("Launch error should preserve backend launch failure; got %v", err)
	}
	if !strings.Contains(err.Error(), reportErr.Error()) {
		t.Fatalf("Launch error = %q, want reporting failure context %q", err.Error(), reportErr.Error())
	}
}

func assertDeniedLaunchReceipt(t *testing.T, got launcherbackend.BackendLaunchReceipt, spec launcherbackend.BackendLaunchSpec) {
	t.Helper()
	if got.LaunchFailureReasonCode != launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch {
		t.Fatalf("launch failure reason = %q, want %q", got.LaunchFailureReasonCode, launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch)
	}
	if got.SessionID != "" {
		t.Fatalf("session_id = %q, want empty for pre-session denial", got.SessionID)
	}
	if got.RuntimeImageDescriptorDigest != spec.Image.DescriptorDigest {
		t.Fatalf("runtime image descriptor digest = %q, want requested descriptor digest", got.RuntimeImageDescriptorDigest)
	}
	if got.Lifecycle == nil {
		t.Fatal("expected denied-launch lifecycle snapshot")
	}
	if got.Lifecycle.TerminateBetweenSteps != spec.LifecyclePolicy.TerminateBetweenSteps {
		t.Fatalf("terminate_between_steps = %v, want %v", got.Lifecycle.TerminateBetweenSteps, spec.LifecyclePolicy.TerminateBetweenSteps)
	}
}
