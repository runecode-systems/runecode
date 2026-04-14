package launcherdaemon

import (
	"context"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

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
