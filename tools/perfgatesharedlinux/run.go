package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/brokerperf"
	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/runnerworkflowperf"
)

func run(args []string) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return err
	}
	return runWithDeps(cfg, deps{
		runRunnerWorkflow: func(repoRoot string, timeout time.Duration) (perfcontracts.CheckOutput, error) {
			return runnerworkflowperf.Run(runnerworkflowperf.HarnessConfig{RepositoryRoot: repoRoot, CommandTimeout: timeout})
		},
		runBrokerPerf: func(repoRoot string, trials int) (perfcontracts.CheckOutput, error) {
			return brokerperf.Run(brokerperf.HarnessConfig{RepositoryRoot: repoRoot, Trials: trials})
		},
		runPhase5Perf: func(repoRoot string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error) {
			return brokerapi.RunPhase5PerformanceHarness(brokerapi.Phase5PerformanceHarnessConfig{RepositoryRoot: repoRoot, Trials: trials, CommandTimeout: timeout})
		},
		runTUIQuiet: func(repoRoot string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error) {
			return measureTUILatency(repoRoot, "tui.empty.v1", trials, timeout)
		},
		runTUIWaiting: func(repoRoot string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error) {
			return measureTUILatency(repoRoot, "tui.waiting.v1", trials, timeout)
		},
		runTUIBench:     measureTUIRenderWaitingBenchmark,
		listRequiredIDs: requiredSharedLinuxMetricIDs,
	})
}

func runWithDeps(cfg config, d deps) error {
	repoRoot, err := resolveRepoRoot(cfg.repository)
	if err != nil {
		return err
	}
	measurements, err := collectMeasurements(cfg, repoRoot, d)
	if err != nil {
		return err
	}
	requiredIDs, err := d.listRequiredIDs(repoRoot)
	if err != nil {
		return err
	}
	measurements, err = selectRequiredMeasurements(measurements, requiredIDs)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(perfcontracts.CheckOutput{SchemaVersion: checkSchemaVersion, Measurements: measurements}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.outputPath, raw, 0o644)
}

func collectMeasurements(cfg config, repoRoot string, d deps) ([]perfcontracts.MeasurementRecord, error) {
	runnerWorkflow, err := d.runRunnerWorkflow(repoRoot, cfg.timeout)
	if err != nil {
		return nil, fmt.Errorf("run runner workflow perf: %w", err)
	}
	brokerOut, err := d.runBrokerPerf(repoRoot, cfg.trials)
	if err != nil {
		return nil, fmt.Errorf("run broker perf: %w", err)
	}
	phase5Out, err := d.runPhase5Perf(repoRoot, cfg.trials, cfg.timeout)
	if err != nil {
		return nil, fmt.Errorf("run phase5 perf: %w", err)
	}
	tuiQuietOut, err := d.runTUIQuiet(repoRoot, cfg.trials, cfg.timeout)
	if err != nil {
		return nil, fmt.Errorf("run tui latency (quiet) perf: %w", err)
	}
	tuiWaitingOut, err := d.runTUIWaiting(repoRoot, cfg.trials, cfg.timeout)
	if err != nil {
		return nil, fmt.Errorf("run tui latency (waiting) perf: %w", err)
	}
	tuiBench, err := d.runTUIBench(repoRoot, cfg.timeout)
	if err != nil {
		return nil, fmt.Errorf("run tui benchmark perf: %w", err)
	}
	return mergedMeasurements(tuiBench, runnerWorkflow, brokerOut, phase5Out, tuiQuietOut, tuiWaitingOut), nil
}

func mergedMeasurements(tuiBench perfcontracts.MeasurementRecord, outputs ...perfcontracts.CheckOutput) []perfcontracts.MeasurementRecord {
	total := 1
	for _, output := range outputs {
		total += len(output.Measurements)
	}
	measurements := make([]perfcontracts.MeasurementRecord, 0, total)
	for _, output := range outputs {
		measurements = append(measurements, output.Measurements...)
	}
	return append(measurements, tuiBench)
}
