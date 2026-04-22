package projectsubstrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdoptExistingReadOnlyCanonicalValid(t *testing.T) {
	root := t.TempDir()
	writeCanonicalV0Anchors(t, root)

	adopted, err := AdoptExisting(AdoptionInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("AdoptExisting returned error: %v", err)
	}
	if adopted.Status != adoptionStatusAdopted {
		t.Fatalf("status = %q, want %q", adopted.Status, adoptionStatusAdopted)
	}
	if adopted.Snapshot.ValidationState != validationStateValid {
		t.Fatalf("validation_state = %q, want %q", adopted.Snapshot.ValidationState, validationStateValid)
	}
}

func TestAdoptExistingBlockedForNonCanonicalState(t *testing.T) {
	root := t.TempDir()
	writeUpgradeableNonVerifiedV0Anchors(t, root)

	adopted, err := AdoptExisting(AdoptionInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("AdoptExisting returned error: %v", err)
	}
	if adopted.Status != adoptionStatusBlocked {
		t.Fatalf("status = %q, want %q", adopted.Status, adoptionStatusBlocked)
	}
	assertHasReason(t, adopted.ReasonCodes, compatibilityReasonNonVerified)
}

func TestAdoptExistingBlockedForUnsupportedTooOldCompatibility(t *testing.T) {
	root := t.TempDir()
	writeCanonicalV0AnchorsWithVersion(t, root, "0.1.0-alpha.11")

	adopted, err := AdoptExisting(AdoptionInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("AdoptExisting returned error: %v", err)
	}
	if adopted.Status != adoptionStatusBlocked {
		t.Fatalf("status = %q, want %q", adopted.Status, adoptionStatusBlocked)
	}
	assertHasReason(t, adopted.ReasonCodes, compatibilityReasonUnsupportedTooOld)
}

func TestAdoptExistingBlockedForUnsupportedTooNewCompatibility(t *testing.T) {
	root := t.TempDir()
	writeCanonicalV0AnchorsWithVersion(t, root, "0.1.0-alpha.99")

	adopted, err := AdoptExisting(AdoptionInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("AdoptExisting returned error: %v", err)
	}
	if adopted.Status != adoptionStatusBlocked {
		t.Fatalf("status = %q, want %q", adopted.Status, adoptionStatusBlocked)
	}
	assertHasReason(t, adopted.ReasonCodes, compatibilityReasonUnsupportedTooNew)
}

func TestPreviewInitializeReadyForMissingCanonicalState(t *testing.T) {
	root := t.TempDir()

	preview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize returned error: %v", err)
	}
	if preview.Status != initPreviewStatusReady {
		t.Fatalf("status = %q, want %q", preview.Status, initPreviewStatusReady)
	}
	if preview.PreviewToken == "" {
		t.Fatal("preview_token empty, want deterministic token")
	}
	if len(preview.FileChanges) != 4 {
		t.Fatalf("file_changes count = %d, want 4", len(preview.FileChanges))
	}
	if preview.ExpectedSnapshot.RuneContextVersion != releaseRecommendedRuneContextVersion {
		t.Fatalf("expected_snapshot.runecontext_version = %q, want %q", preview.ExpectedSnapshot.RuneContextVersion, releaseRecommendedRuneContextVersion)
	}
}

func TestApplyInitializeAppliesCanonicalFiles(t *testing.T) {
	root := t.TempDir()
	result := previewAndApplyInitialize(t, root)
	assertInitializedCanonicalFiles(t, root, result)
}

func TestApplyInitializeIsIdempotent(t *testing.T) {
	root := t.TempDir()
	_ = previewAndApplyInitialize(t, root)

	secondPreview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize(second) returned error: %v", err)
	}
	if secondPreview.Status != initPreviewStatusNoop {
		t.Fatalf("second preview status = %q, want %q", secondPreview.Status, initPreviewStatusNoop)
	}
	second, err := ApplyInitialize(InitApplyInput{Preview: secondPreview, ExpectedPreviewToken: secondPreview.PreviewToken})
	if err != nil {
		t.Fatalf("ApplyInitialize(second) returned error: %v", err)
	}
	if second.Status != initApplyStatusNoop {
		t.Fatalf("second apply status = %q, want %q", second.Status, initApplyStatusNoop)
	}
}

func previewAndApplyInitialize(t *testing.T, root string) InitApplyResult {
	t.Helper()
	preview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize returned error: %v", err)
	}
	result, err := ApplyInitialize(InitApplyInput{Preview: preview, ExpectedPreviewToken: preview.PreviewToken})
	if err != nil {
		t.Fatalf("ApplyInitialize returned error: %v", err)
	}
	return result
}

func assertInitializedCanonicalFiles(t *testing.T, root string, result InitApplyResult) {
	t.Helper()
	if result.Status != initApplyStatusApplied {
		t.Fatalf("status = %q, want %q", result.Status, initApplyStatusApplied)
	}
	if result.ResultingSnapshot.ValidationState != validationStateValid {
		t.Fatalf("resulting_snapshot.validation_state = %q, want %q", result.ResultingSnapshot.ValidationState, validationStateValid)
	}
	if result.ResultingSnapshot.RuneContextVersion != releaseRecommendedRuneContextVersion {
		t.Fatalf("resulting_snapshot.runecontext_version = %q, want %q", result.ResultingSnapshot.RuneContextVersion, releaseRecommendedRuneContextVersion)
	}
	assertCanonicalAssuranceBaseline(t, root)
}

