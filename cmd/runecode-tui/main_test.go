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
		"Usage: runecode-tui [--runtime-dir dir] [--socket-name broker.sock] [--help]",
		"Interactive terminal UI for the local RuneCode broker API.",
		"runecode attach",
		"isolated manual/dev workflows",
	} {
		if !strings.Contains(written, want) {
			t.Fatalf("help output missing %q in %q", want, written)
		}
	}
}

func TestParseCLIConfigRejectsUnexpectedFlagsWithUsageError(t *testing.T) {
	_, err := parseCLIConfig([]string{"--verbose"})
	if err == nil {
		t.Fatal("parseCLIConfig expected error")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("parseCLIConfig error type = %T, want *usageError", err)
	}
	if got := err.Error(); got != "runecode-tui usage: runecode-tui [--runtime-dir dir] [--socket-name broker.sock] [--help]" {
		t.Fatalf("parseCLIConfig error = %q", got)
	}
}

func TestParseCLIConfigParsesIPCOverrides(t *testing.T) {
	cfg, err := parseCLIConfig([]string{"--runtime-dir", "/tmp/runtime", "--socket-name", "broker.dev.sock"})
	if err != nil {
		t.Fatalf("parseCLIConfig returned error: %v", err)
	}
	if cfg.runtimeDir != "/tmp/runtime" || cfg.socketName != "broker.dev.sock" {
		t.Fatalf("parseCLIConfig cfg = %+v, want runtime+socket overrides", cfg)
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
		"Canonical entry: runecode attach",
	} {
		if !strings.Contains(written, want) {
			t.Fatalf("non-interactive output missing %q in %q", want, written)
		}
	}
}
