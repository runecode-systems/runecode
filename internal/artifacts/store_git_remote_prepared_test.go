package artifacts

import (
	"path/filepath"
	"testing"
)

func TestBackupRestorePreservesGitRemotePreparedDurabilityFields(t *testing.T) {
	store := newTestStore(t)
	rec := gitRemotePreparedRecordFixture("prepared-git-1")
	if err := store.GitRemotePreparedUpsert(rec); err != nil {
		t.Fatalf("GitRemotePreparedUpsert returned error: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup-git-remote-prepared")
	if err := store.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup returned error: %v", err)
	}

	restore := newTestStore(t)
	if err := restore.RestoreBackup(backupPath); err != nil {
		t.Fatalf("RestoreBackup returned error: %v", err)
	}
	got, ok := restore.GitRemotePreparedGet(rec.PreparedMutationID)
	if !ok {
		t.Fatalf("GitRemotePreparedGet(%q) missing after restore", rec.PreparedMutationID)
	}
	if got.LastExecuteProviderLease != rec.LastExecuteProviderLease {
		t.Fatalf("last_execute_provider_auth_lease_id=%q, want %q", got.LastExecuteProviderLease, rec.LastExecuteProviderLease)
	}
	if got.LastExecuteAttemptID != rec.LastExecuteAttemptID {
		t.Fatalf("last_execute_attempt_id=%q, want %q", got.LastExecuteAttemptID, rec.LastExecuteAttemptID)
	}
	if got.LastExecuteAttemptReqID != rec.LastExecuteAttemptReqID {
		t.Fatalf("last_execute_attempt_typed_request_hash=%q, want %q", got.LastExecuteAttemptReqID, rec.LastExecuteAttemptReqID)
	}
	if got.LastExecuteSnapshotSegID != rec.LastExecuteSnapshotSegID {
		t.Fatalf("last_execute_snapshot_segment_id=%q, want %q", got.LastExecuteSnapshotSegID, rec.LastExecuteSnapshotSegID)
	}
	if got.LastExecuteSnapshotSeal != rec.LastExecuteSnapshotSeal {
		t.Fatalf("last_execute_snapshot_seal_digest=%q, want %q", got.LastExecuteSnapshotSeal, rec.LastExecuteSnapshotSeal)
	}
	if got.RequiredApprovalDecHash != rec.RequiredApprovalDecHash {
		t.Fatalf("required_approval_decision_hash=%q, want %q", got.RequiredApprovalDecHash, rec.RequiredApprovalDecHash)
	}
}

func gitRemotePreparedRecordFixture(preparedID string) GitRemotePreparedMutationRecord {
	return GitRemotePreparedMutationRecord{
		PreparedMutationID:       preparedID,
		RunID:                    "run-" + preparedID,
		Provider:                 "github",
		DestinationRef:           "github.com/runecode-ai/repo",
		RequestKind:              "git_ref_update",
		TypedRequestSchemaID:     "runecode.protocol.v0.GitRefUpdateRequest",
		TypedRequestSchemaVer:    "0.1.0",
		TypedRequest:             map[string]any{"schema_id": "runecode.protocol.v0.GitRefUpdateRequest", "schema_version": "0.1.0", "request_kind": "git_ref_update", "destination_ref": "github.com/runecode-ai/repo"},
		TypedRequestHash:         testDigest("1"),
		ActionRequestHash:        testDigest("2"),
		PolicyDecisionHash:       testDigest("3"),
		RequiredApprovalID:       testDigest("4"),
		RequiredApprovalReqHash:  testDigest("5"),
		RequiredApprovalDecHash:  testDigest("6"),
		LifecycleState:           "executing",
		ExecutionState:           "not_started",
		DerivedSummary:           map[string]any{"schema_id": "runecode.protocol.v0.GitRemoteMutationDerivedSummary", "schema_version": "0.1.0", "repository_identity": "https://github.com/runecode-ai/repo", "target_refs": []any{"refs/heads/main"}, "referenced_patch_artifact_digests": []any{}, "expected_result_tree_hash": map[string]any{"hash_alg": "sha256", "hash": testDigest("7")[7:]}},
		LastPrepareRequestID:     "req-prepare-1",
		LastGetRequestID:         "req-get-1",
		LastExecuteRequestID:     "req-exec-1",
		LastExecuteProviderLease: "lease-1",
		LastExecuteAttemptID:     testDigest("8"),
		LastExecuteAttemptReqID:  testDigest("1"),
		LastExecuteSnapshotSegID: "segment-000001",
		LastExecuteSnapshotSeal:  testDigest("9"),
		LastExecuteApprovalID:    testDigest("4"),
		LastExecuteApprovalReqID: testDigest("5"),
		LastExecuteApprovalDecID: testDigest("6"),
	}
}
