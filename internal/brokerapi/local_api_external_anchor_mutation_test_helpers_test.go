package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func externalAnchorPrepareRequest(runID, requestID, targetDigest string, deferredPollCount int) ExternalAnchorMutationPrepareRequest {
	return ExternalAnchorMutationPrepareRequest{
		SchemaID:      "runecode.protocol.v0.ExternalAnchorMutationPrepareRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		RunID:         runID,
		TypedRequest:  trustedExternalAnchorTypedRequestWithExecutionMode(targetDigest, deferredPollCount),
	}
}

func externalAnchorExecuteRequest(preparedID, approvalID string, requestDigest, decisionDigest trustpolicy.Digest, leaseID, requestID string, exportReceiptCopy bool) ExternalAnchorMutationExecuteRequest {
	return ExternalAnchorMutationExecuteRequest{
		SchemaID:             "runecode.protocol.v0.ExternalAnchorMutationExecuteRequest",
		SchemaVersion:        "0.1.0",
		RequestID:            requestID,
		PreparedMutationID:   preparedID,
		ApprovalID:           approvalID,
		ApprovalRequestHash:  requestDigest,
		ApprovalDecisionHash: decisionDigest,
		TargetAuthLeaseID:    leaseID,
		ExportReceiptCopy:    exportReceiptCopy,
	}
}

func mustPrepareExternalAnchorMutation(t *testing.T, s *Service, runID, requestID, targetDigest string, deferredPollCount int) ExternalAnchorMutationPrepareResponse {
	t.Helper()
	resp, errResp := s.HandleExternalAnchorMutationPrepare(context.Background(), externalAnchorPrepareRequest(runID, requestID, targetDigest, deferredPollCount), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleExternalAnchorMutationPrepare returned error: %+v", errResp)
	}
	return resp
}

func mustExecuteExternalAnchorMutation(t *testing.T, s *Service, preparedID, approvalID string, requestDigest, decisionDigest trustpolicy.Digest, leaseID, requestID string, exportReceiptCopy bool) ExternalAnchorMutationExecuteResponse {
	t.Helper()
	resp, errResp := s.HandleExternalAnchorMutationExecute(context.Background(), externalAnchorExecuteRequest(preparedID, approvalID, requestDigest, decisionDigest, leaseID, requestID, exportReceiptCopy), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleExternalAnchorMutationExecute returned error: %+v", errResp)
	}
	return resp
}

func executeExternalAnchorMutationError(t *testing.T, s *Service, preparedID, approvalID string, requestDigest, decisionDigest trustpolicy.Digest, leaseID, requestID string) *ErrorResponse {
	t.Helper()
	_, errResp := s.HandleExternalAnchorMutationExecute(context.Background(), externalAnchorExecuteRequest(preparedID, approvalID, requestDigest, decisionDigest, leaseID, requestID, false), RequestContext{})
	return errResp
}

func mustGetExternalAnchorPreparedRecord(t *testing.T, s *Service, preparedID string) artifacts.ExternalAnchorPreparedMutationRecord {
	t.Helper()
	rec, ok := s.ExternalAnchorPreparedGet(preparedID)
	if !ok {
		t.Fatalf("ExternalAnchorPreparedGet(%q) missing", preparedID)
	}
	return rec
}

func tamperExternalAnchorPreparedRecord(t *testing.T, s *Service, preparedID string, mutate func(*artifacts.ExternalAnchorPreparedMutationRecord)) {
	t.Helper()
	rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	mutate(&rec)
	if err := s.ExternalAnchorPreparedUpsert(rec); err != nil {
		t.Fatalf("ExternalAnchorPreparedUpsert returned error: %v", err)
	}
}

func assertExternalAnchorNotStartedState(t *testing.T, s *Service, preparedID string) {
	t.Helper()
	rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	if rec.LifecycleState != gitRemoteMutationLifecyclePrepared {
		t.Fatalf("lifecycle_state=%q, want prepared", rec.LifecycleState)
	}
	if rec.ExecutionState != gitRemoteMutationExecutionNotStarted {
		t.Fatalf("execution_state=%q, want not_started", rec.ExecutionState)
	}
}