func assertCanonicalAssuranceBaseline(t *testing.T, root string) {
	t.Helper()
	baselinePath := filepath.Join(root, canonicalAssuranceBaselinePath)
	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("assurance baseline stat returned error: %v", err)
	}
	baseline, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("ReadFile assurance baseline returned error: %v", err)
	}
	if !strings.Contains(string(baseline), "adoption_commit: \"0000000000000000000000000000000000000000\"") {
		t.Fatalf("assurance baseline missing adoption_commit:\n%s", string(baseline))
	}
}

func TestPreviewInitializeRefusesConflictingCandidateState(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".runecontext"), 0o755); err != nil {
		t.Fatalf("MkdirAll .runecontext returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "runecontext-legacy"), 0o755); err != nil {
		t.Fatalf("MkdirAll runecontext-legacy returned error: %v", err)
	}

	preview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize returned error: %v", err)
	}
	if preview.Status != initPreviewStatusBlocked {
		t.Fatalf("status = %q, want %q", preview.Status, initPreviewStatusBlocked)
	}
	assertHasReason(t, preview.ReasonCodes, reasonInitConflictDetected)
	assertHasReason(t, preview.ReasonCodes, reasonInitConflictPrivateMirror)
	assertHasReason(t, preview.ReasonCodes, reasonInitConflictCandidateState)
	if len(preview.ConflictingPaths) == 0 {
		t.Fatal("conflicting_paths empty, want surfaced conflicts")
	}
}

func TestApplyInitializePreflightsConflictsBeforeCreatingDirectories(t *testing.T) {
	root := t.TempDir()

	preview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, CanonicalConfigPath), []byte("conflict"), 0o644); err != nil {
		t.Fatalf("WriteFile conflict config returned error: %v", err)
	}

	result, err := ApplyInitialize(InitApplyInput{Preview: preview, ExpectedPreviewToken: preview.PreviewToken})
	if err != nil {
		t.Fatalf("ApplyInitialize returned error: %v", err)
	}
	if result.Status != initApplyStatusBlocked {
		t.Fatalf("status = %q, want %q", result.Status, initApplyStatusBlocked)
	}
	assertHasReason(t, result.ReasonCodes, reasonInitSnapshotChanged)
	if _, err := os.Stat(filepath.Join(root, CanonicalSourcePath)); !os.IsNotExist(err) {
		t.Fatalf("canonical source path exists after blocked apply, err = %v, want not exists", err)
	}
	if _, err := os.Stat(filepath.Join(root, CanonicalAssurancePath)); !os.IsNotExist(err) {
		t.Fatalf("canonical assurance path exists after blocked apply, err = %v, want not exists", err)
	}
}

func TestApplyInitializeRequiresExactPreviewToken(t *testing.T) {
	root := t.TempDir()
	preview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize returned error: %v", err)
	}
	result, err := ApplyInitialize(InitApplyInput{Preview: preview})
	if err != nil {
		t.Fatalf("ApplyInitialize returned error: %v", err)
	}
	if result.Status != initApplyStatusBlocked {
		t.Fatalf("status = %q, want %q", result.Status, initApplyStatusBlocked)
	}
	assertHasReason(t, result.ReasonCodes, reasonInitPreviewTokenMismatch)
}

func TestApplyInitializeRollsBackDirectoriesWhenConfigWriteFails(t *testing.T) {
	root := t.TempDir()
	preview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize returned error: %v", err)
	}
	originalWriteCanonicalConfigFile := writeCanonicalConfigFile
	writeCanonicalConfigFile = func(_ string, _ string) error {
		return errors.New("simulated config write failure")
	}
	t.Cleanup(func() { writeCanonicalConfigFile = originalWriteCanonicalConfigFile })
	_, err = ApplyInitialize(InitApplyInput{Preview: preview, ExpectedPreviewToken: preview.PreviewToken})
	if err == nil {
		t.Fatal("ApplyInitialize error = nil, want config write failure")
	}
	if _, statErr := os.Stat(filepath.Join(root, CanonicalSourcePath)); !os.IsNotExist(statErr) {
		t.Fatalf("canonical source path exists after failed init apply, err = %v, want not exists", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, CanonicalAssurancePath)); !os.IsNotExist(statErr) {
		t.Fatalf("canonical assurance path exists after failed init apply, err = %v, want not exists", statErr)
	}
}

func writeCanonicalV0AnchorsWithVersion(t *testing.T, root, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, CanonicalSourcePath), 0o755); err != nil {
		t.Fatalf("MkdirAll source path returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, CanonicalAssurancePath), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance path returned error: %v", err)
	}
	content := "schema_version: 1\nrunecontext_version: \"" + version + "\"\nassurance_tier: verified\nsource:\n  type: embedded\n  path: runecontext\n"
	if err := os.WriteFile(filepath.Join(root, CanonicalConfigPath), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, canonicalAssuranceBaselinePath), []byte("canonicalization: runecontext-canonical-json-v1\ncreated_at: 0\nkind: baseline\nschema_version: 1\nsubject_id: project-root\nvalue:\n  adoption_commit: 0000000000000000000000000000000000000000\n  source_posture: embedded\n"), 0o644); err != nil {
		t.Fatalf("WriteFile assurance baseline returned error: %v", err)
	}
}
