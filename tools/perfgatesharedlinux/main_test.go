package main

import (
	"encoding/json"
	"errors"
	"os"
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
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestParseArgsRequiresOutput(t *testing.T) {
	_, err := parseArgs([]string{})
	if err == nil {
		t.Fatal("parseArgs error = nil, want required output error")
	}
	var usageErr usageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("parseArgs error = %T, want usageError", err)
	}
}

func TestRunWithDepsWritesMergedOutput(t *testing.T) {
	root := repositoryRootForPerfGateTests(t)
	output := filepath.Join(root, "check.json")
	cfg := config{outputPath: output, repository: root, trials: 30, timeout: 2 * time.Second}
	defer os.Remove(output)
	err := runWithDeps(cfg, mergedOutputDeps(t, root))
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
	if len(parsed.Measurements) != 7 {
		t.Fatalf("measurements = %d, want 7", len(parsed.Measurements))
	}
}

func mergedOutputDeps(t *testing.T, root string) deps {
	t.Helper()
	return deps{
		runRunnerWorkflow: expectedHarnessOutput(t, root, 2*time.Second,
			perfcontracts.MeasurementRecord{MetricID: "metric.runner.boundary_check.wall_ms", Value: 50, Unit: "ms"},
			perfcontracts.MeasurementRecord{MetricID: "metric.runner.protocol_fixtures.wall_ms", Value: 75, Unit: "ms"},
		),
		runBrokerPerf: expectedBrokerOutput(t, root, 30,
			perfcontracts.MeasurementRecord{MetricID: "metric.broker.unary.session_list.p95_ms", Value: 1, Unit: "ms"},
		),
		runPhase5Perf: expectedPhase5Output(t, root, 30, 2*time.Second,
			perfcontracts.MeasurementRecord{MetricID: "metric.protocol.schema_validation.wall_ms", Value: 10, Unit: "ms"},
		),
		runTUIQuiet: expectedTUIOutput(t, root, 30, 2*time.Second,
			perfcontracts.MeasurementRecord{MetricID: "metric.tui.attach.quiet.p95_ms", Value: 20, Unit: "ms"},
		),
		runTUIWaiting: expectedTUIOutput(t, root, 30, 2*time.Second,
			perfcontracts.MeasurementRecord{MetricID: "metric.tui.attach.waiting.p95_ms", Value: 22, Unit: "ms"},
		),
		runTUIBench: expectedTUIBench(t, root, 2*time.Second,
			perfcontracts.MeasurementRecord{MetricID: "metric.tui.render.shell_view_waiting.ns_op", Value: 1234, Unit: "ns/op"},
		),
		listRequiredIDs: expectedRequiredIDs(t, root,
			"metric.runner.boundary_check.wall_ms",
			"metric.runner.protocol_fixtures.wall_ms",
			"metric.broker.unary.session_list.p95_ms",
			"metric.protocol.schema_validation.wall_ms",
			"metric.tui.attach.quiet.p95_ms",
			"metric.tui.attach.waiting.p95_ms",
			"metric.tui.render.shell_view_waiting.ns_op",
		),
	}
}

func expectedRequiredIDs(t *testing.T, root string, metricIDs ...string) func(string) ([]string, error) {
	t.Helper()
	return func(repoRoot string) ([]string, error) {
		if repoRoot != root {
			t.Fatalf("repoRoot = %q, want %q", repoRoot, root)
		}
		return metricIDs, nil
	}
}

func expectedHarnessOutput(t *testing.T, root string, wantTimeout time.Duration, items ...perfcontracts.MeasurementRecord) func(string, time.Duration) (perfcontracts.CheckOutput, error) {
	t.Helper()
	return func(repoRoot string, timeout time.Duration) (perfcontracts.CheckOutput, error) {
		if repoRoot != root {
			t.Fatalf("repoRoot = %q, want %q", repoRoot, root)
		}
		if timeout != wantTimeout {
			t.Fatalf("timeout = %s, want %s", timeout, wantTimeout)
		}
		return perfcontracts.CheckOutput{SchemaVersion: checkSchemaVersion, Measurements: items}, nil
	}
}

func expectedBrokerOutput(t *testing.T, root string, wantTrials int, items ...perfcontracts.MeasurementRecord) func(string, int) (perfcontracts.CheckOutput, error) {
	t.Helper()
	return func(repoRoot string, trials int) (perfcontracts.CheckOutput, error) {
		if repoRoot != root {
			t.Fatalf("repoRoot = %q, want %q", repoRoot, root)
		}
		if trials != wantTrials {
			t.Fatalf("trials = %d, want %d", trials, wantTrials)
		}
		return perfcontracts.CheckOutput{SchemaVersion: checkSchemaVersion, Measurements: items}, nil
	}
}

func expectedPhase5Output(t *testing.T, root string, wantTrials int, wantTimeout time.Duration, items ...perfcontracts.MeasurementRecord) func(string, int, time.Duration) (perfcontracts.CheckOutput, error) {
	t.Helper()
	return func(repoRoot string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error) {
		if repoRoot != root {
			t.Fatalf("repoRoot = %q, want %q", repoRoot, root)
		}
		if trials != wantTrials {
			t.Fatalf("trials = %d, want %d", trials, wantTrials)
		}
		if timeout != wantTimeout {
			t.Fatalf("timeout = %s, want %s", timeout, wantTimeout)
		}
		return perfcontracts.CheckOutput{SchemaVersion: checkSchemaVersion, Measurements: items}, nil
	}
}

