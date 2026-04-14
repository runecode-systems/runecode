package launcherdaemon

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type fakeController struct {
	startErr error
	stopErr  error

	starts int
	stops  int

	launchedSpecs []launcherbackend.BackendLaunchSpec

	state InstanceState
}

type failOnceController struct {
	launches      int
	launchedSpecs []launcherbackend.BackendLaunchSpec
}

func (f *failOnceController) Launch(_ context.Context, spec launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	f.launches++
	f.launchedSpecs = append(f.launchedSpecs, spec)
	if f.launches == 1 {
		return nil, errors.New("microvm launch failed")
	}
	updates := make(chan RuntimeUpdate, 1)
	return updates, nil
}

func (f *failOnceController) Terminate(context.Context, InstanceRef) error { return nil }

func (f *failOnceController) GetState(context.Context, InstanceRef) (InstanceState, error) {
	return InstanceState{}, nil
}

func (f *failOnceController) Shutdown(context.Context) error { return nil }

func (f *fakeController) Launch(_ context.Context, spec launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	f.starts++
	f.launchedSpecs = append(f.launchedSpecs, spec)
	if f.startErr != nil {
		return nil, f.startErr
	}
	updates := make(chan RuntimeUpdate, 1)
	return updates, nil
}

func (f *fakeController) Terminate(context.Context, InstanceRef) error {
	f.stops++
	return f.stopErr
}

func (f *fakeController) GetState(context.Context, InstanceRef) (InstanceState, error) {
	return f.state, nil
}

func (f *fakeController) Shutdown(context.Context) error {
	f.stops++
	return f.stopErr
}

type fakeReporter struct {
	facts     []launcherbackend.RuntimeFactsSnapshot
	lifecycle []launcherbackend.RuntimeLifecycleState
	factsErr  error
	stateErr  error
}

func (f *fakeReporter) RecordRuntimeFacts(_ string, facts launcherbackend.RuntimeFactsSnapshot) error {
	if f.factsErr != nil {
		return f.factsErr
	}
	f.facts = append(f.facts, facts)
	return nil
}

func (f *fakeReporter) RecordRuntimeLifecycleState(_ string, lifecycle launcherbackend.RuntimeLifecycleState) error {
	if f.stateErr != nil {
		return f.stateErr
	}
	f.lifecycle = append(f.lifecycle, lifecycle)
	return nil
}

func TestServiceStartStopLifecycle(t *testing.T) {
	controller := &fakeController{}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if got := svc.State(); got != StatePlanned {
		t.Fatalf("initial state = %s, want %s", got, StatePlanned)
	}

	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if got := svc.State(); got != StateServing {
		t.Fatalf("state after start = %s, want %s", got, StateServing)
	}
	if controller.starts != 0 {
		t.Fatalf("controller starts = %d, want 0", controller.starts)
	}

	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if got := svc.State(); got != StateStopped {
		t.Fatalf("state after stop = %s, want %s", got, StateStopped)
	}
	if controller.stops != 1 {
		t.Fatalf("controller stops = %d, want 1", controller.stops)
	}
}

