package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	baselineFileName = ".source-quality-baseline.json"
	configFileName   = ".source-quality-config.json"
)

type runtimeConfig struct {
	repoRoot                   string
	runnerTier1Paths           map[string]struct{}
	tier1SuppressionExceptions map[string]struct{}
	baseline                   map[string]baselineEntry
}

type sourceQualityConfig struct {
	RunnerTier1Paths           []string `json:"runnerTier1Paths"`
	Tier1SuppressionExceptions []string `json:"tier1SuppressionExceptions"`
}

type baselineEntry struct {
	Kind                   string `json:"kind"`
	MaxSloc                int    `json:"maxSloc"`
	MaxFunctionLength      int    `json:"maxFunctionLength"`
	MaxCognitiveComplexity int    `json:"maxCognitiveComplexity"`
	Rationale              string `json:"rationale"`
	FollowUp               string `json:"followUp,omitempty"`
}

type fileTier string

const (
	tierOne fileTier = "tier1"
	tierTwo fileTier = "tier2"
)

type fileKind string

const (
	kindSource fileKind = "source"
	kindTest   fileKind = "test"
)

type fileLanguage string

const (
	languageGo fileLanguage = "go"
	languageJS fileLanguage = "js"
	languageTS fileLanguage = "ts"
)

type fileInfo struct {
	absPath  string
	content  string
	relPath  string
	tier     fileTier
	kind     fileKind
	language fileLanguage
}

func loadRuntimeConfig(repoRoot string) (runtimeConfig, error) {
	configPath := filepath.Join(repoRoot, configFileName)
	baselinePath := filepath.Join(repoRoot, baselineFileName)

	var rawConfig sourceQualityConfig
	if err := loadJSONFile(configPath, &rawConfig); err != nil {
		return runtimeConfig{}, fmt.Errorf("load %s: %w", configFileName, err)
	}

	baseline := make(map[string]baselineEntry)
	if err := loadJSONFile(baselinePath, &baseline); err != nil {
		return runtimeConfig{}, fmt.Errorf("load %s: %w", baselineFileName, err)
	}

	return runtimeConfig{
		repoRoot:                   repoRoot,
		runnerTier1Paths:           normalizePathSet(rawConfig.RunnerTier1Paths),
		tier1SuppressionExceptions: normalizePathSet(rawConfig.Tier1SuppressionExceptions),
		baseline:                   normalizeBaseline(baseline),
	}, nil
}

func loadJSONFile(path string, target any) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(contents, target); err != nil {
		return err
	}

	return nil
}

func normalizePathSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[normalizeRepoPath(value)] = struct{}{}
	}
	return set
}

func normalizeBaseline(entries map[string]baselineEntry) map[string]baselineEntry {
	normalized := make(map[string]baselineEntry, len(entries))
	for path, entry := range entries {
		normalized[normalizeRepoPath(path)] = entry
	}
	return normalized
}

func normalizeRepoPath(path string) string {
	normalized := filepath.ToSlash(filepath.Clean(path))
	return strings.TrimPrefix(normalized, "./")
}

func classifyFile(relPath string, language fileLanguage, cfg runtimeConfig) (fileTier, fileKind, bool) {
	normalized := normalizeRepoPath(relPath)
	kind := classifyKind(normalized, language)

	switch {
	case strings.HasPrefix(normalized, "internal/"):
		return tierOne, kind, true
	case strings.HasPrefix(normalized, "tools/"):
		return tierOne, kind, true
	case strings.HasPrefix(normalized, "protocol/schemas/"):
		return tierOne, kind, true
	case strings.HasPrefix(normalized, "cmd/"):
		return tierTwo, kind, true
	case strings.HasPrefix(normalized, "runner/"):
		if _, ok := cfg.runnerTier1Paths[normalized]; ok {
			return tierOne, kind, true
		}
		return tierTwo, kind, true
	default:
		return "", "", false
	}
}

func classifyKind(relPath string, language fileLanguage) fileKind {
	base := filepath.Base(relPath)
	if language == languageGo && strings.HasSuffix(base, "_test.go") {
		return kindTest
	}

	if language == languageJS || language == languageTS {
		if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
			return kindTest
		}

		if hasPathSegment(filepath.Dir(relPath), "__tests__") || hasPathSegment(filepath.Dir(relPath), "tests") {
			return kindTest
		}
	}

	return kindSource
}

func hasPathSegment(pathValue, segment string) bool {
	for _, part := range strings.Split(filepath.ToSlash(pathValue), "/") {
		if part == segment {
			return true
		}
	}
	return false
}
