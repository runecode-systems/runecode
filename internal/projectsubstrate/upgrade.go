package projectsubstrate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

var revalidateAfterUpgrade = DiscoverAndValidate

func PreviewUpgrade(input UpgradePreviewInput) (UpgradePreview, error) {
	discovered, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: input.RepositoryRoot, Authority: input.Authority})
	if err != nil {
		return UpgradePreview{}, err
	}
	layout := inspectLayout(discovered.RepositoryRoot)
	preview := UpgradePreview{
		SchemaID:        UpgradePreviewSchemaID,
		SchemaVersion:   UpgradePreviewVersion,
		RepositoryRoot:  discovered.RepositoryRoot,
		CurrentSnapshot: discovered.Snapshot,
	}
	cfg, preconditions, reasons, configValid := upgradePreconditions(layout)
	preview.Preconditions = preconditions
	if !configValid || len(reasons) > 0 {
		return blockedUpgradePreview(preview, discovered.Snapshot, reasons), nil
	}

	if discovered.Snapshot.ValidationState == validationStateValid {
		switch discovered.Compatibility.Posture {
		case CompatibilityPostureSupportedCurrent:
			return noopUpgradePreview(preview, discovered.Snapshot), nil
		case CompatibilityPostureSupportedWithUpgrade:
			nextConfig := canonicalV0RunecontextYAML(discovered.Compatibility.Policy.RecommendedRuneContextVersion, discovered.Snapshot.DeclaredSourceType)
			nextLayout := layout
			nextLayout.runecontextYAML = []byte(nextConfig)
			expected := validateLayout(discovered.Contract, nextLayout)
			return readyUpgradePreview(preview, expected, layout.runecontextYAML, []byte(nextConfig)), nil
		default:
			return blockedUpgradePreview(preview, discovered.Snapshot, discovered.Compatibility.ReasonCodes), nil
		}
	}

	nextConfig := canonicalV0RunecontextYAML(cfg.RuneContextVersion, cfg.Source.Type)
	nextLayout := layout
	nextLayout.runecontextYAML = []byte(nextConfig)
	expected := validateLayout(discovered.Contract, nextLayout)
	return readyUpgradePreview(preview, expected, layout.runecontextYAML, []byte(nextConfig)), nil
}

func ApplyUpgrade(input UpgradeApplyInput) (UpgradeApplyResult, error) {
	preview := input.Preview
	if mismatch := blockedUpgradePreviewDigestResult(preview, input.ExpectedPreviewHash); mismatch != nil {
		return *mismatch, nil
	}
	current, authority, err := upgradeCurrentSnapshot(preview)
	if err != nil {
		return UpgradeApplyResult{}, err
	}
	if current.Snapshot.SnapshotDigest != preview.CurrentSnapshot.SnapshotDigest {
		return blockedUpgradeApplyResult(current.RepositoryRoot, current.Snapshot, preview.PreviewDigest, []string{reasonUpgradePreviewDigestMismatch}), nil
	}
	if result, ok := earlyUpgradeApplyResult(preview, current); ok {
		return result, nil
	}
	applyResult, err := applyUpgradePreview(preview, current, authority)
	if err != nil {
		return UpgradeApplyResult{}, err
	}
	if err := appendUpgradeAuditEvent(input.AuditAppender, current.RepositoryRoot, current.Snapshot, applyResult); err != nil {
		applyResult.ReasonCodes = normalizeReasonCodes(append(applyResult.ReasonCodes, reasonAuditAppendFailed))
	}
	return applyResult, nil
}

func upgradeCurrentSnapshot(preview UpgradePreview) (DiscoveryResult, RepoRootAuthority, error) {
	authority := upgradePreviewAuthority(preview)
	current, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: preview.RepositoryRoot, Authority: authority})
	return current, authority, err
}

func earlyUpgradeApplyResult(preview UpgradePreview, current DiscoveryResult) (UpgradeApplyResult, bool) {
	if preview.Status == upgradePreviewStatusNoop {
		return noopUpgradeApplyResult(current.RepositoryRoot, current.Snapshot, preview.PreviewDigest), true
	}
	if preview.Status != upgradePreviewStatusReady {
		return blockedUpgradeApplyResult(current.RepositoryRoot, current.Snapshot, preview.PreviewDigest, preview.ReasonCodes), true
	}
	return UpgradeApplyResult{}, false
}

