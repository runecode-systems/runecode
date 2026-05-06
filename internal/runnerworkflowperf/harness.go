package runnerworkflowperf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

const CheckSchemaVersion = "runecode.performance.check.v1"

type HarnessConfig struct {
	RepositoryRoot string
	CommandTimeout time.Duration
	CommandRunner  func(repoRoot string, timeout time.Duration, args ...string) (float64, error)
}

type runnerMeasurementSpec struct {
	metricID string
	mode     string
	fixture  string
}

func Run(cfg HarnessConfig) (perfcontracts.CheckOutput, error) {
	repoRoot, err := resolveRepoRoot(cfg.RepositoryRoot)
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	timeout := resolvedTimeout(cfg.CommandTimeout)
	runner := cfg.CommandRunner
	if runner == nil {
		runner = measureRunnerCommand
	}
	measurements, err := collectAllMeasurements(repoRoot, timeout, runner)
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	return perfcontracts.CheckOutput{SchemaVersion: CheckSchemaVersion, Measurements: measurements}, nil
}

func resolvedTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 2 * time.Minute
	}
	return timeout
}

func collectAllMeasurements(
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, args ...string) (float64, error),
) ([]perfcontracts.MeasurementRecord, error) {
	measurements := make([]perfcontracts.MeasurementRecord, 0, 12)
	if err := appendRunnerCheckMeasurements(&measurements, repoRoot, timeout, runner); err != nil {
		return nil, err
	}
	if err := appendWorkflowMeasurements(&measurements, repoRoot, timeout, runner); err != nil {
		return nil, err
	}
	if err := appendCHG050Measurements(&measurements, repoRoot, timeout, runner); err != nil {
		return nil, err
	}
	return measurements, nil
}

func appendRunnerCheckMeasurements(
	measurements *[]perfcontracts.MeasurementRecord,
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, args ...string) (float64, error),
) error {
	items, err := collectRunnerCheckMeasurements(repoRoot, timeout, runner)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func appendWorkflowMeasurements(
	measurements *[]perfcontracts.MeasurementRecord,
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, args ...string) (float64, error),
) error {
	items, err := collectMinimalWorkflowMeasurements(repoRoot, timeout, runner)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func appendCHG050Measurements(
	measurements *[]perfcontracts.MeasurementRecord,
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, args ...string) (float64, error),
) error {
	items, err := measureCHG050CompileAndLoad(repoRoot, runner, timeout)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func resolveRepoRoot(explicit string) (string, error) {
	repoRoot := strings.TrimSpace(explicit)
	if repoRoot != "" {
		return validateRepoRoot(repoRoot)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return validateRepoRoot(cwd)
}

func validateRepoRoot(root string) (string, error) {
	clean := filepath.Clean(root)
	if _, err := os.Stat(filepath.Join(clean, "runner", "package.json")); err != nil {
		return "", fmt.Errorf("repository root missing runner/package.json: %w", err)
	}
	if _, err := os.Stat(filepath.Join(clean, "protocol", "schemas")); err != nil {
		return "", fmt.Errorf("repository root missing protocol/schemas: %w", err)
	}
	if _, err := projectsubstrate.DiscoverAndValidate(projectsubstrate.DiscoveryInput{RepositoryRoot: clean, Authority: projectsubstrate.RepoRootAuthorityExplicitConfig}); err != nil {
		return "", fmt.Errorf("repository root validation failed: %w", err)
	}
	return clean, nil
}
