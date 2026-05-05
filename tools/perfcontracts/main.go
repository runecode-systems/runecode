package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

type config struct {
	contractsRoot string
	checkOutput   string
	lane          string
	metricIDs     []string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		var usage usageError
		if errors.As(err, &usage) {
			fmt.Fprintf(os.Stderr, "perfcontracts usage error: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "perfcontracts check failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return err
	}
	manifest, inventory, contracts, baselinesByMetric, err := loadContractSet(cfg.contractsRoot)
	if err != nil {
		return err
	}
	if err := perfcontracts.ValidateWithBaselines(manifest, inventory, contracts, baselinesByMetric); err != nil {
		return err
	}
	checkOutput, err := perfcontracts.LoadCheckOutput(cfg.checkOutput)
	if err != nil {
		return err
	}
	filtered := filterContractsForLane(contracts, cfg.lane, cfg.metricIDs)
	if countMetrics(filtered) == 0 {
		return fmt.Errorf("no required metrics selected for lane %q", cfg.lane)
	}
	violations := perfcontracts.Evaluate(checkOutput, filtered, baselinesByMetric)
	if len(violations) > 0 {
		for _, violation := range violations {
			fmt.Fprintf(os.Stderr, "- %s: %s\n", violation.MetricID, violation.Reason)
		}
		return fmt.Errorf("%d performance contract violation(s)", len(violations))
	}
	fmt.Printf("Performance contracts check passed (%d metrics evaluated).\n", countMetrics(filtered))
	return nil
}

func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("perfcontracts", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	contractsRoot := fs.String("contracts-root", filepath.FromSlash("tools/perfcontracts"), "path to performance contracts root")
	checkOutput := fs.String("check-output", "", "path to performance check output json")
	lane := fs.String("lane", "required_shared_linux", "lane authority to enforce")
	metricIDFlags := multiStringFlag{}
	fs.Var(&metricIDFlags, "metric-id", "optional metric_id to enforce; repeatable")
	if err := fs.Parse(args); err != nil {
		return config{}, usageError{err}
	}
	if strings.TrimSpace(*checkOutput) == "" {
		return config{}, usageError{errors.New("--check-output is required")}
	}
	return config{contractsRoot: strings.TrimSpace(*contractsRoot), checkOutput: strings.TrimSpace(*checkOutput), lane: strings.TrimSpace(*lane), metricIDs: metricIDFlags.values()}, nil
}

func loadContractSet(root string) (perfcontracts.Manifest, perfcontracts.FixtureInventory, []perfcontracts.ContractFile, map[string]perfcontracts.BaselineFile, error) {
	manifest, err := perfcontracts.LoadManifest(root)
	if err != nil {
		return perfcontracts.Manifest{}, perfcontracts.FixtureInventory{}, nil, nil, err
	}
	inventory, err := perfcontracts.LoadFixtureInventory(root, manifest.FixtureInventoryRef)
	if err != nil {
		return perfcontracts.Manifest{}, perfcontracts.FixtureInventory{}, nil, nil, err
	}
	contracts := make([]perfcontracts.ContractFile, 0, len(manifest.Contracts))
	for _, entry := range manifest.Contracts {
		contract, loadErr := perfcontracts.LoadContract(root, entry.Path)
		if loadErr != nil {
			return perfcontracts.Manifest{}, perfcontracts.FixtureInventory{}, nil, nil, loadErr
		}
		contracts = append(contracts, contract)
	}
	baselinesByMetric := map[string]perfcontracts.BaselineFile{}
	for _, entry := range manifest.Baselines {
		baseline, loadErr := perfcontracts.LoadBaseline(root, entry.Path)
		if loadErr != nil {
			return perfcontracts.Manifest{}, perfcontracts.FixtureInventory{}, nil, nil, loadErr
		}
		baselinesByMetric[entry.MetricID] = baseline
	}
	return manifest, inventory, contracts, baselinesByMetric, nil
}

func filterContractsForLane(contracts []perfcontracts.ContractFile, lane string, metricIDs []string) []perfcontracts.ContractFile {
	allowedMetrics := metricFilterSet(metricIDs)

	var filtered []perfcontracts.ContractFile
	for _, contract := range contracts {
		next := perfcontracts.ContractFile{SchemaVersion: contract.SchemaVersion, ContractID: contract.ContractID, Surface: contract.Surface}
		for _, metric := range contract.Metrics {
			if includeMetric(metric, lane, allowedMetrics) {
				next.Metrics = append(next.Metrics, metric)
			}
		}
		if len(next.Metrics) > 0 {
			filtered = append(filtered, next)
		}
	}
	return filtered
}

func metricFilterSet(metricIDs []string) map[string]struct{} {
	allowedMetrics := map[string]struct{}{}
	for _, metricID := range metricIDs {
		trimmed := strings.TrimSpace(metricID)
		if trimmed != "" {
			allowedMetrics[trimmed] = struct{}{}
		}
	}
	return allowedMetrics
}

func includeMetric(metric perfcontracts.MetricContract, lane string, allowedMetrics map[string]struct{}) bool {
	if metric.LaneAuthority != lane || metric.ActivationState != "required" {
		return false
	}
	if len(allowedMetrics) == 0 {
		return true
	}
	_, ok := allowedMetrics[metric.MetricID]
	return ok
}

func countMetrics(contracts []perfcontracts.ContractFile) int {
	total := 0
	for _, contract := range contracts {
		total += len(contract.Metrics)
	}
	return total
}

type usageError struct{ err error }

func (u usageError) Error() string { return u.err.Error() }

func (u usageError) Unwrap() error { return u.err }

type multiStringFlag struct{ items []string }

func (m *multiStringFlag) String() string {
	if m == nil {
		return ""
	}
	return strings.Join(m.items, ",")
}

func (m *multiStringFlag) Set(value string) error {
	m.items = append(m.items, value)
	return nil
}

func (m *multiStringFlag) values() []string {
	out := make([]string, len(m.items))
	copy(out, m.items)
	return out
}
