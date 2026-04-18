package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestGitRemoteMutationPrepareGetExecuteMaintainsTypedAuthorityAndBindings(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	s.gitMutationExecutor = fakeGitRemoteMutationExecutor{}
	runID := "run-git-prepare-execute"
	putTrustedGitGatewayContextForRun(t, s, runID, []any{trustedGitGatewayAllowlistEntry()})

	prepareResp := mustPrepareGitRemoteMutation(t, s, runID)
	assertPreparedGitRemoteMutation(t, prepareResp)
	assertFetchedPreparedGitRemoteMutation(t, s, prepareResp.PreparedMutationID)
	approvalID, approvalRequestDigest, approvalDecisionDigest := resolveApprovalForPreparedMutation(t, s, runID, prepareResp)
	execResp := mustExecutePreparedGitRemoteMutation(t, s, prepareResp.PreparedMutationID, approvalID, approvalRequestDigest, approvalDecisionDigest, "req-git-execute")
	assertExecutedGitRemoteMutation(t, prepareResp, execResp)
}

func TestGitRemoteMutationExecuteFailsClosedOnBindingMismatch(t *testing.T) {
	s := newPreparedGitMutationForExecuteTests(t, "run-git-execute-mismatch")
	s.gitMutationExecutor = fakeGitRemoteMutationExecutor{}
	preparedID, requestDigest, decisionDigest := resolvePreparedApprovalForExecuteTestsForRun(t, s, "")

	wrongRequest := trustpolicy.Digest{HashAlg: requestDigest.HashAlg, Hash: strings.Repeat("a", len(requestDigest.Hash))}
	_, errResp := s.HandleGitRemoteMutationExecute(context.Background(), GitRemoteMutationExecuteRequest{
		SchemaID:             "runecode.protocol.v0.GitRemoteMutationExecuteRequest",
		SchemaVersion:        "0.1.0",
		RequestID:            "req-git-execute-mismatch",
		PreparedMutationID:   preparedID,
		ApprovalID:           mustPreparedApprovalID(t, s, preparedID),
		ApprovalRequestHash:  wrongRequest,
		ApprovalDecisionHash: decisionDigest,
		ProviderAuthLeaseID:  "lease-git-provider",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleGitRemoteMutationExecute expected binding mismatch error")
	}
	if errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("error.code=%q, want broker_approval_state_invalid", errResp.Error.Code)
	}
	if !strings.Contains(errResp.Error.Message, "approval_request_hash") {
		t.Fatalf("error.message=%q, want approval_request_hash mismatch", errResp.Error.Message)
	}
}

func TestGitRemoteMutationPrepareFailsClosedWhenApprovalNotDerivable(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-git-prepare-no-context"
	_, errResp := s.HandleGitRemoteMutationPrepare(context.Background(), GitRemoteMutationPrepareRequest{
		SchemaID:      "runecode.protocol.v0.GitRemoteMutationPrepareRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-git-prepare-no-context",
		RunID:         runID,
		Provider:      "github",
		TypedRequest:  trustedGitRequestPayload(t, "git_ref_update"),
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleGitRemoteMutationPrepare expected fail-closed policy error")
	}
	if errResp.Error.Code != "gateway_failure" && errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error.code=%q, want gateway_failure or broker_limit_policy_rejected", errResp.Error.Code)
	}
}

func TestGitRemoteMutationSummaryIsDerivedAndNonAuthoritative(t *testing.T) {
	s := newPreparedGitMutationForExecuteTests(t, "run-git-summary-derived")
	s.gitMutationExecutor = fakeGitRemoteMutationExecutor{}
	preparedID := mustPreparedMutationIDForRun(t, s, "run-git-summary-derived")
	record, ok := s.GitRemotePreparedGet(preparedID)
	if !ok {
		t.Fatalf("GitRemotePreparedGet(%q) missing", preparedID)
	}
	record.DerivedSummary = map[string]any{
		"schema_id":                         "runecode.protocol.v0.GitRemoteMutationDerivedSummary",
		"schema_version":                    "0.1.0",
		"repository_identity":               "tampered.example/repo",
		"target_refs":                       []any{"refs/heads/tampered"},
		"referenced_patch_artifact_digests": []any{map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}},
		"expected_result_tree_hash":         map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)},
	}
	if err := s.GitRemotePreparedUpsert(record); err != nil {
		t.Fatalf("GitRemotePreparedUpsert(tampered summary) returned error: %v", err)
	}

	preparedID, requestDigest, decisionDigest := resolvePreparedApprovalForExecuteTestsForRun(t, s, "")
	_, errResp := s.HandleGitRemoteMutationExecute(context.Background(), GitRemoteMutationExecuteRequest{
		SchemaID:             "runecode.protocol.v0.GitRemoteMutationExecuteRequest",
		SchemaVersion:        "0.1.0",
		RequestID:            "req-git-execute-derived-summary",
		PreparedMutationID:   preparedID,
		ApprovalID:           mustPreparedApprovalID(t, s, preparedID),
		ApprovalRequestHash:  requestDigest,
		ApprovalDecisionHash: decisionDigest,
		ProviderAuthLeaseID:  "lease-git-provider",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleGitRemoteMutationExecute error: %+v", errResp)
	}
}

