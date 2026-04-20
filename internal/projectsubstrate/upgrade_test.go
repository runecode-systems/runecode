package projectsubstrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingAuditAppender struct {
	events []map[string]interface{}
}

func (r *recordingAuditAppender) AppendTrustedAuditEvent(_ string, _ string, details map[string]interface{}) error {
	r.events = append(r.events, details)
	return nil
}

func TestPreviewUpgradeEnumeratesDeterministicChangeAndExpectedPosture(t *testing.T) {
	root := t.TempDir()
	writeUpgradeableNonVerifiedV0Anchors(t, root)

	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	if preview.Status != upgradePreviewStatusReady {
		t.Fatalf("status = %q, want %q", preview.Status, upgradePreviewStatusReady)
	}
	if len(preview.FileChanges) != 1 {
		t.Fatalf("file_changes count = %d, want 1", len(preview.FileChanges))
	}
	if got := preview.FileChanges[0].Path; got != CanonicalConfigPath {
		t.Fatalf("file_changes[0].path = %q, want %q", got, CanonicalConfigPath)
	}
	if preview.FileChanges[0].BeforeContentSHA == "" || preview.FileChanges[0].AfterContentSHA == "" {
		t.Fatal("file_changes hashes empty, want deterministic content digests")
	}
	if preview.ExpectedSnapshot.ValidationState != validationStateValid {
		t.Fatalf("expected_snapshot.validation_state = %q, want %q", preview.ExpectedSnapshot.ValidationState, validationStateValid)
	}
	if preview.PreviewDigest == "" {
		t.Fatal("preview_digest empty, want digest")
	}
	again, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade(second) returned error: %v", err)
	}
	if again.PreviewDigest != preview.PreviewDigest {
		t.Fatalf("preview_digest drift = %q, want %q", again.PreviewDigest, preview.PreviewDigest)
	}
}

func TestApplyUpgradeMutatesOnlyCanonicalConfigAndRevalidates(t *testing.T) {
	root := t.TempDir()
	writeUpgradeableNonVerifiedV0Anchors(t, root)
	beforeConfig, err := os.ReadFile(filepath.Join(root, CanonicalConfigPath))
	if err != nil {
		t.Fatalf("ReadFile(before config) returned error: %v", err)
	}
	beforeAssuranceMarker := filepath.Join(root, CanonicalAssurancePath, "marker.txt")
	if err := os.WriteFile(beforeAssuranceMarker, []byte("marker"), 0o644); err != nil {
		t.Fatalf("WriteFile(marker) returned error: %v", err)
	}

	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	result, err := ApplyUpgrade(UpgradeApplyInput{Preview: preview, ExpectedPreviewHash: preview.PreviewDigest})
	if err != nil {
		t.Fatalf("ApplyUpgrade returned error: %v", err)
	}
	if result.Status != upgradeApplyStatusApplied {
		t.Fatalf("status = %q, want %q", result.Status, upgradeApplyStatusApplied)
	}
	if result.ResultingSnapshot.ValidationState != validationStateValid {
		t.Fatalf("resulting_snapshot.validation_state = %q, want %q", result.ResultingSnapshot.ValidationState, validationStateValid)
	}
	afterConfig, err := os.ReadFile(filepath.Join(root, CanonicalConfigPath))
	if err != nil {
		t.Fatalf("ReadFile(after config) returned error: %v", err)
	}
	if string(afterConfig) == string(beforeConfig) {
		t.Fatal("runecontext.yaml unchanged, want explicit reviewed upgrade mutation")
	}
	if string(afterConfig) != "schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: verified\nsource:\n  type: \"embedded\"\n  path: \"runecontext\"\n" {
		t.Fatalf("runecontext.yaml after upgrade = %q, want canonical runectx-compatible layout", string(afterConfig))
	}
	if _, err := os.Stat(beforeAssuranceMarker); err != nil {
		t.Fatalf("assurance marker stat returned error: %v", err)
	}
}

func TestPreviewUpgradeSupportedWithUpgradeAvailableTargetsRecommendedVersion(t *testing.T) {
	root := t.TempDir()
	writeCanonicalV0AnchorsWithVersion(t, root, "0.1.0-alpha.13")

	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	if preview.Status != upgradePreviewStatusReady {
		t.Fatalf("status = %q, want %q", preview.Status, upgradePreviewStatusReady)
	}
	if got := preview.ExpectedSnapshot.RuneContextVersion; got != releaseRecommendedRuneContextVersion {
		t.Fatalf("expected_snapshot.runecontext_version = %q, want %q", got, releaseRecommendedRuneContextVersion)
	}
	if len(preview.FileChanges) != 1 {
		t.Fatalf("file_changes count = %d, want 1", len(preview.FileChanges))
	}
}

