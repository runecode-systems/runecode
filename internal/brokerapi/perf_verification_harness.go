package brokerapi

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

const phase5PerfCheckSchemaVersion = "runecode.performance.check.v1"

type Phase5PerformanceHarnessConfig struct {
	RepositoryRoot string
	Trials         int
	CommandTimeout time.Duration
	CommandRunner  func(repoRoot string, timeout time.Duration, command ...string) (float64, error)
}

func RunPhase5PerformanceHarness(cfg Phase5PerformanceHarnessConfig) (perfcontracts.CheckOutput, error) {
	trials := phase5ResolvedTrials(cfg.Trials)
	timeout := phase5ResolvedTimeout(cfg.CommandTimeout)
	repoRoot, err := phase5ResolveRepoRoot(cfg.RepositoryRoot)
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	runner := cfg.CommandRunner
	if runner == nil {
		runner = phase5RunCommand
	}

	measurements, err := phase5CollectMeasurements(trials, repoRoot, timeout, runner)
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	return perfcontracts.CheckOutput{SchemaVersion: phase5PerfCheckSchemaVersion, Measurements: measurements}, nil
}

func phase5ResolvedTrials(trials int) int {
	if trials <= 0 {
		return 10
	}
	return trials
}

func phase5ResolvedTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 2 * time.Minute
	}
	return timeout
}

func phase5ResolveRepoRoot(explicit string) (string, error) {
	repoRoot := strings.TrimSpace(explicit)
	if repoRoot != "" {
		return repoRoot, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return cwd, nil
}

func phase5CollectMeasurements(
	trials int,
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, command ...string) (float64, error),
) ([]perfcontracts.MeasurementRecord, error) {
	measurements := make([]perfcontracts.MeasurementRecord, 0, 24)

	if err := phase5AppendGatewayAndSecrets(&measurements, trials, repoRoot); err != nil {
		return nil, err
	}
	if err := phase5AppendDependencyFlow(&measurements, trials, repoRoot); err != nil {
		return nil, err
	}
	if err := phase5AppendAuditVerification(&measurements, repoRoot, timeout, runner); err != nil {
		return nil, err
	}
	if err := phase5AppendProtocolChecks(&measurements, repoRoot, timeout, runner); err != nil {
		return nil, err
	}
	measurements = append(measurements, measurePhase5ExternalAnchorStubbed(trials)...)
	return measurements, nil
}

func phase5AppendGatewayAndSecrets(measurements *[]perfcontracts.MeasurementRecord, trials int, repoRoot string) error {
	items, err := measurePhase5GatewayAndSecrets(trials, repoRoot)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func phase5AppendDependencyFlow(measurements *[]perfcontracts.MeasurementRecord, trials int, repoRoot string) error {
	items, err := measurePhase5DependencyFlow(trials, repoRoot)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func phase5AppendAuditVerification(
	measurements *[]perfcontracts.MeasurementRecord,
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, command ...string) (float64, error),
) error {
	items, err := measurePhase5AuditVerification(repoRoot, timeout, runner)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func phase5AppendProtocolChecks(
	measurements *[]perfcontracts.MeasurementRecord,
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, command ...string) (float64, error),
) error {
	items, err := measurePhase5ProtocolChecks(repoRoot, timeout, runner)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func phase5DependencyMetric(metricID string, value float64, unit string) perfcontracts.MeasurementRecord {
	return perfcontracts.MeasurementRecord{MetricID: metricID, Value: value, Unit: unit}
}

func phase5DependencyErr(action string, errResp *ErrorResponse) error {
	if errResp == nil {
		return nil
	}
	return fmt.Errorf("%s: %s", action, errResp.Error.Code)
}
