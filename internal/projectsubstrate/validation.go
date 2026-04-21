package projectsubstrate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
	"gopkg.in/yaml.v3"
)

type runecontextConfig struct {
	RuneContextVersion string `json:"runecontext_version" yaml:"runecontext_version"`
	AssuranceTier      string `json:"assurance_tier" yaml:"assurance_tier"`
	Source             struct {
		Type string `json:"type" yaml:"type"`
		Path string `json:"path" yaml:"path"`
	} `json:"source" yaml:"source"`
}

func validateLayout(contract ContractState, layout repositoryLayout) ValidationSnapshot {
	snapshot := ValidationSnapshot{
		SchemaID:        SnapshotSchemaID,
		SchemaVersion:   SnapshotSchemaVersion,
		Contract:        contract,
		ValidationState: validationStateValid,
		Anchors: AnchorStatus{
			HasConfigAnchor:      layout.hasConfigAnchor,
			HasSourceAnchor:      layout.hasSourceAnchor,
			HasAssuranceAnchor:   layout.hasAssuranceAnchor,
			HasAssuranceBaseline: layout.hasAssuranceBaseline,
			HasPrivateTruthCopy:  layout.hasPrivateTruthCopy,
		},
	}
	snapshot.CanonicalCandidatePaths = canonicalCandidatePaths(layout)
	reasons := layoutReasonCodes(layout)
	applyParsedConfig(contract, layout, &snapshot, &reasons)
	snapshot.ReasonCodes = normalizeReasonCodes(reasons)
	if len(snapshot.ReasonCodes) > 0 {
		snapshot.ValidationState = validationStateInvalid
	}
	if !layout.hasConfigAnchor && !layout.hasSourceAnchor && !layout.hasAssuranceAnchor {
		snapshot.ValidationState = validationStateMissing
	}
	digest := digestSnapshot(snapshot)
	snapshot.SnapshotDigest = digest
	snapshot.ProjectContextIdentityDigest = digest
	if snapshot.ValidationState == validationStateValid {
		snapshot.ValidatedSnapshotDigest = digest
	}
	return snapshot
}

func canonicalCandidatePaths(layout repositoryLayout) []string {
	if len(layout.runecontextCandidates) == 0 {
		return nil
	}
	paths := make([]string, 0, len(layout.runecontextCandidates))
	for _, candidate := range layout.runecontextCandidates {
		rel := strings.TrimPrefix(strings.TrimPrefix(candidate, layout.repoRoot), string(filepath.Separator))
		if strings.TrimSpace(rel) == "" {
			rel = candidate
		}
		paths = append(paths, rel)
	}
	if len(paths) > 1 {
		sort.Strings(paths)
	}
	return paths
}

func layoutReasonCodes(layout repositoryLayout) []string {
	reasons := make([]string, 0, 8)
	if !layout.hasConfigAnchor {
		reasons = append(reasons, reasonMissingConfigAnchor)
	}
	if !layout.hasSourceAnchor {
		reasons = append(reasons, reasonMissingSourceAnchor)
	}
	if !layout.hasAssuranceAnchor {
		reasons = append(reasons, reasonMissingAssuranceAnchor)
	}
	if !layout.hasAssuranceBaseline {
		reasons = append(reasons, reasonMissingAssuranceBaseline)
	}
	if layout.hasPrivateTruthCopy {
		reasons = append(reasons, reasonPrivateMirrorDetected)
	}
	return reasons
}

func applyParsedConfig(contract ContractState, layout repositoryLayout, snapshot *ValidationSnapshot, reasons *[]string) {
	if snapshot == nil || reasons == nil {
		return
	}
	cfg, cfgErr := parseRunecontextConfig(layout.runecontextYAML)
	if cfgErr != nil {
		if layout.hasConfigAnchor {
			*reasons = append(*reasons, reasonConfigParseInvalid)
		}
		return
	}
	snapshot.RuneContextVersion = cfg.RuneContextVersion
	snapshot.DeclaredAssuranceTier = cfg.AssuranceTier
	snapshot.DeclaredSourceType = cfg.Source.Type
	snapshot.DeclaredSourcePath = cfg.Source.Path
	snapshot.Anchors.HasVerifiedPosture = strings.EqualFold(strings.TrimSpace(cfg.AssuranceTier), contract.RequiredAssuranceTier)
	snapshot.Anchors.HasCanonicalSource = strings.TrimSpace(cfg.Source.Path) == contract.RequiredSourcePath
	if !snapshot.Anchors.HasVerifiedPosture {
		*reasons = append(*reasons, reasonNonVerifiedPosture)
	}
	if strings.TrimSpace(cfg.Source.Path) == "" {
		*reasons = append(*reasons, reasonConfigMissingSourcePath)
		return
	}
	if !snapshot.Anchors.HasCanonicalSource {
		*reasons = append(*reasons, reasonNonCanonicalSourcePath)
	}
}

func parseRunecontextConfig(data []byte) (runecontextConfig, error) {
	var cfg runecontextConfig
	if len(data) == 0 {
		return cfg, fmt.Errorf("empty")
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return runecontextConfig{}, fmt.Errorf("invalid yaml: %w", err)
	}
	if strings.TrimSpace(cfg.RuneContextVersion) == "" || strings.TrimSpace(cfg.AssuranceTier) == "" {
		return runecontextConfig{}, fmt.Errorf("missing required fields")
	}
	return cfg, nil
}

func digestSnapshot(snapshot ValidationSnapshot) string {
	payload := map[string]any{
		"schema_id":                 snapshot.SchemaID,
		"schema_version":            snapshot.SchemaVersion,
		"contract":                  snapshot.Contract,
		"validation_state":          snapshot.ValidationState,
		"reason_codes":              snapshot.ReasonCodes,
		"runecontext_version":       snapshot.RuneContextVersion,
		"declared_assurance_tier":   snapshot.DeclaredAssuranceTier,
		"declared_source_type":      snapshot.DeclaredSourceType,
		"declared_source_path":      snapshot.DeclaredSourcePath,
		"canonical_candidate_paths": snapshot.CanonicalCandidatePaths,
		"anchors":                   snapshot.Anchors,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}
