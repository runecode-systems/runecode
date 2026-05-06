package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	runRunnerWorkflow func(repoRoot string, timeout time.Duration) (perfcontracts.CheckOutput, error)
	runBrokerPerf     func(repoRoot string, trials int) (perfcontracts.CheckOutput, error)
	runPhase5Perf     func(repoRoot string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error)
	runTUIQuiet       func(repoRoot string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error)
	runTUIWaiting     func(repoRoot string, trials int, timeout time.Duration) (perfcontracts.CheckOutput, error)
	runTUIBench       func(repoRoot string, timeout time.Duration) (perfcontracts.MeasurementRecord, error)
	listRequiredIDs   func(repoRoot string) ([]string, error)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintf(os.Stderr, "perfgatesharedlinux usage error: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "perfgatesharedlinux failed: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("perfgatesharedlinux", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "", "output check json path")
	repository := fs.String("repository-root", "", "repository root path")
	trials := fs.Int("trials", 30, "deterministic broker unary trials")
	timeoutMs := fs.Int("timeout-ms", 120000, "runner command timeout milliseconds")
	if err := fs.Parse(args); err != nil {
		return config{}, usageError{err: err}
	}
	if strings.TrimSpace(*output) == "" {
		return config{}, usageError{err: fmt.Errorf("--output is required")}
	}
	if *trials <= 0 {
		return config{}, usageError{err: fmt.Errorf("--trials must be > 0")}
	}
	timeout := time.Duration(*timeoutMs) * time.Millisecond
	if timeout <= 0 {
		return config{}, usageError{err: fmt.Errorf("--timeout-ms must be > 0")}
	}
	return config{
		outputPath: strings.TrimSpace(*output),
		repository: strings.TrimSpace(*repository),
		trials:     *trials,
		timeout:    timeout,
	}, nil
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

type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }

func (e usageError) Unwrap() error { return e.err }
