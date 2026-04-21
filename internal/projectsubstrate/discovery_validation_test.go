package projectsubstrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAndValidateDeterministicRootNoUpwardSearch(t *testing.T) {
	parent := t.TempDir()
	writeCanonicalV0Anchors(t, parent)
	nested := filepath.Join(parent, "sub", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll nested returned error: %v", err)
	}
	result, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: nested, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("DiscoverAndValidate returned error: %v", err)
	}
	if result.RepositoryRoot != nested {
		t.Fatalf("repository_root = %q, want %q", result.RepositoryRoot, nested)
	}
	if got := result.Snapshot.ValidationState; got != validationStateMissing {
		t.Fatalf("validation_state = %q, want %q", got, validationStateMissing)
	}
	assertHasReason(t, result.Snapshot.ReasonCodes, reasonMissingConfigAnchor)
	assertHasReason(t, result.Snapshot.ReasonCodes, reasonMissingSourceAnchor)
	assertHasReason(t, result.Snapshot.ReasonCodes, reasonMissingAssuranceAnchor)
	assertHasReason(t, result.Snapshot.ReasonCodes, reasonMissingAssuranceBaseline)
}

func TestDiscoverAndValidateRejectsNonAbsoluteRepositoryRoot(t *testing.T) {
	_, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: "relative/root", Authority: RepoRootAuthorityExplicitConfig})
	if err == nil {
		t.Fatal("DiscoverAndValidate error = nil, want root-invalid error")
	}
	if got := err.Error(); got != reasonDiscoveryRootInvalid {
		t.Fatalf("error = %q, want %q", got, reasonDiscoveryRootInvalid)
	}
}

func TestDiscoverAndValidateRejectsEmptyExplicitRepositoryRoot(t *testing.T) {
	_, err := DiscoverAndValidate(DiscoveryInput{Authority: RepoRootAuthorityExplicitConfig})
	if err == nil {
		t.Fatal("DiscoverAndValidate error = nil, want root-invalid error")
	}
	if got := err.Error(); got != reasonDiscoveryRootInvalid {
		t.Fatalf("error = %q, want %q", got, reasonDiscoveryRootInvalid)
	}
}

func TestDiscoverAndValidateCanonicalV0AnchorsValid(t *testing.T) {
	root := t.TempDir()
	writeCanonicalV0Anchors(t, root)
	result, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("DiscoverAndValidate returned error: %v", err)
	}
	if got := result.Snapshot.ValidationState; got != validationStateValid {
		t.Fatalf("validation_state = %q, want %q", got, validationStateValid)
	}
	if got := result.Compatibility.Posture; got != CompatibilityPostureSupportedCurrent {
		t.Fatalf("compatibility_posture = %q, want %q", got, CompatibilityPostureSupportedCurrent)
	}
	if !result.Compatibility.NormalOperationAllowed {
		t.Fatal("normal_operation_allowed = false, want true")
	}
	if result.Snapshot.ValidatedSnapshotDigest == "" {
		t.Fatal("validated_snapshot_digest empty, want digest")
	}
	if result.Snapshot.ProjectContextIdentityDigest != result.Snapshot.ValidatedSnapshotDigest {
		t.Fatalf("project_context_identity_digest = %q, want %q", result.Snapshot.ProjectContextIdentityDigest, result.Snapshot.ValidatedSnapshotDigest)
	}
}

func TestDiscoverAndValidateDigestStableAcrossReasonOrder(t *testing.T) {
	root := t.TempDir()
	writeCanonicalV0Anchors(t, root)
	first, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("DiscoverAndValidate(first) returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".runecontext"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.runecontext) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, CanonicalConfigPath), []byte("schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: plain\nsource:\n  type: embedded\n  path: wrong\n"), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}
	second, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("DiscoverAndValidate(second) returned error: %v", err)
	}
	third, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("DiscoverAndValidate(third) returned error: %v", err)
	}
	if second.Snapshot.SnapshotDigest == first.Snapshot.SnapshotDigest {
		t.Fatalf("snapshot_digest unchanged = %q, want changed when posture changes", second.Snapshot.SnapshotDigest)
	}
	if third.Snapshot.SnapshotDigest != second.Snapshot.SnapshotDigest {
		t.Fatalf("snapshot_digest drift = %q, want %q", third.Snapshot.SnapshotDigest, second.Snapshot.SnapshotDigest)
	}
}

func TestDiscoverAndValidateInvalidCases(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, CanonicalSourcePath), 0o755); err != nil {
		t.Fatalf("MkdirAll source path returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, CanonicalAssurancePath), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance path returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, CanonicalConfigPath), []byte("schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: plain\nsource:\n  type: embedded\n  path: runecontext-custom\n"), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".runecontext"), 0o755); err != nil {
		t.Fatalf("MkdirAll private mirror returned error: %v", err)
	}
	result, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: root, Authority: RepoRootAuthorityExplicitConfig})
	if err != nil {
		t.Fatalf("DiscoverAndValidate returned error: %v", err)
	}
	if got := result.Snapshot.ValidationState; got != validationStateInvalid {
		t.Fatalf("validation_state = %q, want %q", got, validationStateInvalid)
	}
	if got := result.Compatibility.Posture; got != CompatibilityPostureNonVerified {
		t.Fatalf("compatibility_posture = %q, want %q", got, CompatibilityPostureNonVerified)
	}
	if result.Compatibility.NormalOperationAllowed {
		t.Fatal("normal_operation_allowed = true, want false")
	}
	assertHasReason(t, result.Snapshot.ReasonCodes, reasonNonVerifiedPosture)
	assertHasReason(t, result.Snapshot.ReasonCodes, reasonNonCanonicalSourcePath)
	assertHasReason(t, result.Snapshot.ReasonCodes, reasonPrivateMirrorDetected)
	if result.Snapshot.ProjectContextIdentityDigest != result.Snapshot.SnapshotDigest {
		t.Fatalf("project_context_identity_digest = %q, want %q", result.Snapshot.ProjectContextIdentityDigest, result.Snapshot.SnapshotDigest)
	}
}

func writeCanonicalV0Anchors(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, CanonicalSourcePath), 0o755); err != nil {
		t.Fatalf("MkdirAll source path returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, CanonicalAssurancePath), 0o755); err != nil {
		t.Fatalf("MkdirAll assurance path returned error: %v", err)
	}
	content := "schema_version: 1\nrunecontext_version: \"0.1.0-alpha.14\"\nassurance_tier: verified\nsource:\n  type: embedded\n  path: runecontext\n"
	if err := os.WriteFile(filepath.Join(root, CanonicalConfigPath), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile runecontext.yaml returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, canonicalAssuranceBaselinePath), []byte("canonicalization: runecontext-canonical-json-v1\ncreated_at: 0\nkind: baseline\nschema_version: 1\nsubject_id: project-root\nvalue:\n  source_posture: embedded\n"), 0o644); err != nil {
		t.Fatalf("WriteFile assurance baseline returned error: %v", err)
	}
}

func assertHasReason(t *testing.T, reasons []string, want string) {
	t.Helper()
	for _, reason := range reasons {
		if reason == want {
			return
		}
	}
	t.Fatalf("reason_codes = %v, want %q", reasons, want)
}