func assertExternalAnchorPreparedGetState(t *testing.T, s *Service, preparedID, requestID, wantPosture string) {
	t.Helper()
	getResp, getErr := s.HandleExternalAnchorMutationGet(context.Background(), ExternalAnchorMutationGetRequest{SchemaID: "runecode.protocol.v0.ExternalAnchorMutationGetRequest", SchemaVersion: "0.1.0", RequestID: requestID, PreparedMutationID: preparedID}, RequestContext{})
	if getErr != nil {
		t.Fatalf("HandleExternalAnchorMutationGet(%s) error: %+v", requestID, getErr)
	}
	if getResp.Prepared.AnchorPosture != wantPosture {
		t.Fatalf("get.prepared.anchor_posture=%q, want %q", getResp.Prepared.AnchorPosture, wantPosture)
	}
	if getResp.Prepared.ExecutionPathway != "non_workspace_gateway" {
		t.Fatalf("get.prepared.execution_pathway=%q, want non_workspace_gateway", getResp.Prepared.ExecutionPathway)
	}
}

func assertExternalAnchorDeferredExecuteResponse(t *testing.T, resp ExternalAnchorMutationExecuteResponse, leaseID string) {
	t.Helper()
	if resp.Prepared.AnchorPosture != "external_execute_deferred" {
		t.Fatalf("execute response prepared.anchor_posture=%q, want external_execute_deferred", resp.Prepared.AnchorPosture)
	}
	if resp.Prepared.ExecutionPathway != "non_workspace_gateway" {
		t.Fatalf("execute response prepared.execution_pathway=%q, want non_workspace_gateway", resp.Prepared.ExecutionPathway)
	}
	if resp.ExecutionState != gitRemoteMutationExecutionDeferred {
		t.Fatalf("execution_state=%q, want %q", resp.ExecutionState, gitRemoteMutationExecutionDeferred)
	}
	if resp.Prepared.LastExecuteTargetAuthLeaseID != leaseID {
		t.Fatalf("response prepared.last_execute_target_auth_lease_id=%q, want %q", resp.Prepared.LastExecuteTargetAuthLeaseID, leaseID)
	}
	if strings.TrimSpace(resp.Prepared.LastExecuteAttemptID) == "" || resp.Prepared.LastExecuteAttemptSealDigest == nil || resp.Prepared.LastExecuteAttemptTargetID == nil || resp.Prepared.LastExecuteAttemptRequestID == nil || resp.Prepared.LastExecuteSnapshotSealID == nil {
		t.Fatal("response prepared missing deferred execute bindings")
	}
}

func assertExternalAnchorDeferredPreparedRecord(t *testing.T, rec artifacts.ExternalAnchorPreparedMutationRecord, leaseID, requestID string) {
	t.Helper()
	if rec.LifecycleState != gitRemoteMutationLifecyclePrepared {
		t.Fatalf("lifecycle_state=%q, want prepared", rec.LifecycleState)
	}
	if rec.LifecycleReasonCode != gitRemoteMutationLifecycleDeferredReason {
		t.Fatalf("lifecycle_reason_code=%q, want %q", rec.LifecycleReasonCode, gitRemoteMutationLifecycleDeferredReason)
	}
	if rec.ExecutionState != gitRemoteMutationExecutionDeferred {
		t.Fatalf("execution_state=%q, want deferred", rec.ExecutionState)
	}
	if rec.ExecutionReasonCode != "external_anchor_execution_deferred" {
		t.Fatalf("execution_reason_code=%q, want external_anchor_execution_deferred", rec.ExecutionReasonCode)
	}
	if rec.LastExecuteRequestID != requestID {
		t.Fatalf("last_execute_request_id=%q, want %q", rec.LastExecuteRequestID, requestID)
	}
	if rec.LastExecuteTargetAuthLeaseID != leaseID {
		t.Fatalf("last_execute_target_auth_lease_id=%q, want %q", rec.LastExecuteTargetAuthLeaseID, leaseID)
	}
}

