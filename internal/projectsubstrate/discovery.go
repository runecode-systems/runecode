package projectsubstrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DiscoveryInput struct {
	RepositoryRoot string
	Authority      RepoRootAuthority
}

type DiscoveryResult struct {
	RepositoryRoot string
	Contract       ContractState
	Snapshot       ValidationSnapshot
	Compatibility  CompatibilityAssessment
}

type repositoryLayout struct {
	repoRoot              string
	configPath            string
	sourcePath            string
	assurancePath         string
	runecontextCandidates []string
	hasConfigAnchor       bool
	hasSourceAnchor       bool
	hasAssuranceAnchor    bool
	hasPrivateTruthCopy   bool
	runecontextYAML       []byte
}

func DiscoverAndValidate(input DiscoveryInput) (DiscoveryResult, error) {
	repoRoot, err := authoritativeRepoRoot(input)
	if err != nil {
		return DiscoveryResult{}, err
	}
	layout := inspectLayout(repoRoot)
	snapshot := validateLayout(defaultContract(input.Authority), layout)
	compatibility := EvaluateCompatibility(snapshot)
	return DiscoveryResult{
		RepositoryRoot: repoRoot,
		Contract:       snapshot.Contract,
		Snapshot:       snapshot,
		Compatibility:  compatibility,
	}, nil
}

func authoritativeRepoRoot(input DiscoveryInput) (string, error) {
	root := strings.TrimSpace(input.RepositoryRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("determine repository root: %w", err)
		}
		root = cwd
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("%s", reasonDiscoveryRootInvalid)
	}
	clean := filepath.Clean(root)
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("%s", reasonDiscoveryRootInvalid)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s", reasonDiscoveryRootInvalid)
	}
	return clean, nil
}

func inspectLayout(repoRoot string) repositoryLayout {
	layout := repositoryLayout{
		repoRoot:      repoRoot,
		configPath:    filepath.Join(repoRoot, CanonicalConfigPath),
		sourcePath:    filepath.Join(repoRoot, CanonicalSourcePath),
		assurancePath: filepath.Join(repoRoot, CanonicalAssurancePath),
	}
	layout.runecontextCandidates = discoverRunecontextCandidates(repoRoot)
	readConfigAnchor(&layout)
	statDirectoryAnchor(layout.sourcePath, &layout.hasSourceAnchor)
	statDirectoryAnchor(layout.assurancePath, &layout.hasAssuranceAnchor)
	statDirectoryAnchor(filepath.Join(repoRoot, ".runecontext"), &layout.hasPrivateTruthCopy)
	return layout
}

func discoverRunecontextCandidates(repoRoot string) []string {
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return nil
	}
	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if !isRunecontextCandidate(name) {
			continue
		}
		candidates = append(candidates, filepath.Join(repoRoot, name))
	}
	return candidates
}

func isRunecontextCandidate(name string) bool {
	if name == "" {
		return false
	}
	lower := strings.ToLower(name)
	if lower == strings.ToLower(CanonicalConfigPath) || lower == strings.ToLower(CanonicalSourcePath) || lower == ".runecontext" {
		return false
	}
	return strings.Contains(lower, "runecontext")
}

func readConfigAnchor(layout *repositoryLayout) {
	if layout == nil {
		return
	}
	b, err := os.ReadFile(layout.configPath)
	if err != nil {
		return
	}
	layout.hasConfigAnchor = true
	layout.runecontextYAML = b
}

func statDirectoryAnchor(path string, target *bool) {
	if target == nil {
		return
	}
	info, err := os.Stat(path)
	*target = err == nil && info.IsDir()
}