func TestGitRemoteMutationExecuteFailsClosedOnDriftAndPersistsFailedState(t *testing.T) {
	s := newPreparedGitMutationForExecuteTests(t, "run-git-execute-drift")
	s.gitMutationExecutor = fakeGitRemoteMutationExecutorErr{err: &gitRemoteExecutionError{code: "broker_approval_state_invalid", category: "auth", reasonCode: "git_remote_drift_detected", message: "drift", executionState: gitRemoteMutationExecutionBlocked}}
	preparedID, requestDigest, decisionDigest := resolvePreparedApprovalForExecuteTestsForRun(t, s, "run-git-execute-drift")

	_, errResp := s.HandleGitRemoteMutationExecute(context.Background(), GitRemoteMutationExecuteRequest{
		SchemaID:             "runecode.protocol.v0.GitRemoteMutationExecuteRequest",
		SchemaVersion:        "0.1.0",
		RequestID:            "req-git-execute-drift",
		PreparedMutationID:   preparedID,
		ApprovalID:           mustPreparedApprovalID(t, s, preparedID),
		ApprovalRequestHash:  requestDigest,
		ApprovalDecisionHash: decisionDigest,
		ProviderAuthLeaseID:  "lease-git-provider",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleGitRemoteMutationExecute expected drift fail-closed error")
	}
	if errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("error.code=%q, want broker_approval_state_invalid", errResp.Error.Code)
	}
	rec, ok := s.GitRemotePreparedGet(preparedID)
	if !ok {
		t.Fatalf("GitRemotePreparedGet(%q) missing", preparedID)
	}
	if rec.ExecutionState != gitRemoteMutationExecutionBlocked {
		t.Fatalf("execution_state=%q, want blocked", rec.ExecutionState)
	}
	if rec.ExecutionReasonCode != "git_remote_drift_detected" {
		t.Fatalf("execution_reason_code=%q, want git_remote_drift_detected", rec.ExecutionReasonCode)
	}
}

func TestGitRemoteMutationExecuteFailsClosedOnTreeMismatchAndPersistsFailedState(t *testing.T) {
	s := newPreparedGitMutationForExecuteTests(t, "run-git-execute-tree")
	s.gitMutationExecutor = fakeGitRemoteMutationExecutorErr{err: &gitRemoteExecutionError{code: "broker_approval_state_invalid", category: "auth", reasonCode: "git_result_tree_hash_mismatch", message: "tree mismatch", executionState: gitRemoteMutationExecutionBlocked}}
	preparedID, requestDigest, decisionDigest := resolvePreparedApprovalForExecuteTestsForRun(t, s, "run-git-execute-tree")

	_, errResp := s.HandleGitRemoteMutationExecute(context.Background(), GitRemoteMutationExecuteRequest{
		SchemaID:             "runecode.protocol.v0.GitRemoteMutationExecuteRequest",
		SchemaVersion:        "0.1.0",
		RequestID:            "req-git-execute-tree",
		PreparedMutationID:   preparedID,
		ApprovalID:           mustPreparedApprovalID(t, s, preparedID),
		ApprovalRequestHash:  requestDigest,
		ApprovalDecisionHash: decisionDigest,
		ProviderAuthLeaseID:  "lease-git-provider",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleGitRemoteMutationExecute expected tree mismatch fail-closed error")
	}
	if errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("error.code=%q, want broker_approval_state_invalid", errResp.Error.Code)
	}
	rec, ok := s.GitRemotePreparedGet(preparedID)
	if !ok {
		t.Fatalf("GitRemotePreparedGet(%q) missing", preparedID)
	}
	if rec.ExecutionState != gitRemoteMutationExecutionBlocked {
		t.Fatalf("execution_state=%q, want blocked", rec.ExecutionState)
	}
	if rec.ExecutionReasonCode != "git_result_tree_hash_mismatch" {
		t.Fatalf("execution_reason_code=%q, want git_result_tree_hash_mismatch", rec.ExecutionReasonCode)
	}
}

