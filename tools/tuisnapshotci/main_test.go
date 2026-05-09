package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLocalOutputDirDefaultsToTempDir(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	if err := run([]string{"local-output-dir"}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	want := filepath.Join(os.TempDir(), localSnapshotDirName)
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Fatalf("local-output-dir = %q, want %q", got, want)
	}
}

func TestRunCleanupLocalDirRemovesTrustedSnapshotDir(t *testing.T) {
	outputDir := filepath.Join(os.TempDir(), "runecode-custom-cleanup", t.Name())
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "artifact.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	if err := run([]string{"cleanup-local-dir", "--output-dir", outputDir}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Fatalf("outputDir still exists after cleanup, stat err = %v", err)
	}
}

func TestRunCleanupLocalDirRejectsTrustedTempRoot(t *testing.T) {
	outputDir := os.TempDir()
	var stdout strings.Builder
	var stderr strings.Builder
	err := run([]string{"cleanup-local-dir", "--output-dir", outputDir}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run returned nil error, want cleanup trust failure")
	}
	if !strings.Contains(err.Error(), "must not be the trusted temporary root") {
		t.Fatalf("run error = %v, want cleanup trust failure", err)
	}
}

func TestValidateSnapshotArtifactsAcceptsExpectedManifestWithoutPNGFiles(t *testing.T) {
	outputDir := t.TempDir()
	manifest := snapshotManifest{
		Version:  manifestVersion,
		Bundle:   snapshotBundle,
		Viewport: snapshotViewport,
		Width:    160,
		Height:   48,
		Coverage: testCoverage(),
	}
	for _, scenario := range expectedScenarioNames {
		base := scenario + "." + snapshotViewport
		manifest.Scenarios = append(manifest.Scenarios, snapshotManifestEntry{
			Name:     scenario,
			Viewport: snapshotViewport,
			Width:    160,
			Height:   48,
			Route:    "dashboard",
			Artifacts: snapshotManifestArtifacts{
				ANSI: writeTestArtifact(t, outputDir, base+".ansi", scenario+" ansi\n"),
				Text: writeTestArtifact(t, outputDir, base+".txt", scenario+" text\n"),
				SVG:  writeTestArtifact(t, outputDir, base+".svg", "<svg>"+scenario+"</svg>\n"),
				PNG:  snapshotManifestArtifact{Path: base + ".png"},
			},
		})
	}
	writeManifest(t, outputDir, manifest)

	if err := validateSnapshotArtifacts(outputDir); err != nil {
		t.Fatalf("validateSnapshotArtifacts returned error: %v", err)
	}
}

