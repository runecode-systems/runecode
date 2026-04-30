//go:build linux

package launcherdaemon

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type ContainerControllerConfig struct {
	WorkRoot string
}

type containerController struct {
	workRoot  string
	mu        sync.RWMutex
	instances map[string]InstanceState
}

func NewContainerController(cfg ContainerControllerConfig) Controller {
	return &containerController{workRoot: strings.TrimSpace(cfg.WorkRoot), instances: map[string]InstanceState{}}
}

func (c *containerController) Launch(_ context.Context, spec launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	hardening, err := validateContainerLaunchSpec(spec)
	if err != nil {
		return nil, err
	}
	admittedImage, err := admitRuntimeImage(c.workRoot, spec.Image)
	if err != nil {
		return nil, err
	}
	isoID, sessionID, nonce, err := makeRuntimeIdentity(spec.RunID)
	if err != nil {
		return nil, backendError(launcherbackend.BackendErrorCodeHandshakeFailed, "failed to generate runtime identity")
	}
	ref := InstanceRef{RunID: spec.RunID, StageID: spec.StageID, RoleInstanceID: spec.RoleInstanceID}
	return c.buildContainerRuntimeUpdates(ref, spec, hardening, admittedImage.admissionRecord, isoID, sessionID, nonce), nil
}

func (c *containerController) Terminate(_ context.Context, ref InstanceRef) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.instances, instanceKey(ref))
	return nil
}

func (c *containerController) GetState(_ context.Context, ref InstanceRef) (InstanceState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.instances[instanceKey(ref)], nil
}

func (c *containerController) Shutdown(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances = map[string]InstanceState{}
	return nil
}

func containerBaselineHardeningPosture() launcherbackend.AppliedHardeningPosture {
	return launcherbackend.AppliedHardeningPosture{
		Requested:                 launcherbackend.HardeningRequestedHardened,
		Effective:                 launcherbackend.HardeningEffectiveDegraded,
		DegradedReasons:           []string{"container_runtime_enforcement_not_verified_mvp_v0"},
		ExecutionIdentityPosture:  launcherbackend.HardeningExecutionIdentityUnprivileged,
		RootlessPosture:           launcherbackend.HardeningRootlessEnabled,
		FilesystemExposurePosture: launcherbackend.HardeningFilesystemExposureRestricted,
		WritableLayersPosture:     launcherbackend.HardeningWritableLayersEphemeral,
		NetworkExposurePosture:    launcherbackend.HardeningNetworkExposureNone,
		NetworkNamespacePosture:   launcherbackend.HardeningNetworkNamespacePerRole,
		NetworkDefaultPosture:     launcherbackend.HardeningNetworkDefaultNone,
		EgressEnforcementPosture:  launcherbackend.HardeningEgressEnforcementHostLevel,
		SyscallFilteringPosture:   launcherbackend.HardeningSyscallFilteringSeccomp,
		CapabilitiesPosture:       launcherbackend.HardeningCapabilitiesDropped,
		DeviceSurfacePosture:      launcherbackend.HardeningDeviceSurfaceAllowlist,
		ControlChannelKind:        launcherbackend.TransportKindNotApplicable,
		AccelerationKind:          launcherbackend.AccelerationKindNotApplicable,
		BackendEvidenceRefs:       []string{"container-hardening:mvp-v0"},
	}
}

