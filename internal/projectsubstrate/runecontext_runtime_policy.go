package projectsubstrate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	releaseSupportedRuneContextVersionMin = "0.1.0-alpha.13"
	releaseSupportedRuneContextVersionMax = "0.1.0-alpha.16"
	releaseRecommendedRuneContextVersion  = "0.1.0-alpha.14"

	runeContextMetadataCommandTimeout = 2 * time.Second
	runeContextBinaryName             = "runectx"
)

type runtimePolicySnapshot struct {
	SupportedRuneContextVersionMin string
	SupportedRuneContextVersionMax string
	RecommendedRuneContextVersion  string
	LocalRunectxVersion            string
}

type runtimePolicyProvider interface {
	RuntimePolicy() (runtimePolicySnapshot, error)
}

type execCommandFactory func(context.Context, string, ...string) *exec.Cmd
type runectxPathResolver func() (string, error)

var resolveRunectxBinaryPath runectxPathResolver = secureRunectxBinaryPath

var runtimeRuneContextPolicyProvider runtimePolicyProvider = newCachedRuntimePolicyProvider(execRuntimePolicyProvider{})

func runtimeCompatibilityPolicy() runtimePolicySnapshot {
	policy, err := runtimeRuneContextPolicyProvider.RuntimePolicy()
	if err != nil {
		log.Printf("projectsubstrate: runectx runtime policy unavailable, using release fallback error=%v", err)
		return releaseFallbackRuntimePolicy()
	}
	if err := validateRuntimePolicy(policy); err != nil {
		log.Printf("projectsubstrate: runectx runtime policy invalid, using release fallback error=%v", err)
		return releaseFallbackRuntimePolicy()
	}
	return policy
}

func releaseFallbackRuntimePolicy() runtimePolicySnapshot {
	return runtimePolicySnapshot{
		SupportedRuneContextVersionMin: releaseSupportedRuneContextVersionMin,
		SupportedRuneContextVersionMax: releaseSupportedRuneContextVersionMax,
		RecommendedRuneContextVersion:  releaseRecommendedRuneContextVersion,
	}
}

func validateRuntimePolicy(policy runtimePolicySnapshot) error {
	if _, err := compareVersion(policy.SupportedRuneContextVersionMin, policy.SupportedRuneContextVersionMax); err != nil {
		return err
	}
	cmpMin, minErr := compareVersion(policy.RecommendedRuneContextVersion, policy.SupportedRuneContextVersionMin)
	cmpMax, maxErr := compareVersion(policy.RecommendedRuneContextVersion, policy.SupportedRuneContextVersionMax)
	if minErr != nil || maxErr != nil {
		return fmt.Errorf("recommended version invalid")
	}
	if cmpMin < 0 || cmpMax > 0 {
		return fmt.Errorf("recommended version out of supported range")
	}
	return nil
}

type cachedRuntimePolicyProvider struct {
	once     sync.Once
	provider runtimePolicyProvider
	policy   runtimePolicySnapshot
	err      error
}

func newCachedRuntimePolicyProvider(provider runtimePolicyProvider) runtimePolicyProvider {
	return &cachedRuntimePolicyProvider{provider: provider}
}

func (p *cachedRuntimePolicyProvider) RuntimePolicy() (runtimePolicySnapshot, error) {
	p.once.Do(func() {
		if p.provider == nil {
			p.err = fmt.Errorf("runtime policy provider unavailable")
			return
		}
		p.policy, p.err = p.provider.RuntimePolicy()
	})
	return p.policy, p.err
}

type execRuntimePolicyProvider struct{}

type runeContextMetadataEnvelope struct {
	Release struct {
		Version string `json:"version"`
	} `json:"release"`
	Compatibility struct {
		DefaultProjectVersion           string   `json:"default_project_version"`
		DirectlySupportedProjectVersion []string `json:"directly_supported_project_versions"`
		UpgradeableFromProjectVersion   []string `json:"upgradeable_from_project_versions"`
	} `json:"compatibility"`
}

