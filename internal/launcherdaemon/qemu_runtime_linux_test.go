//go:build linux

package launcherdaemon

import (
	"context"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestQEMUMonitorInstanceKillsAndReapsOnRuntimeMaterialParseFailure(t *testing.T) {
	controller := &qemuController{cfg: QEMUControllerConfig{}, instances: map[string]*qemuInstance{}}
	spec := validSpecForTests()
	ref := InstanceRef{RunID: spec.RunID, StageID: spec.StageID, RoleInstanceID: spec.RoleInstanceID}
	cmd, stdout := qemuParseFailureCommand(t)
	inst := registerQEMUTestInstance(controller, ref, cmd)
	waitForQEMUMonitor(t, controller, inst, preparedLaunchState{
		stdout:  stdout,
		receipt: launcherbackend.BackendLaunchReceipt{RunID: spec.RunID, IsolateID: "iso-1", SessionID: "session-1"},
		spec:    spec,
		cmd:     cmd,
	})
	assertQEMUParseFailureCleanup(t, controller, inst, ref, cmd)
}

func TestQEMURecordHelloLineSynchronizesStateMutation(t *testing.T) {
	controller := &qemuController{cfg: QEMUControllerConfig{}, instances: map[string]*qemuInstance{}}
	ref := InstanceRef{RunID: "run-1", StageID: "stage-1", RoleInstanceID: "role-1"}
	inst := &qemuInstance{ref: ref, state: InstanceState{Ref: ref, Active: true}}
	controller.mu.Lock()
	controller.instances[instanceKey(ref)] = inst
	controller.mu.Unlock()

	const workers = 32
	const iterations = 64
	runConcurrentHelloStateWork(t, controller, inst, ref, workers, iterations)
	assertHelloStateVisible(t, controller, inst, ref)
}

func TestRecordHelloLineRejectsSubstringMatch(t *testing.T) {
	controller := &qemuController{cfg: QEMUControllerConfig{}, instances: map[string]*qemuInstance{}}
	inst := &qemuInstance{state: InstanceState{Active: true}}
	if recordHelloLine(controller, inst, "boot "+helloWorldToken) {
		t.Fatal("recordHelloLine accepted substring match, want exact hello line only")
	}
}

func qemuParseFailureCommand(t *testing.T) (*exec.Cmd, launchStateStdout) {
	t.Helper()
	cmd := exec.Command("sh", "-c", "printf 'RUNE_POST_HANDSHAKE_MATERIAL=!!!\\n'; sleep 30")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe returned error: %v", err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	return cmd, stdout
}

func registerQEMUTestInstance(controller *qemuController, ref InstanceRef, cmd *exec.Cmd) *qemuInstance {
	inst := &qemuInstance{ref: ref, state: InstanceState{Ref: ref, Active: true}, cmd: cmd, updates: make(chan RuntimeUpdate, 16)}
	controller.mu.Lock()
	controller.instances[instanceKey(ref)] = inst
	controller.mu.Unlock()
	return inst
}

func waitForQEMUMonitor(t *testing.T, controller *qemuController, inst *qemuInstance, launchState preparedLaunchState) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		controller.monitorInstance(context.Background(), inst, launchState)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("monitorInstance did not return after parse failure")
	}
}

func assertQEMUParseFailureCleanup(t *testing.T, controller *qemuController, inst *qemuInstance, ref InstanceRef, cmd *exec.Cmd) {
	t.Helper()
	if cmd.ProcessState == nil {
		t.Fatal("process state is nil, want reaped process after parse failure")
	}
	lastLifecycle := lastQEMULifecycleUpdate(inst.updates)
	if lastLifecycle == nil || lastLifecycle.BackendLifecycle == nil {
		t.Fatal("missing terminal lifecycle update")
	}
	if got, want := lastLifecycle.BackendLifecycle.CurrentState, launcherbackend.BackendLifecycleStateTerminated; got != want {
		t.Fatalf("terminal lifecycle current state = %q, want %q", got, want)
	}
	controller.mu.RLock()
	_, stillTracked := controller.instances[instanceKey(ref)]
	controller.mu.RUnlock()
	if stillTracked {
		t.Fatal("instance still tracked after terminal cleanup")
	}
}

func lastQEMULifecycleUpdate(updates <-chan RuntimeUpdate) *launcherbackend.RuntimeLifecycleState {
	var lastLifecycle *launcherbackend.RuntimeLifecycleState
	for update := range updates {
		if update.Lifecycle != nil {
			lastLifecycle = update.Lifecycle
		}
	}
	return lastLifecycle
}

func runConcurrentHelloStateWork(t *testing.T, controller *qemuController, inst *qemuInstance, ref InstanceRef, workers, iterations int) {
	t.Helper()
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				runHelloStateWorkerIteration(t, controller, inst, ref, i)
			}
		}(i)
	}
	wg.Wait()
}

func runHelloStateWorkerIteration(t *testing.T, controller *qemuController, inst *qemuInstance, ref InstanceRef, worker int) {
	t.Helper()
	if worker%2 == 0 {
		if !recordHelloLine(controller, inst, helloWorldToken) {
			t.Error("recordHelloLine did not accept hello token line")
		}
		return
	}
	if _, err := controller.GetState(context.Background(), ref); err != nil {
		t.Errorf("GetState returned error: %v", err)
	}
}

func assertHelloStateVisible(t *testing.T, controller *qemuController, inst *qemuInstance, ref InstanceRef) {
	t.Helper()
	state, err := controller.GetState(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if !state.HelloWorldSeen {
		t.Fatal("HelloWorldSeen = false, want true after hello output")
	}
	controller.mu.RLock()
	helloSeen := inst.helloSeen
	controller.mu.RUnlock()
	if !helloSeen {
		t.Fatal("inst.helloSeen = false, want true after hello output")
	}
}
