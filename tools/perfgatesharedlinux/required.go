package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func requiredSharedLinuxMetricIDs(repoRoot string) ([]string, error) {
	root := filepath.Join(repoRoot, "tools", "perfcontracts")
	manifest, err := perfcontracts.LoadManifest(root)
	if err != nil {
		return nil, err
	}
	required, err := requiredMetricSet(root, manifest.Contracts)
	if err != nil {
		return nil, err
	}
	return sortedMetricIDs(required), nil
}

func requiredMetricSet(root string, entries []perfcontracts.ManifestContract) (map[string]struct{}, error) {
	required := map[string]struct{}{}
	for _, entry := range entries {
		contract, err := perfcontracts.LoadContract(root, entry.Path)
		if err != nil {
			return nil, err
		}
		collectRequiredMetrics(required, contract.Metrics)
	}
	return required, nil
}

func collectRequiredMetrics(required map[string]struct{}, metrics []perfcontracts.MetricContract) {
	for _, metric := range metrics {
		if metric.LaneAuthority == "required_shared_linux" && metric.ActivationState == "required" {
			required[metric.MetricID] = struct{}{}
		}
	}
}

func sortedMetricIDs(required map[string]struct{}) []string {
	out := make([]string, 0, len(required))
	for metricID := range required {
		out = append(out, metricID)
	}
	sort.Strings(out)
	return out
}

func selectRequiredMeasurements(measurements []perfcontracts.MeasurementRecord, requiredIDs []string) ([]perfcontracts.MeasurementRecord, error) {
	byMetric := map[string]perfcontracts.MeasurementRecord{}
	for _, measurement := range measurements {
		byMetric[measurement.MetricID] = measurement
	}
	selected := make([]perfcontracts.MeasurementRecord, 0, len(requiredIDs))
	missing := make([]string, 0)
	for _, metricID := range requiredIDs {
		measurement, ok := byMetric[metricID]
		if !ok {
			missing = append(missing, metricID)
			continue
		}
		selected = append(selected, measurement)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("required_shared_linux metrics missing from aggregated output: %s", strings.Join(missing, ", "))
	}
	return selected, nil
}