func validateContainerLaunchSpec(spec launcherbackend.BackendLaunchSpec) (launcherbackend.AppliedHardeningPosture, error) {
	if err := spec.Validate(); err != nil {
		return launcherbackend.AppliedHardeningPosture{}, err
	}
	if strings.TrimSpace(strings.ToLower(spec.RequestedBackend)) != launcherbackend.BackendKindContainer {
		return launcherbackend.AppliedHardeningPosture{}, backendError(launcherbackend.BackendErrorCodeContainerAutomaticFallbackDisallowed, "container controller only supports explicit container backend requests")
	}
	if !strings.EqualFold(strings.TrimSpace(spec.RoleFamily), "workspace") {
		return launcherbackend.AppliedHardeningPosture{}, backendError(launcherbackend.BackendErrorCodeRequiredHardeningUnavailable, "container backend v0 only supports role_family=workspace")
	}
	if os.Geteuid() == 0 {
		return launcherbackend.AppliedHardeningPosture{}, backendError(launcherbackend.BackendErrorCodeRequiredHardeningUnavailable, "container backend requires rootless launcher execution")
	}
	hardening := containerBaselineHardeningPosture()
	if err := hardening.Validate(); err != nil {
		return launcherbackend.AppliedHardeningPosture{}, backendError(launcherbackend.BackendErrorCodeRequiredHardeningUnavailable, "container hardening baseline validation failed")
	}
	if err := enforceContainerHardeningBaseline(hardening); err != nil {
		return launcherbackend.AppliedHardeningPosture{}, backendError(launcherbackend.BackendErrorCodeRequiredHardeningUnavailable, err.Error())
	}
	return hardening, nil
}

func (c *containerController) storeLaunchedContainerInstance(ref InstanceRef) {
	c.mu.Lock()
	c.instances[instanceKey(ref)] = InstanceState{Ref: ref, Active: true, LifecycleState: launcherbackend.RuntimeLifecycleState{BackendLifecycle: &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateActive, PreviousState: launcherbackend.BackendLifecycleStateBinding, TerminateBetweenSteps: true, TransitionCount: 3}}}
	c.mu.Unlock()
}

func (c *containerController) buildContainerRuntimeUpdates(ref InstanceRef, spec launcherbackend.BackendLaunchSpec, hardening launcherbackend.AppliedHardeningPosture, admission launcherbackend.RuntimeAdmissionRecord, isolateID string, sessionID string, nonce string) <-chan RuntimeUpdate {
	updates := make(chan RuntimeUpdate, 3)
	receipt, err := containerLaunchReceipt(spec, admission, isolateID, sessionID, nonce)
	if err != nil {
		updates <- RuntimeUpdate{RunID: spec.RunID, Facts: &launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: spec.RunID, StageID: spec.StageID, RoleInstanceID: spec.RoleInstanceID, BackendKind: launcherbackend.BackendKindContainer, IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded, LaunchFailureReasonCode: launcherbackend.BackendErrorCodeHandshakeFailed}, HardeningPosture: hardening}}
		close(updates)
		return updates
	}
	c.storeLaunchedContainerInstance(ref)
	facts := launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: receipt, HardeningPosture: hardening}
	updates <- RuntimeUpdate{RunID: spec.RunID, Facts: &facts}
	started := lifecycleUpdate(launcherbackend.BackendLifecycleStateStarted, launcherbackend.BackendLifecycleStateLaunching, 2, "")
	active := lifecycleUpdate(launcherbackend.BackendLifecycleStateActive, launcherbackend.BackendLifecycleStateStarted, 3, "")
	updates <- RuntimeUpdate{RunID: spec.RunID, Lifecycle: &started}
	updates <- RuntimeUpdate{RunID: spec.RunID, Lifecycle: &active}
	close(updates)
	return updates
}

