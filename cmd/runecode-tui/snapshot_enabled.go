//go:build runecode_tui_snapshot

package main

import (
	"flag"
	"fmt"
	"strings"
)

type snapshotViewportPreset string

const (
	snapshotViewportDesktop snapshotViewportPreset = "desktop"
	snapshotViewportCompact snapshotViewportPreset = "compact"
	snapshotViewportMobile  snapshotViewportPreset = "mobile"
)

type snapshotViewportSpec struct {
	Name   snapshotViewportPreset
	Width  int
	Height int
}

type tuiSnapshotConfig struct {
	enabled   bool
	scenario  string
	bundle    string
	outputDir string
	viewport  snapshotViewportPreset
	width     int
	height    int
	theme     themePreset
}

type tuiSnapshotFlags struct {
	scenario  *string
	bundle    *string
	outputDir *string
	viewport  *string
	width     *int
	height    *int
	theme     *string
}

func registerSnapshotFlags(fs *flag.FlagSet) tuiSnapshotFlags {
	return tuiSnapshotFlags{
		scenario:  fs.String("snapshot-scenario", "", "hidden dev option: deterministic TUI snapshot scenario"),
		bundle:    fs.String("snapshot-bundle", "", "hidden dev option: snapshot audit bundle"),
		outputDir: fs.String("snapshot-output-dir", snapshotDefaultOutputDir(), "hidden dev option: snapshot output directory"),
		viewport:  fs.String("snapshot-viewport", string(snapshotViewportDesktop), "hidden dev option: snapshot viewport preset"),
		width:     fs.Int("snapshot-width", 0, "hidden dev option: snapshot terminal width override"),
		height:    fs.Int("snapshot-height", 0, "hidden dev option: snapshot terminal height override"),
		theme:     fs.String("snapshot-theme", string(themePresetDark), "hidden dev option: snapshot theme preset"),
	}
}

func applySnapshotCLIConfig(cfg *tuiCLIConfig, flags tuiSnapshotFlags) {
	if *flags.scenario == "" && *flags.bundle == "" {
		return
	}
	cfg.snapshot = tuiSnapshotConfig{
		enabled:   true,
		scenario:  *flags.scenario,
		bundle:    *flags.bundle,
		outputDir: *flags.outputDir,
		viewport:  snapshotViewportPreset(strings.TrimSpace(*flags.viewport)),
		width:     *flags.width,
		height:    *flags.height,
		theme:     normalizeThemePreset(themePreset(*flags.theme)),
	}
}

func resolveSnapshotViewportPreset(name snapshotViewportPreset) (snapshotViewportSpec, error) {
	switch snapshotViewportPreset(strings.TrimSpace(string(name))) {
	case "", snapshotViewportDesktop:
		return snapshotViewportSpec{Name: snapshotViewportDesktop, Width: 160, Height: 48}, nil
	case snapshotViewportCompact:
		return snapshotViewportSpec{Name: snapshotViewportCompact, Width: 120, Height: 36}, nil
	case snapshotViewportMobile:
		return snapshotViewportSpec{Name: snapshotViewportMobile, Width: 80, Height: 32}, nil
	default:
		return snapshotViewportSpec{}, fmt.Errorf("unknown snapshot viewport %q", name)
	}
}
