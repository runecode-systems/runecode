//go:build linux

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

func run(args []string) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return err
	}
	return runMode(cfg)
}

func runMode(cfg config) error {
	switch cfg.mode {
	case "cpu":
		return runCPUMode(cfg)
	case "latency":
		return runLatencyMode(cfg)
	case "bench-parse":
		return runBenchParseMode(cfg)
	default:
		return usageError{err: fmt.Errorf("unsupported mode %q", cfg.mode)}
	}
}

func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("tuiperf", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	mode := fs.String("mode", "", "cpu|latency|bench-parse")
	output := fs.String("output", "", "output check json path")
	fixtureID := fs.String("fixture-id", "", "tui.empty.v1|tui.waiting.v1")
	runtimeDir := fs.String("runtime-dir", "", "isolated runtime dir")
	socketName := fs.String("socket-name", "", "isolated socket name")
	stateRoot := fs.String("state-root", "", "isolated broker state root")
	auditLedgerRoot := fs.String("audit-ledger-root", "", "isolated broker audit ledger root")
	targetAlias := fs.String("target-alias", "", "RUNECODE_TUI_BROKER_TARGET alias")
	trials := fs.Int("trials", 30, "latency trials")
	warmupMs := fs.Int("warmup-ms", 3000, "cpu warmup millis")
	windowMs := fs.Int("window-ms", 20000, "cpu observation window millis")
	windows := fs.Int("windows", 3, "cpu observation windows")
	timeoutMs := fs.Int("timeout-ms", 120000, "mode timeout millis")
	benchOutput := fs.String("bench-output", "", "go test bench output path for bench-parse mode")
	if err := fs.Parse(args); err != nil {
		return config{}, usageError{err: err}
	}
	return buildConfig(mode, output, fixtureID, runtimeDir, socketName, stateRoot, auditLedgerRoot, targetAlias, trials, warmupMs, windowMs, windows, timeoutMs, benchOutput)
}

func buildConfig(
	mode *string,
	output *string,
	fixtureID *string,
	runtimeDir *string,
	socketName *string,
	stateRoot *string,
	auditLedgerRoot *string,
	targetAlias *string,
	trials *int,
	warmupMs *int,
	windowMs *int,
	windows *int,
	timeoutMs *int,
	benchOutput *string,
) (config, error) {
	if strings.TrimSpace(*mode) == "" || strings.TrimSpace(*output) == "" {
		return config{}, usageError{err: errors.New("--mode and --output are required")}
	}
	timeout := time.Duration(*timeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return config{
		mode:            strings.TrimSpace(*mode),
		outputPath:      strings.TrimSpace(*output),
		fixtureID:       strings.TrimSpace(*fixtureID),
		runtimeDir:      strings.TrimSpace(*runtimeDir),
		socketName:      strings.TrimSpace(*socketName),
		stateRoot:       strings.TrimSpace(*stateRoot),
		auditLedgerRoot: strings.TrimSpace(*auditLedgerRoot),
		targetAlias:     strings.TrimSpace(*targetAlias),
		trials:          *trials,
		warmup:          time.Duration(*warmupMs) * time.Millisecond,
		window:          time.Duration(*windowMs) * time.Millisecond,
		windows:         *windows,
		timeout:         timeout,
		benchOutput:     strings.TrimSpace(*benchOutput),
	}, nil
}

type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }

func (e usageError) Unwrap() error { return e.err }
