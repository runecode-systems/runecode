package localbootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeriveProductInstanceIDNormalization(t *testing.T) {
	t.Parallel()

	baseSlash := "/tmp/repo/project"
	baseOS := filepath.FromSlash(baseSlash)
	withWhitespace := "  " + baseOS + "  "

	canonical := DeriveProductInstanceID(baseOS)
	if canonical == unknownRepoID {
		t.Fatalf("DeriveProductInstanceID(%q) = %q, want hashed repo identity", baseOS, canonical)
	}
	if got := DeriveProductInstanceID(withWhitespace); got != canonical {
		t.Fatalf("DeriveProductInstanceID(%q) = %q, want %q", withWhitespace, got, canonical)
	}
	if got := DeriveProductInstanceID(baseSlash); got != canonical {
		t.Fatalf("DeriveProductInstanceID(%q) = %q, want %q", baseSlash, got, canonical)
	}
}

func TestDeriveProductInstanceIDEmptyReturnsUnknown(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", "   ", "\t\n"} {
		if got := DeriveProductInstanceID(input); got != unknownRepoID {
			t.Fatalf("DeriveProductInstanceID(%q) = %q, want %q", input, got, unknownRepoID)
		}
	}
}

func TestResolveAuthoritativeRepoRootUsesVCSAnchor(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) returned error: %v", err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll nested returned error: %v", err)
	}

	resolved, err := ResolveAuthoritativeRepoRoot(nested)
	if err != nil {
		t.Fatalf("ResolveAuthoritativeRepoRoot returned error: %v", err)
	}
	if resolved != root {
		t.Fatalf("resolved root = %q, want %q", resolved, root)
	}
}

func TestResolveAuthoritativeRepoRootFallsBackToInputWhenNoVCSAnchor(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll nested returned error: %v", err)
	}

	resolved, err := ResolveAuthoritativeRepoRoot(nested)
	if err != nil {
		t.Fatalf("ResolveAuthoritativeRepoRoot returned error: %v", err)
	}
	if resolved != nested {
		t.Fatalf("resolved root = %q, want %q", resolved, nested)
	}
}

func TestResolveRepoScopeDerivesRepoScopedPaths(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) returned error: %v", err)
	}
	runtimeBase := filepath.Join(t.TempDir(), "xdg-runtime")
	if err := os.MkdirAll(runtimeBase, 0o755); err != nil {
		t.Fatalf("MkdirAll runtime base returned error: %v", err)
	}
	t.Setenv("XDG_RUNTIME_DIR", runtimeBase)

	scope, err := ResolveRepoScope(ResolveInput{RepositoryRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}
	if scope.RepositoryRoot != repoRoot {
		t.Fatalf("repository root = %q, want %q", scope.RepositoryRoot, repoRoot)
	}
	if !strings.HasPrefix(scope.ProductInstance, repoIDPrefix) {
		t.Fatalf("product instance = %q, want %q prefix", scope.ProductInstance, repoIDPrefix)
	}
	if !strings.Contains(filepath.ToSlash(scope.StateRoot), "/runecode/repos/") {
		t.Fatalf("state root = %q, want repo-scoped runecode/repos path", scope.StateRoot)
	}
	if !strings.Contains(filepath.ToSlash(scope.AuditLedgerRoot), "/runecode/repos/") {
		t.Fatalf("audit ledger root = %q, want repo-scoped runecode/repos path", scope.AuditLedgerRoot)
	}
	if !strings.HasPrefix(scope.LocalRuntimeDir, runtimeBase) {
		t.Fatalf("local runtime dir = %q, want prefix %q", scope.LocalRuntimeDir, runtimeBase)
	}
	if scope.LocalSocketName != defaultSocketName {
		t.Fatalf("local socket name = %q, want %q", scope.LocalSocketName, defaultSocketName)
	}
}

func TestResolveAuthoritativeRepoRootIgnoresSymlinkGitAnchor(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(root, ".git")); err != nil {
		t.Skipf("symlink unsupported on this filesystem: %v", err)
	}
	nested := filepath.Join(root, "child")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll child returned error: %v", err)
	}

	resolved, err := ResolveAuthoritativeRepoRoot(nested)
	if err != nil {
		t.Fatalf("ResolveAuthoritativeRepoRoot returned error: %v", err)
	}
	if resolved != nested {
		t.Fatalf("resolved root = %q, want %q when .git anchor is symlink", resolved, nested)
	}
}

func TestUserRuntimeBaseDirFallsBackToUserCacheDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	base, err := userRuntimeBaseDir()
	if err != nil {
		t.Fatalf("userRuntimeBaseDir returned error: %v", err)
	}
	if strings.Contains(filepath.ToSlash(base), "/tmp/runecode") {
		t.Fatalf("runtime base = %q, want non-shared user-scoped fallback", base)
	}
}
