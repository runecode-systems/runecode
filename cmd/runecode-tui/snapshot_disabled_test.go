//go:build !runecode_tui_snapshot

package main

import (
	"strings"
	"testing"
)

func TestParseCLIConfigRejectsSnapshotFlagsWithoutBuildTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "scenario", args: []string{"--snapshot-scenario", "all"}},
		{name: "output-dir", args: []string{"--snapshot-output-dir", "/tmp/snaps"}},
		{name: "viewport", args: []string{"--snapshot-viewport", "mobile"}},
		{name: "width", args: []string{"--snapshot-width", "140"}},
		{name: "height", args: []string{"--snapshot-height", "40"}},
		{name: "theme", args: []string{"--snapshot-theme", "dusk"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCLIConfig(tc.args)
			if err == nil {
				t.Fatal("parseCLIConfig expected error")
			}
			if _, ok := err.(*usageError); !ok {
				t.Fatalf("parseCLIConfig error type = %T, want *usageError", err)
			}
			if got := err.Error(); got != "runecode-tui usage: runecode-tui [--runtime-dir dir] [--socket-name broker.sock] [--help]" {
				t.Fatalf("parseCLIConfig error = %q", got)
			}
		})
	}
}

func TestWriteSnapshotArtifactsReturnsGracefulErrorWithoutBuildTag(t *testing.T) {
	t.Parallel()

	err := writeSnapshotArtifacts(tuiSnapshotConfig{enabled: true})
	if err == nil {
		t.Fatal("writeSnapshotArtifacts returned nil error, want graceful failure")
	}
	if !strings.Contains(err.Error(), "runecode_tui_snapshot") {
		t.Fatalf("writeSnapshotArtifacts error = %q, want build-tag guidance", err)
	}
}
