package launcherdaemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type fakeController struct {
	startErr error
	stopErr  error

	starts int
	stops  int

	state InstanceState
}

func (f *fakeController) Launch(context.Context, launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	f.starts++
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
				launcherbackend.AttachmentRoleLaunchContext:  {ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelReadOnlyChannel, RequiredDigests: []string{"sha256:" + repeatHex('e')}},
				launcherbackend.AttachmentRoleWorkspace:      {ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelVirtualDisk},
				launcherbackend.AttachmentRoleInputArtifacts: {ReadOnly: true, ChannelKind: launcherbackend.AttachmentChannelVirtualDisk, RequiredDigests: []string{"sha256:" + repeatHex('f')}},
				launcherbackend.AttachmentRoleScratch:        {ReadOnly: false, ChannelKind: launcherbackend.AttachmentChannelEphemeralDisk},
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
