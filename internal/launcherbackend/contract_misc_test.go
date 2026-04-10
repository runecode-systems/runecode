package launcherbackend

import "testing"

func TestBackendLifecycleTransitionFailClosed(t *testing.T) {
	if err := ValidateBackendLifecycleTransition(BackendLifecycleStatePlanned, BackendLifecycleStateLaunching); err != nil {
		t.Fatalf("ValidateBackendLifecycleTransition returned error: %v", err)
	}
	if err := ValidateBackendLifecycleTransition(BackendLifecycleStateStarted, BackendLifecycleStateActive); err == nil {
		t.Fatal("ValidateBackendLifecycleTransition expected started->active rejection")
	}
	if err := ValidateBackendLifecycleTransition("unknown", BackendLifecycleStateActive); err == nil {
		t.Fatal("ValidateBackendLifecycleTransition expected unknown state rejection")
	}
}

func TestRuntimeImageDescriptorRejectsUnknownBackendKind(t *testing.T) {
	descriptor := RuntimeImageDescriptor{
		DescriptorDigest:      testDigest("1"),
		BackendKind:           "qemu",
		PlatformCompatibility: RuntimeImagePlatformCompat{OS: "linux", Architecture: "amd64"},
		BootContractVersion:   "v1",
		ComponentDigests: map[string]string{
			"kernel": testDigest("2"),
			"rootfs": testDigest("3"),
		},
	}
	if err := descriptor.Validate(); err == nil {
		t.Fatal("Validate expected unknown backend kind error")
	}
}
