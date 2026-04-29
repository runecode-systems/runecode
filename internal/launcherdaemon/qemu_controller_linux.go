//go:build linux

package launcherdaemon

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

const helloWorldToken = "RUNE_HELLO_WORLD"

type QEMUControllerConfig struct {
	QEMUBinary string
	KernelPath string
	WorkRoot   string
	Now        func() time.Time
}

type qemuController struct {
	cfg QEMUControllerConfig

	mu        sync.RWMutex
	instances map[string]*qemuInstance
}

type qemuInstance struct {
	ref       InstanceRef
	state     InstanceState
	updates   chan RuntimeUpdate
	launchDir string

	cmd    *exec.Cmd
	cancel context.CancelFunc

	helloSeen bool
	errText   string
}

func NewQEMUController(cfg QEMUControllerConfig) Controller {
	if strings.TrimSpace(cfg.QEMUBinary) == "" {
		cfg.QEMUBinary = "/usr/bin/qemu-system-x86_64"
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &qemuController{cfg: cfg, instances: map[string]*qemuInstance{}}
}

func (c *qemuController) Launch(ctx context.Context, spec launcherbackend.BackendLaunchSpec) (<-chan RuntimeUpdate, error) {
	launchState, err := c.prepareLaunchState(spec)
	if err != nil {
		return nil, err
	}
	instance := c.registerLaunchState(spec, launchState)
	go c.monitorInstance(context.Background(), instance, spec, launchState.stdout, launchState.hardening, launchState.receipt)
	return instance.updates, nil
}

func (c *qemuController) Terminate(_ context.Context, ref InstanceRef) error {
	inst := c.instanceByRef(ref)
	if inst == nil {
		return nil
	}
	c.terminateInstance(inst)
	return nil
}

func (c *qemuController) GetState(_ context.Context, ref InstanceRef) (InstanceState, error) {
	inst := c.instanceByRef(ref)
	if inst == nil {
		return InstanceState{}, nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return inst.state, nil
}

func (c *qemuController) Shutdown(_ context.Context) error {
	c.mu.RLock()
	refs := make([]InstanceRef, 0, len(c.instances))
	for _, inst := range c.instances {
		refs = append(refs, inst.ref)
	}
	c.mu.RUnlock()
	for _, ref := range refs {
		_ = c.Terminate(context.Background(), ref)
	}
	return nil
}

type preparedLaunchState struct {
	stdout    io.Reader
	launchDir string
	receipt   launcherbackend.BackendLaunchReceipt
	hardening launcherbackend.AppliedHardeningPosture
	cmd       *exec.Cmd
	cancel    context.CancelFunc
}

func (c *qemuController) prepareLaunchState(spec launcherbackend.BackendLaunchSpec) (preparedLaunchState, error) {
	if err := c.validateLaunchPrereqs(spec); err != nil {
		return preparedLaunchState{}, err
	}
	qemuPath := strings.TrimSpace(c.cfg.QEMUBinary)
	admittedImage, err := admitRuntimeImage(c.cfg.WorkRoot, spec.Image)
	if err != nil {
		return preparedLaunchState{}, err
	}
	launchDir, err := c.prepareLaunchDir(spec)
	if err != nil {
		return preparedLaunchState{}, backendError(launcherbackend.BackendErrorCodeAttachmentPlanInvalid, "failed to materialize attachments")
	}
	kernelPath := admittedImage.componentPaths["kernel"]
	initrdPath := admittedImage.componentPaths["initrd"]
	isoID, sessionID, nonce, err := makeRuntimeIdentity(spec.RunID)
	if err != nil {
		return preparedLaunchState{}, backendError(launcherbackend.BackendErrorCodeHypervisorLaunchFailed, "failed to generate runtime identity")
	}
	qemuVersion, qemuBuild := detectQEMUProvenance(qemuPath)
	cmd, stdout, cancel, err := c.startQEMUProcess(qemuPath, kernelPath, initrdPath, spec.ResourceLimits)
	if err != nil {
		return preparedLaunchState{}, err
	}
	return preparedLaunchState{
		stdout:    stdout,
		launchDir: launchDir,
		receipt:   buildLaunchReceipt(spec, isoID, sessionID, nonce, qemuVersion, qemuBuild, admittedImage.cacheEvidence),
		hardening: buildHardeningPosture(),
		cmd:       cmd,
		cancel:    cancel,
	}, nil
}

func (c *qemuController) validateLaunchPrereqs(spec launcherbackend.BackendLaunchSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(strings.ToLower(spec.RequestedBackend)) != launcherbackend.BackendKindMicroVM {
		return backendError(launcherbackend.BackendErrorCodeContainerAutomaticFallbackDisallowed, "launcher backend only supports microvm for this slice")
	}
	if os.Geteuid() == 0 {
		return backendError(launcherbackend.BackendErrorCodeRequiredHardeningUnavailable, "launcher must run unprivileged")
	}
	if _, err := os.Stat("/dev/kvm"); err != nil {
		return backendError(launcherbackend.BackendErrorCodeAccelerationUnavailable, "kvm unavailable")
	}
	if _, err := os.Stat(strings.TrimSpace(c.cfg.QEMUBinary)); err != nil {
		return backendError(launcherbackend.BackendErrorCodeAccelerationUnavailable, "qemu binary unavailable")
	}
	return nil
}

func (c *qemuController) startQEMUProcess(qemuPath, kernelPath, initrdPath string, limits launcherbackend.BackendResourceLimits) (*exec.Cmd, io.Reader, context.CancelFunc, error) {
	argv := buildQEMUArgv(qemuPath, kernelPath, initrdPath, limits)
	launchCtx, launchCancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(launchCtx, argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		launchCancel()
		return nil, nil, nil, backendError(launcherbackend.BackendErrorCodeHypervisorLaunchFailed, "failed to prepare qemu output stream")
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		launchCancel()
		if strings.Contains(strings.ToLower(err.Error()), "kvm") {
			return nil, nil, nil, backendError(launcherbackend.BackendErrorCodeAccelerationUnavailable, "kvm initialization failed")
		}
		return nil, nil, nil, backendError(launcherbackend.BackendErrorCodeHypervisorLaunchFailed, "qemu launch failed")
	}
	return cmd, stdout, launchCancel, nil
}

func (c *qemuController) registerLaunchState(spec launcherbackend.BackendLaunchSpec, launchState preparedLaunchState) *qemuInstance {
	ref := InstanceRef{RunID: spec.RunID, StageID: spec.StageID, RoleInstanceID: spec.RoleInstanceID}
	updates := make(chan RuntimeUpdate, 8)
	instance := &qemuInstance{
		ref: ref,
		state: InstanceState{
			Ref:            ref,
			Active:         true,
			LifecycleState: launcherbackend.RuntimeLifecycleState{BackendLifecycle: &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateLaunching, TerminateBetweenSteps: true, TransitionCount: 1}},
		},
		updates:   updates,
		launchDir: launchState.launchDir,
		cmd:       launchState.cmd,
		cancel:    launchState.cancel,
	}
	var existing *qemuInstance
	c.mu.Lock()
	existing = c.instances[instanceKey(ref)]
	c.instances[instanceKey(ref)] = instance
	c.mu.Unlock()
	if existing != nil {
		c.terminateInstance(existing)
	}
	updates <- RuntimeUpdate{RunID: spec.RunID, Facts: &launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launchState.receipt, HardeningPosture: launchState.hardening}}
	started := lifecycleUpdate(launcherbackend.BackendLifecycleStateStarted, launcherbackend.BackendLifecycleStateLaunching, 2, "")
	active := lifecycleUpdate(launcherbackend.BackendLifecycleStateActive, launcherbackend.BackendLifecycleStateStarted, 3, "")
	updates <- RuntimeUpdate{RunID: spec.RunID, Lifecycle: &started}
	updates <- RuntimeUpdate{RunID: spec.RunID, Lifecycle: &active}
	c.mu.Lock()
	instance.state.LifecycleState = active
	c.mu.Unlock()
	return instance
}

func (c *qemuController) terminateInstance(inst *qemuInstance) {
	inst.cancel()
	if inst.cmd == nil || inst.cmd.Process == nil {
		return
	}
	_ = inst.cmd.Process.Signal(syscall.SIGTERM)
	time.Sleep(2 * time.Second)
	_ = inst.cmd.Process.Kill()
}

func (c *qemuController) instanceByRef(ref InstanceRef) *qemuInstance {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if ref.RunID == "" && len(c.instances) == 1 {
		for _, inst := range c.instances {
			return inst
		}
	}
	return c.instances[instanceKey(ref)]
}
