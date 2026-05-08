package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/runnerworkflowperf"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintf(os.Stderr, "runnerworkflowperf usage error: %v\n", err)
			os.Exit(2)
		}
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
		return usageError{err: err}
	}
	if strings.TrimSpace(*output) == "" {
		return usageError{err: fmt.Errorf("--output is required")}
	}
	if strings.TrimSpace(*repositoryRoot) == "" {
		return usageError{err: fmt.Errorf("--repository-root is required")}
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

type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }

func (e usageError) Unwrap() error { return e.err }
