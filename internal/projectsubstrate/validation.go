package projectsubstrate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
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

type assuranceBaseline struct {
	SchemaVersion    *int   `yaml:"schema_version"`
	Kind             string `yaml:"kind"`
	SubjectID        string `yaml:"subject_id"`
	CreatedAt        *int64 `yaml:"created_at"`
	Canonicalization string `yaml:"canonicalization"`
	Value            struct {
		AdoptionCommit string `yaml:"adoption_commit"`
		SourcePosture  string `yaml:"source_posture"`
	} `yaml:"value"`
}

var adoptionCommitPattern = regexp.MustCompile("^[a-f0-9]{40}$")

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
	applyParsedAssuranceBaseline(layout, &reasons)
	snapshot.ReasonCodes = normalizeReasonCodes(reasons)
	if len(snapshot.ReasonCodes) > 0 {
		snapshot.ValidationState = validationStateInvalid
	}
	if !layout.hasConfigAnchor && !layout.hasSourceAnchor && !layout.hasAssuranceAnchor && !layout.hasAssuranceBaseline && missingOnlyReasonCodes(snapshot.ReasonCodes) {
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

func missingOnlyReasonCodes(reasons []string) bool {
	for _, reason := range reasons {
		switch reason {
		case reasonMissingConfigAnchor, reasonMissingSourceAnchor, reasonMissingAssuranceAnchor, reasonMissingAssuranceBaseline:
			continue
		default:
			return false
		}
	}
	return true
}

func applyParsedAssuranceBaseline(layout repositoryLayout, reasons *[]string) {
	if reasons == nil || !layout.hasAssuranceBaseline {
		return
	}
	if _, err := parseAssuranceBaseline(layout.assuranceBaselineYAML); err != nil {
		*reasons = append(*reasons, reasonAssuranceBaselineInvalid)
	}
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

func parseAssuranceBaseline(data []byte) (assuranceBaseline, error) {
	var baseline assuranceBaseline
	if len(data) == 0 {
		return baseline, fmt.Errorf("empty")
	}
	if err := yaml.Unmarshal(data, &baseline); err != nil {
		return assuranceBaseline{}, fmt.Errorf("invalid yaml: %w", err)
	}
	if err := validateAssuranceBaselineMetadata(baseline); err != nil {
		return assuranceBaseline{}, err
	}
	if err := validateAssuranceBaselineValue(baseline.Value.AdoptionCommit, baseline.Value.SourcePosture); err != nil {
		return assuranceBaseline{}, err
	}
	return baseline, nil
}

func validateAssuranceBaselineMetadata(baseline assuranceBaseline) error {
	if baseline.SchemaVersion == nil || *baseline.SchemaVersion != 1 {
		return fmt.Errorf("invalid schema_version")
	}
	if baseline.CreatedAt == nil {
		return fmt.Errorf("missing created_at")
	}
	if strings.TrimSpace(baseline.Kind) != "baseline" {
		return fmt.Errorf("invalid kind")
	}
	if strings.TrimSpace(baseline.SubjectID) == "" {
		return fmt.Errorf("missing subject_id")
	}
	if strings.TrimSpace(baseline.Canonicalization) != "runecontext-canonical-json-v1" {
		return fmt.Errorf("invalid canonicalization")
	}
	return nil
}

func validateAssuranceBaselineValue(adoptionCommit, sourcePosture string) error {
	trimmedCommit := strings.TrimSpace(adoptionCommit)
	if trimmedCommit == "" {
		return fmt.Errorf("missing adoption_commit")
	}
	if !adoptionCommitPattern.MatchString(trimmedCommit) {
		return fmt.Errorf("invalid adoption_commit")
	}
	if !supportedSourcePosture(sourcePosture) {
		return fmt.Errorf("invalid source_posture")
	}
	return nil
}

func supportedSourcePosture(value string) bool {
	switch strings.TrimSpace(value) {
	case "embedded", "git", "path":
		return true
	default:
		return false
	}
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
