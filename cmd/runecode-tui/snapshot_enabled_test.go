//go:build runecode_tui_snapshot

package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var snapshotSHA256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func TestParseCLIConfigParsesSnapshotOptionsWithBuildTag(t *testing.T) {
	cfg, err := parseCLIConfig([]string{"--snapshot-scenario", "dashboard", "--snapshot-output-dir", "/tmp/snaps", "--snapshot-width", "140", "--snapshot-height", "40", "--snapshot-theme", "dusk", "--snapshot-viewport", "mobile"})
	if err != nil {
		t.Fatalf("parseCLIConfig returned error: %v", err)
	}
	if !cfg.snapshot.enabled || cfg.snapshot.scenario != "dashboard" || cfg.snapshot.outputDir != "/tmp/snaps" || cfg.snapshot.viewport != snapshotViewportMobile || cfg.snapshot.width != 140 || cfg.snapshot.height != 40 || cfg.snapshot.theme != themePresetDusk {
		t.Fatalf("parseCLIConfig snapshot cfg = %+v, want snapshot options populated", cfg.snapshot)
	}
}

func TestParseCLIConfigParsesSnapshotBundleWithBuildTag(t *testing.T) {
	cfg, err := parseCLIConfig([]string{"--snapshot-bundle", "full-audit", "--snapshot-output-dir", "/tmp/snaps"})
	if err != nil {
		t.Fatalf("parseCLIConfig returned error: %v", err)
	}
	if !cfg.snapshot.enabled || cfg.snapshot.bundle != "full-audit" || cfg.snapshot.outputDir != "/tmp/snaps" {
		t.Fatalf("parseCLIConfig snapshot cfg = %+v, want bundle populated", cfg.snapshot)
	}
}

func TestNormalizeSnapshotConfigUsesViewportPresetDefaults(t *testing.T) {
	cfg, err := normalizeSnapshotConfig(tuiSnapshotConfig{enabled: true, scenario: "dashboard", viewport: snapshotViewportCompact})
	if err != nil {
		t.Fatalf("normalizeSnapshotConfig returned error: %v", err)
	}
	if cfg.viewport != snapshotViewportCompact || cfg.width != 120 || cfg.height != 36 {
		t.Fatalf("normalizeSnapshotConfig cfg = %+v, want compact preset dimensions", cfg)
	}
}

func TestNormalizeSnapshotConfigKeepsExplicitDimensionOverrides(t *testing.T) {
	cfg, err := normalizeSnapshotConfig(tuiSnapshotConfig{enabled: true, scenario: "dashboard", viewport: snapshotViewportMobile, width: 90, height: 28})
	if err != nil {
		t.Fatalf("normalizeSnapshotConfig returned error: %v", err)
	}
	if cfg.viewport != snapshotViewportMobile || cfg.width != 90 || cfg.height != 28 {
		t.Fatalf("normalizeSnapshotConfig cfg = %+v, want explicit overrides preserved", cfg)
	}
}

func TestNormalizeSnapshotConfigAllowsRealTempPathWhenTempDirIsSymlink(t *testing.T) {
	baseDir := t.TempDir()
	realTempDir := filepath.Join(baseDir, "real-temp")
	if err := os.MkdirAll(realTempDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(realTempDir) returned error: %v", err)
	}
	linkedTempDir := filepath.Join(baseDir, "temp-link")
	if err := os.Symlink(realTempDir, linkedTempDir); err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink creation not permitted: %v", err)
		}
		t.Fatalf("Symlink(linkedTempDir) returned error: %v", err)
	}
	t.Setenv("TMPDIR", linkedTempDir)

	outputDir := filepath.Join(realTempDir, "snapshots")
	cfg, err := normalizeSnapshotConfig(tuiSnapshotConfig{enabled: true, scenario: "dashboard", outputDir: outputDir, viewport: snapshotViewportCompact})
	if err != nil {
		t.Fatalf("normalizeSnapshotConfig returned error: %v", err)
	}
	if cfg.outputDir != outputDir {
		t.Fatalf("normalizeSnapshotConfig outputDir = %q, want %q", cfg.outputDir, outputDir)
	}
}

