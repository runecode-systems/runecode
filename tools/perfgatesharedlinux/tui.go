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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

const tuiRenderBenchmarkSamples = 10

func measureTUILatency(repoRoot, fixtureID string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error) {
	tmpDir, err := os.MkdirTemp("", "runecode-perfgate-tui-latency-")
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	defer os.RemoveAll(tmpDir)
	outputPath := filepath.Join(tmpDir, "latency.json")
	ctx, cancel := context.WithTimeout(context.Background(), latencyBatchTimeout(timeout, trials))
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

func latencyBatchTimeout(perTrialTimeout time.Duration, trials int) time.Duration {
	if trials < 1 {
		trials = 1
	}
	estimated := 30*time.Second + time.Duration(trials)*7*time.Second
	if estimated > perTrialTimeout {
		return estimated
	}
	return perTrialTimeout
}

func measureTUIRenderWaitingBenchmark(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./cmd/runecode-tui", "-run", "^$", "-bench", "BenchmarkShellViewWaitingSession$", "-benchmem", "-count", strconv.Itoa(tuiRenderBenchmarkSamples))
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return perfcontracts.MeasurementRecord{}, commandFailure(err, string(out), "run bench BenchmarkShellViewWaitingSession")
	}
	value, err := parseBenchmarkMedianNSOp(string(out), "BenchmarkShellViewWaitingSession", tuiRenderBenchmarkSamples)
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

func parseBenchmarkMedianNSOp(output, benchmark string, wantSamples int) (float64, error) {
	values, err := parseBenchmarkNSOps(output, benchmark)
	if err != nil {
		return 0, err
	}
	if wantSamples > 0 && len(values) != wantSamples {
		return 0, fmt.Errorf("benchmark %s ns/op sample count = %d, want %d", benchmark, len(values), wantSamples)
	}
	return medianFloat64(values), nil
}

func parseBenchmarkNSOps(output, benchmark string) ([]float64, error) {
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(benchmark) + `-\d+\s+\d+\s+([0-9]+(?:\.[0-9]+)?)\s+ns/op`)
	matches := pattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("benchmark %s ns/op missing from output", benchmark)
	}
	values := make([]float64, 0, len(matches))
	for _, match := range matches {
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse benchmark %s ns/op: %w", benchmark, err)
		}
		values = append(values, value)
	}
	return values, nil
}

func medianFloat64(values []float64) float64 {
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 0 {
		return (cp[mid-1] + cp[mid]) / 2
	}
	return cp[mid]
}
