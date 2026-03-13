// Command checksourcequality enforces RuneCode source-quality guardrails.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintf(os.Stderr, "source-quality usage error: %v\n", err)
			os.Exit(2)
		}

		fmt.Fprintln(os.Stderr, "Source quality check failed:")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	root, err := parseRootArg(args)
	if err != nil {
		return err
	}

	cfg, err := loadRuntimeConfig(root)
	if err != nil {
		return err
	}

	files, err := collectEligibleFiles(root, cfg)
	if err != nil {
		return err
	}

	violations, err := checkFiles(files, cfg)
	if err != nil {
		return err
	}

	if err := reportViolations(violations); err != nil {
		return err
	}

	fmt.Printf("Source quality check passed (%d files scanned).\n", len(files))
	return nil
}

func parseRootArg(args []string) (string, error) {
	fs := flag.NewFlagSet("checksourcequality", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	repoRoot := fs.String("root", ".", "repository root to scan")
	if err := fs.Parse(args); err != nil {
		return "", usageError{err: err}
	}

	root, err := filepath.Abs(*repoRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}
	if err := validateRootWithinWorkspace(root); err != nil {
		return "", err
	}

	return root, nil
}

func reportViolations(violations []violation) error {
	if len(violations) == 0 {
		return nil
	}

	for _, violation := range violations {
		fmt.Fprintf(os.Stderr, "- [%s] %s\n", violation.rule, violation.format())
	}

	return fmt.Errorf("%d source-quality violation(s)", len(violations))
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

func validateRootWithinWorkspace(root string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}

	workspaceRoot, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("resolve workspace root: %w", err)
	}

	rel, err := filepath.Rel(workspaceRoot, root)
	if err != nil {
		return usageError{err: fmt.Errorf("root must stay within %s", workspaceRoot)}
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return usageError{err: fmt.Errorf("root must stay within %s", workspaceRoot)}
	}

	return nil
}