func TestRunRejectsPositionalArguments(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	err := run([]string{"unexpected"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run returned nil error, want positional argument failure")
	}
	if !strings.Contains(err.Error(), "accepts no positional arguments") {
		t.Fatalf("run error = %v, want positional argument failure", err)
	}
}

func TestValidateSnapshotArtifactsAcceptsSymlinkedOutputDir(t *testing.T) {
	realOutputDir := t.TempDir()
	linkRoot := t.TempDir()
	symlinkPath := filepath.Join(linkRoot, "snapshots-link")
	if err := os.Symlink(realOutputDir, symlinkPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	manifest := snapshotManifest{
		Version:  manifestVersion,
		Bundle:   snapshotBundle,
		Viewport: snapshotViewport,
		Width:    160,
		Height:   48,
		Coverage: testCoverage(),
	}
	for _, scenario := range expectedScenarioNames {
		base := scenario + "." + snapshotViewport
		manifest.Scenarios = append(manifest.Scenarios, snapshotManifestEntry{
			Name:     scenario,
			Viewport: snapshotViewport,
			Width:    160,
			Height:   48,
			Route:    "dashboard",
			Artifacts: snapshotManifestArtifacts{
				ANSI: writeTestArtifact(t, realOutputDir, base+".ansi", scenario+" ansi\n"),
				Text: writeTestArtifact(t, realOutputDir, base+".txt", scenario+" text\n"),
				SVG:  writeTestArtifact(t, realOutputDir, base+".svg", "<svg>"+scenario+"</svg>\n"),
				PNG:  snapshotManifestArtifact{Path: base + ".png"},
			},
		})
	}
	writeManifest(t, realOutputDir, manifest)

	if err := validateSnapshotArtifacts(symlinkPath); err != nil {
		t.Fatalf("validateSnapshotArtifacts returned error for symlink path: %v", err)
	}
}

func TestValidateSnapshotArtifactsRejectsMissingScenarioEntries(t *testing.T) {
	outputDir := t.TempDir()
	base := expectedScenarioNames[0] + "." + snapshotViewport
	manifest := snapshotManifest{
		Version:  manifestVersion,
		Bundle:   snapshotBundle,
		Viewport: snapshotViewport,
		Width:    160,
		Height:   48,
		Coverage: testCoverage(),
		Scenarios: []snapshotManifestEntry{{
			Name:     expectedScenarioNames[0],
			Viewport: snapshotViewport,
			Width:    160,
			Height:   48,
			Route:    "dashboard",
			Artifacts: snapshotManifestArtifacts{
				ANSI: writeTestArtifact(t, outputDir, base+".ansi", "ansi\n"),
				Text: writeTestArtifact(t, outputDir, base+".txt", "text\n"),
				SVG:  writeTestArtifact(t, outputDir, base+".svg", "<svg/>\n"),
				PNG:  snapshotManifestArtifact{Path: base + ".png"},
			},
		}},
	}
	writeManifest(t, outputDir, manifest)

	err := validateSnapshotArtifacts(outputDir)
	if err == nil {
		t.Fatal("validateSnapshotArtifacts returned nil error, want missing scenario failure")
	}
	if !strings.Contains(err.Error(), "missing expected scenarios") {
		t.Fatalf("validateSnapshotArtifacts error = %v, want missing scenario failure", err)
	}
}

func TestValidateSnapshotArtifactsRejectsDuplicateScenarioEntries(t *testing.T) {
	outputDir := t.TempDir()
	manifest := testManifestForAllScenarios(t, outputDir)
	manifest.Scenarios = append(manifest.Scenarios, duplicateScenarioEntry(t, outputDir, expectedScenarioNames[0]))
	writeManifest(t, outputDir, manifest)

	err := validateSnapshotArtifacts(outputDir)
	if err == nil {
		t.Fatal("validateSnapshotArtifacts returned nil error, want duplicate scenario failure")
	}
	if !strings.Contains(err.Error(), "duplicate scenario entries") {
		t.Fatalf("validateSnapshotArtifacts error = %v, want duplicate scenario failure", err)
	}
}

func TestValidateSnapshotArtifactsRejectsAbsoluteArtifactPaths(t *testing.T) {
	outputDir := t.TempDir()
	manifest := testManifestForAllScenarios(t, outputDir)
	manifest.Scenarios[0].Artifacts.SVG.Path = filepath.Join(outputDir, expectedScenarioNames[0]+"."+snapshotViewport+".svg")
	writeManifest(t, outputDir, manifest)

	err := validateSnapshotArtifacts(outputDir)
	if err == nil {
		t.Fatal("validateSnapshotArtifacts returned nil error, want absolute path failure")
	}
	if !strings.Contains(err.Error(), "want \""+expectedScenarioNames[0]+"."+snapshotViewport+".svg\"") {
		t.Fatalf("validateSnapshotArtifacts error = %v, want relative path expectation", err)
	}
}

func TestValidateSnapshotArtifactsRejectsScenarioNamesWithSeparatorsOrTraversal(t *testing.T) {
	tests := []struct {
		name         string
		scenarioName string
		want         string
	}{
		{name: "slash separator", scenarioName: "nested/name", want: "must not contain path separators"},
		{name: "backslash separator", scenarioName: `nested\\name`, want: "must not contain path separators"},
		{name: "dot traversal", scenarioName: "..", want: "must not be a path traversal component"},
		{name: "separator and traversal", scenarioName: "../escape", want: "must not contain path separators"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := t.TempDir()
			manifest := testManifestForAllScenarios(t, outputDir)
			manifest.Scenarios[0].Name = tt.scenarioName
			writeManifest(t, outputDir, manifest)

			err := validateSnapshotArtifacts(outputDir)
			if err == nil {
				t.Fatal("validateSnapshotArtifacts returned nil error, want invalid scenario name failure")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateSnapshotArtifacts error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateScenarioEntryRejectsInvalidScenarioNameBeforeArtifactValidation(t *testing.T) {
	err := validateScenarioEntry(t.TempDir(), snapshotManifestEntry{
		Name:     "../escape",
		Viewport: snapshotViewport,
		Width:    160,
		Height:   48,
		Route:    "dashboard",
	})
	if err == nil {
		t.Fatal("validateScenarioEntry returned nil error, want invalid scenario name failure")
	}
	if !strings.Contains(err.Error(), "scenario name") || !strings.Contains(err.Error(), "must not contain path separators") {
		t.Fatalf("validateScenarioEntry error = %v, want explicit scenario name validation failure", err)
	}
}

func TestValidateSnapshotArtifactsRejectsPNGDirectory(t *testing.T) {
	outputDir := t.TempDir()
	manifest := testManifestForAllScenarios(t, outputDir)
	pngPath := filepath.Join(outputDir, manifest.Scenarios[0].Artifacts.PNG.Path)
	if err := os.Mkdir(pngPath, 0o755); err != nil {
		t.Fatalf("Mkdir(%s) returned error: %v", pngPath, err)
	}
	writeManifest(t, outputDir, manifest)

	err := validateSnapshotArtifacts(outputDir)
	if err == nil {
		t.Fatal("validateSnapshotArtifacts returned nil error, want PNG regular file failure")
	}
	if !strings.Contains(err.Error(), "optional PNG must be a regular file when present") {
		t.Fatalf("validateSnapshotArtifacts error = %v, want PNG regular file failure", err)
	}
}

func TestPrepareWorkDirCanonicalizesSymlinkedOutputDir(t *testing.T) {
	realRoot := t.TempDir()
	linkRoot := t.TempDir()
	symlinkPath := filepath.Join(linkRoot, "workspace-link")
	if err := os.Symlink(realRoot, symlinkPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	requestedOutputDir := filepath.Join(symlinkPath, "nested", "artifacts")
	got, cleanup, err := prepareWorkDir(requestedOutputDir)
	if err != nil {
		t.Fatalf("prepareWorkDir returned error: %v", err)
	}
	defer cleanup()

	want := filepath.Join(realRoot, "nested", "artifacts")
	if got != want {
		t.Fatalf("prepareWorkDir returned %q, want %q", got, want)
	}
}

func TestPrepareWorkDirRejectsOutputDirOutsideTrustedRoot(t *testing.T) {
	workspaceDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	outsideDir := filepath.Join(workspaceDir, ".tmp-tuisnapshotci-outside-workdir")
	defer os.RemoveAll(outsideDir)
	outsideDir, err = filepath.Abs(outsideDir)
	if err != nil {
		t.Fatalf("Abs returned error: %v", err)
	}
	got, cleanup, err := prepareWorkDir(outsideDir)
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Fatalf("prepareWorkDir returned path %q, want trusted root failure", got)
	}
	if !strings.Contains(err.Error(), "output directory must stay under") {
		t.Fatalf("prepareWorkDir error = %v, want trusted root failure", err)
	}
}

func TestPrepareArtifactsDirRejectsOutputDirOutsideTrustedRoot(t *testing.T) {
	workspaceDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	outsideDir := filepath.Join(workspaceDir, ".tmp-tuisnapshotci-outside-artifacts")
	defer os.RemoveAll(outsideDir)
	outsideDir, err = filepath.Abs(outsideDir)
	if err != nil {
		t.Fatalf("Abs returned error: %v", err)
	}
	got, err := prepareArtifactsDir(outsideDir, outsideDir)
	if err == nil {
		t.Fatalf("prepareArtifactsDir returned path %q, want trusted root failure", got)
	}
	if !strings.Contains(err.Error(), "artifact directory must stay under") {
		t.Fatalf("prepareArtifactsDir error = %v, want trusted root failure", err)
	}
}

func TestRequirePathWithinTrustedRootAcceptsCanonicalTmp(t *testing.T) {
	if _, err := os.Stat("/tmp"); err != nil {
		t.Skip("/tmp unavailable on this platform")
	}
	candidate := filepath.Join("/tmp", "runecode-tui-snapshots")
	if err := requirePathWithinTrustedRoot("artifact directory", candidate); err != nil {
		t.Fatalf("requirePathWithinTrustedRoot returned error for /tmp path: %v", err)
	}
}

func testManifestForAllScenarios(t *testing.T, outputDir string) snapshotManifest {
	t.Helper()
	manifest := snapshotManifest{
		Version:  manifestVersion,
		Bundle:   snapshotBundle,
		Viewport: snapshotViewport,
		Width:    160,
		Height:   48,
		Coverage: testCoverage(),
	}
	for _, scenario := range expectedScenarioNames {
		manifest.Scenarios = append(manifest.Scenarios, scenarioEntryForDir(t, outputDir, scenario))
	}
	return manifest
}

func scenarioEntryForDir(t *testing.T, outputDir string, scenario string) snapshotManifestEntry {
	t.Helper()
	base := scenario + "." + snapshotViewport
	return snapshotManifestEntry{
		Name:     scenario,
		Viewport: snapshotViewport,
		Width:    160,
		Height:   48,
		Route:    routeForScenario(scenario),
		Artifacts: snapshotManifestArtifacts{
			ANSI: writeTestArtifact(t, outputDir, base+".ansi", scenario+" ansi\n"),
			Text: writeTestArtifact(t, outputDir, base+".txt", scenario+" text\n"),
			SVG:  writeTestArtifact(t, outputDir, base+".svg", "<svg>"+scenario+"</svg>\n"),
			PNG:  snapshotManifestArtifact{Path: base + ".png"},
		},
	}
}

func testCoverage() snapshotCoverage {
	return snapshotCoverage{Bundle: snapshotBundle, Routes: expectedCoverageRoutes(), Viewports: []string{snapshotViewport}, Scenarios: append([]string(nil), expectedScenarioNames...)}
}

func routeForScenario(scenario string) string {
	switch scenario {
	case "dashboard-healthy-empty", "dashboard-approval-waiting", "dashboard-blocked", "dashboard-degraded":
		return "dashboard"
	case "chat-active-session":
		return "chat"
	case "runs-active-detail":
		return "runs"
	case "approvals-pending-detail":
		return "approvals"
	case "action-center-triage":
		return "action-center"
	case "audit-degraded-detail":
		return "audit"
	case "status-ready-overview":
		return "status"
	case "model-providers-credential-needed":
		return "model-providers"
	case "git-setup-identity-needed":
		return "git-setup"
	case "git-remote-approval-ready":
		return "git-remote-mutation"
	default:
		return "dashboard"
	}
}

func duplicateScenarioEntry(t *testing.T, outputDir string, scenario string) snapshotManifestEntry {
	t.Helper()
	base := scenario + "-duplicate." + snapshotViewport
	return snapshotManifestEntry{
		Name:     scenario,
		Viewport: snapshotViewport,
		Width:    160,
		Height:   48,
		Route:    "dashboard-duplicate",
		Artifacts: snapshotManifestArtifacts{
			ANSI: writeTestArtifact(t, outputDir, base+".ansi", scenario+" duplicate ansi\n"),
			Text: writeTestArtifact(t, outputDir, base+".txt", scenario+" duplicate text\n"),
			SVG:  writeTestArtifact(t, outputDir, base+".svg", "<svg>"+scenario+" duplicate</svg>\n"),
			PNG:  snapshotManifestArtifact{Path: base + ".png"},
		},
	}
}

func writeManifest(t *testing.T, outputDir string, manifest snapshotManifest) {
	t.Helper()
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "manifest.json"), raw, 0o644); err != nil {
		t.Fatalf("WriteFile(manifest.json) returned error: %v", err)
	}
}

func writeTestArtifact(t *testing.T, outputDir string, relativePath string, content string) snapshotManifestArtifact {
	t.Helper()
	path := filepath.Join(outputDir, relativePath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
	sum := sha256.Sum256([]byte(content))
	return snapshotManifestArtifact{Path: relativePath, SHA256: hex.EncodeToString(sum[:])}
}