func prepareInlineExternalAnchorExecuteFixture(t *testing.T, runID, prepareRequestID, targetDigest string) (*Service, string, string, trustpolicy.Digest, trustpolicy.Digest, string) {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedExternalAnchorGatewayContextForRun(t, s, runID, targetDigest)
	prepareResp := mustPrepareExternalAnchorMutation(t, s, runID, prepareRequestID, targetDigest, 0)
	setExternalAnchorInlineCompletionRuntime(s)
	preparedID := strings.TrimSpace(prepareResp.PreparedMutationID)
	approvalID := strings.TrimSpace(prepareResp.Prepared.RequiredApprovalID)
	requestDigest, decisionDigest := approveExternalAnchorForExecuteTests(t, s, preparedID, approvalID)
	leaseID := mustIssueExternalAnchorGatewayLease(t, s, preparedID)
	return s, preparedID, approvalID, requestDigest, decisionDigest, leaseID
}

func mustExecutePreparedExternalAnchor(t *testing.T, s *Service, preparedID, requestID string, exportReceiptCopy bool) ExternalAnchorMutationExecuteResponse {
	t.Helper()
	rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	approvalID := strings.TrimSpace(rec.RequiredApprovalID)
	requestDigest, decisionDigest := approveExternalAnchorForExecuteTests(t, s, preparedID, approvalID)
	leaseID := mustIssueExternalAnchorGatewayLease(t, s, preparedID)
	return mustExecuteExternalAnchorMutation(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, requestID, exportReceiptCopy)
}

func setExternalAnchorInlineCompletionRuntime(s *Service) {
	s.externalAnchorRuntime = externalAnchorExecutionRuntimeFunc(func(_ context.Context, input externalAnchorExecutionInput) externalAnchorExecutionOutcome {
		_ = input
		return externalAnchorExecutionOutcome{ExecutionState: gitRemoteMutationExecutionCompleted, LifecycleState: gitRemoteMutationLifecycleExecuted}
	})
}

func putExternalAnchorAttestationEvidence(t *testing.T, s *Service, runID string) {
	t.Helper()
	facts := launcherbackend.DefaultRuntimeFacts(runID)
	evidence := launcherbackend.RuntimeEvidenceSnapshot{Attestation: &launcherbackend.IsolateAttestationEvidence{RunID: runID, EvidenceDigest: "sha256:" + strings.Repeat("a", 64)}, AttestationVerification: &launcherbackend.IsolateAttestationVerificationRecord{AttestationEvidenceDigest: "sha256:" + strings.Repeat("a", 64), VerificationDigest: "sha256:" + strings.Repeat("b", 64), VerificationResult: launcherbackend.AttestationVerificationResultValid, ReplayVerdict: launcherbackend.AttestationReplayVerdictOriginal}}
	if err := s.store.RecordRuntimeEvidenceState(runID, facts, evidence, launcherbackend.RuntimeLifecycleState{}); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState returned error: %v", err)
	}
	_, evidenceSnapshot, _, _, ok := s.store.RuntimeEvidenceState(runID)
	if !ok || evidenceSnapshot.Attestation == nil {
		t.Fatal("runtime evidence attestation missing after RecordRuntimeFacts")
	}
}

func startBlockingExternalAnchorExecution(t *testing.T, s *Service, preparedID, approvalID string, requestDigest, decisionDigest trustpolicy.Digest, leaseID, requestID string) (chan struct{}, chan struct{}, chan struct {
	resp ExternalAnchorMutationExecuteResponse
	err  *ErrorResponse
}) {
	t.Helper()
	started := make(chan struct{}, 1)
	continueCh := make(chan struct{}, 1)
	s.externalAnchorRuntime = externalAnchorExecutionRuntimeFunc(func(_ context.Context, input externalAnchorExecutionInput) externalAnchorExecutionOutcome {
		_ = input
		started <- struct{}{}
		<-continueCh
		return externalAnchorExecutionOutcome{ExecutionState: gitRemoteMutationExecutionCompleted, LifecycleState: gitRemoteMutationLifecycleExecuted}
	})
	resultCh := make(chan struct {
		resp ExternalAnchorMutationExecuteResponse
		err  *ErrorResponse
	}, 1)
	go func() {
		resp, err := s.HandleExternalAnchorMutationExecute(context.Background(), externalAnchorExecuteRequest(preparedID, approvalID, requestDigest, decisionDigest, leaseID, requestID, false), RequestContext{})
		resultCh <- struct {
			resp ExternalAnchorMutationExecuteResponse
			err  *ErrorResponse
		}{resp: resp, err: err}
	}()
	return started, continueCh, resultCh
}

