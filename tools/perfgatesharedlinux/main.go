package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerperf"
	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

const checkSchemaVersion = "runecode.performance.check.v1"

type config struct {
	outputPath string
	repository string
	trials     int
	timeout    time.Duration
}

type deps struct {
	runRunnerBoundary func(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error)
	runRunnerFixtures func(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error)
	runBrokerUnary    func(repoRoot string, trials int) (perfcontracts.MeasurementRecord, error)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "perfgatesharedlinux failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return err
	}
	return runWithDeps(cfg, deps{
		runRunnerBoundary: func(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error) {
			return measureRunnerCommand(repoRoot, timeout, "metric.runner.boundary_check.wall_ms", "npm", "run", "boundary-check")
		},
		runRunnerFixtures: func(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error) {
			return measureRunnerCommand(repoRoot, timeout, "metric.runner.protocol_fixtures.wall_ms", "node", "--test", "scripts/protocol-fixtures.test.js")
		},
		runBrokerUnary: measureBrokerUnarySessionList,
	})
}

func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("perfgatesharedlinux", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "", "output check json path")
	repository := fs.String("repository-root", "", "repository root path")
	trials := fs.Int("trials", 30, "deterministic broker unary trials")
	timeoutMs := fs.Int("timeout-ms", 120000, "runner command timeout milliseconds")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if strings.TrimSpace(*output) == "" {
		return config{}, fmt.Errorf("--output is required")
	}
	if *trials <= 0 {
		return config{}, fmt.Errorf("--trials must be > 0")
	}
	timeout := time.Duration(*timeoutMs) * time.Millisecond
	if timeout <= 0 {
		return config{}, fmt.Errorf("--timeout-ms must be > 0")
	}
	return config{
		outputPath: strings.TrimSpace(*output),
		repository: strings.TrimSpace(*repository),
		trials:     *trials,
		timeout:    timeout,
	}, nil
}

func runWithDeps(cfg config, d deps) error {
	repoRoot, err := resolveRepoRoot(cfg.repository)
	if err != nil {
		return err
	}
	runnerBoundary, err := d.runRunnerBoundary(repoRoot, cfg.timeout)
	if err != nil {
		return fmt.Errorf("run runner boundary-check perf: %w", err)
	}
	runnerFixtures, err := d.runRunnerFixtures(repoRoot, cfg.timeout)
	if err != nil {
		return fmt.Errorf("run runner protocol-fixtures perf: %w", err)
	}
	brokerUnary, err := d.runBrokerUnary(repoRoot, cfg.trials)
	if err != nil {
		return fmt.Errorf("run broker unary perf: %w", err)
	}
	measurements := []perfcontracts.MeasurementRecord{runnerBoundary, runnerFixtures, brokerUnary}

	raw, err := json.MarshalIndent(perfcontracts.CheckOutput{SchemaVersion: checkSchemaVersion, Measurements: measurements}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.outputPath, raw, 0o644)
}

func measureRunnerCommand(repoRoot string, timeout time.Duration, metricID string, args ...string) (perfcontracts.MeasurementRecord, error) {
	if len(args) == 0 {
		return perfcontracts.MeasurementRecord{}, fmt.Errorf("command arguments required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = filepath.Join(filepath.Clean(repoRoot), "runner")
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	start := time.Now()
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return perfcontracts.MeasurementRecord{}, fmt.Errorf("%s failed: %s", strings.Join(args, " "), msg)
	}
	return perfcontracts.MeasurementRecord{MetricID: metricID, Value: float64(time.Since(start).Milliseconds()), Unit: "ms"}, nil
}

func resolveRepoRoot(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return validateRepoRoot(strings.TrimSpace(explicit))
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

func measureBrokerUnarySessionList(repoRoot string, trials int) (perfcontracts.MeasurementRecord, error) {
	out, err := brokerperf.Run(brokerperf.HarnessConfig{RepositoryRoot: repoRoot, Trials: trials})
	if err != nil {
		return perfcontracts.MeasurementRecord{}, err
	}
	for _, measurement := range out.Measurements {
		if measurement.MetricID == "metric.broker.unary.session_list.p95_ms" {
			return measurement, nil
		}
	}
	return perfcontracts.MeasurementRecord{}, fmt.Errorf("metric.broker.unary.session_list.p95_ms missing from broker harness output")
}
