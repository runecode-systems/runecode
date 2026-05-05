package runnerworkflowperf

import (
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func TestRunProducesPhase4RunnerWorkflowMetrics(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(path.Join(filepath.Dir(file), "..", ".."))
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

func hasMetric(measurements []perfcontracts.MeasurementRecord, metricID, unit string) bool {
	for _, m := range measurements {
		if m.MetricID == metricID && m.Unit == unit {
			return true
		}
	}
	return false
}
