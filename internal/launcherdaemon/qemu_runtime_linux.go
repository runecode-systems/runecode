//go:build linux

package launcherdaemon

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (c *qemuController) monitorInstance(parent context.Context, inst *qemuInstance, launchState preparedLaunchState) {
	defer removeLaunchDir(inst.launchDir)
	currentReceipt := launchState.receipt
	var currentPostHandshake *launcherbackend.PostHandshakeRuntimeAttestationInput
	runtimeMaterial, helloSeen, waitErr := waitForHelloAndRuntimeMaterial(parent, inst, launchState)
	if waitErr != nil {
		c.finishQEMUInstanceWithTerminal(inst, launchState, currentReceipt, currentPostHandshake, helloSeen)
		return
	}
	launchState.material = mergeQEMURuntimePostHandshakeMaterial(launchState.material, runtimeMaterial)
	postHandshakeFailed := false
	if helloSeen {
		if update, err := c.runtimePostHandshakeUpdate(launchState); err == nil {
			if update.Facts != nil {
				currentReceipt = update.Facts.LaunchReceipt
				currentPostHandshake = update.Facts.PostHandshakeAttestationInput
			}
			inst.updates <- update
			active := lifecycleUpdate(launcherbackend.BackendLifecycleStateActive, launcherbackend.BackendLifecycleStateBinding, 4, "")
			inst.updates <- RuntimeUpdate{RunID: launchState.spec.RunID, Lifecycle: &active}
			c.mu.Lock()
			inst.state.LifecycleState = active
			c.mu.Unlock()
		} else {
			postHandshakeFailed = true
			inst.errText = launcherbackend.BackendErrorCodeHandshakeFailed
			_ = inst.cmd.Process.Kill()
		}
	}
	_ = inst.cmd.Wait()
	c.finishQEMUInstanceWithTerminal(inst, launchState, currentReceipt, currentPostHandshake, helloSeen && !postHandshakeFailed)
}

func (c *qemuController) finishQEMUInstanceWithTerminal(inst *qemuInstance, launchState preparedLaunchState, currentReceipt launcherbackend.BackendLaunchReceipt, currentPostHandshake *launcherbackend.PostHandshakeRuntimeAttestationInput, helloSeen bool) {
	term := buildTerminalReport(launchState.spec, currentReceipt, helloSeen, inst.errText)
	facts := launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: currentReceipt, PostHandshakeAttestationInput: currentPostHandshake, HardeningPosture: launchState.hardening, TerminalReport: &term}
	inst.updates <- RuntimeUpdate{RunID: launchState.spec.RunID, Facts: &facts}
	terminating, terminated := terminalLifecycleUpdates(term)
	inst.updates <- RuntimeUpdate{RunID: launchState.spec.RunID, Lifecycle: &terminating}
	inst.updates <- RuntimeUpdate{RunID: launchState.spec.RunID, Lifecycle: &terminated}
	c.finishInstance(inst, terminated, term.FailureReasonCode)
	close(inst.updates)

	c.mu.Lock()
	if current := c.instances[instanceKey(inst.ref)]; current == inst {
		delete(c.instances, instanceKey(inst.ref))
	}
	c.mu.Unlock()
}

func removeLaunchDir(dir string) {
	if dir == "" {
		return
	}
	_ = os.RemoveAll(dir)
}

func (c *qemuController) finishInstance(inst *qemuInstance, terminated launcherbackend.RuntimeLifecycleState, failureReason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	inst.state.Active = false
	inst.state.LifecycleState = terminated
	inst.state.LastError = failureReason
}

func waitForHelloAndRuntimeMaterial(parent context.Context, inst *qemuInstance, launchState preparedLaunchState) (*launcherbackend.RuntimePostHandshakeMaterial, bool, error) {
	timer := time.NewTimer(activeTimeout(launchState.spec.ResourceLimits))
	defer timer.Stop()
	lineCh := make(chan string, 16)
	errCh := make(chan error, 1)
	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(launchState.stdout)
		const maxFrame = 1024 * 1024
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, maxFrame)
		for scanner.Scan() {
			lineCh <- scanner.Text()
		}
		errCh <- scanner.Err()
	}()
	var runtimeMaterial *launcherbackend.RuntimePostHandshakeMaterial
	for {
		select {
		case <-parent.Done():
			return qemuWaitCancelled(inst, runtimeMaterial, parent.Err())
		case <-timer.C:
			return qemuWaitTimedOut(inst, runtimeMaterial)
		case line, ok := <-lineCh:
			updatedMaterial, done, err := handleQEMURuntimeOutputLine(inst, runtimeMaterial, line, ok, errCh)
			if err != nil || done {
				return updatedMaterial, inst.helloSeen, err
			}
			runtimeMaterial = updatedMaterial
		}
	}
}

func qemuWaitCancelled(inst *qemuInstance, runtimeMaterial *launcherbackend.RuntimePostHandshakeMaterial, err error) (*launcherbackend.RuntimePostHandshakeMaterial, bool, error) {
	_ = inst.cmd.Process.Kill()
	inst.errText = launcherbackend.BackendErrorCodeHandshakeFailed
	return runtimeMaterial, inst.helloSeen, err
}

func qemuWaitTimedOut(inst *qemuInstance, runtimeMaterial *launcherbackend.RuntimePostHandshakeMaterial) (*launcherbackend.RuntimePostHandshakeMaterial, bool, error) {
	_ = inst.cmd.Process.Kill()
	inst.errText = launcherbackend.BackendErrorCodeWatchdogTimeout
	return runtimeMaterial, inst.helloSeen, fmt.Errorf("watchdog timeout waiting for runtime material")
}

