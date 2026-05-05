package runnerworkflowperf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		{metricID: "metric.runner.cold_start.minimal_workflow.wall_ms", mode: "cold-start"},
		{metricID: "metric.workflow.mvp_execution.supported_path.wall_ms", mode: "workflow-path"},
		{metricID: "metric.workflow.chg049.first_party_beta_slice.wall_ms", mode: "first-party-beta"},
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
		wallMS, err := runner(repoRoot, timeout, "node", "--experimental-strip-types", "scripts/perf-runner-workflow.js", "--mode", spec.mode, "--runplan", runPlanPath)
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
	return float64(time.Since(started).Milliseconds()), nil
}