func TestNormalizeSnapshotConfigAllowsCanonicalTmpWhenTempDirDiffers(t *testing.T) {
	if _, err := os.Stat("/tmp"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("/tmp is not present on this platform")
		}
		t.Fatalf("Stat(/tmp) returned error: %v", err)
	}

	customTempDir := filepath.Join(t.TempDir(), "custom-temp")
	if err := os.MkdirAll(customTempDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(customTempDir) returned error: %v", err)
	}
	t.Setenv("TMPDIR", customTempDir)

	outputDir := filepath.Join("/tmp", ".tui-snapshots", t.Name())
	cfg, err := normalizeSnapshotConfig(tuiSnapshotConfig{enabled: true, scenario: "dashboard", outputDir: outputDir, viewport: snapshotViewportCompact})
	if err != nil {
		t.Fatalf("normalizeSnapshotConfig returned error: %v", err)
	}
	wantOutputDir, err := snapshotCanonicalPath(outputDir)
	if err != nil {
		t.Fatalf("snapshotCanonicalPath returned error: %v", err)
	}
	if cfg.outputDir != wantOutputDir {
		t.Fatalf("normalizeSnapshotConfig outputDir = %q, want %q", cfg.outputDir, wantOutputDir)
	}
}

func TestNormalizeSnapshotConfigDefaultsToRepoSnapshotDir(t *testing.T) {
	cfg, err := normalizeSnapshotConfig(tuiSnapshotConfig{enabled: true, scenario: "dashboard", viewport: snapshotViewportCompact})
	if err != nil {
		t.Fatalf("normalizeSnapshotConfig returned error: %v", err)
	}
	wantOutputDir, err := snapshotCanonicalPath(snapshotDefaultOutputDir())
	if err != nil {
		t.Fatalf("snapshotCanonicalPath(snapshotDefaultOutputDir()) returned error: %v", err)
	}
	if cfg.outputDir != wantOutputDir {
		t.Fatalf("normalizeSnapshotConfig outputDir = %q, want %q", cfg.outputDir, wantOutputDir)
	}
}

func TestNormalizeSnapshotConfigRejectsOutputDirEscapingTempViaSymlink(t *testing.T) {
	baseDir := t.TempDir()
	realTempDir := filepath.Join(baseDir, "real-temp")
	if err := os.MkdirAll(realTempDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(realTempDir) returned error: %v", err)
	}
	linkedTempDir := filepath.Join(baseDir, "temp-link")
	if err := os.Symlink(realTempDir, linkedTempDir); err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink creation not permitted: %v", err)
		}
		t.Fatalf("Symlink(linkedTempDir) returned error: %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	outsideDir, err := os.MkdirTemp(wd, "snapshot-outside-")
	if err != nil {
		t.Fatalf("MkdirTemp(outsideDir) returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(outsideDir)
	})
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(outsideDir) returned error: %v", err)
	}
	escapeLink := filepath.Join(realTempDir, "escape")
	if err := os.Symlink(outsideDir, escapeLink); err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink creation not permitted: %v", err)
		}
		t.Fatalf("Symlink(escapeLink) returned error: %v", err)
	}
	t.Setenv("TMPDIR", linkedTempDir)

	_, err = normalizeSnapshotConfig(tuiSnapshotConfig{enabled: true, scenario: "dashboard", outputDir: filepath.Join(escapeLink, "snapshots"), viewport: snapshotViewportCompact})
	if err == nil {
		t.Fatalf("normalizeSnapshotConfig returned nil error, want rejection for symlink escape")
	}
}

