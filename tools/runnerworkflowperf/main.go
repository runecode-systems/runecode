package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/runnerworkflowperf"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "runnerworkflowperf failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("runnerworkflowperf", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "", "output check json path")
	repositoryRoot := fs.String("repository-root", "", "repository root")
	timeoutMs := fs.Int("timeout-ms", 120000, "per-command timeout milliseconds")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--output is required")
	}
	if strings.TrimSpace(*repositoryRoot) == "" {
		return fmt.Errorf("--repository-root is required")
	}
	out, err := runnerworkflowperf.Run(runnerworkflowperf.HarnessConfig{
		RepositoryRoot: strings.TrimSpace(*repositoryRoot),
		CommandTimeout: time.Duration(*timeoutMs) * time.Millisecond,
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
