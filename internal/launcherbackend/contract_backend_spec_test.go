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
