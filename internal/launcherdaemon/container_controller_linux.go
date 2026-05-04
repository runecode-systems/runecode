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
	WorkRoot                             string
	Now                                  func() time.Time
	RuntimePostHandshakeMaterialProvider func(launcherbackend.BackendLaunchSpec, launcherbackend.BackendLaunchReceipt) (*launcherbackend.RuntimePostHandshakeMaterial, error)
}

type containerController struct {
	workRoot                             string
	now                                  func() time.Time
	runtimePostHandshakeMaterialProvider func(launcherbackend.BackendLaunchSpec, launcherbackend.BackendLaunchReceipt) (*launcherbackend.RuntimePostHandshakeMaterial, error)
	mu                                   sync.RWMutex
	instances                            map[string]InstanceState
}

func NewContainerController(cfg ContainerControllerConfig) Controller {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &containerController{
		workRoot:                             strings.TrimSpace(cfg.WorkRoot),
		now:                                  cfg.Now,
		runtimePostHandshakeMaterialProvider: cfg.RuntimePostHandshakeMaterialProvider,
		instances:                            map[string]InstanceState{},
	}
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

func (c *containerController) storeContainerInstanceState(ref InstanceRef, state launcherbackend.RuntimeLifecycleState, active bool, lastErr string) {
	c.mu.Lock()
	c.instances[instanceKey(ref)] = InstanceState{Ref: ref, Active: active, LifecycleState: state, LastError: lastErr}
	c.mu.Unlock()
}

func (c *containerController) buildContainerRuntimeUpdates(ref InstanceRef, spec launcherbackend.BackendLaunchSpec, hardening launcherbackend.AppliedHardeningPosture, admission launcherbackend.RuntimeAdmissionRecord, isolateID string, sessionID string, nonce string) <-chan RuntimeUpdate {
	updates := make(chan RuntimeUpdate, 8)
	receipt, err := containerLaunchReceipt(spec, admission, isolateID, sessionID, nonce)
	if err != nil {
		updates <- RuntimeUpdate{RunID: spec.RunID, Facts: &launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{RunID: spec.RunID, StageID: spec.StageID, RoleInstanceID: spec.RoleInstanceID, BackendKind: launcherbackend.BackendKindContainer, IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded, LaunchFailureReasonCode: launcherbackend.BackendErrorCodeHandshakeFailed}, HardeningPosture: hardening}}
		close(updates)
		return updates
	}
	c.emitContainerLaunchProgress(ref, spec.RunID, receipt, hardening, updates)
	material, err := c.containerRuntimePostHandshakeMaterial(spec, receipt)
	if err != nil {
		c.emitContainerHandshakeFailure(ref, spec.RunID, updates)
		close(updates)
		return updates
	}
	postHandshake, err := runtimePostHandshakeFactsUpdate(spec.RunID, receipt, admission, hardening, material)
	if err != nil {
		c.emitContainerHandshakeFailure(ref, spec.RunID, updates)
		close(updates)
		return updates
	}
	updates <- postHandshake
	c.emitContainerActive(ref, spec.RunID, updates)
	close(updates)
	return updates
}

func (c *containerController) emitContainerLaunchProgress(ref InstanceRef, runID string, receipt launcherbackend.BackendLaunchReceipt, hardening launcherbackend.AppliedHardeningPosture, updates chan<- RuntimeUpdate) {
	launching := lifecycleUpdate(launcherbackend.BackendLifecycleStateLaunching, launcherbackend.BackendLifecycleStatePlanned, 1, "")
	c.storeContainerInstanceState(ref, launching, true, "")
	facts := launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: receipt, HardeningPosture: hardening}
	updates <- RuntimeUpdate{RunID: runID, Facts: &facts}
	started := lifecycleUpdate(launcherbackend.BackendLifecycleStateStarted, launcherbackend.BackendLifecycleStateLaunching, 2, "")
	binding := lifecycleUpdate(launcherbackend.BackendLifecycleStateBinding, launcherbackend.BackendLifecycleStateStarted, 3, "")
	c.storeContainerInstanceState(ref, started, true, "")
	updates <- RuntimeUpdate{RunID: runID, Lifecycle: &started}
	c.storeContainerInstanceState(ref, binding, true, "")
	updates <- RuntimeUpdate{RunID: runID, Lifecycle: &binding}
}

func (c *containerController) containerRuntimePostHandshakeMaterial(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt) (*launcherbackend.RuntimePostHandshakeMaterial, error) {
	if c.runtimePostHandshakeMaterialProvider == nil {
		return nil, fmt.Errorf("runtime post-handshake material not provided")
	}
	return c.runtimePostHandshakeMaterialProvider(spec, receipt)
}

func (c *containerController) emitContainerHandshakeFailure(ref InstanceRef, runID string, updates chan<- RuntimeUpdate) {
	terminating := lifecycleUpdate(launcherbackend.BackendLifecycleStateTerminating, launcherbackend.BackendLifecycleStateBinding, 4, launcherbackend.BackendErrorCodeHandshakeFailed)
	terminated := lifecycleUpdate(launcherbackend.BackendLifecycleStateTerminated, launcherbackend.BackendLifecycleStateTerminating, 5, launcherbackend.BackendErrorCodeHandshakeFailed)
	c.storeContainerInstanceState(ref, terminating, false, launcherbackend.BackendErrorCodeHandshakeFailed)
	updates <- RuntimeUpdate{RunID: runID, Lifecycle: &terminating}
	c.storeContainerInstanceState(ref, terminated, false, launcherbackend.BackendErrorCodeHandshakeFailed)
	updates <- RuntimeUpdate{RunID: runID, Lifecycle: &terminated}
}

func (c *containerController) emitContainerActive(ref InstanceRef, runID string, updates chan<- RuntimeUpdate) {
	active := lifecycleUpdate(launcherbackend.BackendLifecycleStateActive, launcherbackend.BackendLifecycleStateBinding, 4, "")
	c.storeContainerInstanceState(ref, active, true, "")
	updates <- RuntimeUpdate{RunID: runID, Lifecycle: &active}
}

func containerLaunchReceipt(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord, isolateID string, sessionID string, nonce string) (launcherbackend.BackendLaunchReceipt, error) {
	sessionBinding, err := deriveRuntimeSessionBinding(spec, admission.DescriptorDigest, isolateID, sessionID, nonce)
	if err != nil {
		return launcherbackend.BackendLaunchReceipt{}, err
	}
	receipt := launcherbackend.BackendLaunchReceipt{
		RunID:                            spec.RunID,
		StageID:                          spec.StageID,
		RoleInstanceID:                   spec.RoleInstanceID,
		RoleFamily:                       spec.RoleFamily,
		RoleKind:                         spec.RoleKind,
		BackendKind:                      launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel:          launcherbackend.IsolationAssuranceDegraded,
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
	populateRuntimeSessionBinding(&receipt, sessionBinding)
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