type fakeGitRemoteMutationExecutor struct{}

func (fakeGitRemoteMutationExecutor) executePreparedMutation(_ context.Context, req gitRemoteExecutionRequest) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	typedHash, _ := digestFromIdentity(req.Record.TypedRequestHash)
	return gitRuntimeProofPayload{
		SchemaID:               "runecode.protocol.v0.GitRuntimeProof",
		SchemaVersion:          "0.1.0",
		TypedRequestHash:       typedHash,
		PatchArtifactDigests:   []trustpolicy.Digest{{HashAlg: "sha256", Hash: strings.Repeat("5", 64)}},
		ExpectedOldObjectID:    strings.Repeat("a", 40),
		ObservedOldObjectID:    strings.Repeat("a", 40),
		ExpectedResultTreeHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("4", 64)},
		ObservedResultTreeHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("4", 64)},
		SparseCheckoutApplied:  true,
		DriftDetected:          false,
		DestructiveRefMutation: false,
		ProviderKind:           "github",
	}, nil
}

type fakeGitRemoteMutationExecutorErr struct {
	err *gitRemoteExecutionError
}

func (f fakeGitRemoteMutationExecutorErr) executePreparedMutation(_ context.Context, req gitRemoteExecutionRequest) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	typedHash, _ := digestFromIdentity(req.Record.TypedRequestHash)
	proof := gitRuntimeProofPayload{SchemaID: "runecode.protocol.v0.GitRuntimeProof", SchemaVersion: "0.1.0", TypedRequestHash: typedHash, ExpectedOldObjectID: strings.Repeat("a", 40), ObservedOldObjectID: strings.Repeat("b", 40), ExpectedResultTreeHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("4", 64)}, ObservedResultTreeHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("f", 64)}, SparseCheckoutApplied: true, DriftDetected: true, ProviderKind: "github"}
	return proof, f.err
}

func newPreparedGitMutationForExecuteTests(t *testing.T, runID string) *Service {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedGitGatewayContextForRun(t, s, runID, []any{trustedGitGatewayAllowlistEntry()})
	prepareResp, errResp := prepareGitMutationForRun(t, s, runID, "req-prepare-"+runID)
	if errResp != nil {
		t.Fatalf("HandleGitRemoteMutationPrepare returned error: %+v", errResp)
	}
	if prepareResp.PreparedMutationID == "" {
		t.Fatalf("prepared_mutation_id empty for run %q", runID)
	}
	return s
}

func resolvePreparedApprovalForExecuteTestsForRun(t *testing.T, s *Service, runID string) (string, trustpolicy.Digest, trustpolicy.Digest) {
	t.Helper()
	preparedID := mustPreparedMutationIDForRun(t, s, runID)
	approvalID := mustPreparedApprovalID(t, s, preparedID)
	approvalAfterResolve := resolveApprovalForPreparedID(t, s, approvalID, approvalBoundScopeFromApprovalRecord(t, s, approvalID))
	requestDigest := mustDigestFromIdentity(t, approvalAfterResolve.RequestDigest, "request digest")
	decisionDigest := mustDigestFromIdentity(t, approvalAfterResolve.DecisionDigest, "decision digest")
	return preparedID, requestDigest, decisionDigest
}

func prepareGitMutationForRun(t *testing.T, s *Service, runID, requestID string) (GitRemoteMutationPrepareResponse, *ErrorResponse) {
	t.Helper()
	return s.HandleGitRemoteMutationPrepare(context.Background(), GitRemoteMutationPrepareRequest{
		SchemaID:      "runecode.protocol.v0.GitRemoteMutationPrepareRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		RunID:         runID,
		Provider:      "github",
		TypedRequest:  trustedGitRequestPayload(t, "git_ref_update"),
	}, RequestContext{})
}

