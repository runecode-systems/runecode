//go:build linux

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type runningHarness struct {
	ctx       context.Context
	brokerCmd *exec.Cmd
	tuiCmd    *exec.Cmd
	tuiOut    io.ReadCloser
	tuiIn     io.WriteCloser
}

func startTUIHarness(cfg config) (context.Context, context.CancelFunc, runningHarness, error) {
	if err := requireTUIFixtureConfig(cfg); err != nil {
		return nil, nil, runningHarness{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	if err := prepareTUIIsolation(cfg); err != nil {
		cancel()
		return nil, nil, runningHarness{}, err
	}
	harness, err := startHarnessProcesses(ctx, cfg)
	if err != nil {
		cancel()
		return nil, nil, runningHarness{}, err
	}
	return ctx, cancel, harness, nil
}

func startHarnessProcesses(ctx context.Context, cfg config) (runningHarness, error) {
	brokerCmd, err := startBrokerProcess(ctx, cfg)
	if err != nil {
		return runningHarness{}, err
	}
	tuiCmd, tuiOut, tuiIn, err := startTUIProcess(ctx, cfg)
	if err != nil {
		terminateProcess(brokerCmd.Process)
		return runningHarness{}, err
	}
	return runningHarness{ctx: ctx, brokerCmd: brokerCmd, tuiCmd: tuiCmd, tuiOut: tuiOut, tuiIn: tuiIn}, nil
}

func requireTUIFixtureConfig(cfg config) error {
	if err := requireIsolationInputs(cfg); err != nil {
		return err
	}
	if cfg.fixtureID != "tui.empty.v1" && cfg.fixtureID != "tui.waiting.v1" {
		return fmt.Errorf("mode requires --fixture-id tui.empty.v1|tui.waiting.v1")
	}
	if cfg.trials <= 0 {
		return fmt.Errorf("--trials must be > 0")
	}
	return nil
}

func prepareTUIIsolation(cfg config) error {
	if err := seedFixture(cfg.stateRoot, cfg.fixtureID); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.runtimeDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(cfg.runtimeDir, 0o700); err != nil {
		return err
	}
	return os.MkdirAll(cfg.auditLedgerRoot, 0o755)
}

func startBrokerProcess(ctx context.Context, cfg config) (*exec.Cmd, error) {
	brokerCmd := exec.CommandContext(ctx, "go", "run", "./cmd/runecode-broker", "--state-root", cfg.stateRoot, "--audit-ledger-root", cfg.auditLedgerRoot, "serve-local", "--runtime-dir", cfg.runtimeDir, "--socket-name", cfg.socketName)
	brokerCmd.Env = os.Environ()
	brokerCmd.Stdout = io.Discard
	brokerCmd.Stderr = io.Discard
	if err := brokerCmd.Start(); err != nil {
		return nil, err
	}
	if err := waitForSocket(filepath.Join(cfg.runtimeDir, cfg.socketName), 5*time.Second); err != nil {
		terminateProcess(brokerCmd.Process)
		return nil, err
	}
	return brokerCmd, nil
}

func startTUIProcess(ctx context.Context, cfg config) (*exec.Cmd, io.ReadCloser, io.WriteCloser, error) {
	tuiCmd := exec.CommandContext(ctx, "script", "-q", "-f", "-c", fmt.Sprintf("RUNECODE_TUI_BROKER_TARGET=%s go run ./cmd/runecode-tui --runtime-dir %s --socket-name %s", shellEscape(cfg.targetAlias), shellEscape(cfg.runtimeDir), shellEscape(cfg.socketName)), "/dev/null")
	tuiCmd.Env = os.Environ()
	tuiOut, err := tuiCmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	tuiIn, err := tuiCmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	tuiCmd.Stderr = io.Discard
	if err := tuiCmd.Start(); err != nil {
		return nil, nil, nil, err
	}
	return tuiCmd, tuiOut, tuiIn, nil
}

func stopHarness(h runningHarness) {
	terminateProcess(h.tuiCmd.Process)
	terminateProcess(h.brokerCmd.Process)
}

func requireIsolationInputs(cfg config) error {
	if cfg.runtimeDir == "" || cfg.socketName == "" || cfg.stateRoot == "" || cfg.auditLedgerRoot == "" || cfg.targetAlias == "" {
		return fmt.Errorf("isolation inputs required: --runtime-dir --socket-name --state-root --audit-ledger-root --target-alias")
	}
	return nil
}

func waitForSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		info, err := os.Stat(path)
		if err == nil && (info.Mode()&os.ModeSocket) != 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("socket not ready: %s", path)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func terminateProcess(p *os.Process) {
	if p == nil {
		return
	}
	_ = p.Signal(syscall.SIGTERM)
	_, _ = p.Wait()
}
