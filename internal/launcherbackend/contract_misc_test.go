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
		BootContractVersion:   BootProfileMicroVMLinuxKernelInitrdV1,
		ComponentDigests: map[string]string{
			"kernel": testDigest("2"),
			"initrd": testDigest("3"),
		},
	}
	if err := descriptor.Validate(); err == nil {
		t.Fatal("Validate expected unknown backend kind error")
	}
}

func TestRuntimeImageDescriptorBindsDescriptorDigestToCanonicalSignedPayload(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	if err := descriptor.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	descriptor.ComponentDigests["initrd"] = testDigest("f")
	if err := descriptor.Validate(); err == nil {
		t.Fatal("Validate expected descriptor digest mismatch after signed payload change")
	}
}

func TestRuntimeImageDescriptorRejectsLegacyMicroVMBootShape(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	descriptor.BootContractVersion = "v1"
	if err := descriptor.Validate(); err == nil {
		t.Fatal("Validate expected legacy boot contract rejection")
	}
}

func TestRuntimeImageDescriptorRequiresVerifierSetDigest(t *testing.T) {
	descriptor := validRuntimeImageDescriptorForContractTests()
	descriptor.Signing.VerifierSetRef = "verifier-set:runtime-image"
	if err := descriptor.Validate(); err == nil {
		t.Fatal("Validate expected verifier_set_ref digest rejection")
	}
}