func parseQEMURuntimeMaterialUpdate(line string) (*launcherbackend.RuntimePostHandshakeMaterial, bool, error) {
	material, err := parseQEMURuntimeMaterialLine(line)
	if err != nil {
		return nil, false, err
	}
	return material, material != nil, nil
}

func handleQEMURuntimeOutputLine(inst *qemuInstance, runtimeMaterial *launcherbackend.RuntimePostHandshakeMaterial, line string, ok bool, errCh <-chan error) (*launcherbackend.RuntimePostHandshakeMaterial, bool, error) {
	if !ok {
		return runtimeMaterial, true, <-errCh
	}
	material, handled, err := parseQEMURuntimeMaterialUpdate(line)
	if err != nil {
		return nil, true, err
	}
	if handled {
		return material, false, nil
	}
	if recordHelloLine(inst, line) {
		return runtimeMaterial, false, nil
	}
	return runtimeMaterial, false, nil
}

func activeTimeout(limits launcherbackend.BackendResourceLimits) time.Duration {
	timeout := time.Duration(limits.ActiveTimeoutSeconds) * time.Second
	if timeout <= 0 {
		return 20 * time.Second
	}
	return timeout
}

func recordHelloLine(inst *qemuInstance, line string) bool {
	if !strings.Contains(line, helloWorldToken) {
		return false
	}
	// caller synchronizes around lifecycle updates; this flag is per-instance
	if inst.helloSeen {
		return true
	}
	inst.helloSeen = true
	inst.state.HelloWorldSeen = true
	return true
}

func (c *qemuController) runtimePostHandshakeUpdate(launchState preparedLaunchState) (RuntimeUpdate, error) {
	runtimeMaterial, err := c.runtimePostHandshakeMaterialForQEMU(launchState)
	if err != nil {
		return RuntimeUpdate{}, err
	}
	if runtimeMaterial == nil || runtimeMaterial.SecureSession == nil {
		return RuntimeUpdate{}, backendError(launcherbackend.BackendErrorCodeHandshakeFailed, "runtime secure-session material not provided")
	}
	summary, launchContextDigest, err := validateSecureSessionAndBuildSummary(launchState.receipt, runtimeMaterial.SecureSession)
	if err != nil {
		return RuntimeUpdate{}, err
	}
	receipt := launchState.receipt
	if err := recordValidatedSecureSession(&receipt, summary, launchContextDigest); err != nil {
		return RuntimeUpdate{}, err
	}
	postHandshake, err := buildPostHandshakeAttestationProgressFromMaterial(receipt, launchState.admission, runtimeMaterial)
	if err != nil {
		return RuntimeUpdate{}, err
	}
	if err := recordPostHandshakeAttestationProgress(&receipt, postHandshake); err != nil {
		return RuntimeUpdate{}, err
	}
	return RuntimeUpdate{RunID: launchState.spec.RunID, Facts: &launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: receipt, PostHandshakeAttestationInput: postHandshake, HardeningPosture: launchState.hardening}}, nil
}

func (c *qemuController) runtimePostHandshakeMaterialForQEMU(launchState preparedLaunchState) (*launcherbackend.RuntimePostHandshakeMaterial, error) {
	if launchState.material == nil {
		return nil, nil
	}
	return launchState.material, nil
}

func buildTerminalReport(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt, helloSeen bool, errText string) launcherbackend.BackendTerminalReport {
	report := launcherbackend.BackendTerminalReport{
		RunID:           spec.RunID,
		StageID:         spec.StageID,
		RoleInstanceID:  spec.RoleInstanceID,
		IsolateID:       receipt.IsolateID,
		SessionID:       receipt.SessionID,
		FailClosed:      true,
		FallbackPosture: launcherbackend.BackendFallbackPostureNoAutomaticFallback,
		TerminatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if strings.TrimSpace(errText) == "" && helloSeen {
		report.TerminationKind = launcherbackend.BackendTerminationKindCompleted
		return report
	}
	report.TerminationKind = launcherbackend.BackendTerminationKindFailed
	if errText == launcherbackend.BackendErrorCodeWatchdogTimeout {
		report.FailureReasonCode = launcherbackend.BackendErrorCodeWatchdogTimeout
		return report
	}
	if errText == launcherbackend.BackendErrorCodeHandshakeFailed {
		report.FailureReasonCode = launcherbackend.BackendErrorCodeHandshakeFailed
		return report
	}
	report.FailureReasonCode = launcherbackend.BackendErrorCodeHypervisorLaunchFailed
	return report
}

func terminalLifecycleUpdates(term launcherbackend.BackendTerminalReport) (launcherbackend.RuntimeLifecycleState, launcherbackend.RuntimeLifecycleState) {
	terminating := lifecycleUpdate(launcherbackend.BackendLifecycleStateTerminating, launcherbackend.BackendLifecycleStateActive, 4, term.FailureReasonCode)
	terminated := lifecycleUpdate(launcherbackend.BackendLifecycleStateTerminated, launcherbackend.BackendLifecycleStateTerminating, 5, term.FailureReasonCode)
	return terminating, terminated
}

func lifecycleUpdate(current, previous string, count int, failure string) launcherbackend.RuntimeLifecycleState {
	return launcherbackend.RuntimeLifecycleState{
		BackendLifecycle:        &launcherbackend.BackendLifecycleSnapshot{CurrentState: current, PreviousState: previous, TerminateBetweenSteps: true, TransitionCount: count},
		LaunchFailureReasonCode: failure,
	}
}
