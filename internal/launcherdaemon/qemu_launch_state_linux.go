//go:build linux

package launcherdaemon

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type launchStateStdout = io.Reader
type launchStateCmd = *exec.Cmd

func (c *qemuController) prepareLaunchState(ctx context.Context, spec launcherbackend.BackendLaunchSpec) (preparedLaunchState, error) {
	if err := ctx.Err(); err != nil {
		return preparedLaunchState{}, err
	}
	if err := c.validateLaunchPrereqs(spec); err != nil {
		return preparedLaunchState{}, err
	}
	qemuPath := strings.TrimSpace(c.cfg.QEMUBinary)
	admittedImage, launchDir, kernelPath, initrdPath, err := c.prepareLaunchAssets(ctx, qemuPath, spec)
	if err != nil {
		return preparedLaunchState{}, err
	}
	receipt, err := c.prepareLaunchReceipt(spec, admittedImage.admissionRecord, qemuPath, admittedImage.cacheEvidence)
	if err != nil {
		return preparedLaunchState{}, err
	}
	guestMaterialArg, err := c.prepareQEMUGuestMaterialArg(spec, receipt)
	if err != nil {
		return preparedLaunchState{}, err
	}
	cmd, stdout, cancel, err := c.startQEMUProcess(ctx, qemuPath, kernelPath, initrdPath, spec.ResourceLimits, guestMaterialArg)
	if err != nil {
		return preparedLaunchState{}, err
	}
	return preparedLaunchState{
		stdout:    stdout,
		launchDir: launchDir,
		receipt:   receipt,
		hardening: buildHardeningPosture(),
		admission: admittedImage.admissionRecord,
		material:  nil,
		spec:      spec,
		cmd:       cmd,
		cancel:    cancel,
	}, nil
}

func (c *qemuController) prepareLaunchReceipt(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord, qemuPath string, cacheEvidence *launcherbackend.BackendCacheEvidence) (launcherbackend.BackendLaunchReceipt, error) {
	isoID, sessionID, nonce, err := makeRuntimeIdentity(spec.RunID)
	if err != nil {
		return launcherbackend.BackendLaunchReceipt{}, backendError(launcherbackend.BackendErrorCodeHypervisorLaunchFailed, "failed to generate runtime identity")
	}
	qemuVersion, qemuBuild := detectQEMUProvenance(qemuPath)
	receipt, err := buildLaunchReceipt(spec, admission, isoID, sessionID, nonce, qemuVersion, qemuBuild, cacheEvidence)
	if err != nil {
		return launcherbackend.BackendLaunchReceipt{}, backendError(launcherbackend.BackendErrorCodeHandshakeFailed, err.Error())
	}
	return receipt, nil
}

func (c *qemuController) prepareQEMUGuestMaterialArg(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt) (string, error) {
	seed, err := c.runtimePostHandshakeSeed(spec, receipt)
	if err != nil {
		return "", err
	}
	runtimeMaterialLine, err := encodeQEMURuntimePostHandshakeMaterialLine(seed)
	if err != nil {
		return "", backendError(launcherbackend.BackendErrorCodeHandshakeFailed, err.Error())
	}
	return qemuGuestRuntimeMaterialKernelArg(runtimeMaterialLine), nil
}

func (c *qemuController) runtimePostHandshakeSeed(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt) (*launcherbackend.RuntimePostHandshakeMaterial, error) {
	if c.cfg.RuntimePostHandshakeMaterialProvider == nil {
		return nil, nil
	}
	seed, err := c.cfg.RuntimePostHandshakeMaterialProvider(spec, receipt)
	if err != nil {
		return nil, backendError(launcherbackend.BackendErrorCodeHandshakeFailed, err.Error())
	}
	return seed, nil
}

func (c *qemuController) prepareLaunchAssets(ctx context.Context, qemuPath string, spec launcherbackend.BackendLaunchSpec) (admittedRuntimeImage, string, string, string, error) {
	admittedImage, err := admitRuntimeImage(c.cfg.WorkRoot, spec.Image)
	if err != nil {
		return admittedRuntimeImage{}, "", "", "", err
	}
	launchDir, err := c.prepareLaunchDir(spec)
	if err != nil {
		return admittedRuntimeImage{}, "", "", "", backendError(launcherbackend.BackendErrorCodeAttachmentPlanInvalid, "failed to materialize attachments")
	}
	keepLaunchDir := false
	defer func() {
		if !keepLaunchDir {
			_ = os.RemoveAll(launchDir)
		}
	}()
	if err := ctx.Err(); err != nil {
		return admittedRuntimeImage{}, "", "", "", err
	}
	if err := verifyRuntimeToolchainArtifact(qemuPath, admittedImage.toolchain); err != nil {
		return admittedRuntimeImage{}, "", "", "", backendError(launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch, err.Error())
	}
	keepLaunchDir = true
	return admittedImage, launchDir, admittedImage.componentPaths["kernel"], admittedImage.componentPaths["initrd"], nil
}

func (c *qemuController) startQEMUProcess(ctx context.Context, qemuPath, kernelPath, initrdPath string, limits launcherbackend.BackendResourceLimits, guestMaterialArg string) (*exec.Cmd, io.Reader, context.CancelFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	argv := buildQEMUArgv(qemuPath, kernelPath, initrdPath, limits, guestMaterialArg)
	cmd := exec.Command(argv[0], argv[1:]...)
	launchCancel := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
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
	if err := ctx.Err(); err != nil {
		launchCancel()
		_ = cmd.Process.Kill()
		return nil, nil, nil, err
	}
	return cmd, stdout, launchCancel, nil
}
