package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	supportedExtensions = map[string]fileLanguage{
		".go":  languageGo,
		".js":  languageJS,
		".cjs": languageJS,
		".mjs": languageJS,
		".ts":  languageTS,
		".cts": languageTS,
		".mts": languageTS,
		".tsx": languageTS,
	}
	excludedDirs = map[string]struct{}{
		".direnv":      {},
		".git":         {},
		".turbo":       {},
		"coverage":     {},
		"dist":         {},
		"node_modules": {},
	}
	generatedFilePattern = regexp.MustCompile(`(?i)code generated .*do not edit`)
)

func collectEligibleFiles(repoRoot string, cfg runtimeConfig) ([]fileInfo, error) {
	files := make([]fileInfo, 0)

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		return visitPath(repoRoot, path, d, walkErr, cfg, &files)
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].relPath < files[j].relPath
	})

	if len(files) == 0 {
		return nil, fmt.Errorf("no eligible source files found under %s", repoRoot)
	}

	return files, nil
}

func visitPath(repoRoot, path string, d fs.DirEntry, walkErr error, cfg runtimeConfig, files *[]fileInfo) error {
	if walkErr != nil {
		return walkErr
	}

	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return err
	}
	relPath = normalizeRepoPath(relPath)

	if d.IsDir() {
		if _, excluded := excludedDirs[d.Name()]; excluded {
			return filepath.SkipDir
		}
		return nil
	}

	if strings.HasPrefix(relPath, "protocol/fixtures/") {
		return nil
	}

	file, err := classifyEligibleFile(path, relPath, cfg)
	if err != nil || file == nil {
		return err
	}

	*files = append(*files, *file)
	return nil
}

func classifyEligibleFile(path, relPath string, cfg runtimeConfig) (*fileInfo, error) {
	language, ok := supportedExtensions[filepath.Ext(path)]
	if !ok {
		return nil, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", relPath, err)
	}
	if isGeneratedFile(string(contents)) {
		return nil, nil
	}

	tier, kind, eligible := classifyFile(relPath, language, cfg)
	if !eligible {
		return nil, nil
	}

	return &fileInfo{
		absPath:  path,
		content:  string(contents),
		relPath:  relPath,
		tier:     tier,
		kind:     kind,
		language: language,
	}, nil
}

func isGeneratedFile(contents string) bool {
	seenNonBlank := 0
	for _, line := range strings.Split(contents, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		seenNonBlank++
		if generatedFilePattern.MatchString(trimmed) {
			return true
		}
		if seenNonBlank >= 10 {
			break
		}
	}
	return false
}
