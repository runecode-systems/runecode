package launcherdaemon

import (
	"context"
	"errors"
	"runtime"
	"sync"
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

type fakeContainerController struct {
	launchedSpecs []launcherbackend.BackendLaunchSpec
	state         InstanceState
}

func (f *fakeContainerController) Launch(_ context.Context, spec launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	f.launchedSpecs = append(f.launchedSpecs, spec)
	updates := make(chan RuntimeUpdate, 1)
	close(updates)
	return updates, nil
}

func (f *fakeContainerController) Terminate(context.Context, InstanceRef) error { return nil }

func (f *fakeContainerController) GetState(context.Context, InstanceRef) (InstanceState, error) {
	return f.state, nil
}

func (f *fakeContainerController) Shutdown(context.Context) error { return nil }

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
	mu        sync.RWMutex
	facts     []launcherbackend.RuntimeFactsSnapshot
	lifecycle []launcherbackend.RuntimeLifecycleState
	factsErr  error
	stateErr  error
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

	_, err = svc.Launch(context.Background(), validSpecForTests())
	assertSecondLaunchOutcomeByPlatform(t, controller, err)
}

func assertSecondLaunchOutcomeByPlatform(t *testing.T, controller *failOnceController, secondLaunchErr error) {
	t.Helper()
	if runtime.GOOS == "linux" {
		assertLinuxSecondMicroVMLaunchSucceeds(t, controller, secondLaunchErr)
		return
	}
	assertNonLinuxSecondMicroVMLaunchFailsValidation(t, controller, secondLaunchErr)
}