func TestApplyUpgradeForSupportedWithUpgradeAvailableAppliesRecommendedVersion(t *testing.T) {
	root := t.TempDir()
	writeCanonicalV0AnchorsWithVersion(t, root, "0.1.0-alpha.13")

	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	if preview.Status != upgradePreviewStatusReady {
		t.Fatalf("status = %q, want %q", preview.Status, upgradePreviewStatusReady)
	}

	result, err := ApplyUpgrade(UpgradeApplyInput{Preview: preview, ExpectedPreviewHash: preview.PreviewDigest})
	if err != nil {
		t.Fatalf("ApplyUpgrade returned error: %v", err)
	}
	if result.Status != upgradeApplyStatusApplied {
		t.Fatalf("status = %q, want %q", result.Status, upgradeApplyStatusApplied)
	}
	if got := result.ResultingSnapshot.RuneContextVersion; got != releaseRecommendedRuneContextVersion {
		t.Fatalf("resulting_snapshot.runecontext_version = %q, want %q", got, releaseRecommendedRuneContextVersion)
	}
	if result.ResultingSnapshot.ValidationState != validationStateValid {
		t.Fatalf("resulting_snapshot.validation_state = %q, want %q", result.ResultingSnapshot.ValidationState, validationStateValid)
	}
}

func TestApplyUpgradeNoopIsIdempotent(t *testing.T) {
	root := t.TempDir()
	writeCanonicalV0Anchors(t, root)
	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	if preview.Status != upgradePreviewStatusNoop {
		t.Fatalf("status = %q, want %q", preview.Status, upgradePreviewStatusNoop)
	}
	first, err := ApplyUpgrade(UpgradeApplyInput{Preview: preview, ExpectedPreviewHash: preview.PreviewDigest})
	if err != nil {
		t.Fatalf("ApplyUpgrade(first) returned error: %v", err)
	}
	second, err := ApplyUpgrade(UpgradeApplyInput{Preview: preview, ExpectedPreviewHash: preview.PreviewDigest})
	if err != nil {
		t.Fatalf("ApplyUpgrade(second) returned error: %v", err)
	}
	if first.Status != upgradeApplyStatusNoop || second.Status != upgradeApplyStatusNoop {
		t.Fatalf("noop apply statuses = %q/%q, want %q", first.Status, second.Status, upgradeApplyStatusNoop)
	}
	if first.ResultingSnapshot.SnapshotDigest != second.ResultingSnapshot.SnapshotDigest {
		t.Fatalf("snapshot digest drift = %q vs %q", first.ResultingSnapshot.SnapshotDigest, second.ResultingSnapshot.SnapshotDigest)
	}
}

func TestApplyUpgradePreviewDigestMismatchBlocked(t *testing.T) {
	root := t.TempDir()
	writeUpgradeableNonVerifiedV0Anchors(t, root)
	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	result, err := ApplyUpgrade(UpgradeApplyInput{Preview: preview, ExpectedPreviewHash: "sha256:" + "0"})
	if err != nil {
		t.Fatalf("ApplyUpgrade returned error: %v", err)
	}
	if result.Status != upgradeApplyStatusBlocked {
		t.Fatalf("status = %q, want %q", result.Status, upgradeApplyStatusBlocked)
	}
	assertHasReason(t, result.ReasonCodes, reasonUpgradePreviewDigestMismatch)
	config, err := os.ReadFile(filepath.Join(root, CanonicalConfigPath))
	if err != nil {
		t.Fatalf("ReadFile config returned error: %v", err)
	}
	if string(config) != "schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: plain\nsource:\n  type: embedded\n  path: runecontext\n" {
		t.Fatalf("config changed on blocked apply: %q", string(config))
	}
}

func TestApplyUpgradeAllowsEmptyExpectedPreviewDigest(t *testing.T) {
	root := t.TempDir()
	writeUpgradeableNonVerifiedV0Anchors(t, root)

	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}

	result, err := ApplyUpgrade(UpgradeApplyInput{Preview: preview})
	if err != nil {
		t.Fatalf("ApplyUpgrade returned error: %v", err)
	}
	if result.Status != upgradeApplyStatusApplied {
		t.Fatalf("status = %q, want %q", result.Status, upgradeApplyStatusApplied)
	}
	for _, reason := range result.ReasonCodes {
		if reason == reasonUpgradePreviewDigestMismatch {
			t.Fatalf("reason_codes unexpectedly contains %q: %v", reasonUpgradePreviewDigestMismatch, result.ReasonCodes)
		}
	}
}

