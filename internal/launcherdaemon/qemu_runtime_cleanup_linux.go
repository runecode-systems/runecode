//go:build linux

package launcherdaemon

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func qemuKillAndReap(inst *qemuInstance) {
	if inst == nil || inst.cmd == nil {
		return
	}
	if inst.cmd.Process != nil {
		_ = inst.cmd.Process.Kill()
	}
	_ = inst.cmd.Wait()
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
