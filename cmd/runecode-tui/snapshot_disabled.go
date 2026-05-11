//go:build !runecode_tui_snapshot

package main

import (
	"flag"
	"fmt"
)

type tuiSnapshotConfig struct {
	enabled  bool
	viewport snapshotViewportPreset
}

type snapshotViewportPreset string

type tuiSnapshotFlags struct{}

func registerSnapshotFlags(*flag.FlagSet) tuiSnapshotFlags {
	return tuiSnapshotFlags{}
}

func applySnapshotCLIConfig(*tuiCLIConfig, tuiSnapshotFlags) {}

func writeSnapshotArtifacts(tuiSnapshotConfig) error {
	return fmt.Errorf("snapshot support requires the runecode_tui_snapshot build tag")
}
