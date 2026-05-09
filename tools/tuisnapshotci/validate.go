package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func validateSnapshotArtifacts(outputDir string) error {
	resolvedOutputDir, err := canonicalPath(outputDir)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	if err := requirePathWithinTrustedRoot("output directory", resolvedOutputDir); err != nil {
		return err
	}
	outputDir = resolvedOutputDir

	manifest, err := readSnapshotManifest(outputDir)
	if err != nil {
		return err
	}
	entriesByName, err := validateManifestStructure(manifest)
	if err != nil {
		return err
	}
	return validateScenarioArtifacts(outputDir, entriesByName)
}

func readSnapshotManifest(outputDir string) (snapshotManifest, error) {
	manifestPath := filepath.Join(outputDir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return snapshotManifest{}, fmt.Errorf("read manifest %q: %w", manifestPath, err)
	}
	var manifest snapshotManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return snapshotManifest{}, fmt.Errorf("decode manifest %q: %w", manifestPath, err)
	}
	return manifest, nil
}

func validateManifestStructure(manifest snapshotManifest) (map[string]snapshotManifestEntry, error) {
	if manifest.Version != manifestVersion {
		return nil, fmt.Errorf("manifest version = %d, want %d", manifest.Version, manifestVersion)
	}
	if manifest.Bundle != snapshotBundle {
		return nil, fmt.Errorf("manifest bundle = %q, want %q", manifest.Bundle, snapshotBundle)
	}
	if manifest.Viewport != snapshotViewport {
		return nil, fmt.Errorf("manifest viewport = %q, want %q", manifest.Viewport, snapshotViewport)
	}
	if manifest.Width <= 0 || manifest.Height <= 0 {
		return nil, fmt.Errorf("manifest dimensions must be positive, got %dx%d", manifest.Width, manifest.Height)
	}
	if err := validateCoverage(manifest.Coverage); err != nil {
		return nil, err
	}
	entriesByName, err := manifestEntriesByName(manifest.Scenarios)
	if err != nil {
		return nil, err
	}
	if err := validateScenarioSet(entriesByName); err != nil {
		return nil, err
	}
	return entriesByName, nil
}

func validateCoverage(coverage snapshotCoverage) error {
	if coverage.Bundle != snapshotBundle {
		return fmt.Errorf("coverage bundle = %q, want %q", coverage.Bundle, snapshotBundle)
	}
	if len(coverage.Viewports) != 1 || coverage.Viewports[0] != snapshotViewport {
		return fmt.Errorf("coverage viewports = %v, want [%s]", coverage.Viewports, snapshotViewport)
	}
	routes := append([]string(nil), coverage.Routes...)
	slices.Sort(routes)
	if !slices.Equal(routes, expectedCoverageRoutes()) {
		return fmt.Errorf("coverage routes = %v, want %v", routes, expectedCoverageRoutes())
	}
	scenarios := append([]string(nil), coverage.Scenarios...)
	slices.Sort(scenarios)
	wantScenarios := append([]string(nil), expectedScenarioNames...)
	slices.Sort(wantScenarios)
	if !slices.Equal(scenarios, wantScenarios) {
		return fmt.Errorf("coverage scenarios = %v, want %v", scenarios, wantScenarios)
	}
	return nil
}

func expectedCoverageRoutes() []string {
	routes := []string{"action-center", "approvals", "audit", "chat", "dashboard", "git-remote-mutation", "git-setup", "model-providers", "runs", "status"}
	slices.Sort(routes)
	return routes
}

func manifestEntriesByName(entries []snapshotManifestEntry) (map[string]snapshotManifestEntry, error) {
	entriesByName := make(map[string]snapshotManifestEntry, len(entries))
	for _, entry := range entries {
		if err := validateScenarioName(entry.Name); err != nil {
			return nil, err
		}
		if _, exists := entriesByName[entry.Name]; exists {
			return nil, fmt.Errorf("manifest contains duplicate scenario entries: %s", entry.Name)
		}
		entriesByName[entry.Name] = entry
	}
	return entriesByName, nil
}

