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

func testDigestIdentity(ch string) string {
	return "sha256:" + strings.Repeat(ch, 64)
}

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

func TestLookupSecretMetadataReturnsMetadataWithoutMaterial(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	imported, err := svc.ImportSecret("secrets/prod/api", strings.NewReader("api-secret"))
	if err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	meta, ok := svc.LookupSecretMetadata("secrets/prod/api")
	if !ok {
		t.Fatal("LookupSecretMetadata returned ok=false, want true")
	}
	if meta.SecretRef != imported.SecretRef || meta.SecretID != imported.SecretID || meta.MaterialDigest != imported.MaterialDigest {
		t.Fatalf("LookupSecretMetadata returned %+v, want imported metadata", meta)
	}
	if _, ok := svc.LookupSecretMetadata("secrets/prod/missing"); ok {
		t.Fatal("LookupSecretMetadata(missing) returned ok=true, want false")
	}
}

func TestGitLeaseBindingEnforcedFailClosed(t *testing.T) {
	svc, binding, lease := issueBoundGitLeaseForTest(t)
	if lease.DeliveryKind != "git_gateway" {
		t.Fatalf("lease delivery_kind = %q, want git_gateway", lease.DeliveryKind)
	}
	if lease.GitBinding == nil {
		t.Fatal("lease git_binding missing")
	}
	if got := lease.GitBinding.RepositoryIdentity; got != binding.RepositoryIdentity {
		t.Fatalf("lease git_binding.repository_identity = %q, want %q", got, binding.RepositoryIdentity)
	}
	assertGitLeaseRetrieveAllowed(t, svc, lease, gitLeaseUseContext(binding, "push_ref", binding.ActionRequestHash, binding.PolicyContextHash))
	assertGitLeaseRetrieveDenied(t, svc, lease, gitLeaseUseContext(binding, "push_ref", binding.ActionRequestHash, binding.PolicyContextHash, withGitRepositoryIdentity("git_remote:github.com/runecode-ai/other")))
	assertGitLeaseRetrieveDenied(t, svc, lease, gitLeaseUseContext(binding, "delete_ref", binding.ActionRequestHash, binding.PolicyContextHash))
	assertGitLeaseRetrieveDenied(t, svc, lease, gitLeaseUseContext(binding, "push_ref", testDigestIdentity("c"), binding.PolicyContextHash))
	assertGitLeaseRetrieveDenied(t, svc, lease, gitLeaseUseContext(binding, "push_ref", binding.ActionRequestHash, testDigestIdentity("d")))
}

func TestGitLeaseRevocationByRepositoryBinding(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	importGitProviderSecret(t, svc)
	repo := "git_remote:github.com/runecode-ai/runecode"
	actionHash := testDigestIdentity("a")
	policyHash := testDigestIdentity("b")
	first := mustIssueGitLease(t, svc, gitLeaseIssueRequest(repo, "run:git-1", "push_ref", actionHash, policyHash))
	second := mustIssueGitLease(t, svc, gitLeaseIssueRequest(repo, "run:git-1", "create_pull_request", actionHash, policyHash))
	third := mustIssueGitLease(t, svc, gitLeaseIssueRequest(repo, "run:git-2", "push_ref", testDigestIdentity("e"), policyHash))

	revoked, err := svc.RevokeGitLeases(RevokeGitLeasesRequest{RepositoryIdentity: repo, ActionRequestHash: actionHash, PolicyContextHash: policyHash, Reason: "git operation completed"})
	if err != nil {
		t.Fatalf("RevokeGitLeases returned error: %v", err)
	}
	if len(revoked) != 2 {
		t.Fatalf("revoked lease count = %d, want 2", len(revoked))
	}
	assertGitLeaseRetrieveDenied(t, svc, first, gitLeaseUseContext(first.GitBinding, "push_ref", actionHash, policyHash))
	assertGitLeaseRetrieveDenied(t, svc, second, gitLeaseUseContext(second.GitBinding, "create_pull_request", actionHash, policyHash))
	assertGitLeaseRetrieveAllowed(t, svc, third, gitLeaseUseContext(third.GitBinding, "push_ref", testDigestIdentity("e"), policyHash))
}

