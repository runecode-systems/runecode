package artifacts

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupRestorePreservesPolicyDecisionState(t *testing.T) {
	store := newTestStore(t)
	rec := basePolicyDecisionRecord("run-backup-policy", map[string]any{"precedence": "invariants_first"})
	if err := store.RecordPolicyDecision(rec); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	storedDecision, ok := firstPolicyDecisionRecord(store)
	if !ok {
		t.Fatal("policy decision missing from source state")
	}

	backupPath := filepath.Join(t.TempDir(), "backup-policy.json")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup returned error: %v", err)
	}

	restoreStore := newTestStore(t)
	if err := restoreStore.RestoreBackup(backupPath); err != nil {
		t.Fatalf("RestoreBackup returned error: %v", err)
	}

	restoredDecision, ok := restoreStore.state.PolicyDecisions[storedDecision.Digest]
	if !ok {
		t.Fatalf("restored policy decision %q missing", storedDecision.Digest)
	}
	if restoredDecision.RunID != rec.RunID {
		t.Fatalf("restored run_id = %q, want %q", restoredDecision.RunID, rec.RunID)
	}
	refs := restoreStore.PolicyDecisionRefsForRun(rec.RunID)
	if len(refs) != 1 || refs[0] != storedDecision.Digest {
		t.Fatalf("restored PolicyDecisionRefsForRun = %v, want [%s]", refs, storedDecision.Digest)
	}
}

func TestRestoreRejectsPolicyDecisionDigestMismatch(t *testing.T) {
	store := newTestStore(t)
	backupPath := writeBackupWithPolicyDecisionDigestMismatch(t, store)
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err == nil {
		t.Fatal("RestoreBackup expected policy decision digest mismatch error")
	}
	if !strings.Contains(err.Error(), "policy decision digest mismatch") {
		t.Fatalf("RestoreBackup error = %v, want policy decision digest mismatch", err)
	}
}

func TestRestoreRejectsBoundApprovalMissingPolicyDecisionHash(t *testing.T) {
	store := newTestStore(t)
	backupPath := writeBackupWithBoundApprovalMissingPolicyDecisionHash(t, store)
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err != ErrApprovalPolicyDecisionRequired {
		t.Fatalf("RestoreBackup error = %v, want %v", err, ErrApprovalPolicyDecisionRequired)
	}
}

func TestRestoreRejectsBoundApprovalWithDanglingPolicyDecisionHash(t *testing.T) {
	store := newTestStore(t)
	backupPath := writeBackupWithBoundApprovalDanglingPolicyDecisionHash(t, store)
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err == nil {
		t.Fatal("RestoreBackup expected ErrApprovalPolicyDecisionRequired for dangling policy decision hash")
	}
	if !strings.Contains(err.Error(), ErrApprovalPolicyDecisionRequired.Error()) {
		t.Fatalf("RestoreBackup error = %v, want %v", err, ErrApprovalPolicyDecisionRequired)
	}
}

func TestRestoreRejectsBoundApprovalPolicyDecisionBindingMismatch(t *testing.T) {
	store := newTestStore(t)
	backupPath := writeBackupWithBoundApprovalPolicyDecisionBindingMismatch(t, store)
	restoreStore := newTestStore(t)
	err := restoreStore.RestoreBackup(backupPath)
	if err == nil {
		t.Fatal("RestoreBackup expected ErrApprovalPolicyDecisionRequired for policy decision binding mismatch")
	}
	if !strings.Contains(err.Error(), ErrApprovalPolicyDecisionRequired.Error()) {
		t.Fatalf("RestoreBackup error = %v, want %v", err, ErrApprovalPolicyDecisionRequired)
	}
}