func TestApplyUpgradeEmitsAuditEvidenceWhenAppenderProvided(t *testing.T) {
	root := t.TempDir()
	writeUpgradeableNonVerifiedV0Anchors(t, root)
	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	auditor := &recordingAuditAppender{}
	_, err = ApplyUpgrade(UpgradeApplyInput{Preview: preview, ExpectedPreviewHash: preview.PreviewDigest, AuditAppender: auditor})
	if err != nil {
		t.Fatalf("ApplyUpgrade returned error: %v", err)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	if got := auditor.events[0]["preview_digest"]; got != preview.PreviewDigest {
		t.Fatalf("audit preview_digest = %v, want %q", got, preview.PreviewDigest)
	}
}

func TestApplyInitializeEmitsAuditEvidenceWhenAppenderProvided(t *testing.T) {
	root := t.TempDir()
	preview, err := PreviewInitialize(InitPreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewInitialize returned error: %v", err)
	}
	auditor := &recordingAuditAppender{}
	_, err = ApplyInitialize(InitApplyInput{Preview: preview, ExpectedPreviewToken: preview.PreviewToken, AuditAppender: auditor})
	if err != nil {
		t.Fatalf("ApplyInitialize returned error: %v", err)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	if got := auditor.events[0]["preview_token"]; got != preview.PreviewToken {
		t.Fatalf("audit preview_token = %v, want %q", got, preview.PreviewToken)
	}
}

func TestApplyUpgradeRestoresConfigWhenPostWriteValidationErrors(t *testing.T) {
	root := t.TempDir()
	writeUpgradeableNonVerifiedV0Anchors(t, root)
	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	beforeConfig, err := os.ReadFile(filepath.Join(root, CanonicalConfigPath))
	if err != nil {
		t.Fatalf("ReadFile(before config) returned error: %v", err)
	}
	originalRevalidate := revalidateAfterUpgrade
	revalidateAfterUpgrade = func(input DiscoveryInput) (DiscoveryResult, error) {
		return DiscoveryResult{}, os.ErrInvalid
	}
	t.Cleanup(func() { revalidateAfterUpgrade = originalRevalidate })
	_, err = ApplyUpgrade(UpgradeApplyInput{Preview: preview, ExpectedPreviewHash: preview.PreviewDigest})
	if err == nil {
		t.Fatal("ApplyUpgrade error = nil, want post-write revalidation failure")
	}
	if !strings.Contains(err.Error(), "revalidate upgraded project substrate") {
		t.Fatalf("ApplyUpgrade error = %q, want revalidation context", err.Error())
	}
	afterConfig, err := os.ReadFile(filepath.Join(root, CanonicalConfigPath))
	if err != nil {
		t.Fatalf("ReadFile(after config) returned error: %v", err)
	}
	if string(afterConfig) != string(beforeConfig) {
		t.Fatalf("runecontext.yaml mutated after failed post-write validation:\nbefore:\n%s\nafter:\n%s", string(beforeConfig), string(afterConfig))
	}
}

func TestPreviewUpgradeBlocksRemediationOnlyStatesWithStableCodes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, CanonicalSourcePath), 0o755); err != nil {
		t.Fatalf("MkdirAll source returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, CanonicalAssurancePath), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, CanonicalConfigPath), []byte("schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: verified\nsource:\n  type: embedded\n  path: runecontext-custom\n"), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}

	preview, err := PreviewUpgrade(UpgradePreviewInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("PreviewUpgrade returned error: %v", err)
	}
	if preview.Status != upgradePreviewStatusBlockedRemediation {
		t.Fatalf("status = %q, want %q", preview.Status, upgradePreviewStatusBlockedRemediation)
	}
	assertHasReason(t, preview.ReasonCodes, reasonRemediationFlowRequired)
	assertHasReason(t, preview.ReasonCodes, reasonRemediationDeclaredSourceConflicts)
	if len(preview.FileChanges) != 0 {
		t.Fatalf("file_changes count = %d, want 0", len(preview.FileChanges))
	}
	if len(preview.RequiredFollowUp) == 0 {
		t.Fatal("required_follow_up empty, want remediation guidance")
	}
}

func writeUpgradeableNonVerifiedV0Anchors(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, CanonicalSourcePath), 0o755); err != nil {
		t.Fatalf("MkdirAll source path returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, CanonicalAssurancePath), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance path returned error: %v", err)
	}
	content := "schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: plain\nsource:\n  type: embedded\n  path: runecontext\n"
	if err := os.WriteFile(filepath.Join(root, CanonicalConfigPath), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}
}