func issueBoundGitLeaseForTest(t *testing.T) (*Service, *GitLeaseBinding, Lease) {
	t.Helper()
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	importGitProviderSecret(t, svc)
	binding := &GitLeaseBinding{
		RepositoryIdentity: "git_remote:github.com/runecode-ai/runecode",
		AllowedOperations:  []string{"push_ref", "create_pull_request"},
		ActionRequestHash:  testDigestIdentity("a"),
		PolicyContextHash:  testDigestIdentity("b"),
	}
	lease := mustIssueGitLease(t, svc, IssueLeaseRequest{SecretRef: "secrets/prod/git/provider-token", ConsumerID: "principal:gateway:git:1", RoleKind: "git-gateway", Scope: "run:git-1", DeliveryKind: "git_gateway", GitBinding: binding, TTLSeconds: 120})
	return svc, binding, lease
}

func importGitProviderSecret(t *testing.T, svc *Service) {
	t.Helper()
	if _, err := svc.ImportSecret("secrets/prod/git/provider-token", strings.NewReader("git-provider-token")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
}

func gitLeaseIssueRequest(repo, scope, operation, actionHash, policyHash string) IssueLeaseRequest {
	return IssueLeaseRequest{SecretRef: "secrets/prod/git/provider-token", ConsumerID: "principal:gateway:git:1", RoleKind: "git-gateway", Scope: scope, DeliveryKind: "git_gateway", GitBinding: &GitLeaseBinding{RepositoryIdentity: repo, AllowedOperations: []string{operation}, ActionRequestHash: actionHash, PolicyContextHash: policyHash}, TTLSeconds: 120}
}

func mustIssueGitLease(t *testing.T, svc *Service, req IssueLeaseRequest) Lease {
	t.Helper()
	lease, err := svc.IssueLease(req)
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	return lease
}

type gitLeaseUseContextOption func(*GitLeaseUseContext)

func withGitRepositoryIdentity(repositoryIdentity string) gitLeaseUseContextOption {
	return func(ctx *GitLeaseUseContext) {
		ctx.RepositoryIdentity = repositoryIdentity
	}
}

func gitLeaseUseContext(binding *GitLeaseBinding, operation, actionHash, policyHash string, opts ...gitLeaseUseContextOption) *GitLeaseUseContext {
	ctx := &GitLeaseUseContext{RepositoryIdentity: binding.RepositoryIdentity, Operation: operation, ActionRequestHash: actionHash, PolicyContextHash: policyHash}
	for _, opt := range opts {
		opt(ctx)
	}
	return ctx
}

func assertGitLeaseRetrieveAllowed(t *testing.T, svc *Service, lease Lease, ctx *GitLeaseUseContext) {
	t.Helper()
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:gateway:git:1", RoleKind: "git-gateway", Scope: lease.Scope, DeliveryKind: "git_gateway", GitUseContext: ctx}); err != nil {
		t.Fatalf("Retrieve returned error for valid git context: %v", err)
	}
}

func assertGitLeaseRetrieveDenied(t *testing.T, svc *Service, lease Lease, ctx *GitLeaseUseContext) {
	t.Helper()
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:gateway:git:1", RoleKind: "git-gateway", Scope: lease.Scope, DeliveryKind: "git_gateway", GitUseContext: ctx}); err == nil {
		t.Fatal("Retrieve expected deny for git lease mismatch")
	}
}