func writeBackupWithBoundApprovalMissingPolicyDecisionHash(t *testing.T, store *Store) string {
	t.Helper()
	approval := ApprovalRecord{ApprovalID: testDigest("a"), Status: "pending", WorkspaceID: "workspace-local", RunID: "run-restore-policy-hash", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", RequestedAt: time.Now().UTC(), ApprovalTriggerCode: "excerpt_promotion", ChangesIfApproved: approvalChangesIfApprovedDefault, ApprovalAssuranceLevel: "session_authenticated", PresenceMode: "os_confirmation", ManifestHash: testDigest("1"), ActionRequestHash: testDigest("2")}
	manifest := BackupManifest{Schema: "runecode.backup.artifacts.v1", ExportedAt: time.Now().UTC(), StorageProtection: "encrypted_at_rest_default", Policy: DefaultPolicy(), Runs: map[string]string{"run-restore-policy-hash": "active"}, Artifacts: []ArtifactRecord{}, Approvals: []ApprovalRecord{approval}}
	backupPath := filepath.Join(t.TempDir(), "backup-bound-approval-missing-policy-hash.json")
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	return backupPath
}

func writeBackupWithPolicyDecisionDigestMismatch(t *testing.T, store *Store) string {
	t.Helper()
	decision := basePolicyDecisionRecord("run-restore-digest-mismatch", map[string]any{"precedence": "deny"})
	decision.Digest = testDigest("f")
	manifest := BackupManifest{Schema: "runecode.backup.artifacts.v1", ExportedAt: time.Now().UTC(), StorageProtection: "encrypted_at_rest_default", Policy: DefaultPolicy(), Runs: map[string]string{"run-restore-digest-mismatch": "active"}, Artifacts: []ArtifactRecord{}, PolicyDecisions: []PolicyDecisionRecord{decision}}
	backupPath := filepath.Join(t.TempDir(), "backup-policy-decision-digest-mismatch.json")
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	return backupPath
}

func writeBackupWithBoundApprovalDanglingPolicyDecisionHash(t *testing.T, store *Store) string {
	t.Helper()
	approval := ApprovalRecord{ApprovalID: testDigest("a"), Status: "pending", WorkspaceID: "workspace-local", RunID: "run-restore-policy-hash", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", RequestedAt: time.Now().UTC(), ApprovalTriggerCode: "excerpt_promotion", ChangesIfApproved: approvalChangesIfApprovedDefault, ApprovalAssuranceLevel: "session_authenticated", PresenceMode: "os_confirmation", ManifestHash: testDigest("1"), ActionRequestHash: testDigest("2"), PolicyDecisionHash: testDigest("d")}
	manifest := BackupManifest{Schema: "runecode.backup.artifacts.v1", ExportedAt: time.Now().UTC(), StorageProtection: "encrypted_at_rest_default", Policy: DefaultPolicy(), Runs: map[string]string{"run-restore-policy-hash": "active"}, Artifacts: []ArtifactRecord{}, Approvals: []ApprovalRecord{approval}, PolicyDecisions: []PolicyDecisionRecord{}}
	backupPath := filepath.Join(t.TempDir(), "backup-bound-approval-dangling-policy-hash.json")
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	return backupPath
}

func writeBackupWithBoundApprovalPolicyDecisionBindingMismatch(t *testing.T, store *Store) string {
	t.Helper()
	decision := basePolicyDecisionRecord("run-restore-policy-hash", map[string]any{"precedence": "deny"})
	if _, payloadBytes, err := canonicalizePolicyDecisionRecord(decision); err != nil {
		t.Fatalf("canonicalizePolicyDecisionRecord returned error: %v", err)
	} else {
		decision.Digest = digestBytes(payloadBytes)
	}
	approval := ApprovalRecord{ApprovalID: testDigest("a"), Status: "pending", WorkspaceID: "workspace-local", RunID: "run-restore-policy-hash", StageID: "artifact_flow", StepID: "step-1", ActionKind: "promotion", RequestedAt: time.Now().UTC(), ApprovalTriggerCode: "excerpt_promotion", ChangesIfApproved: approvalChangesIfApprovedDefault, ApprovalAssuranceLevel: "session_authenticated", PresenceMode: "os_confirmation", ManifestHash: testDigest("f"), ActionRequestHash: decision.ActionRequestHash, PolicyDecisionHash: decision.Digest}
	manifest := BackupManifest{Schema: "runecode.backup.artifacts.v1", ExportedAt: time.Now().UTC(), StorageProtection: "encrypted_at_rest_default", Policy: DefaultPolicy(), Runs: map[string]string{"run-restore-policy-hash": "active"}, Artifacts: []ArtifactRecord{}, Approvals: []ApprovalRecord{approval}, PolicyDecisions: []PolicyDecisionRecord{decision}}
	backupPath := filepath.Join(t.TempDir(), "backup-bound-approval-policy-hash-binding-mismatch.json")
	writeBackupManifestWithSignature(t, store, backupPath, manifest)
	return backupPath
}
