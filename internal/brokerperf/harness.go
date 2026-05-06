package brokerperf

import (
	"fmt"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

const CheckSchemaVersion = "runecode.performance.check.v1"

type HarnessConfig struct {
	Trials         int
	RepositoryRoot string
}

type latencySpec struct {
	metricID string
	call     func() error
}

type watchSpec struct {
	latencyMetricID string
	payloadMetricID string
	countMetricID   string
	call            func() (any, error)
}

func Run(cfg HarnessConfig) (perfcontracts.CheckOutput, error) {
	repoRoot, err := brokerPerfResolveRepoRoot(cfg.RepositoryRoot)
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	trials := brokerPerfResolvedTrials(cfg.Trials)
	measurements, err := brokerPerfCollectMeasurements(trials, repoRoot)
	if err != nil {
		return perfcontracts.CheckOutput{}, err
	}
	return perfcontracts.CheckOutput{SchemaVersion: CheckSchemaVersion, Measurements: measurements}, nil
}

func brokerPerfResolveRepoRoot(explicit string) (string, error) {
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

func brokerPerfResolvedTrials(trials int) int {
	if trials <= 0 {
		return 30
	}
	return trials
}

func brokerPerfCollectMeasurements(trials int, repoRoot string) ([]perfcontracts.MeasurementRecord, error) {
	measurements := make([]perfcontracts.MeasurementRecord, 0, 26)
	if err := brokerPerfAppendUnary(&measurements, trials, repoRoot); err != nil {
		return nil, err
	}
	if err := brokerPerfAppendWatches(&measurements, trials, repoRoot); err != nil {
		return nil, err
	}
	if err := brokerPerfAppendMutations(&measurements, trials, repoRoot); err != nil {
		return nil, err
	}
	if err := brokerPerfAppendAttachResume(&measurements, trials, repoRoot); err != nil {
		return nil, err
	}
	return measurements, nil
}

func brokerPerfAppendUnary(measurements *[]perfcontracts.MeasurementRecord, trials int, repoRoot string) error {
	items, err := measureUnary(trials, repoRoot)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func brokerPerfAppendWatches(measurements *[]perfcontracts.MeasurementRecord, trials int, repoRoot string) error {
	items, err := measureWatches(trials, repoRoot)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func brokerPerfAppendMutations(measurements *[]perfcontracts.MeasurementRecord, trials int, repoRoot string) error {
	items, err := measureMutations(trials, repoRoot)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func brokerPerfAppendAttachResume(measurements *[]perfcontracts.MeasurementRecord, trials int, repoRoot string) error {
	items, err := measureAttachResume(trials, repoRoot)
	if err != nil {
		return err
	}
	*measurements = append(*measurements, items...)
	return nil
}

func unaryErr(errResp *brokerapi.ErrorResponse, label string) error {
	if errResp == nil {
		return nil
	}
	return fmt.Errorf("%s: %s", label, errResp.Error.Code)
}
