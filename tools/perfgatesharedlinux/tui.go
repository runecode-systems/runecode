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
	"strconv"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func measureTUILatency(repoRoot, fixtureID string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error) {
	tmpDir, err := os.MkdirTemp("", "runecode-perfgate-tui-latency-")
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	defer os.RemoveAll(tmpDir)
	outputPath := filepath.Join(tmpDir, "latency.json")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx,
		"go", "run", "./tools/tuiperf",
		"--mode", "latency",
		"--output", outputPath,
		"--fixture-id", fixtureID,
		"--runtime-dir", filepath.Join(tmpDir, "runtime"),
		"--socket-name", "runecode.sock",
		"--state-root", filepath.Join(tmpDir, "state"),
		"--audit-ledger-root", filepath.Join(tmpDir, "audit-ledger"),
		"--target-alias", "default",
		"--trials", strconv.Itoa(trials),
		"--timeout-ms", strconv.Itoa(int(timeout.Milliseconds())),
	)
	cmd.Dir = repoRoot
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return perfcontracts.CheckOutput{}, commandFailure(err, stderr.String(), fmt.Sprintf("tuiperf latency %s", fixtureID))
	}
	return perfcontracts.LoadCheckOutput(outputPath)
}

func measureTUIRenderWaitingBenchmark(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./cmd/runecode-tui", "-run", "^$", "-bench", "BenchmarkShellViewWaitingSession$", "-benchmem", "-count=1")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return perfcontracts.MeasurementRecord{}, commandFailure(err, string(out), "run bench BenchmarkShellViewWaitingSession")
	}
	value, err := parseBenchmarkNSOp(string(out), "BenchmarkShellViewWaitingSession")
	if err != nil {
		return perfcontracts.MeasurementRecord{}, err
	}
	return perfcontracts.MeasurementRecord{MetricID: "metric.tui.render.shell_view_waiting.ns_op", Value: value, Unit: "ns/op"}, nil
}
func commandFailure(err error, output, label string) error {
	msg := strings.TrimSpace(output)
	if msg == "" {
		msg = err.Error()
	}
	return fmt.Errorf("%s failed: %s", label, msg)
}

func parseBenchmarkNSOp(output, benchmark string) (float64, error) {
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(benchmark) + `-\d+\s+\d+\s+([0-9]+(?:\.[0-9]+)?)\s+ns/op`)
	matches := pattern.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("benchmark %s ns/op missing from output", benchmark)
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse benchmark %s ns/op: %w", benchmark, err)
	}
	return value, nil
}