func resolveApprovalForPreparedID(t *testing.T, s *Service, approvalID string, boundScope ApprovalBoundScope) artifacts.ApprovalRecord {
	t.Helper()
	approvalRecord, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	requestEnv, decisionEnv, verifier := signedResolveEnvelopesForStoredPendingRequest(t, *approvalRecord.RequestEnvelope, "human", "approve")
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	_, resolveErr := s.HandleApprovalResolve(context.Background(), ApprovalResolveRequest{
		SchemaID:               "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              "req-resolve-" + approvalID,
		ApprovalID:             approvalID,
		BoundScope:             boundScope,
		UnapprovedDigest:       approvalRecord.SourceDigest,
		Approver:               "human",
		RepoPath:               "repo/file.txt",
		Commit:                 "abc123",
		ExtractorToolVersion:   "tool-v1",
		FullContentVisible:     true,
		SignedApprovalRequest:  requestEnv,
		SignedApprovalDecision: decisionEnv,
	}, RequestContext{})
	if resolveErr != nil {
		t.Fatalf("HandleApprovalResolve returned error: %+v", resolveErr)
	}
	approvalAfterResolve, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing after resolve", approvalID)
	}
	return approvalAfterResolve
}

func mustPreparedMutationIDForRun(t *testing.T, s *Service, runID string) string {
	t.Helper()
	if runID != "" {
		refs := s.GitRemotePreparedRefsForRun(runID)
		if len(refs) == 0 {
			t.Fatalf("GitRemotePreparedRefsForRun(%q) empty", runID)
		}
		return refs[0]
	}
	for _, candidateRun := range []string{"run-git-prepare-execute", "run-git-execute-mismatch", "run-git-summary-derived"} {
		refs := s.GitRemotePreparedRefsForRun(candidateRun)
		if len(refs) > 0 {
			return refs[0]
		}
	}
	t.Fatal("no prepared mutation refs found")
	return ""
}

func mustPreparedApprovalID(t *testing.T, s *Service, preparedID string) string {
	t.Helper()
	rec, ok := s.GitRemotePreparedGet(preparedID)
	if !ok {
		t.Fatalf("GitRemotePreparedGet(%q) missing", preparedID)
	}
	if strings.TrimSpace(rec.RequiredApprovalID) == "" {
		t.Fatalf("prepared mutation %q missing required_approval_id", preparedID)
	}
	return rec.RequiredApprovalID
}

