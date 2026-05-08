package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherperf"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintf(os.Stderr, "launcherperf usage error: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "launcherperf failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("launcherperf", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "", "output check json path")
	if err := fs.Parse(args); err != nil {
		return usageError{err: err}
	}
	if strings.TrimSpace(*output) == "" {
		return usageError{err: fmt.Errorf("--output is required")}
	}
	out, err := launcherperf.Run(launcherperf.HarnessConfig{})
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
