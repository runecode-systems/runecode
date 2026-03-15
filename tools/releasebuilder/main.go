// Command releasebuilder creates deterministic release artifacts for Nix builds.
package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintf(os.Stderr, "releasebuilder usage error: %v\n", err)
			os.Exit(2)
		}

		fmt.Fprintf(os.Stderr, "releasebuilder failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError{err: fmt.Errorf("expected subcommand: zip or manifest")}
	}

	switch args[0] {
	case "zip":
		return runZip(args[1:])
	case "manifest":
		return runManifest(args[1:])
	default:
		return usageError{err: fmt.Errorf("unknown subcommand %q", args[0])}
	}
}

type usageError struct {
	err error
}

func (e usageError) Error() string {
	return e.err.Error()
}

func (e usageError) Unwrap() error {
	return e.err
}
