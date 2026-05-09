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

func TestParseCLIConfigParsesSnapshotOptions(t *testing.T) {
	cfg, err := parseCLIConfig([]string{"--snapshot-scenario", "dashboard", "--snapshot-output-dir", "/tmp/snaps", "--snapshot-width", "140", "--snapshot-height", "40", "--snapshot-theme", "dusk"})
	if err != nil {
		t.Fatalf("parseCLIConfig returned error: %v", err)
	}
	if !cfg.snapshot.enabled || cfg.snapshot.scenario != "dashboard" || cfg.snapshot.outputDir != "/tmp/snaps" || cfg.snapshot.width != 140 || cfg.snapshot.height != 40 || cfg.snapshot.theme != themePresetDusk {
		t.Fatalf("parseCLIConfig snapshot cfg = %+v, want snapshot options populated", cfg.snapshot)
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
