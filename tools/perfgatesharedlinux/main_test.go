package main

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func repositoryRootForPerfGateTests(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(path.Join(filepath.Dir(file), "..", ".."))
}

func TestParseArgsRequiresOutput(t *testing.T) {
	if _, err := parseArgs([]string{}); err == nil {
		t.Fatal("parseArgs error = nil, want required output error")
	}
}

func TestRunWithDepsWritesMergedOutput(t *testing.T) {
	root := repositoryRootForPerfGateTests(t)
	output := filepath.Join(root, "check.json")
	cfg := config{outputPath: output, repository: root, trials: 30, timeout: 2 * time.Second}
	defer os.Remove(output)
	err := runWithDeps(cfg, deps{
		runRunnerBoundary: expectedRunnerMetric(t, root, 2*time.Second, "metric.runner.boundary_check.wall_ms", 50),
		runRunnerFixtures: expectedRunnerMetric(t, root, 2*time.Second, "metric.runner.protocol_fixtures.wall_ms", 75),
		runBrokerUnary:    expectedBrokerMetric(t, root, 30, "metric.broker.unary.session_list.p95_ms", 1),
	})
	if err != nil {
		t.Fatalf("runWithDeps error: %v", err)
	}

	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile output: %v", err)
	}
	var parsed perfcontracts.CheckOutput
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if parsed.SchemaVersion != checkSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", parsed.SchemaVersion, checkSchemaVersion)
	}
	if len(parsed.Measurements) != 3 {
		t.Fatalf("measurements = %d, want 3", len(parsed.Measurements))
	}
}

func expectedRunnerMetric(t *testing.T, root string, wantTimeout time.Duration, metricID string, value float64) func(string, time.Duration) (perfcontracts.MeasurementRecord, error) {
	t.Helper()
	return func(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error) {
		if repoRoot != root {
			t.Fatalf("repoRoot = %q, want %q", repoRoot, root)
		}
		if timeout != wantTimeout {
			t.Fatalf("timeout = %s, want %s", timeout, wantTimeout)
		}
		return perfcontracts.MeasurementRecord{MetricID: metricID, Value: value, Unit: "ms"}, nil
	}
}

func expectedBrokerMetric(t *testing.T, root string, wantTrials int, metricID string, value float64) func(string, int) (perfcontracts.MeasurementRecord, error) {
	t.Helper()
	return func(repoRoot string, trials int) (perfcontracts.MeasurementRecord, error) {
		if repoRoot != root {
			t.Fatalf("repoRoot = %q, want %q", repoRoot, root)
		}
		if trials != wantTrials {
			t.Fatalf("trials = %d, want %d", trials, wantTrials)
		}
		return perfcontracts.MeasurementRecord{MetricID: metricID, Value: value, Unit: "ms"}, nil
	}
}

func TestRunWithDepsPropagatesHarnessError(t *testing.T) {
	root := repositoryRootForPerfGateTests(t)
	cfg := config{outputPath: filepath.Join(root, "check.json"), repository: root, trials: 30, timeout: time.Second}
	err := runWithDeps(cfg, deps{
		runRunnerBoundary: func(_ string, _ time.Duration) (perfcontracts.MeasurementRecord, error) {
			return perfcontracts.MeasurementRecord{}, assertErr("runner boundary boom")
		},
		runRunnerFixtures: func(_ string, _ time.Duration) (perfcontracts.MeasurementRecord, error) {
			return perfcontracts.MeasurementRecord{}, nil
		},
		runBrokerUnary: func(_ string, _ int) (perfcontracts.MeasurementRecord, error) {
			return perfcontracts.MeasurementRecord{}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "run runner boundary-check perf") {
		t.Fatalf("runWithDeps error = %v, want runner boundary context", err)
	}
}

func TestRunWithDepsPropagatesBrokerError(t *testing.T) {
	root := repositoryRootForPerfGateTests(t)
	cfg := config{outputPath: filepath.Join(root, "check.json"), repository: root, trials: 30, timeout: time.Second}
	err := runWithDeps(cfg, deps{
		runRunnerBoundary: func(_ string, _ time.Duration) (perfcontracts.MeasurementRecord, error) {
			return perfcontracts.MeasurementRecord{MetricID: "metric.runner.boundary_check.wall_ms", Value: 1, Unit: "ms"}, nil
		},
		runRunnerFixtures: func(_ string, _ time.Duration) (perfcontracts.MeasurementRecord, error) {
			return perfcontracts.MeasurementRecord{MetricID: "metric.runner.protocol_fixtures.wall_ms", Value: 1, Unit: "ms"}, nil
		},
		runBrokerUnary: func(_ string, _ int) (perfcontracts.MeasurementRecord, error) {
			return perfcontracts.MeasurementRecord{}, assertErr("broker boom")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "run broker unary perf") {
		t.Fatalf("runWithDeps error = %v, want broker context", err)
	}
}

func TestRunWithDepsRejectsInvalidRepositoryRoot(t *testing.T) {
	t.Parallel()
	cfg := config{outputPath: filepath.Join(t.TempDir(), "check.json"), repository: t.TempDir(), trials: 30, timeout: time.Second}
	err := runWithDeps(cfg, deps{})
	if err == nil || !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("runWithDeps error = %v, want repository root validation failure", err)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