func expectedTUIOutput(t *testing.T, root string, wantTrials int, wantTimeout time.Duration, items ...perfcontracts.MeasurementRecord) func(string, int, time.Duration) (perfcontracts.CheckOutput, error) {
	t.Helper()
	return func(repoRoot string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error) {
		if repoRoot != root {
			t.Fatalf("repoRoot = %q, want %q", repoRoot, root)
		}
		if trials != wantTrials {
			t.Fatalf("trials = %d, want %d", trials, wantTrials)
		}
		if timeout != wantTimeout {
			t.Fatalf("timeout = %s, want %s", timeout, wantTimeout)
		}
		return perfcontracts.CheckOutput{SchemaVersion: checkSchemaVersion, Measurements: items}, nil
	}
}

func expectedTUIBench(t *testing.T, root string, wantTimeout time.Duration, item perfcontracts.MeasurementRecord) func(string, time.Duration) (perfcontracts.MeasurementRecord, error) {
	t.Helper()
	return func(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error) {
		if repoRoot != root {
			t.Fatalf("repoRoot = %q, want %q", repoRoot, root)
		}
		if timeout != wantTimeout {
			t.Fatalf("timeout = %s, want %s", timeout, wantTimeout)
		}
		return item, nil
	}
}

func TestRunWithDepsPropagatesHarnessError(t *testing.T) {
	root := repositoryRootForPerfGateTests(t)
	cfg := config{outputPath: filepath.Join(root, "check.json"), repository: root, trials: 30, timeout: time.Second}
	err := runWithDeps(cfg, deps{
		runRunnerWorkflow: func(_ string, _ time.Duration) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, assertErr("runner workflow boom")
		},
		runBrokerPerf: func(_ string, _ int) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, nil
		},
		runPhase5Perf: func(_ string, _ int, _ time.Duration) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, nil
		},
		runTUIQuiet: func(_ string, _ int, _ time.Duration) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, nil
		},
		runTUIWaiting: func(_ string, _ int, _ time.Duration) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, nil
		},
		runTUIBench: func(_ string, _ time.Duration) (perfcontracts.MeasurementRecord, error) {
			return perfcontracts.MeasurementRecord{}, nil
		},
		listRequiredIDs: func(_ string) ([]string, error) { return nil, nil },
	})
	if err == nil || !strings.Contains(err.Error(), "run runner workflow perf") {
		t.Fatalf("runWithDeps error = %v, want runner workflow context", err)
	}
}

func TestRunWithDepsPropagatesBrokerError(t *testing.T) {
	root := repositoryRootForPerfGateTests(t)
	cfg := config{outputPath: filepath.Join(root, "check.json"), repository: root, trials: 30, timeout: time.Second}
	err := runWithDeps(cfg, deps{
		runRunnerWorkflow: func(_ string, _ time.Duration) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{SchemaVersion: checkSchemaVersion, Measurements: []perfcontracts.MeasurementRecord{{MetricID: "metric.runner.boundary_check.wall_ms", Value: 1, Unit: "ms"}}}, nil
		},
		runBrokerPerf: func(_ string, _ int) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, assertErr("broker boom")
		},
		runPhase5Perf: func(_ string, _ int, _ time.Duration) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, nil
		},
		runTUIQuiet: func(_ string, _ int, _ time.Duration) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, nil
		},
		runTUIWaiting: func(_ string, _ int, _ time.Duration) (perfcontracts.CheckOutput, error) {
			return perfcontracts.CheckOutput{}, nil
		},
		runTUIBench: func(_ string, _ time.Duration) (perfcontracts.MeasurementRecord, error) {
			return perfcontracts.MeasurementRecord{}, nil
		},
		listRequiredIDs: func(_ string) ([]string, error) { return nil, nil },
	})
	if err == nil || !strings.Contains(err.Error(), "run broker perf") {
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

func TestRunWithDepsFailsWhenRequiredMetricMissingFromAggregation(t *testing.T) {
	root := repositoryRootForPerfGateTests(t)
	cfg := config{outputPath: filepath.Join(root, "check.json"), repository: root, trials: 30, timeout: time.Second}
	err := runWithDeps(cfg, deps{
		runRunnerWorkflow: expectedHarnessOutput(t, root, time.Second,
			perfcontracts.MeasurementRecord{MetricID: "metric.runner.boundary_check.wall_ms", Value: 1, Unit: "ms"},
		),
		runBrokerPerf: expectedBrokerOutput(t, root, 30),
		runPhase5Perf: expectedPhase5Output(t, root, 30, time.Second),
		runTUIQuiet:   expectedTUIOutput(t, root, 30, time.Second),
		runTUIWaiting: expectedTUIOutput(t, root, 30, time.Second),
		runTUIBench:   expectedTUIBench(t, root, time.Second, perfcontracts.MeasurementRecord{MetricID: "metric.tui.render.shell_view_waiting.ns_op", Value: 1, Unit: "ns/op"}),
		listRequiredIDs: func(_ string) ([]string, error) {
			return []string{"metric.runner.boundary_check.wall_ms", "metric.missing"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "required_shared_linux metrics missing") {
		t.Fatalf("runWithDeps error = %v, want missing required metric error", err)
	}
}

func TestParseBenchmarkNSOp(t *testing.T) {
	out := "BenchmarkShellViewWaitingSession-8  12493  8912 ns/op  1200 B/op  10 allocs/op\n"
	value, err := parseBenchmarkNSOp(out, "BenchmarkShellViewWaitingSession")
	if err != nil {
		t.Fatalf("parseBenchmarkNSOp returned error: %v", err)
	}
	if value != 8912 {
		t.Fatalf("value = %v, want 8912", value)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
