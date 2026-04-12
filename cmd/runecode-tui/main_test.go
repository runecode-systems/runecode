package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteHelpDescribesInteractiveBrokerBackedUI(t *testing.T) {
	var out bytes.Buffer
	if err := writeHelp(&out); err != nil {
		t.Fatalf("writeHelp returned error: %v", err)
	}
	written := out.String()
	for _, want := range []string{
		"Usage: runecode-tui [--help]",
		"Interactive terminal UI for the local RuneCode broker API.",
		"runecode-broker serve-local",
	} {
		if !strings.Contains(written, want) {
			t.Fatalf("help output missing %q in %q", want, written)
		}
	}
}

func TestValidateArgsRejectsUnexpectedFlagsWithUsageError(t *testing.T) {
	err := validateArgs([]string{"--verbose"})
	if err == nil {
		t.Fatal("validateArgs expected error")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("validateArgs error type = %T, want *usageError", err)
	}
	if got := err.Error(); got != "runecode-tui accepts no arguments; use --help for usage" {
		t.Fatalf("validateArgs error = %q", got)
	}
}

func TestWriteNonInteractiveMessageIncludesBrokerRemediation(t *testing.T) {
	var out bytes.Buffer
	if err := writeNonInteractiveMessage(&out); err != nil {
		t.Fatalf("writeNonInteractiveMessage returned error: %v", err)
	}
	written := out.String()
	for _, want := range []string{
		"interactive terminal UI",
		"Interactive terminal required to launch UI.",
		"runecode-broker serve-local",
	} {
		if !strings.Contains(written, want) {
			t.Fatalf("non-interactive output missing %q in %q", want, written)
		}
	}
}