func assertLinuxSecondMicroVMLaunchSucceeds(t *testing.T, controller *failOnceController, secondLaunchErr error) {
	t.Helper()
	if secondLaunchErr != nil {
		t.Fatalf("second Launch returned error: %v", secondLaunchErr)
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

func assertNonLinuxSecondMicroVMLaunchFailsValidation(t *testing.T, controller *failOnceController, secondLaunchErr error) {
	t.Helper()
	if secondLaunchErr == nil {
		t.Fatal("second Launch expected microvm acceleration unsupported error")
	}
	if len(controller.launchedSpecs) != 0 {
		t.Fatalf("launch count = %d, want 0", len(controller.launchedSpecs))
	}
}

func TestServiceInstanceScopedBackendSelectionAffectsFutureLaunchesOnly(t *testing.T) {
	state := setupServiceWithFakeController(t)
	assertInitialMicroVMSelection(t, state.service)
	if runtime.GOOS == "linux" {
		state.launchAndAssertBackend(t, validSpecForTests(), launcherbackend.BackendKindMicroVM)
	} else {
		if _, err := state.service.Launch(context.Background(), validSpecForTests()); err == nil {
			t.Fatal("Launch expected microvm acceleration unsupported error")
		}
		if len(state.controller.launchedSpecs) != 0 {
			t.Fatalf("launch count = %d, want 0", len(state.controller.launchedSpecs))
		}
	}
	assertNoLiveMigrationOnPostureChange(t, state)
	state.launchAndAssertBackend(t, validContainerSpecForTests(), launcherbackend.BackendKindContainer)
	if runtime.GOOS == "linux" {
		state.assertFirstLaunchRemainsMicroVM(t)
	}
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
	if got := svc.ActiveInstanceID(); got == "" {
		t.Fatal("active instance id should be minted after start")
	}
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
	before := svc.ActiveInstanceID()
	if before == "" {
		t.Fatal("initial instance id should be set")
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
	after := svc.ActiveInstanceID()
	if after == "" || after == before {
		t.Fatalf("instance id after restart = %q, want new non-empty instance id different from %q", after, before)
	}
	if got := svc.InstanceBackendKind(); got != launcherbackend.BackendKindMicroVM {
		t.Fatalf("instance backend after restart = %q, want %q", got, launcherbackend.BackendKindMicroVM)
	}
}

func TestServiceApplyInstanceBackendPostureRequiresMatchingInstanceID(t *testing.T) {
	controller := &fakeController{}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := svc.ApplyInstanceBackendPosture("launcher-instance-wrong", launcherbackend.BackendKindContainer); err == nil {
		t.Fatal("ApplyInstanceBackendPosture expected instance mismatch error")
	}
	if err := svc.ApplyInstanceBackendPosture(svc.ActiveInstanceID(), launcherbackend.BackendKindContainer); err != nil {
		t.Fatalf("ApplyInstanceBackendPosture returned error: %v", err)
	}
	if got := svc.InstanceBackendKind(); got != launcherbackend.BackendKindContainer {
		t.Fatalf("instance backend after apply = %q, want %q", got, launcherbackend.BackendKindContainer)
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

func TestServiceRoutesContainerLaunchToContainerController(t *testing.T) {
	micro := &fakeController{}
	container := &fakeContainerController{}
	svc, err := New(Config{MicroVMController: micro, ContainerController: container})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := svc.SetInstanceBackendKind(launcherbackend.BackendKindContainer); err != nil {
		t.Fatalf("SetInstanceBackendKind(container) returned error: %v", err)
	}
	if _, err := svc.Launch(context.Background(), validContainerSpecForTests()); err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	if len(container.launchedSpecs) != 1 {
		t.Fatalf("container launch count = %d, want 1", len(container.launchedSpecs))
	}
	if len(micro.launchedSpecs) != 0 {
		t.Fatalf("microvm launch count = %d, want 0 for container route", len(micro.launchedSpecs))
	}
}

func TestServiceLaunchRejectsBackendConflictWithActiveInstancePosture(t *testing.T) {
	micro := &fakeController{}
	container := &fakeContainerController{}
	svc, err := New(Config{MicroVMController: micro, ContainerController: container})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := svc.SetInstanceBackendKind(launcherbackend.BackendKindContainer); err != nil {
		t.Fatalf("SetInstanceBackendKind(container) returned error: %v", err)
	}
	if _, err := svc.Launch(context.Background(), validSpecForTests()); err == nil {
		t.Fatal("Launch expected backend conflict error")
	}
	if len(micro.launchedSpecs) != 0 {
		t.Fatalf("microvm launch count = %d, want 0", len(micro.launchedSpecs))
	}
	if len(container.launchedSpecs) != 0 {
		t.Fatalf("container launch count = %d, want 0", len(container.launchedSpecs))
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
	for reporter.factsCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if reporter.factsCount() == 0 {
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
	svc.mu.Lock()
	svc.instanceBackendByKey[instanceKey(ref)] = launcherbackend.BackendKindMicroVM
	svc.mu.Unlock()
	state, err := svc.GetState(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if !state.Active {
		t.Fatal("GetState active=false, want true")
	}
	if err := svc.Terminate(context.Background(), ref); err != nil {
		t.Fatalf("Terminate returned error: %v", err)
	}
	if controller.stops != 1 {
		t.Fatalf("controller stops = %d, want 1", controller.stops)
	}
	if _, err := svc.GetState(context.Background(), ref); err == nil {
		t.Fatal("GetState expected unknown backend routing error after terminate removed routing")
	}
}

func TestServiceTerminateAndGetStateFailClosedWhenBackendRoutingUnknown(t *testing.T) {
	controller := &fakeController{state: InstanceState{Active: true}}
	svc, err := New(Config{Controller: controller})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	ref := InstanceRef{RunID: "run-1", StageID: "stage-1", RoleInstanceID: "role-1"}
	if _, err := svc.GetState(context.Background(), ref); err == nil {
		t.Fatal("GetState expected unknown backend routing error")
	}
	if err := svc.Terminate(context.Background(), ref); err == nil {
		t.Fatal("Terminate expected unknown backend routing error")
	}
}