func validateScenarioName(name string) error {
	if strings.ContainsAny(name, `/\\`) {
		return fmt.Errorf("manifest scenario name %q must not contain path separators", name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("manifest scenario name %q must not be a path traversal component", name)
	}
	return nil
}

func validateScenarioSet(entriesByName map[string]snapshotManifestEntry) error {
	var missing []string
	for _, expected := range expectedScenarioNames {
		if _, ok := entriesByName[expected]; !ok {
			missing = append(missing, expected)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("manifest missing expected scenarios: %s", strings.Join(missing, ", "))
	}

	var unexpected []string
	for name := range entriesByName {
		if !slices.Contains(expectedScenarioNames, name) {
			unexpected = append(unexpected, name)
		}
	}
	if len(unexpected) > 0 {
		slices.Sort(unexpected)
		return fmt.Errorf("manifest contains unexpected scenarios: %s", strings.Join(unexpected, ", "))
	}
	return nil
}

func validateScenarioArtifacts(outputDir string, entriesByName map[string]snapshotManifestEntry) error {
	for _, scenario := range expectedScenarioNames {
		entry := entriesByName[scenario]
		if err := validateScenarioEntry(outputDir, entry); err != nil {
			return fmt.Errorf("scenario %q: %w", scenario, err)
		}
	}
	return nil
}

func validateScenarioEntry(outputDir string, entry snapshotManifestEntry) error {
	if err := validateScenarioName(entry.Name); err != nil {
		return fmt.Errorf("scenario name: %w", err)
	}
	if entry.Viewport != snapshotViewport {
		return fmt.Errorf("viewport = %q, want %q", entry.Viewport, snapshotViewport)
	}
	if entry.Width <= 0 || entry.Height <= 0 {
		return fmt.Errorf("dimensions must be positive, got %dx%d", entry.Width, entry.Height)
	}
	if strings.TrimSpace(entry.Route) == "" {
		return errors.New("route must be present")
	}

	base := entry.Name + "." + snapshotViewport
	if err := validateHashedArtifact(outputDir, entry.Artifacts.ANSI, base+".ansi"); err != nil {
		return fmt.Errorf("ansi artifact: %w", err)
	}
	if err := validateHashedArtifact(outputDir, entry.Artifacts.Text, base+".txt"); err != nil {
		return fmt.Errorf("text artifact: %w", err)
	}
	if err := validateHashedArtifact(outputDir, entry.Artifacts.SVG, base+".svg"); err != nil {
		return fmt.Errorf("svg artifact: %w", err)
	}
	if err := validateOptionalPNGArtifact(outputDir, entry.Artifacts.PNG, base+".png"); err != nil {
		return fmt.Errorf("png artifact: %w", err)
	}
	return nil
}

func validateHashedArtifact(outputDir string, artifact snapshotManifestArtifact, expectedPath string) error {
	if artifact.Path != expectedPath {
		return fmt.Errorf("path = %q, want %q", artifact.Path, expectedPath)
	}
	if strings.TrimSpace(artifact.SHA256) == "" {
		return errors.New("sha256 must be present")
	}
	content, err := os.ReadFile(filepath.Join(outputDir, expectedPath))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(content)
	if artifact.SHA256 != hex.EncodeToString(sum[:]) {
		return fmt.Errorf("sha256 = %q, want %q", artifact.SHA256, hex.EncodeToString(sum[:]))
	}
	return nil
}

func validateOptionalPNGArtifact(outputDir string, artifact snapshotManifestArtifact, expectedPath string) error {
	if artifact.Path != expectedPath {
		return fmt.Errorf("path = %q, want %q", artifact.Path, expectedPath)
	}
	if artifact.SHA256 != "" {
		return fmt.Errorf("sha256 = %q, want empty for optional PNG", artifact.SHA256)
	}
	info, err := os.Lstat(filepath.Join(outputDir, expectedPath))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("optional PNG must be a regular file when present")
	}
	return nil
}
