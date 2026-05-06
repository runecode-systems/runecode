package runnerworkflowperf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/perffixtures"
)

func collectRunnerCheckMeasurements(repoRoot string, timeout time.Duration, runner func(repoRoot string, timeout time.Duration, args ...string) (float64, error)) ([]perfcontracts.MeasurementRecord, error) {
	boundaryMS, err := runner(repoRoot, timeout, "npm", "run", "boundary-check")
	if err != nil {
		return nil, fmt.Errorf("measure boundary-check: %w", err)
	}
	fixturesMS, err := runner(repoRoot, timeout, "node", "--test", "scripts/protocol-fixtures.test.js")
	if err != nil {
		return nil, fmt.Errorf("measure protocol-fixtures: %w", err)
	}
	return []perfcontracts.MeasurementRecord{{MetricID: "metric.runner.boundary_check.wall_ms", Value: boundaryMS, Unit: "ms"}, {MetricID: "metric.runner.protocol_fixtures.wall_ms", Value: fixturesMS, Unit: "ms"}}, nil
}

func collectMinimalWorkflowMeasurements(repoRoot string, timeout time.Duration, runner func(repoRoot string, timeout time.Duration, args ...string) (float64, error)) ([]perfcontracts.MeasurementRecord, error) {
	minimal, cleanup, err := buildMinimalWorkflowFixture()
	if err != nil {
		return nil, err
	}
	defer cleanup()
	specs := []runnerMeasurementSpec{
		{metricID: "metric.runner.cold_start.minimal_workflow.wall_ms", mode: "cold-start", fixture: minimal.FixtureID},
		{metricID: "metric.workflow.mvp_execution.supported_path.wall_ms", mode: "workflow-path", fixture: minimal.FixtureID},
		{metricID: "metric.workflow.chg049.first_party_beta_slice.wall_ms", mode: "first-party-beta", fixture: minimal.FixtureID},
	}
	return collectWorkflowSpecs(repoRoot, timeout, runner, minimal.RunPlan, specs)
}

func collectWorkflowSpecs(
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, args ...string) (float64, error),
	runPlanPath string,
	specs []runnerMeasurementSpec,
) ([]perfcontracts.MeasurementRecord, error) {
	measurements := make([]perfcontracts.MeasurementRecord, 0, len(specs))
	for _, spec := range specs {
		args := []string{"node", "--experimental-strip-types", "scripts/perf-runner-workflow.js", "--mode", spec.mode, "--runplan", runPlanPath}
		if fixtureID := strings.TrimSpace(spec.fixture); fixtureID != "" {
			args = append(args, "--fixture", fixtureID)
		}
		wallMS, err := runner(repoRoot, timeout, args...)
		if err != nil {
			return nil, fmt.Errorf("measure %s: %w", spec.mode, err)
		}
		measurements = append(measurements, perfcontracts.MeasurementRecord{MetricID: spec.metricID, Value: wallMS, Unit: "ms"})
	}
	return measurements, nil
}

func buildMinimalWorkflowFixture() (perffixtures.WorkflowFixtureResult, func(), error) {
	root, err := os.MkdirTemp("", "runecode-runnerworkflowperf-minimal-")
	if err != nil {
		return perffixtures.WorkflowFixtureResult{}, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }
	fixture, err := perffixtures.BuildWorkflowFixture(root, perffixtures.FixtureWorkflowFirstPartyMinimal)
	if err != nil {
		cleanup()
		return perffixtures.WorkflowFixtureResult{}, nil, fmt.Errorf("build minimal workflow fixture: %w", err)
	}
	return fixture, cleanup, nil
}

func measureRunnerCommand(repoRoot string, timeout time.Duration, args ...string) (float64, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("command arguments required")
	}
	if err := validateRunnerExecutable(args[0]); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	runnerDir := filepath.Join(repoRoot, "runner")
	started := time.Now()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = runnerDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return 0, fmt.Errorf("%s failed: %s", strings.Join(args, " "), msg)
	}
	if expectsRunnerScriptMeasurement(args) {
		return parseRunnerMeasurement(out)
	}
	return float64(time.Since(started).Milliseconds()), nil
}

func validateRunnerExecutable(name string) error {
	switch strings.TrimSpace(name) {
	case "npm", "node":
		return nil
	default:
		return fmt.Errorf("unsupported runner executable %q", name)
	}
}

func expectsRunnerScriptMeasurement(args []string) bool {
	for _, arg := range args {
		if arg == "scripts/perf-runner-workflow.js" {
			return true
		}
	}
	return false
}

func parseRunnerMeasurement(out []byte) (float64, error) {
	value := strings.TrimSpace(string(out))
	if value == "" {
		return 0, fmt.Errorf("runner workflow script returned empty measurement output")
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse runner workflow measurement %q: %w", value, err)
	}
	return parsed, nil
}