func mustPrepareGitRemoteMutation(t *testing.T, s *Service, runID string) GitRemoteMutationPrepareResponse {
	t.Helper()
	prepareResp, errResp := s.HandleGitRemoteMutationPrepare(context.Background(), GitRemoteMutationPrepareRequest{
		SchemaID:       "runecode.protocol.v0.GitRemoteMutationPrepareRequest",
		SchemaVersion:  "0.1.0",
		RequestID:      "req-git-prepare",
		RunID:          runID,
		Provider:       "github",
		DestinationRef: "",
		TypedRequest:   trustedGitRequestPayload(t, "git_ref_update"),
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleGitRemoteMutationPrepare error: %+v", errResp)
	}
	return prepareResp
}

func assertPreparedGitRemoteMutation(t *testing.T, prepareResp GitRemoteMutationPrepareResponse) {
	t.Helper()
	if prepareResp.PreparedMutationID == "" {
		t.Fatal("prepared_mutation_id empty")
	}
	if prepareResp.Prepared.RequestKind != "git_ref_update" {
		t.Fatalf("request_kind=%q, want git_ref_update", prepareResp.Prepared.RequestKind)
	}
	typedHashIdentity, _ := prepareResp.TypedRequestHash.Identity()
	if typedHashIdentity == "" {
		t.Fatal("typed_request_hash identity empty")
	}
	if prepareResp.Prepared.TypedRequestSchemaID != "runecode.protocol.v0.GitRefUpdateRequest" {
		t.Fatalf("typed_request_schema_id=%q, want runecode.protocol.v0.GitRefUpdateRequest", prepareResp.Prepared.TypedRequestSchemaID)
	}
	if prepareResp.Prepared.TypedRequestSchemaVersion != "0.1.0" {
		t.Fatalf("typed_request_schema_version=%q, want 0.1.0", prepareResp.Prepared.TypedRequestSchemaVersion)
	}
	if prepareResp.Prepared.DerivedSummary.CommitSubject == "" {
		t.Fatal("derived_summary.commit_subject empty")
	}
}

func assertFetchedPreparedGitRemoteMutation(t *testing.T, s *Service, preparedMutationID string) {
	t.Helper()
	getResp, getErr := s.HandleGitRemoteMutationGet(context.Background(), GitRemoteMutationGetRequest{
		SchemaID:           "runecode.protocol.v0.GitRemoteMutationGetRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-git-get",
		PreparedMutationID: preparedMutationID,
	}, RequestContext{})
	if getErr != nil {
		t.Fatalf("HandleGitRemoteMutationGet error: %+v", getErr)
	}
	if getResp.Prepared.PreparedMutationID != preparedMutationID {
		t.Fatalf("prepared_mutation_id=%q, want %q", getResp.Prepared.PreparedMutationID, preparedMutationID)
	}
	if getResp.Prepared.LastGetRequestID != "req-git-get" {
		t.Fatalf("last_get_request_id=%q, want req-git-get", getResp.Prepared.LastGetRequestID)
	}
}

func resolveApprovalForPreparedMutation(t *testing.T, s *Service, runID string, prepareResp GitRemoteMutationPrepareResponse) (string, trustpolicy.Digest, trustpolicy.Digest) {
	t.Helper()
	approvalID := prepareResp.Prepared.RequiredApprovalID
	approvalAfterResolve := resolveApprovalForPreparedID(t, s, approvalID, ApprovalBoundScope{
		SchemaID:           "runecode.protocol.v0.ApprovalBoundScope",
		SchemaVersion:      "0.1.0",
		WorkspaceID:        mustApprovalWorkspaceID(t, s, approvalID),
		RunID:              runID,
		ActionKind:         "gateway_egress",
		PolicyDecisionHash: prepareResp.Prepared.PolicyDecisionHash.HashAlg + ":" + prepareResp.Prepared.PolicyDecisionHash.Hash,
	})
	return approvalID, mustDigestFromIdentity(t, approvalAfterResolve.RequestDigest, "approval request"), mustDigestFromIdentity(t, approvalAfterResolve.DecisionDigest, "approval decision")
}

func approvalBoundScopeFromApprovalRecord(t *testing.T, s *Service, approvalID string) ApprovalBoundScope {
	t.Helper()
	approvalRecord, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	return ApprovalBoundScope{
		SchemaID:           "runecode.protocol.v0.ApprovalBoundScope",
		SchemaVersion:      "0.1.0",
		WorkspaceID:        approvalRecord.WorkspaceID,
		RunID:              approvalRecord.RunID,
		ActionKind:         approvalRecord.ActionKind,
		PolicyDecisionHash: approvalRecord.PolicyDecisionHash,
	}
}

func mustApprovalWorkspaceID(t *testing.T, s *Service, approvalID string) string {
	t.Helper()
	approvalRecord, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	return approvalRecord.WorkspaceID
}

func mustDigestFromIdentity(t *testing.T, identity, label string) trustpolicy.Digest {
	t.Helper()
	digest, err := digestFromIdentity(identity)
	if err != nil {
		t.Fatalf("digestFromIdentity(%s) returned error: %v", label, err)
	}
	return digest
}

func mustExecutePreparedGitRemoteMutation(t *testing.T, s *Service, preparedMutationID, approvalID string, approvalRequestDigest, approvalDecisionDigest trustpolicy.Digest, requestID string) GitRemoteMutationExecuteResponse {
	t.Helper()
	execResp, execErr := s.HandleGitRemoteMutationExecute(context.Background(), GitRemoteMutationExecuteRequest{
		SchemaID:             "runecode.protocol.v0.GitRemoteMutationExecuteRequest",
		SchemaVersion:        "0.1.0",
		RequestID:            requestID,
		PreparedMutationID:   preparedMutationID,
		ApprovalID:           approvalID,
		ApprovalRequestHash:  approvalRequestDigest,
		ApprovalDecisionHash: approvalDecisionDigest,
		ProviderAuthLeaseID:  "lease-git-provider",
	}, RequestContext{})
	if execErr != nil {
		t.Fatalf("HandleGitRemoteMutationExecute error: %+v", execErr)
	}
	return execResp
}

func assertExecutedGitRemoteMutation(t *testing.T, prepareResp GitRemoteMutationPrepareResponse, execResp GitRemoteMutationExecuteResponse) {
	t.Helper()
	if execResp.ExecutionState != gitRemoteMutationExecutionCompleted {
		t.Fatalf("execution_state=%q, want %q", execResp.ExecutionState, gitRemoteMutationExecutionCompleted)
	}
	if execResp.Prepared.ExecutionReasonCode != "" {
		t.Fatalf("execution_reason_code=%q, want empty", execResp.Prepared.ExecutionReasonCode)
	}
	if execResp.Prepared.LastExecuteRequestID != "req-git-execute" {
		t.Fatalf("last_execute_request_id=%q, want req-git-execute", execResp.Prepared.LastExecuteRequestID)
	}
	if execResp.Prepared.TypedRequestHash.Hash != prepareResp.Prepared.TypedRequestHash.Hash {
		t.Fatalf("typed_request_hash changed between prepare and execute")
	}
}
