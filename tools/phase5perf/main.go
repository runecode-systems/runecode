package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "phase5perf failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("phase5perf", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "", "output check json path")
	repositoryRoot := fs.String("repository-root", "", "repository root")
	trials := fs.Int("trials", 10, "deterministic trial count for p95 metrics")
	timeoutMS := fs.Int("timeout-ms", 120000, "per-command timeout milliseconds")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--output is required")
	}
	out, err := brokerapi.RunPhase5PerformanceHarness(brokerapi.Phase5PerformanceHarnessConfig{
		RepositoryRoot: strings.TrimSpace(*repositoryRoot),
		Trials:         *trials,
		CommandTimeout: time.Duration(*timeoutMS) * time.Millisecond,
	})
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(strings.TrimSpace(*output), raw, 0o644)
}
