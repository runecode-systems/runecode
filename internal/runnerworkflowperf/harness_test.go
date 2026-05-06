package runnerworkflowperf

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func TestRunProducesPhase4RunnerWorkflowMetrics(t *testing.T) {
	repoRoot := runnerWorkflowRepoRoot(t)
	out, err := Run(HarnessConfig{RepositoryRoot: repoRoot, CommandRunner: deterministicCommandRunner, CommandTimeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if out.SchemaVersion != CheckSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", out.SchemaVersion, CheckSchemaVersion)
	}
	required := map[string]string{
		"metric.runner.boundary_check.wall_ms":                          "ms",
		"metric.runner.protocol_fixtures.wall_ms":                       "ms",
		"metric.runner.cold_start.minimal_workflow.wall_ms":             "ms",
		"metric.workflow.mvp_execution.supported_path.wall_ms":          "ms",
		"metric.workflow.chg049.first_party_beta_slice.wall_ms":         "ms",
		"metric.workflow.chg050.compile.wall_ms":                        "ms",
		"metric.workflow.chg050.validation_canonicalization.wall_ms":    "ms",
		"metric.workflow.chg050.runplan_persist_load.wall_ms":           "ms",
		"metric.workflow.chg050.runner_start_immutable_runplan.wall_ms": "ms",
	}
	for metricID, unit := range required {
		if !hasMetric(out.Measurements, metricID, unit) {
			t.Fatalf("missing metric %s (%s)", metricID, unit)
		}
	}
}

func TestRunPassesExpectedWorkflowFixtureForSupportedPathMetrics(t *testing.T) {
	repoRoot := runnerWorkflowRepoRoot(t)
	var calls [][]string
	runner := func(_ string, _ time.Duration, args ...string) (float64, error) {
		copied := append([]string(nil), args...)
		calls = append(calls, copied)
		return deterministicCommandRunner("", 0, args...)
	}
	if _, err := Run(HarnessConfig{RepositoryRoot: repoRoot, CommandRunner: runner, CommandTimeout: 10 * time.Second}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	assertModeFixtureArg(t, calls, "workflow-path")
	assertModeFixtureArg(t, calls, "first-party-beta")
}

func runnerWorkflowRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func assertModeFixtureArg(t *testing.T, calls [][]string, mode string) {
	t.Helper()
	for _, call := range calls {
		if !containsArg(call, "--mode", mode) {
			continue
		}
		fixture, ok := argValue(call, "--fixture")
		if !ok {
			t.Fatalf("mode %s missing --fixture argument", mode)
		}
		if fixture != "workflow.first-party-minimal.v1" {
			t.Fatalf("mode %s fixture %q, want %q", mode, fixture, "workflow.first-party-minimal.v1")
		}
		return
	}
	t.Fatalf("no invocation found for mode %s", mode)
}

func deterministicCommandRunner(_ string, _ time.Duration, args ...string) (float64, error) {
	if len(args) == 0 {
		return 0, nil
	}
	if args[0] == "npm" {
		return 1100, nil
	}
	if args[0] != "node" {
		return 100, nil
	}
	if modeLatency, ok := deterministicNodeModeLatency(args); ok {
		return modeLatency, nil
	}
	if isNodeProtocolFixtureCommand(args) {
		return 2700, nil
	}
	return 250, nil
}

func deterministicNodeModeLatency(args []string) (float64, bool) {
	for _, token := range args {
		switch token {
		case "cold-start":
			return 220, true
		case "workflow-path":
			return 340, true
		case "first-party-beta":
			return 280, true
		case "immutable-startup":
			return 310, true
		}
	}
	return 0, false
}

func isNodeProtocolFixtureCommand(args []string) bool {
	return len(args) >= 3 && args[1] == "--test"
}

func containsArg(args []string, key, value string) bool {
	for idx := 0; idx < len(args)-1; idx++ {
		if strings.TrimSpace(args[idx]) == key && strings.TrimSpace(args[idx+1]) == value {
			return true
		}
	}
	return false
}

func argValue(args []string, key string) (string, bool) {
	for idx := 0; idx < len(args)-1; idx++ {
		if strings.TrimSpace(args[idx]) != key {
			continue
		}
		value := strings.TrimSpace(args[idx+1])
		if value == "" {
			return "", false
		}
		return value, true
	}
	return "", false
}

func hasMetric(measurements []perfcontracts.MeasurementRecord, metricID, unit string) bool {
	for _, m := range measurements {
		if m.MetricID == metricID && m.Unit == unit {
			return true
		}
	}
	return false
}

func TestValidateRunnerExecutableRejectsUnexpectedBinary(t *testing.T) {
	if err := validateRunnerExecutable("bash"); err == nil {
		t.Fatal("validateRunnerExecutable error = nil, want rejection")
	}
}

func TestParseRunnerMeasurement(t *testing.T) {
	value, err := parseRunnerMeasurement([]byte("340\n"))
	if err != nil {
		t.Fatalf("parseRunnerMeasurement returned error: %v", err)
	}
	if value != 340 {
		t.Fatalf("value = %v, want 340", value)
	}
}

func TestDeterministicCommandRunnerUsesScriptMeasurementOutput(t *testing.T) {
	value, err := deterministicCommandRunner("", 0, "node", "--experimental-strip-types", "scripts/perf-runner-workflow.js", "--mode", "workflow-path")
	if err != nil {
		t.Fatalf("deterministicCommandRunner returned error: %v", err)
	}
	if value != 340 {
		t.Fatalf("value = %v, want 340", value)
	}
}
