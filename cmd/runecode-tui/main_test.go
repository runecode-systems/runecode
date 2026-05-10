package main

import (
	"bytes"
	"os"
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

func TestParseCLIConfigShowsHelpWhenCombinedWithOtherFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--help", "--runtime-dir", "/tmp/runtime"},
		{"--runtime-dir", "/tmp/runtime", "--help"},
		{"--help", "--verbose"},
	} {
		cfg, err := parseCLIConfig(args)
		if err != nil {
			t.Fatalf("parseCLIConfig(%v) returned error: %v", args, err)
		}
		if !cfg.showHelp {
			t.Fatalf("parseCLIConfig(%v) cfg = %+v, want showHelp", args, cfg)
		}
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

type flushTrackingWorkbenchStore struct {
	memoryWorkbenchStateStore
	flushed bool
	err     error
}

func (s *flushTrackingWorkbenchStore) Flush() {
	s.flushed = true
}

func (s *flushTrackingWorkbenchStore) LastError() error {
	return s.err
}

func TestRunMainFlushesWorkbenchStateBeforeExit(t *testing.T) {
	store := &flushTrackingWorkbenchStore{}
	origModel := newShellModelFunc
	origTerm := isTerminalFunc
	origRunner := runShellProgram
	newShellModelFunc = func() shellModel { return newShellModelWithWorkbenchStore(store) }
	isTerminalFunc = func(int) bool { return true }
	runShellProgram = func(model shellModel) (shellModel, error) { return model, nil }
	defer func() {
		newShellModelFunc = origModel
		isTerminalFunc = origTerm
		runShellProgram = origRunner
	}()

	var stderr bytes.Buffer
	code := runMain(nil, os.Stdin, os.Stdout, &stderr)
	if code != 0 {
		t.Fatalf("runMain() exit code = %d stderr=%q", code, stderr.String())
	}
	if !store.flushed {
		t.Fatal("expected runMain to flush workbench state before exit")
	}
}

func TestRunMainReturnsFailureWhenWorkbenchFlushFails(t *testing.T) {
	store := &flushTrackingWorkbenchStore{err: errForcedFlushFailure{}}
	origModel := newShellModelFunc
	origTerm := isTerminalFunc
	origRunner := runShellProgram
	newShellModelFunc = func() shellModel { return newShellModelWithWorkbenchStore(store) }
	isTerminalFunc = func(int) bool { return true }
	runShellProgram = func(model shellModel) (shellModel, error) { return model, nil }
	defer func() {
		newShellModelFunc = origModel
		isTerminalFunc = origTerm
		runShellProgram = origRunner
	}()

	var stderr bytes.Buffer
	code := runMain(nil, os.Stdin, os.Stdout, &stderr)
	if code != 1 {
		t.Fatalf("runMain() exit code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "runecode-tui failed") {
		t.Fatalf("expected flush failure on stderr, got %q", stderr.String())
	}
}

type errForcedFlushFailure struct{}

func (errForcedFlushFailure) Error() string { return "forced flush failure" }
