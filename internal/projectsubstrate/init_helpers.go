package projectsubstrate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func blockedInitPreviewTokenResult(preview InitPreview, expectedPreviewToken string) *InitApplyResult {
	expected := strings.TrimSpace(expectedPreviewToken)
	if expected != "" && expected == preview.PreviewToken {
		return nil
	}
	result := blockedInitApplyResult(preview.RepositoryRoot, preview.CurrentSnapshot, preview.PreviewToken, []string{reasonInitPreviewTokenMismatch})
	return &result
}

func initPreviewAuthority(preview InitPreview) RepoRootAuthority {
	if strings.TrimSpace(preview.CurrentSnapshot.Contract.RepoRootAuthority) == string(RepoRootAuthorityProcessWorkingDirectory) {
		return RepoRootAuthorityProcessWorkingDirectory
	}
	return RepoRootAuthorityExplicitConfig
}

func blockedInitSnapshotResult(repoRoot string, snapshot ValidationSnapshot, previewToken string) InitApplyResult {
	return blockedInitApplyResult(repoRoot, snapshot, previewToken, []string{reasonInitSnapshotChanged})
}

func noopInitApplyResult(repoRoot string, snapshot ValidationSnapshot, previewToken string) InitApplyResult {
	return InitApplyResult{
		SchemaID:          InitApplySchemaID,
		SchemaVersion:     InitApplyVersion,
		RepositoryRoot:    repoRoot,
		Status:            initApplyStatusNoop,
		CurrentSnapshot:   snapshot,
		ResultingSnapshot: snapshot,
		PreviewToken:      previewToken,
	}
}

func blockedInitApplyResult(repoRoot string, snapshot ValidationSnapshot, previewToken string, reasonCodes []string) InitApplyResult {
	blockedReasons := normalizeReasonCodes(append([]string{}, reasonCodes...))
	if len(blockedReasons) == 0 {
		blockedReasons = []string{reasonInitConflictDetected}
	}
	return InitApplyResult{
		SchemaID:          InitApplySchemaID,
		SchemaVersion:     InitApplyVersion,
		RepositoryRoot:    repoRoot,
		Status:            initApplyStatusBlocked,
		ReasonCodes:       blockedReasons,
		CurrentSnapshot:   snapshot,
		ResultingSnapshot: snapshot,
		PreviewToken:      previewToken,
	}
}

func applyCanonicalInitialization(root string) error {
	configPath := filepath.Join(root, CanonicalConfigPath)
	sourcePath := filepath.Join(root, CanonicalSourcePath)
	assurancePath := filepath.Join(root, CanonicalAssurancePath)
	baselinePath := filepath.Join(root, canonicalAssuranceBaselinePath)
	mutation, err := canonicalInitialization(recommendedRuneContextVersionTarget(), "embedded")
	if err != nil {
		return err
	}
	createdSource := !pathExists(sourcePath)
	createdAssurance := !pathExists(assurancePath)
	createdBaseline := !pathExists(baselinePath)
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		return fmt.Errorf("create canonical source directory: %w", err)
	}
	if err := os.MkdirAll(assurancePath, 0o755); err != nil {
		rollbackCanonicalInitialization(createdSource, false, false, sourcePath, assurancePath, baselinePath)
		return fmt.Errorf("create canonical assurance directory: %w", err)
	}
	if err := writeCanonicalConfigFile(baselinePath, mutation.AssuranceBaselineYML); err != nil {
		rollbackCanonicalInitialization(createdSource, createdAssurance, false, sourcePath, assurancePath, baselinePath)
		return fmt.Errorf("write canonical assurance baseline: %w", err)
	}
	if err := writeCanonicalConfigFile(configPath, mutation.ConfigYAML); err != nil {
		rollbackCanonicalInitialization(createdSource, createdAssurance, createdBaseline, sourcePath, assurancePath, baselinePath)
		return err
	}
	return nil
}

func rollbackCanonicalInitialization(createdSource, createdAssurance, createdBaseline bool, sourcePath, assurancePath, baselinePath string) {
	if createdBaseline {
		_ = os.Remove(baselinePath)
	}
	if createdAssurance {
		_ = os.Remove(assurancePath)
	}
	if createdSource {
		_ = os.Remove(sourcePath)
	}
}

func initApplyConflicts(root string) ([]string, []string, error) {
	reasons := make([]string, 0, 2)
	paths := make([]string, 0, 3)
	if err := appendInitApplyConflict(root, CanonicalConfigPath, false, &reasons, &paths); err != nil {
		return nil, nil, err
	}
	if err := appendInitApplyConflict(root, CanonicalSourcePath, true, &reasons, &paths); err != nil {
		return nil, nil, err
	}
	if err := appendInitApplyConflict(root, CanonicalAssurancePath, true, &reasons, &paths); err != nil {
		return nil, nil, err
	}
	if err := appendInitApplyConflict(root, canonicalAssuranceBaselinePath, false, &reasons, &paths); err != nil {
		return nil, nil, err
	}
	return normalizeReasonCodes(reasons), normalizeReasonCodes(paths), nil
}