func TestServiceLaunchFailureReturnsError(t *testing.T) {
	svc, err := New(Config{Controller: &fakeController{startErr: errors.New("boom")}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if _, err := svc.Launch(context.Background(), validSpecForTests()); err == nil {
		t.Fatal("Launch expected error")
	}
}

func TestServiceMicroVMLaunchFailureDoesNotAutoSwitchToContainerMode(t *testing.T) {
	controller := &failOnceController{}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if _, err := svc.Launch(context.Background(), validSpecForTests()); err == nil {
		t.Fatal("first Launch expected microvm failure")
	}
	if got := svc.InstanceBackendKind(); got != launcherbackend.BackendKindMicroVM {
		t.Fatalf("instance backend after failure = %q, want %q", got, launcherbackend.BackendKindMicroVM)
	}

	if _, err := svc.Launch(context.Background(), validSpecForTests()); err != nil {
		t.Fatalf("second Launch returned error: %v", err)
	}
	if len(controller.launchedSpecs) != 2 {
		t.Fatalf("launch count = %d, want 2", len(controller.launchedSpecs))
	}
	if controller.launchedSpecs[0].RequestedBackend != launcherbackend.BackendKindMicroVM {
		t.Fatalf("first launch requested_backend = %q, want %q", controller.launchedSpecs[0].RequestedBackend, launcherbackend.BackendKindMicroVM)
	}
	if controller.launchedSpecs[1].RequestedBackend != launcherbackend.BackendKindMicroVM {
		t.Fatalf("second launch requested_backend = %q, want %q", controller.launchedSpecs[1].RequestedBackend, launcherbackend.BackendKindMicroVM)
	}
}

func TestServiceInstanceScopedBackendSelectionAffectsFutureLaunchesOnly(t *testing.T) {
	state := setupServiceWithFakeController(t)
	assertInitialMicroVMSelection(t, state.service)
	state.launchAndAssertBackend(t, validSpecForTests(), launcherbackend.BackendKindMicroVM)
	assertNoLiveMigrationOnPostureChange(t, state)
	state.launchAndAssertBackend(t, validContainerSpecForTests(), launcherbackend.BackendKindContainer)
	state.assertFirstLaunchRemainsMicroVM(t)
}

type serviceHarness struct {
	service    *Service
	controller *fakeController
}

func setupServiceWithFakeController(t *testing.T) serviceHarness {
	t.Helper()
	controller := &fakeController{}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	return serviceHarness{service: svc, controller: controller}
}

func assertInitialMicroVMSelection(t *testing.T, svc *Service) {
	t.Helper()
	if got := svc.InstanceBackendKind(); got != launcherbackend.BackendKindMicroVM {
		t.Fatalf("initial instance backend = %q, want %q", got, launcherbackend.BackendKindMicroVM)
	}
}

func (h serviceHarness) launchAndAssertBackend(t *testing.T, spec launcherbackend.BackendLaunchSpec, wantBackend string) {
	t.Helper()
	if _, err := h.service.Launch(context.Background(), spec); err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	if got := len(h.controller.launchedSpecs); got == 0 {
		t.Fatal("launch count = 0, want at least 1")
	}
	last := h.controller.launchedSpecs[len(h.controller.launchedSpecs)-1]
	if last.RequestedBackend != wantBackend {
		t.Fatalf("last launch requested_backend = %q, want %q", last.RequestedBackend, wantBackend)
	}
}

func assertNoLiveMigrationOnPostureChange(t *testing.T, h serviceHarness) {
	t.Helper()
	if err := h.service.SetInstanceBackendKind(launcherbackend.BackendKindContainer); err != nil {
		t.Fatalf("SetInstanceBackendKind(container) returned error: %v", err)
	}
	if got := h.service.InstanceBackendKind(); got != launcherbackend.BackendKindContainer {
		t.Fatalf("instance backend after switch = %q, want %q", got, launcherbackend.BackendKindContainer)
	}
	if h.controller.stops != 0 {
		t.Fatalf("controller stops after posture change = %d, want 0 (no live migration)", h.controller.stops)
	}
}

func (h serviceHarness) assertFirstLaunchRemainsMicroVM(t *testing.T) {
	t.Helper()
	if len(h.controller.launchedSpecs) != 2 {
		t.Fatalf("launch count = %d, want 2", len(h.controller.launchedSpecs))
	}
	if h.controller.launchedSpecs[0].RequestedBackend != launcherbackend.BackendKindMicroVM {
		t.Fatalf("first launch backend mutated to %q, want %q", h.controller.launchedSpecs[0].RequestedBackend, launcherbackend.BackendKindMicroVM)
	}
}

func TestServiceRestartResetsInstanceBackendToPreferredMicroVM(t *testing.T) {
	controller := &fakeController{}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := svc.SetInstanceBackendKind(launcherbackend.BackendKindContainer); err != nil {
		t.Fatalf("SetInstanceBackendKind(container) returned error: %v", err)
	}
	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start(restart) returned error: %v", err)
	}
	if got := svc.InstanceBackendKind(); got != launcherbackend.BackendKindMicroVM {
		t.Fatalf("instance backend after restart = %q, want %q", got, launcherbackend.BackendKindMicroVM)
	}
}

func TestServiceLaunchRejectsContainerBackendOutsideWorkspaceRoleFamily(t *testing.T) {
	controller := &fakeController{}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := svc.SetInstanceBackendKind(launcherbackend.BackendKindContainer); err != nil {
		t.Fatalf("SetInstanceBackendKind(container) returned error: %v", err)
	}
	spec := validContainerSpecForTests()
	spec.RoleFamily = "gateway"
	if _, err := svc.Launch(context.Background(), spec); err == nil {
		t.Fatal("Launch expected role-family scope error for container backend")
	}
	if len(controller.launchedSpecs) != 0 {
		t.Fatalf("controller launches = %d, want 0", len(controller.launchedSpecs))
	}
}

func TestServiceStopFromInvalidStateFails(t *testing.T) {
	svc, err := New(Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Stop(context.Background()); err == nil {
		t.Fatal("Stop expected invalid-state error")
	}
}

func TestServiceLaunchConsumesRuntimeUpdates(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("microvm/kvm launch validation is linux-only in MVP")
	}
	reporter := &fakeReporter{}
	controller := &scriptedController{}
	svc, err := New(Config{Controller: controller, Reporter: reporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	spec := validSpecForTests()
	if _, err := svc.Launch(context.Background(), spec); err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for len(reporter.facts) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(reporter.facts) == 0 {
		t.Fatal("expected runtime facts update")
	}
}

func TestServiceReporterFailureMarksServiceFailed(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("microvm/kvm launch validation is linux-only in MVP")
	}
	reporter := &fakeReporter{factsErr: errors.New("persist failed")}
	controller := &scriptedController{}
	svc, err := New(Config{Controller: controller, Reporter: reporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if _, err := svc.Launch(context.Background(), validSpecForTests()); err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for svc.State() != StateFailed && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := svc.State(); got != StateFailed {
		t.Fatalf("service state = %s, want %s", got, StateFailed)
	}
}

func TestServiceTerminateAndGetState(t *testing.T) {
	controller := &fakeController{state: InstanceState{Active: true}}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	ref := InstanceRef{RunID: "run-1", StageID: "stage-1", RoleInstanceID: "role-1"}
	if err := svc.Terminate(context.Background(), ref); err != nil {
		t.Fatalf("Terminate returned error: %v", err)
	}
	if controller.stops != 1 {
		t.Fatalf("controller stops = %d, want 1", controller.stops)
	}
	state, err := svc.GetState(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if !state.Active {
		t.Fatal("GetState active=false, want true")
	}
}

type scriptedController struct{}

func (scriptedController) Launch(context.Context, launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	updates := make(chan RuntimeUpdate, 2)
	facts := launcherbackend.DefaultRuntimeFacts("run-1")
	facts.LaunchReceipt.BackendKind = launcherbackend.BackendKindMicroVM
	updates <- RuntimeUpdate{RunID: "run-1", Facts: &facts}
	updates <- RuntimeUpdate{RunID: "run-1", Lifecycle: &launcherbackend.RuntimeLifecycleState{BackendLifecycle: &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateActive, PreviousState: launcherbackend.BackendLifecycleStateBinding, TerminateBetweenSteps: true, TransitionCount: 3}}}
	close(updates)
	return updates, nil
}

func (scriptedController) Terminate(context.Context, InstanceRef) error { return nil }

func (scriptedController) GetState(context.Context, InstanceRef) (InstanceState, error) {
	return InstanceState{}, nil
}

func (scriptedController) Shutdown(context.Context) error { return nil }

func repeatHex(ch byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

func validSpecForTests() launcherbackend.BackendLaunchSpec {
	return launcherbackend.BackendLaunchSpec{
		RunID:                     "run-1",
		StageID:                   "stage-1",
		RoleInstanceID:            "role-1",
		RoleFamily:                "role",
		RoleKind:                  "hello",
		RequestedBackend:          launcherbackend.BackendKindMicroVM,
		RequestedAccelerationKind: launcherbackend.AccelerationKindKVM,
		ControlTransportKind:      launcherbackend.TransportKindVSock,
		Image: launcherbackend.RuntimeImageDescriptor{
			DescriptorDigest:      "sha256:" + repeatHex('a'),
			BackendKind:           launcherbackend.BackendKindMicroVM,
			BootContractVersion:   "v1",
			PlatformCompatibility: launcherbackend.RuntimeImagePlatformCompat{OS: "linux", Architecture: "amd64", AccelerationKind: launcherbackend.AccelerationKindKVM},
			ComponentDigests:      map[string]string{"kernel": "sha256:" + repeatHex('b'), "rootfs": "sha256:" + repeatHex('c')},
			Signing:               &launcherbackend.RuntimeImageSigningHooks{SignerRef: "test", SignatureDigest: "sha256:" + repeatHex('d')},
		},
		Attachments: launcherbackend.AttachmentPlan{
			ByRole: map[string]launcherbackend.AttachmentBinding{
				launcherbackend.AttachmentRoleLaunchContext:  {ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelReadOnlyVolume, RequiredDigests: []string{"sha256:" + repeatHex('e')}},
				launcherbackend.AttachmentRoleWorkspace:      {ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelWritableVolume},
				launcherbackend.AttachmentRoleInputArtifacts: {ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelArtifactImage, RequiredDigests: []string{"sha256:" + repeatHex('f')}},
				launcherbackend.AttachmentRoleScratch:        {ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelEphemeralVolume},
			},
			Constraints:         launcherbackend.AttachmentRealizationConstraints{NoHostFilesystemMounts: true},
			WorkspaceEncryption: &launcherbackend.WorkspaceEncryptionPosture{Required: true, AtRestProtection: launcherbackend.WorkspaceAtRestProtectionHostManagedEncryption, KeyProtectionPosture: launcherbackend.WorkspaceKeyProtectionOSKeystore, Effective: true},
		},
		ResourceLimits:  launcherbackend.BackendResourceLimits{VCPUCount: 1, MemoryMiB: 256, DiskMiB: 128, LaunchTimeoutSeconds: 30, BindTimeoutSeconds: 10, ActiveTimeoutSeconds: 30, TerminationGraceSeconds: 2},
		WatchdogPolicy:  launcherbackend.BackendWatchdogPolicy{Enabled: true, TerminateOnMisbehavior: true, HeartbeatTimeoutSeconds: 5, NoProgressTimeoutSeconds: 10},
		LifecyclePolicy: launcherbackend.BackendLifecyclePolicy{TerminateBetweenSteps: true},
		CachePosture:    launcherbackend.BackendCachePosture{ResetOrDestroyBeforeReuse: true, DigestPinned: true, SignaturePinned: true},
	}
}

func validContainerSpecForTests() launcherbackend.BackendLaunchSpec {
	spec := validSpecForTests()
	spec.RoleFamily = "workspace"
	spec.RequestedBackend = launcherbackend.BackendKindContainer
	spec.RequestedAccelerationKind = ""
	spec.ControlTransportKind = ""
	spec.Image.BackendKind = launcherbackend.BackendKindContainer
	spec.Image.PlatformCompatibility.AccelerationKind = ""
	spec.Image.ComponentDigests = map[string]string{"image": "sha256:" + repeatHex('b')}
	return spec
}