func TestWriteSnapshotArtifactsIncludesPerScenarioViewportAndDimensions(t *testing.T) {
	outputDir := t.TempDir()
	if err := writeSnapshotArtifacts(tuiSnapshotConfig{enabled: true, scenario: "dashboard-healthy-empty", outputDir: outputDir, viewport: snapshotViewportMobile}); err != nil {
		t.Fatalf("writeSnapshotArtifacts returned error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(outputDir, "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile(manifest.json) returned error: %v", err)
	}
	var manifest snapshotManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("json.Unmarshal(manifest.json) returned error: %v", err)
	}
	if manifest.Viewport != string(snapshotViewportMobile) || manifest.Width != 80 || manifest.Height != 32 {
		t.Fatalf("manifest = %+v, want mobile top-level metadata", manifest)
	}
	if manifest.Version != 2 {
		t.Fatalf("manifest version = %d, want 2", manifest.Version)
	}
	if len(manifest.Scenarios) != 1 {
		t.Fatalf("manifest scenarios len = %d, want 1", len(manifest.Scenarios))
	}
	entry := manifest.Scenarios[0]
	if entry.Viewport != string(snapshotViewportMobile) || entry.Width != 80 || entry.Height != 32 {
		t.Fatalf("manifest entry = %+v, want mobile per-scenario dimensions", entry)
	}
	if entry.Artifacts.SVG.Path != "dashboard-healthy-empty.mobile.svg" || entry.Artifacts.Text.Path != "dashboard-healthy-empty.mobile.txt" || entry.Artifacts.ANSI.Path != "dashboard-healthy-empty.mobile.ansi" || entry.Artifacts.PNG.Path != "dashboard-healthy-empty.mobile.png" {
		t.Fatalf("manifest entry paths = %+v, want relative viewport-qualified artifact names", entry)
	}
	if !snapshotSHA256Pattern.MatchString(entry.Artifacts.ANSI.SHA256) || !snapshotSHA256Pattern.MatchString(entry.Artifacts.Text.SHA256) || !snapshotSHA256Pattern.MatchString(entry.Artifacts.SVG.SHA256) {
		t.Fatalf("manifest artifact hashes = %+v, want sha256 hex digests", entry.Artifacts)
	}
	if entry.Artifacts.PNG.SHA256 != "" {
		t.Fatalf("manifest png hash = %q, want empty when png is not generated by Go tool", entry.Artifacts.PNG.SHA256)
	}
}

func TestWriteSnapshotArtifactsIncludesBundleCoverageMetadata(t *testing.T) {
	outputDir := t.TempDir()
	if err := writeSnapshotArtifacts(tuiSnapshotConfig{enabled: true, bundle: "dashboard-audit", outputDir: outputDir, viewport: snapshotViewportDesktop}); err != nil {
		t.Fatalf("writeSnapshotArtifacts returned error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(outputDir, "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile(manifest.json) returned error: %v", err)
	}
	var manifest snapshotManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("json.Unmarshal(manifest.json) returned error: %v", err)
	}
	if manifest.Bundle != "dashboard-audit" || manifest.Coverage.Bundle != "dashboard-audit" {
		t.Fatalf("manifest bundle coverage = %+v, want dashboard-audit", manifest)
	}
	if len(manifest.Coverage.Routes) != 1 || manifest.Coverage.Routes[0] != "dashboard" {
		t.Fatalf("manifest coverage routes = %+v, want dashboard only", manifest.Coverage)
	}
	if len(manifest.Coverage.Scenarios) != 6 {
		t.Fatalf("manifest coverage scenarios = %+v, want 6 dashboard scenarios", manifest.Coverage)
	}
}

func TestWriteSnapshotArtifactsRemovesStalePNG(t *testing.T) {
	outputDir := t.TempDir()
	stalePNG := filepath.Join(outputDir, "dashboard-healthy-empty.mobile.png")
	if err := os.WriteFile(stalePNG, []byte("stale"), 0o644); err != nil {
		t.Fatalf("WriteFile(stalePNG) returned error: %v", err)
	}

	if err := writeSnapshotArtifacts(tuiSnapshotConfig{enabled: true, scenario: "dashboard-healthy-empty", outputDir: outputDir, viewport: snapshotViewportMobile}); err != nil {
		t.Fatalf("writeSnapshotArtifacts returned error: %v", err)
	}
	if _, err := os.Stat(stalePNG); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(stalePNG) error = %v, want not exist after regeneration", err)
	}
}
