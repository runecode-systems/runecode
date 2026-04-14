package launcherbackend

import (
	"runtime"
	"testing"
)

func TestBackendLaunchSpecRejectsUnknownAttachmentRole(t *testing.T) {
	spec := validMicroVMSpecForContractTests()
	spec.Attachments.ByRole = map[string]AttachmentBinding{
		"not_allowed": {ReadOnly: true, ChannelKind: AttachmentChannelVirtualDisk, RequiredDigests: []string{testDigest("3")}},
	}
	if err := spec.Validate(); err == nil {
		t.Fatal("Validate expected unknown attachment role error")
	}
}

func TestBackendLaunchSpecRejectsInvalidResourceLimitsLifecycleAndCache(t *testing.T) {
	spec := validMicroVMSpecForContractTests()
	spec.ResourceLimits.VCPUCount = 0
	if err := spec.Validate(); err == nil {
		t.Fatal("Validate expected resource limit rejection")
	}

	spec = validMicroVMSpecForContractTests()
	spec.LifecyclePolicy.TerminateBetweenSteps = false
	if err := spec.Validate(); err == nil {
		t.Fatal("Validate expected lifecycle terminate_between_steps rejection")
	}

	spec = validMicroVMSpecForContractTests()
	spec.CachePosture.ReusePriorSessionIdentityKeys = true
	if err := spec.Validate(); err == nil {
		t.Fatal("Validate expected cache identity key reuse rejection")
	}
}

func TestBackendLaunchSpecRejectsRoleMultiplicityTokens(t *testing.T) {
	spec := validMicroVMSpecForContractTests()
	spec.RoleKind = "workspace-edit workspace-read"
	if err := spec.Validate(); err == nil {
		t.Fatal("Validate expected role token multiplicity error")
	}
}

func TestBackendLaunchSpecRejectsUnsupportedAccelerationForPlatform(t *testing.T) {
	spec := validMicroVMSpecForContractTests()
	spec.RequestedAccelerationKind = unsupportedAccelerationForPlatform(runtime.GOOS)
	if err := spec.Validate(); err == nil {
		t.Fatalf("Validate expected unsupported acceleration error on %s", runtime.GOOS)
	}
}

func TestBackendLaunchSpecAcceptsPlatformMVPAccelerationAndTransport(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("MVP microvm acceleration contract currently supports linux only, got %s", runtime.GOOS)
	}
	spec := validMicroVMSpecForContractTests()
	spec.RequestedAccelerationKind = AccelerationKindKVM
	spec.ControlTransportKind = TransportKindVirtioSerial
	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestContainerLaunchSpecRejectsMicroVMSpecificAccelerationAndTransportFields(t *testing.T) {
	spec := validMicroVMSpecForContractTests()
	spec.RequestedBackend = BackendKindContainer
	spec.RequestedAccelerationKind = AccelerationKindKVM
	spec.ControlTransportKind = TransportKindVSock
	spec.Image = RuntimeImageDescriptor{
		DescriptorDigest:      testDigest("7"),
		BackendKind:           BackendKindContainer,
		PlatformCompatibility: RuntimeImagePlatformCompat{OS: "linux", Architecture: "amd64"},
		BootContractVersion:   "v1",
		ComponentDigests:      map[string]string{"image": testDigest("8")},
		Signing:               &RuntimeImageSigningHooks{SignerRef: "signer:trusted-ci", SignatureDigest: testDigest("9")},
	}
	if err := spec.Validate(); err == nil {
		t.Fatal("Validate expected container rejection for non-empty acceleration/transport")
	}

	spec.RequestedAccelerationKind = ""
	if err := spec.Validate(); err == nil {
		t.Fatal("Validate expected container rejection for non-empty transport")
	}

	spec.ControlTransportKind = ""
	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate returned error after clearing microvm-only fields: %v", err)
	}
}
