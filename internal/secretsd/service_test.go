package secretsd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestLeaseLifecycleAndBinding(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if _, err := svc.ImportSecret("secrets/prod/db", strings.NewReader("db-secret")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	lease, err := svc.IssueLease(IssueLeaseRequest{SecretRef: "secrets/prod/db", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 120})
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:other", RoleKind: "runner", Scope: "stage:alpha"}); err == nil {
		t.Fatal("Retrieve expected deny for consumer mismatch")
	}
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha"}); err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if _, err := svc.RenewLease(RenewLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 180}); err != nil {
		t.Fatalf("RenewLease returned error: %v", err)
	}
	if _, err := svc.RevokeLease(RevokeLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", Reason: "operator"}); err != nil {
		t.Fatalf("RevokeLease returned error: %v", err)
	}
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha"}); err == nil {
		t.Fatal("Retrieve expected deny after revoke")
	}
}

func TestRecoveryFailClosedWithCorruptState(t *testing.T) {
	root := t.TempDir()
	svc, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if _, err := svc.ImportSecret("secrets/prod/api", strings.NewReader("api-secret")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, stateFileName), []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if _, err := Open(root); err == nil {
		t.Fatal("Open expected fail-closed recovery error")
	} else if !errors.Is(err, ErrStateRecoveryFailed) {
		t.Fatalf("Open error = %v, want ErrStateRecoveryFailed", err)
	}
}

func TestImportSecretUsesDistinctSecretIDPrefix(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	metadata, err := svc.ImportSecret("secrets/prod/api", strings.NewReader("api-secret"))
	if err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	lease, err := svc.IssueLease(IssueLeaseRequest{SecretRef: "secrets/prod/api", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 60})
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	if !strings.HasPrefix(metadata.SecretID, "secret_") {
		t.Fatalf("secret_id = %q, want secret_ prefix", metadata.SecretID)
	}
	if !strings.HasPrefix(lease.LeaseID, "lease_") {
		t.Fatalf("lease_id = %q, want lease_ prefix", lease.LeaseID)
	}
}

func TestWriteFileAtomicReplacesExistingStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), stateFileName)
	if err := os.WriteFile(path, []byte("old-state"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := writeFileAtomic(path, []byte("new-state"), 0o600); err != nil {
		t.Fatalf("writeFileAtomic returned error: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(b) != "new-state" {
		t.Fatalf("state file contents = %q, want new-state", string(b))
	}
}

func TestReplaceFileRestoresDestinationWhenSecondRenameFails(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "state.json.tmp")
	dst := filepath.Join(dir, "state.json")
	if err := os.WriteFile(src, []byte(`{"next":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(src) returned error: %v", err)
	}
	if err := os.WriteFile(dst, []byte(`{"current":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(dst) returned error: %v", err)
	}
	originalRename := renameFile
	renameFile = func(srcPath, dstPath string) error {
		if srcPath == src && dstPath == dst {
			return errors.New("forced rename failure")
		}
		return originalRename(srcPath, dstPath)
	}
	t.Cleanup(func() {
		renameFile = originalRename
	})

	err := replaceFile(src, dst)
	if err == nil {
		t.Fatal("replaceFile expected rename failure")
	}
	b, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("ReadFile(dst) returned error: %v", readErr)
	}
	if string(b) != `{"current":true}` {
		t.Fatalf("dst contents = %q, want original contents", string(b))
	}
}

func TestReplaceFileFallbackPromotesSourceAndRemovesBackup(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "state.json.tmp")
	dst := filepath.Join(dir, "state.json")
	if err := os.WriteFile(src, []byte(`{"next":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(src) returned error: %v", err)
	}
	if err := os.WriteFile(dst, []byte(`{"current":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(dst) returned error: %v", err)
	}
	originalRename := renameFile
	firstPromote := true
	renameFile = func(srcPath, dstPath string) error {
		if srcPath == src && dstPath == dst && firstPromote {
			firstPromote = false
			return errors.New("forced first promote failure")
		}
		return originalRename(srcPath, dstPath)
	}
	t.Cleanup(func() {
		renameFile = originalRename
	})

	if err := replaceFile(src, dst); err != nil {
		t.Fatalf("replaceFile returned error: %v", err)
	}
	b, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("ReadFile(dst) returned error: %v", readErr)
	}
	if string(b) != `{"next":true}` {
		t.Fatalf("dst contents = %q, want promoted contents", string(b))
	}
	if _, statErr := os.Stat(dst + ".bak"); !os.IsNotExist(statErr) {
		t.Fatalf("backup presence err = %v, want not exist", statErr)
	}
}

func TestSignAuditAnchorRejectsUnsupportedScopeFailClosed(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	req := testAuditAnchorSignRequest(t)
	req.LogicalScope = "user"
	if _, err := svc.SignAuditAnchor(req); err == nil {
		t.Fatal("SignAuditAnchor expected fail-closed scope validation error")
	}
}

func TestSignAuditAnchorRejectsPassphraseWithoutExplicitSupport(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Setenv(envAuditAnchorPresenceMode, "passphrase")
	req := testAuditAnchorSignRequest(t)
	if _, err := svc.SignAuditAnchor(req); err == nil {
		t.Fatal("SignAuditAnchor expected passphrase opt-in enforcement")
	}
}

func TestSignAuditAnchorRejectsInconsistentPassphrasePosture(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Setenv(envAuditAnchorAllowPassphrase, "true")
	t.Setenv(envAuditAnchorPresenceMode, "passphrase")
	t.Setenv(envAuditAnchorKeyPosture, "os_keystore")
	req := testAuditAnchorSignRequest(t)
	if _, err := svc.SignAuditAnchor(req); err == nil {
		t.Fatal("SignAuditAnchor expected fail-closed inconsistent passphrase posture error")
	}
}

func testAuditAnchorSignRequest(t *testing.T) AuditAnchorSignRequest {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	sum := sha256.Sum256(pub)
	return AuditAnchorSignRequest{
		PayloadCanonicalBytes: []byte("anchor-payload"),
		TargetSealDigest: trustpolicy.Digest{
			HashAlg: "sha256",
			Hash:    hex.EncodeToString(sum[:]),
		},
		LogicalScope: "node",
	}
}