func (execRuntimePolicyProvider) RuntimePolicy() (runtimePolicySnapshot, error) {
	resolvedPath, err := resolveRunectxBinaryPath()
	if err != nil {
		return runtimePolicySnapshot{}, fmt.Errorf("resolve runectx metadata binary: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), runeContextMetadataCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, resolvedPath, "metadata")
	out, err := cmd.Output()
	if err != nil {
		return runtimePolicySnapshot{}, fmt.Errorf("invoke runectx metadata: %w", err)
	}
	var metadata runeContextMetadataEnvelope
	if err := json.Unmarshal(out, &metadata); err != nil {
		return runtimePolicySnapshot{}, fmt.Errorf("decode runectx metadata: %w", err)
	}
	return deriveRuntimePolicy(metadata)
}

func deriveRuntimePolicy(metadata runeContextMetadataEnvelope) (runtimePolicySnapshot, error) {
	recommended := strings.TrimSpace(metadata.Compatibility.DefaultProjectVersion)
	if recommended == "" {
		return runtimePolicySnapshot{}, fmt.Errorf("runectx metadata missing default project version")
	}
	if _, err := parseVersion(recommended); err != nil {
		return runtimePolicySnapshot{}, fmt.Errorf("runectx metadata default project version invalid: %w", err)
	}

	maxCandidates := append([]string{}, metadata.Compatibility.DirectlySupportedProjectVersion...)
	maxCandidates = append(maxCandidates, recommended)
	maxVersion, err := lowestOrHighestVersion(maxCandidates, true)
	if err != nil {
		return runtimePolicySnapshot{}, fmt.Errorf("derive supported max: %w", err)
	}

	minCandidates := append([]string{}, metadata.Compatibility.UpgradeableFromProjectVersion...)
	minCandidates = append(minCandidates, recommended)
	minVersion, err := lowestOrHighestVersion(minCandidates, false)
	if err != nil {
		return runtimePolicySnapshot{}, fmt.Errorf("derive supported min: %w", err)
	}

	policy := runtimePolicySnapshot{
		SupportedRuneContextVersionMin: minVersion,
		SupportedRuneContextVersionMax: maxVersion,
		RecommendedRuneContextVersion:  recommended,
		LocalRunectxVersion:            strings.TrimSpace(metadata.Release.Version),
	}
	if err := validateRuntimePolicy(policy); err != nil {
		return runtimePolicySnapshot{}, err
	}
	return policy, nil
}

func lowestOrHighestVersion(candidates []string, wantHighest bool) (string, error) {
	selected := ""
	for _, candidate := range candidates {
		v, ok := normalizedVersionCandidate(candidate)
		if !ok {
			continue
		}
		if selected == "" {
			selected = v
			continue
		}
		if shouldReplaceSelectedVersion(v, selected, wantHighest) {
			selected = v
		}
	}
	if selected == "" {
		return "", fmt.Errorf("no parseable versions")
	}
	return selected, nil
}

func normalizedVersionCandidate(candidate string) (string, bool) {
	v := strings.TrimSpace(candidate)
	if v == "" {
		return "", false
	}
	if _, err := parseVersion(v); err != nil {
		return "", false
	}
	return v, true
}

func compareVersionCandidate(left, right string) (int, bool) {
	cmp, err := compareVersion(left, right)
	if err != nil {
		return 0, false
	}
	return cmp, true
}

func shouldReplaceSelectedVersion(candidate, selected string, wantHighest bool) bool {
	cmp, ok := compareVersionCandidate(candidate, selected)
	if !ok {
		return false
	}
	if wantHighest {
		return cmp > 0
	}
	return cmp < 0
}

func secureRunectxBinaryPath() (string, error) {
	resolvedPath, err := exec.LookPath(runeContextBinaryName)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(resolvedPath) {
		return "", fmt.Errorf("runectx binary path must be absolute")
	}
	exePath, err := exec.LookPath(filepath.Base(resolvedPath))
	if err != nil {
		return "", err
	}
	if filepath.Clean(exePath) != filepath.Clean(resolvedPath) {
		return "", fmt.Errorf("runectx binary resolution unstable")
	}
	return resolvedPath, nil
}