func waitForExternalAnchorRuntimeEntry(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("execute runtime was not entered")
	}
}

func mustCanonicalExternalAnchorTypedRequestHash(t *testing.T, req map[string]any) string {
	t.Helper()
	hash, err := canonicalExternalAnchorTypedRequestHash(req)
	if err != nil {
		t.Fatalf("canonicalExternalAnchorTypedRequestHash returned error: %v", err)
	}
	return hash
}

func prepareExternalAnchorExecuteFixture(t *testing.T, runID, targetDigest string) (*Service, string, string, trustpolicy.Digest, trustpolicy.Digest, string) {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedExternalAnchorGatewayContextForRun(t, s, runID, targetDigest)
	prepareResp := mustPrepareExternalAnchorMutation(t, s, runID, "req-anchor-prepare-"+runID, targetDigest, 0)
	preparedID := strings.TrimSpace(prepareResp.PreparedMutationID)
	if preparedID == "" {
		t.Fatal("prepared_mutation_id empty")
	}
	approvalID := strings.TrimSpace(prepareResp.Prepared.RequiredApprovalID)
	if approvalID == "" {
		t.Fatal("required_approval_id empty")
	}
	requestDigest, decisionDigest := approveExternalAnchorForExecuteTests(t, s, preparedID, approvalID)
	leaseID := mustIssueExternalAnchorGatewayLease(t, s, preparedID)
	return s, preparedID, approvalID, requestDigest, decisionDigest, leaseID
}

func approveExternalAnchorForExecuteTests(t *testing.T, s *Service, preparedID, approvalID string) (trustpolicy.Digest, trustpolicy.Digest) {
	t.Helper()
	prepared := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	approval, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	approval.Status = "approved"
	approval.RequestDigest = strings.TrimSpace(prepared.RequiredApprovalReqHash)
	if approval.RequestDigest == "" {
		t.Fatal("required approval request hash empty")
	}
	approval.DecisionDigest = "sha256:" + strings.Repeat("3", 64)
	if err := s.RecordApproval(approval); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return mustDigestFromIdentity(t, approval.RequestDigest, "approval request"), mustDigestFromIdentity(t, approval.DecisionDigest, "approval decision")
}

func mustIssueExternalAnchorGatewayLease(t *testing.T, s *Service, preparedID string) string {
	t.Helper()
	rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	if _, err := s.secretsSvc.ImportSecret("secrets/prod/git/provider-token", strings.NewReader("anchor-token")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	lease, err := s.secretsSvc.IssueLease(secretsd.IssueLeaseRequest{
		SecretRef:    "secrets/prod/git/provider-token",
		ConsumerID:   "principal:gateway:git:1",
		RoleKind:     "git-gateway",
		Scope:        "run:" + rec.RunID,
		DeliveryKind: "git_gateway",
		TTLSeconds:   120,
		GitBinding: &secretsd.GitLeaseBinding{
			RepositoryIdentity: rec.DestinationRef,
			AllowedOperations:  []string{"external_anchor_submit"},
			ActionRequestHash:  rec.ActionRequestHash,
			PolicyContextHash:  rec.PolicyDecisionHash,
		},
	})
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	return lease.LeaseID
}

func mustIssueExternalAnchorNonGatewayLease(t *testing.T, s *Service, runID string) string {
	t.Helper()
	if _, err := s.secretsSvc.ImportSecret("secrets/prod/git/provider-token", strings.NewReader("anchor-token")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	lease, err := s.secretsSvc.IssueLease(secretsd.IssueLeaseRequest{SecretRef: "secrets/prod/git/provider-token", ConsumerID: "principal:gateway:git:1", RoleKind: "git-gateway", Scope: "run:" + runID, TTLSeconds: 120})
	if err != nil {
		t.Fatalf("IssueLease(non-gateway) returned error: %v", err)
	}
	return lease.LeaseID
}
