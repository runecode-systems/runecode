package brokerapi

import (
	"context"
	"testing"
)

func TestGitRemoteMutationGetAllowsLegacyExecutedPreparedStateWithoutNewExecuteBindings(t *testing.T) {
	s := newPreparedGitMutationForExecuteTests(t, "run-git-legacy-executed")
	preparedID := mustPreparedMutationIDForRun(t, s, "run-git-legacy-executed")
	persistLegacyExecutedGitPreparedState(t, s, preparedID)

	resp, errResp := s.HandleGitRemoteMutationGet(context.Background(), GitRemoteMutationGetRequest{
		SchemaID:           "runecode.protocol.v0.GitRemoteMutationGetRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-git-get-legacy-executed",
		PreparedMutationID: preparedID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleGitRemoteMutationGet returned error: %+v", errResp)
	}
	assertLegacyExecutedGitPreparedState(t, resp.Prepared)
}

func persistLegacyExecutedGitPreparedState(t *testing.T, s *Service, preparedID string) {
	t.Helper()
	rec, ok := s.GitRemotePreparedGet(preparedID)
	if !ok {
		t.Fatalf("GitRemotePreparedGet(%q) missing", preparedID)
	}
	rec.LifecycleState = gitRemoteMutationLifecycleExecuted
	rec.ExecutionState = gitRemoteMutationExecutionCompleted
	rec.LastExecuteRequestID = "req-git-execute-legacy"
	rec.LastExecuteProviderLease = ""
	rec.LastExecuteAttemptID = ""
	rec.LastExecuteAttemptReqID = ""
	rec.LastExecuteSnapshotSegID = ""
	rec.LastExecuteSnapshotSeal = ""
	if err := s.GitRemotePreparedUpsert(rec); err != nil {
		t.Fatalf("GitRemotePreparedUpsert(legacy executed record) returned error: %v", err)
	}
}

func assertLegacyExecutedGitPreparedState(t *testing.T, prepared GitRemoteMutationPreparedState) {
	t.Helper()
	if got := prepared.LastGetRequestID; got != "req-git-get-legacy-executed" {
		t.Fatalf("last_get_request_id=%q, want req-git-get-legacy-executed", got)
	}
	if got := prepared.LastExecuteRequestID; got != "req-git-execute-legacy" {
		t.Fatalf("last_execute_request_id=%q, want req-git-execute-legacy", got)
	}
	if prepared.LastExecuteProviderLeaseID != "" {
		t.Fatalf("last_execute_provider_auth_lease_id=%q, want empty for legacy record", prepared.LastExecuteProviderLeaseID)
	}
	if prepared.LastExecuteAttemptID != "" {
		t.Fatalf("last_execute_attempt_id=%q, want empty for legacy record", prepared.LastExecuteAttemptID)
	}
	if prepared.LastExecuteAttemptRequestID != nil {
		t.Fatalf("last_execute_attempt_typed_request_hash=%+v, want nil for legacy record", prepared.LastExecuteAttemptRequestID)
	}
}