func appendInitApplyConflict(root, relativePath string, requireDir bool, reasons, paths *[]string) error {
	info, err := os.Stat(filepath.Join(root, relativePath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", relativePath, err)
	}
	appendUniqueString(reasons, reasonInitConflictCanonicalExists)
	appendUniqueString(paths, relativePath)
	if requireDir && !info.IsDir() {
		return nil
	}
	return nil
}

func initConflicts(snapshot ValidationSnapshot) ([]string, []string) {
	reasons := make([]string, 0, 4)
	paths := make([]string, 0, 8)
	appendCanonicalInitConflicts(snapshot, &reasons, &paths)
	appendPrivateMirrorConflict(snapshot, &reasons, &paths)
	appendCandidateInitConflicts(snapshot, &reasons, &paths)
	if snapshot.ValidationState == validationStateInvalid && len(reasons) == 0 {
		appendUniqueString(&reasons, reasonInitConflictCanonicalExists)
	}
	sort.Strings(paths)
	return normalizeReasonCodes(reasons), paths
}

func appendCanonicalInitConflicts(snapshot ValidationSnapshot, reasons *[]string, paths *[]string) {
	if !snapshot.Anchors.HasConfigAnchor && !snapshot.Anchors.HasSourceAnchor && !snapshot.Anchors.HasAssuranceAnchor && !snapshot.Anchors.HasAssuranceBaseline {
		return
	}
	appendUniqueString(reasons, reasonInitConflictCanonicalExists)
	appendAnchorConflictPath(snapshot.Anchors.HasConfigAnchor, CanonicalConfigPath, paths)
	appendAnchorConflictPath(snapshot.Anchors.HasSourceAnchor, CanonicalSourcePath, paths)
	appendAnchorConflictPath(snapshot.Anchors.HasAssuranceAnchor, CanonicalAssurancePath, paths)
	appendAnchorConflictPath(snapshot.Anchors.HasAssuranceBaseline, canonicalAssuranceBaselinePath, paths)
}

func appendAnchorConflictPath(present bool, path string, paths *[]string) {
	if present {
		appendUniqueString(paths, path)
	}
}

func appendPrivateMirrorConflict(snapshot ValidationSnapshot, reasons *[]string, paths *[]string) {
	if !snapshot.Anchors.HasPrivateTruthCopy {
		return
	}
	appendUniqueString(reasons, reasonInitConflictPrivateMirror)
	appendUniqueString(paths, ".runecontext")
}

func appendCandidateInitConflicts(snapshot ValidationSnapshot, reasons *[]string, paths *[]string) {
	if len(snapshot.CanonicalCandidatePaths) == 0 {
		return
	}
	appendUniqueString(reasons, reasonInitConflictCandidateState)
	for _, candidate := range snapshot.CanonicalCandidatePaths {
		appendUniqueString(paths, candidate)
	}
}

func appendUniqueString(target *[]string, value string) {
	if target == nil {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || containsString(*target, trimmed) {
		return
	}
	*target = append(*target, trimmed)
}

func initRemediationFollowUp(reasons []string) []string {
	followUp := []string{}
	add := func(value string) {
		if !containsString(followUp, value) {
			followUp = append(followUp, value)
		}
	}
	for _, code := range reasons {
		switch strings.TrimSpace(code) {
		case reasonInitConflictCanonicalExists:
			add("adopt_existing_canonical_substrate")
			add("run_manual_remediation_for_partial_canonical_state")
		case reasonInitConflictPrivateMirror:
			add("remove_private_runecontext_mirror")
		case reasonInitConflictCandidateState:
			add("inspect_and_rename_noncanonical_runecontext_candidates")
		}
	}
	if len(followUp) == 0 {
		add("inspect_project_substrate_diagnostics")
	}
	return followUp
}

func digestInitPreview(preview InitPreview) string {
	payload := map[string]any{
		"schema_id":          preview.SchemaID,
		"schema_version":     preview.SchemaVersion,
		"repository_root":    preview.RepositoryRoot,
		"status":             preview.Status,
		"reason_codes":       preview.ReasonCodes,
		"current_snapshot":   preview.CurrentSnapshot,
		"expected_snapshot":  preview.ExpectedSnapshot,
		"file_changes":       preview.FileChanges,
		"conflicting_paths":  preview.ConflictingPaths,
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

func appendInitAuditEvent(appender AuditEventAppender, repoRoot string, current ValidationSnapshot, applyResult InitApplyResult) error {
	if appender == nil {
		return nil
	}
	details := map[string]interface{}{
		"repository_root":           repoRoot,
		"preview_token":             applyResult.PreviewToken,
		"status":                    applyResult.Status,
		"reason_codes":              append([]string{}, applyResult.ReasonCodes...),
		"before_snapshot_digest":    current.SnapshotDigest,
		"result_snapshot_digest":    applyResult.ResultingSnapshot.SnapshotDigest,
		"validated_snapshot_digest": applyResult.ResultingSnapshot.ValidatedSnapshotDigest,
	}
	return appender.AppendTrustedAuditEvent("project_substrate_init_event", "projectsubstrate", details)
}

func yamlScalar(value string) string {
	return strconv.Quote(strings.TrimSpace(value))
}