func TestForbiddenSecretDeliveryInjectionChannels(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if _, err := svc.ImportSecret("secrets/prod/api", strings.NewReader("api-secret")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	if _, err := svc.IssueLease(IssueLeaseRequest{SecretRef: "secrets/prod/api", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", DeliveryKind: "environment_variable", TTLSeconds: 120}); err == nil {
		t.Fatal("IssueLease expected reject for environment_variable delivery")
	}
	if _, err := svc.IssueLease(IssueLeaseRequest{SecretRef: "secrets/prod/api", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", DeliveryKind: "cli_argument", TTLSeconds: 120}); err == nil {
		t.Fatal("IssueLease expected reject for cli_argument delivery")
	}
	lease, err := svc.IssueLease(IssueLeaseRequest{SecretRef: "secrets/prod/api", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 120})
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", DeliveryKind: "environment_variable"}); err == nil {
		t.Fatal("Retrieve expected reject for environment_variable delivery")
	}
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", DeliveryKind: "cli_argument"}); err == nil {
		t.Fatal("Retrieve expected reject for cli_argument delivery")
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

func TestSignAuditAnchorSucceedsWithValidPresenceAttestation(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	req := testAuditAnchorSignRequest(t)
	req.PresenceAttestation = mustAuditAnchorPresenceAttestationForSignTest(t, svc, "os_confirmation", req.TargetSealDigest)
	if _, err := svc.SignAuditAnchor(req); err != nil {
		t.Fatalf("SignAuditAnchor returned error: %v", err)
	}
}

func TestSignAuditAnchorFailsClosedWithoutPresenceAttestation(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	req := testAuditAnchorSignRequest(t)
	if _, err := svc.SignAuditAnchor(req); err == nil {
		t.Fatal("SignAuditAnchor expected fail-closed missing presence attestation error")
	}
}

func TestSignAuditAnchorHardwareTouchFailsClosedWithoutPresenceAttestation(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Setenv(envAuditAnchorPresenceMode, "hardware_touch")
	req := testAuditAnchorSignRequest(t)
	if _, err := svc.SignAuditAnchor(req); err == nil {
		t.Fatal("SignAuditAnchor expected fail-closed missing presence attestation error for hardware_touch")
	}
}

func TestSignAuditAnchorFailsClosedWithInvalidPresenceAttestation(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	req := testAuditAnchorSignRequest(t)
	req.PresenceAttestation = &AuditAnchorPresenceAttestation{Challenge: "presence-challenge", AcknowledgmentToken: "deadbeef"}
	if _, err := svc.SignAuditAnchor(req); err == nil {
		t.Fatal("SignAuditAnchor expected fail-closed invalid presence attestation error")
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

func TestSignAuditAnchorPassphraseOptInBehaviorUnchanged(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Setenv(envAuditAnchorAllowPassphrase, "true")
	t.Setenv(envAuditAnchorPresenceMode, "passphrase")
	t.Setenv(envAuditAnchorKeyPosture, "passphrase_wrapped")
	req := testAuditAnchorSignRequest(t)
	if _, err := svc.SignAuditAnchor(req); err != nil {
		t.Fatalf("SignAuditAnchor returned error: %v", err)
	}
}

func TestComputeAuditAnchorPresenceAcknowledgmentTokenRejectsInvalidInlineKey(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Setenv(envAuditAnchorPresenceKeyB64, "invalid-base64")
	if _, err := svc.ComputeAuditAnchorPresenceAcknowledgmentToken("os_confirmation", testAuditAnchorSignRequest(t).TargetSealDigest, "presence-challenge"); err == nil {
		t.Fatal("ComputeAuditAnchorPresenceAcknowledgmentToken expected decode failure for invalid inline key")
	}
}

func TestComputeAuditAnchorPresenceAcknowledgmentTokenRejectsWrongKeyLength(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Setenv(envAuditAnchorPresenceKeyB64, "YQ==")
	if _, err := svc.ComputeAuditAnchorPresenceAcknowledgmentToken("os_confirmation", testAuditAnchorSignRequest(t).TargetSealDigest, "presence-challenge"); err == nil {
		t.Fatal("ComputeAuditAnchorPresenceAcknowledgmentToken expected key length validation error")
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

func mustAuditAnchorPresenceAttestationForSignTest(t *testing.T, svc *Service, mode string, sealDigest trustpolicy.Digest) *AuditAnchorPresenceAttestation {
	t.Helper()
	if svc == nil {
		t.Fatal("secrets service is required")
	}
	challenge := "presence-challenge"
	challenge += "-seed-0001"
	token, err := svc.ComputeAuditAnchorPresenceAcknowledgmentToken(mode, sealDigest, challenge)
	if err != nil {
		t.Fatalf("ComputeAuditAnchorPresenceAcknowledgmentToken returned error: %v", err)
	}
	return &AuditAnchorPresenceAttestation{Challenge: challenge, AcknowledgmentToken: token}
}
