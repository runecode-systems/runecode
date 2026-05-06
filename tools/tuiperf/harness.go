//go:build linux

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

const terminateGracePeriod = 2 * time.Second

var (
	absPathPattern      = regexp.MustCompile(`(?:[A-Za-z]:\\|/)[^\s"']+`)
	longTokenPattern    = regexp.MustCompile(`\b[A-Za-z0-9_=-]{24,}\b`)
	wsPattern           = regexp.MustCompile(`\s+`)
	nonPrintablePattern = regexp.MustCompile(`[^[:print:]\t\n\r]`)
)

type runningHarness struct {
	ctx       context.Context
	cancel    context.CancelFunc
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
	preparedCfg, err := prepareHarnessBinaries(cfg)
	if err != nil {
		cancel()
		return nil, nil, runningHarness{}, err
	}
	cancelWithCleanup := func() {
		cancel()
		cleanupHarnessBinaries(preparedCfg)
	}
	if err := prepareTUIIsolation(cfg); err != nil {
		cancelWithCleanup()
		return nil, nil, runningHarness{}, err
	}
	harness, err := startHarnessProcesses(ctx, preparedCfg)
	if err != nil {
		cancelWithCleanup()
		return nil, nil, runningHarness{}, err
	}
	return ctx, cancelWithCleanup, harness, nil
}

func prepareHarnessBinaries(cfg config) (config, error) {
	if cfg.repoRoot == "" {
		return cfg, fmt.Errorf("repository root required")
	}
	binDir, err := os.MkdirTemp("", "runecode-tuiperf-bin-")
	if err != nil {
		return cfg, err
	}
	brokerBin := filepath.Join(binDir, "runecode-broker")
	tuiBin := filepath.Join(binDir, "runecode-tui")
	cfg.harnessBinDir = binDir
	if err := buildHarnessBinary(cfg.repoRoot, "./cmd/runecode-broker", brokerBin); err != nil {
		_ = os.RemoveAll(binDir)
		return cfg, err
	}
	if err := buildHarnessBinary(cfg.repoRoot, "./cmd/runecode-tui", tuiBin); err != nil {
		_ = os.RemoveAll(binDir)
		return cfg, err
	}
	cfg.brokerBin = brokerBin
	cfg.tuiBin = tuiBin
	return cfg, nil
}

func cleanupHarnessBinaries(cfg config) {
	if cfg.harnessBinDir != "" {
		_ = os.RemoveAll(cfg.harnessBinDir)
	}
}

func buildHarnessBinary(repoRoot, pkg, output string) error {
	cmd := exec.Command("go", "build", "-o", output, pkg)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build %s: %s", pkg, strings.TrimSpace(string(out)))
	}
	return nil
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
		return usageError{err: fmt.Errorf("mode requires --fixture-id tui.empty.v1|tui.waiting.v1")}
	}
	if cfg.trials <= 0 {
		return usageError{err: fmt.Errorf("--trials must be > 0")}
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
	return os.MkdirAll(cfg.auditLedgerRoot, 0o700)
}

func startBrokerProcess(ctx context.Context, cfg config) (*exec.Cmd, error) {
	brokerBin := cfg.brokerBin
	if brokerBin == "" {
		brokerBin = "go"
	}
	brokerArgs := []string{"--state-root", cfg.stateRoot, "--audit-ledger-root", cfg.auditLedgerRoot, "serve-local", "--runtime-dir", cfg.runtimeDir, "--socket-name", cfg.socketName}
	brokerCmd := exec.CommandContext(ctx, brokerBin, brokerArgs...)
	if cfg.brokerBin == "" {
		brokerCmd = exec.CommandContext(ctx, "go", append([]string{"run", "./cmd/runecode-broker"}, brokerArgs...)...)
	}
	brokerCmd.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	brokerCmd.Stdout = &stdout
	brokerCmd.Stderr = &stderr
	if err := brokerCmd.Start(); err != nil {
		return nil, err
	}
	if err := waitForSocket(filepath.Join(cfg.runtimeDir, cfg.socketName), 5*time.Second); err != nil {
		terminateProcess(brokerCmd.Process)
		return nil, fmt.Errorf("%w; broker startup summary: %s", err, summarizeBrokerStartupOutput(stdout.String(), stderr.String()))
	}
	return brokerCmd, nil
}

func startTUIProcess(ctx context.Context, cfg config) (*exec.Cmd, io.ReadCloser, io.WriteCloser, error) {
	tuiBin := cfg.tuiBin
	tuiArgs := []string{"--runtime-dir", cfg.runtimeDir, "--socket-name", cfg.socketName}
	tuiCmd := exec.CommandContext(ctx, tuiBin, tuiArgs...)
	if cfg.tuiBin == "" {
		tuiCmd = exec.CommandContext(ctx, "go", append([]string{"run", "./cmd/runecode-tui"}, tuiArgs...)...)
	}
	tuiCmd.Env = stableTTYEnv(os.Environ())
	tuiCmd.Env = append(tuiCmd.Env, "RUNECODE_TUI_BROKER_TARGET="+cfg.targetAlias)
	tty, err := pty.Start(tuiCmd)
	if err != nil {
		return nil, nil, nil, err
	}
	return tuiCmd, newTerminalQueryResponder(tty, tty), tty, nil
}

func stopHarness(h runningHarness) {
	terminateProcess(processOf(h.tuiCmd))
	terminateProcess(processOf(h.brokerCmd))
}

func processOf(cmd *exec.Cmd) *os.Process {
	if cmd == nil {
		return nil
	}
	return cmd.Process
}

func requireIsolationInputs(cfg config) error {
	if cfg.runtimeDir == "" || cfg.socketName == "" || cfg.stateRoot == "" || cfg.auditLedgerRoot == "" || cfg.targetAlias == "" {
		return usageError{err: fmt.Errorf("isolation inputs required: --runtime-dir --socket-name --state-root --audit-ledger-root --target-alias")}
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
	done := make(chan struct{})
	go func() {
		_, _ = p.Wait()
		close(done)
	}()
	_ = p.Signal(syscall.SIGTERM)
	select {
	case <-done:
		return
	case <-time.After(terminateGracePeriod):
	}
	_ = p.Signal(syscall.SIGKILL)
	select {
	case <-done:
	case <-time.After(terminateGracePeriod):
	}
}

func summarizeBrokerStartupOutput(stdoutRaw, stderrRaw string) string {
	return fmt.Sprintf("stdout=%s stderr=%s", summarizeStartupOutput(stdoutRaw), summarizeStartupOutput(stderrRaw))
}

func summarizeStartupOutput(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "<empty>"
	}
	text = nonPrintablePattern.ReplaceAllString(text, "")
	text = absPathPattern.ReplaceAllString(text, "<path>")
	text = longTokenPattern.ReplaceAllString(text, "<redacted>")
	text = wsPattern.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	if text == "" {
		return "<empty>"
	}
	const maxLen = 200
	if len(text) <= maxLen {
		return text
	}
	return "…" + text[len(text)-maxLen:]
}