func applyUpgradePreview(preview UpgradePreview, current DiscoveryResult, authority RepoRootAuthority) (UpgradeApplyResult, error) {
	layout := inspectLayout(current.RepositoryRoot)
	cfg, err := parseRunecontextConfig(layout.runecontextYAML)
	if err != nil {
		return blockedUpgradeApplyResult(current.RepositoryRoot, current.Snapshot, preview.PreviewDigest, []string{reasonRemediationConfigInvalid}), nil
	}
	originalConfig := append([]byte{}, layout.runecontextYAML...)
	targetVersion := strings.TrimSpace(preview.ExpectedSnapshot.RuneContextVersion)
	if targetVersion == "" {
		targetVersion = cfg.RuneContextVersion
	}
	nextConfig := canonicalV0RunecontextYAML(targetVersion, cfg.Source.Type)
	if err := writeUpgradedConfig(current.RepositoryRoot, nextConfig); err != nil {
		return UpgradeApplyResult{}, err
	}
	result, err := revalidateAfterUpgrade(DiscoveryInput{RepositoryRoot: current.RepositoryRoot, Authority: authority})
	if err != nil {
		if restoreErr := writeUpgradedConfig(current.RepositoryRoot, string(originalConfig)); restoreErr != nil {
			return UpgradeApplyResult{}, fmt.Errorf("revalidate upgraded project substrate: %w (restore config: %v)", err, restoreErr)
		}
		return UpgradeApplyResult{}, fmt.Errorf("revalidate upgraded project substrate: %w", err)
	}
	applyResult := appliedUpgradeResult(current.RepositoryRoot, current.Snapshot, result.Snapshot, preview)
	if applyResult.ResultingSnapshot.ValidationState != validationStateValid {
		applyResult.Status = upgradeApplyStatusAppliedValidationError
		applyResult.ReasonCodes = []string{reasonUpgradePostValidationFailed}
	}
	return applyResult, nil
}

func writeUpgradedConfig(repoRoot, nextConfig string) error {
	configPath := filepath.Join(repoRoot, CanonicalConfigPath)
	configFile, tempPath, err := openAtomicConfigTemp(configPath)
	if err != nil {
		return fmt.Errorf("write upgrade config: %w", err)
	}
	if _, err := configFile.WriteString(nextConfig); err != nil {
		_ = configFile.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write upgrade config: %w", err)
	}
	if err := configFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("write upgrade config: %w", err)
	}
	if err := replaceConfigFile(tempPath, configPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("write upgrade config: %w", err)
	}
	return nil
}

func remediationFollowUpForReasons(reasons []string) []string {
	followUp := []string{}
	seen := map[string]struct{}{}
	add := func(value string) {
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		followUp = append(followUp, value)
	}
	for _, code := range reasons {
		switch code {
		case reasonRemediationConfigAnchorRequired, reasonRemediationInitRequired:
			add("initialize_canonical_runecontext_substrate")
		case reasonPrivateMirrorDetected:
			add("remove_private_runecontext_mirror")
		case reasonRemediationConfigInvalid:
			add("repair_runecontext_yaml")
		case reasonRemediationDeclaredSourceConflicts:
			add("align_source_path_to_canonical_runecontext")
		}
	}
	if len(followUp) == 0 {
		add("inspect_project_substrate_diagnostics")
	}
	return followUp
}

func canonicalV0RunecontextYAML(version, sourceType string) string {
	v := strings.TrimSpace(version)
	if v == "" {
		v = "0.1.0-alpha.14"
	}
	t := strings.TrimSpace(sourceType)
	if t == "" {
		t = "embedded"
	}
	return fmt.Sprintf("schema_version: 1\nrunecontext_version: %s\nassurance_tier: verified\nsource:\n  type: %s\n  path: %s\n", yamlScalar(v), yamlScalar(t), yamlScalar(CanonicalSourcePath))
}

func sha256Hex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestUpgradePreview(preview UpgradePreview) string {
	payload := map[string]any{
		"schema_id":          preview.SchemaID,
		"schema_version":     preview.SchemaVersion,
		"repository_root":    preview.RepositoryRoot,
		"status":             preview.Status,
		"reason_codes":       preview.ReasonCodes,
		"current_snapshot":   preview.CurrentSnapshot,
		"expected_snapshot":  preview.ExpectedSnapshot,
		"file_changes":       preview.FileChanges,
		"preconditions":      preview.Preconditions,
		"required_follow_up": preview.RequiredFollowUp,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "digest:error"
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "digest:error"
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}
