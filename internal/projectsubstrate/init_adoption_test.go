package projectsubstrate

import (
	"os"
	"path/filepath"
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
	assertHasReason(t, adopted.ReasonCodes, reasonAdoptionNotCanonical)
	assertHasReason(t, adopted.ReasonCodes, reasonAdoptionNotVerified)
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
	if len(preview.FileChanges) != 3 {
		t.Fatalf("file_changes count = %d, want 3", len(preview.FileChanges))
	}
}

func TestApplyInitializeAppliesCanonicalFilesAndIsIdempotent(t *testing.T) {
	root := t.TempDir()

	preview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize returned error: %v", err)
	}
	first, err := ApplyInitialize(InitApplyInput{Preview: preview, ExpectedPreviewToken: preview.PreviewToken})
	if err != nil {
		t.Fatalf("ApplyInitialize(first) returned error: %v", err)
	}
	if first.Status != initApplyStatusApplied {
		t.Fatalf("status = %q, want %q", first.Status, initApplyStatusApplied)
	}
	if first.ResultingSnapshot.ValidationState != validationStateValid {
		t.Fatalf("resulting_snapshot.validation_state = %q, want %q", first.ResultingSnapshot.ValidationState, validationStateValid)
	}

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
	assertHasReason(t, result.ReasonCodes, reasonInitPreviewTokenMismatch)
	if _, err := os.Stat(filepath.Join(root, CanonicalSourcePath)); !os.IsNotExist(err) {
		t.Fatalf("canonical source path exists after blocked apply, err = %v, want not exists", err)
	}
	if _, err := os.Stat(filepath.Join(root, CanonicalAssurancePath)); !os.IsNotExist(err) {
		t.Fatalf("canonical assurance path exists after blocked apply, err = %v, want not exists", err)
	}
}
