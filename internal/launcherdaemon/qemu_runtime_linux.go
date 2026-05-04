//go:build linux

package launcherdaemon

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func (c *qemuController) monitorInstance(parent context.Context, inst *qemuInstance, launchState preparedLaunchState) {
	defer removeLaunchDir(inst.launchDir)
	scanStop := make(chan struct{})
	defer close(scanStop)
	currentReceipt := launchState.receipt
	var currentPostHandshake *launcherbackend.PostHandshakeRuntimeAttestationInput
	lineCh, scanDone := scanQEMUOutput(launchState.stdout, scanStop)
	helloSeen := c.waitForHelloOrExit(parent, inst, launchState, lineCh, scanDone)
	postHandshakeFailed := false
	if helloSeen {
		if update, err := runtimePostHandshakeUpdate(launchState); err == nil {
			if update.Facts != nil {
				currentReceipt = update.Facts.LaunchReceipt
				currentPostHandshake = update.Facts.PostHandshakeAttestationInput
			}
			inst.updates <- update
		} else {
			postHandshakeFailed = true
			inst.errText = launcherbackend.BackendErrorCodeHandshakeFailed
			_ = inst.cmd.Process.Kill()
		}
	}
	_ = inst.cmd.Wait()
	term := buildTerminalReport(launchState.spec, currentReceipt, helloSeen && !postHandshakeFailed, inst.errText)
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

func scanQEMUOutput(out io.Reader, stop <-chan struct{}) (<-chan string, <-chan struct{}) {
	lineCh := make(chan string, 16)
	scanDone := make(chan struct{})
	go func() {
		defer close(lineCh)
		defer close(scanDone)
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case lineCh <- line:
			case <-stop:
				return
			}
		}
	}()
	return lineCh, scanDone
}

func (c *qemuController) waitForHelloOrExit(parent context.Context, inst *qemuInstance, launchState preparedLaunchState, lineCh <-chan string, scanDone <-chan struct{}) bool {
	timer := time.NewTimer(activeTimeout(launchState.spec.ResourceLimits))
	defer timer.Stop()
	for {
		select {
		case <-parent.Done():
			_ = inst.cmd.Process.Kill()
			return inst.helloSeen
		case <-timer.C:
			_ = inst.cmd.Process.Kill()
			inst.errText = launcherbackend.BackendErrorCodeWatchdogTimeout
			return inst.helloSeen
		case line, ok := <-lineCh:
			if !ok {
				return inst.helloSeen
			}
			if c.recordHelloLine(inst, launchState, line) {
				continue
			}
		case <-scanDone:
			return inst.helloSeen
		}
	}
}

func activeTimeout(limits launcherbackend.BackendResourceLimits) time.Duration {
	timeout := time.Duration(limits.ActiveTimeoutSeconds) * time.Second
	if timeout <= 0 {
		return 20 * time.Second
	}
	return timeout
}

func (c *qemuController) recordHelloLine(inst *qemuInstance, launchState preparedLaunchState, line string) bool {
	if !strings.Contains(line, helloWorldToken) {
		return false
	}
	c.mu.Lock()
	if inst.helloSeen {
		c.mu.Unlock()
		return true
	}
	inst.helloSeen = true
	inst.state.HelloWorldSeen = true
	c.mu.Unlock()
	return true
}

func runtimePostHandshakeUpdate(launchState preparedLaunchState) (RuntimeUpdate, error) {
	summary, launchContextDigest, err := validateSecureSessionAndBuildSummary(launchState.spec, launchState.receipt)
	if err != nil {
		return RuntimeUpdate{}, err
	}
	receipt := launchState.receipt
	if err := recordValidatedSecureSession(&receipt, summary, launchContextDigest); err != nil {
		return RuntimeUpdate{}, err
	}
	postHandshake, err := buildPostHandshakeAttestationProgress(receipt, launchState.admission)
	if err != nil {
		return RuntimeUpdate{}, err
	}
	if err := recordPostHandshakeAttestationProgress(&receipt, postHandshake); err != nil {
		return RuntimeUpdate{}, err
	}
	return RuntimeUpdate{RunID: launchState.spec.RunID, Facts: &launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: receipt, PostHandshakeAttestationInput: postHandshake, HardeningPosture: launchState.hardening}}, nil
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
	if helloSeen {
		report.TerminationKind = launcherbackend.BackendTerminationKindCompleted
		return report
	}
	report.TerminationKind = launcherbackend.BackendTerminationKindFailed
	if errText == launcherbackend.BackendErrorCodeWatchdogTimeout {
		report.FailureReasonCode = launcherbackend.BackendErrorCodeWatchdogTimeout
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