func containerLaunchReceipt(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord, isolateID string, sessionID string, nonce string) (launcherbackend.BackendLaunchReceipt, error) {
	receipt := launcherbackend.BackendLaunchReceipt{
		RunID:                            spec.RunID,
		StageID:                          spec.StageID,
		RoleInstanceID:                   spec.RoleInstanceID,
		RoleFamily:                       spec.RoleFamily,
		RoleKind:                         spec.RoleKind,
		BackendKind:                      launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel:          launcherbackend.IsolationAssuranceDegraded,
		ProvisioningPosture:              launcherbackend.ProvisioningPostureAttested,
		HypervisorImplementation:         launcherbackend.HypervisorImplementationNotApplicable,
		AccelerationKind:                 launcherbackend.AccelerationKindNotApplicable,
		TransportKind:                    launcherbackend.TransportKindNotApplicable,
		RuntimeImageDescriptorDigest:     admission.DescriptorDigest,
		RuntimeImageBootProfile:          admission.BootContractVersion,
		RuntimeImageSignerRef:            admission.RuntimeImageSignerRef,
		RuntimeImageVerifierRef:          admission.RuntimeImageVerifierSetRef,
		RuntimeImageSignatureDigest:      admission.RuntimeImageSignatureDigest,
		RuntimeToolchainDescriptorDigest: admission.RuntimeToolchainDescriptorDigest,
		RuntimeToolchainSignerRef:        admission.RuntimeToolchainSignerRef,
		RuntimeToolchainVerifierRef:      admission.RuntimeToolchainVerifierSetRef,
		RuntimeToolchainSignatureDigest:  admission.RuntimeToolchainSignatureDigest,
		AuthorityStateDigest:             admission.AuthorityStateDigest,
		AuthorityStateRevision:           admission.AuthorityStateRevision,
		BootComponentDigestByName:        cloneMap(admission.ComponentDigests),
		BootComponentDigests:             componentDigestValues(admission.ComponentDigests),
		AttachmentPlanSummary:            summarizeAttachments(spec.Attachments),
		WorkspaceEncryptionPosture:       containerWorkspaceEncryptionPosture(),
		Lifecycle:                        &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateLaunching, TerminateBetweenSteps: true, TransitionCount: 1},
	}
	populateRuntimeSessionBinding(&receipt, spec, admission.DescriptorDigest, isolateID, sessionID, nonce)
	if err := applyTrustedRuntimeAttestation(&receipt, admission, time.Now()); err != nil {
		return launcherbackend.BackendLaunchReceipt{}, err
	}
	return receipt, nil
}

func containerWorkspaceEncryptionPosture() *launcherbackend.WorkspaceEncryptionPosture {
	return &launcherbackend.WorkspaceEncryptionPosture{
		Required:             true,
		AtRestProtection:     launcherbackend.WorkspaceAtRestProtectionHostManagedEncryption,
		KeyProtectionPosture: launcherbackend.WorkspaceKeyProtectionOSKeystore,
		Effective:            true,
	}
}

func enforceContainerHardeningBaseline(posture launcherbackend.AppliedHardeningPosture) error {
	required := []struct {
		ok  bool
		msg string
	}{
		{posture.ExecutionIdentityPosture == launcherbackend.HardeningExecutionIdentityUnprivileged, "rootless execution identity is required"},
		{posture.RootlessPosture == launcherbackend.HardeningRootlessEnabled, "rootless posture must be enabled"},
		{posture.SyscallFilteringPosture == launcherbackend.HardeningSyscallFilteringSeccomp, "seccomp syscall filtering is required"},
		{posture.CapabilitiesPosture == launcherbackend.HardeningCapabilitiesDropped, "linux capabilities must be dropped"},
		{posture.FilesystemExposurePosture == launcherbackend.HardeningFilesystemExposureRestricted, "read-only root filesystem posture is required"},
		{posture.WritableLayersPosture == launcherbackend.HardeningWritableLayersEphemeral, "ephemeral writable layers posture is required"},
		{posture.NetworkNamespacePosture == launcherbackend.HardeningNetworkNamespacePerRole, "per-role network namespace is required"},
		{posture.NetworkDefaultPosture == launcherbackend.HardeningNetworkDefaultNone || posture.NetworkDefaultPosture == launcherbackend.HardeningNetworkDefaultLoopbackOnly, "deny-by-default network posture is required"},
		{posture.NetworkExposurePosture == launcherbackend.HardeningNetworkExposureNone, "workspace-role container v0 must deny network exposure by default"},
		{posture.EgressEnforcementPosture == launcherbackend.HardeningEgressEnforcementHostLevel, "host-level egress enforcement posture is required"},
	}
	for _, check := range required {
		if !check.ok {
			return fmt.Errorf("%s", check.msg)
		}
	}
	return nil
}
