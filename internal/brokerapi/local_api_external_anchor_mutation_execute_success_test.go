package brokerapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestExternalAnchorMutationExecutePersistsDurableStateOnDeferredCompletion(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareExternalAnchorExecuteFixture(t, "run-anchor-execute-deferred", "sha256:"+strings.Repeat("5", 64))
	assertExternalAnchorPreparedGetState(t, s, preparedID, "req-anchor-get-before-execute-deferred", "external_configured_not_run")
	resp := mustExecuteExternalAnchorMutation(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-deferred", false)
	assertExternalAnchorDeferredExecuteResponse(t, resp, leaseID)
	rec, ok := s.ExternalAnchorPreparedGet(preparedID)
	if !ok {
		t.Fatalf("ExternalAnchorPreparedGet(%q) missing", preparedID)
	}
	assertExternalAnchorDeferredPreparedRecord(t, rec, leaseID, "req-anchor-execute-deferred")
}

func TestExternalAnchorMutationExecuteCanCompleteInline(t *testing.T) {
	s, preparedID, _, _, _, _ := prepareInlineExternalAnchorExecuteFixture(t, "run-anchor-execute-inline", "req-anchor-prepare-inline", "sha256:"+strings.Repeat("4", 64))
	resp := mustExecutePreparedExternalAnchor(t, s, preparedID, "req-anchor-execute-inline", false)
	assertExternalAnchorInlineCompletionResponse(t, resp)
	rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	assertExternalAnchorInlineCompletionRecord(t, rec)
	assertExternalAnchorEvidenceSidecarsPresent(t, s, rec)
}

func TestExternalAnchorMutationExecuteCanExportReceiptCopyBestEffort(t *testing.T) {
	s, preparedID, _, _, _, _ := prepareInlineExternalAnchorExecuteFixture(t, "run-anchor-execute-inline-export", "req-anchor-prepare-inline-export", "sha256:"+strings.Repeat("4", 64))
	mustExecutePreparedExternalAnchor(t, s, preparedID, "req-anchor-execute-inline-export", true)
	rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	if strings.TrimSpace(rec.LastAnchorReceiptDigest) == "" {
		t.Fatal("last_anchor_receipt_digest empty")
	}
	receiptDigest, err := digestFromIdentity(rec.LastAnchorReceiptDigest)
	if err != nil {
		t.Fatalf("digestFromIdentity(last_anchor_receipt_digest) returned error: %v", err)
	}
	assertExternalAnchorReceiptCopyExported(t, s, rec.LastAnchorReceiptDigest)
	if _, err := s.auditLedger.ReceiptEnvelopeByDigest(receiptDigest); err != nil {
		t.Fatalf("ReceiptEnvelopeByDigest returned error for authoritative sidecar receipt: %v", err)
	}
}

func TestExternalAnchorMutationExecuteRecordsAttestationAndProjectContextReferencesInEvidence(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareExternalAnchorExecuteFixture(t, "run-anchor-evidence-attestation-context", "sha256:"+strings.Repeat("6", 64))
	rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	setExternalAnchorInlineCompletionRuntime(s)
	putExternalAnchorAttestationEvidence(t, s, rec.RunID)
	s.projectSubstrate.Snapshot.ProjectContextIdentityDigest = "sha256:" + strings.Repeat("9", 64)
	mustExecuteExternalAnchorMutation(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-evidence-attestation-context", false)
	payload := readExternalAnchorEvidencePayload(t, s, mustGetExternalAnchorPreparedRecord(t, s, preparedID))
	assertExternalAnchorEvidenceContextRefs(t, payload)
}

func TestExternalAnchorMutationExecuteDeferredPollCompletesInBackground(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareDeferredPollExternalAnchorExecuteFixture(t)
	resp := mustExecuteExternalAnchorMutation(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-deferred-poll", false)
	if resp.ExecutionState != gitRemoteMutationExecutionDeferred {
		t.Fatalf("execution_state=%q, want deferred", resp.ExecutionState)
	}
	final := waitForExternalAnchorCompletion(t, s, preparedID)
	if final.LifecycleState != gitRemoteMutationLifecycleExecuted {
		t.Fatalf("final lifecycle_state=%q, want executed", final.LifecycleState)
	}
	if final.LastExecuteDeferredPolls != 0 {
		t.Fatalf("final deferred polls remaining=%d, want 0", final.LastExecuteDeferredPolls)
	}
}

func TestExternalAnchorMutationExecuteDeferredPollResumesAfterServiceRestart(t *testing.T) {
	root := t.TempDir()
	storeRoot := filepath.Join(root, "store")
	ledgerRoot := filepath.Join(root, "ledger")
	cfg := APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t), ExternalAnchor: ExternalAnchorConfig{MaxParallelExecutions: 1}}
	s, err := NewServiceWithConfig(storeRoot, ledgerRoot, cfg)
	if err != nil {
		t.Fatalf("NewServiceWithConfig(initial) returned error: %v", err)
	}
	runID := "run-anchor-execute-deferred-restart"
	targetDigest := "sha256:" + strings.Repeat("7", 64)
	putTrustedExternalAnchorGatewayContextForRun(t, s, runID, targetDigest)
	prepareResp := mustPrepareExternalAnchorMutation(t, s, runID, "req-anchor-prepare-deferred-restart", targetDigest, 2)
	preparedID := strings.TrimSpace(prepareResp.PreparedMutationID)
	approvalID := strings.TrimSpace(prepareResp.Prepared.RequiredApprovalID)
	requestDigest, decisionDigest := approveExternalAnchorForExecuteTests(t, s, preparedID, approvalID)
	leaseID := mustIssueExternalAnchorGatewayLease(t, s, preparedID)
	// Simulate a shutdown before background deferred workers can drain.
	s.externalAnchorQueue.close()
	resp := mustExecuteExternalAnchorMutation(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-deferred-restart", false)
	if resp.ExecutionState != gitRemoteMutationExecutionDeferred {
		t.Fatalf("execution_state=%q, want deferred", resp.ExecutionState)
	}
	restarted, err := NewServiceWithConfig(storeRoot, ledgerRoot, cfg)
	if err != nil {
		t.Fatalf("NewServiceWithConfig(restart) returned error: %v", err)
	}
	final := waitForExternalAnchorCompletion(t, restarted, preparedID)
	if final.LifecycleState != gitRemoteMutationLifecycleExecuted {
		t.Fatalf("final lifecycle_state=%q, want executed", final.LifecycleState)
	}
	if final.ExecutionState != gitRemoteMutationExecutionCompleted {
		t.Fatalf("final execution_state=%q, want completed", final.ExecutionState)
	}
}

func TestExternalAnchorMutationExecuteIdempotentReplayReturnsStoredOutcome(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareExternalAnchorExecuteFixture(t, "run-anchor-execute-replay", "sha256:"+strings.Repeat("2", 64))
	setExternalAnchorInlineCompletionRuntime(s)
	first := mustExecuteExternalAnchorMutation(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-replay-1", false)
	second := mustExecuteExternalAnchorMutation(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-replay-2", false)
	if first.ExecutionState != gitRemoteMutationExecutionCompleted || second.ExecutionState != gitRemoteMutationExecutionCompleted {
		t.Fatalf("execution states first=%q second=%q, want both completed", first.ExecutionState, second.ExecutionState)
	}
	if strings.TrimSpace(first.Prepared.LastExecuteAttemptID) == "" || first.Prepared.LastExecuteAttemptID != second.Prepared.LastExecuteAttemptID {
		t.Fatalf("attempt ids first=%q second=%q, want same non-empty", first.Prepared.LastExecuteAttemptID, second.Prepared.LastExecuteAttemptID)
	}
}

func TestExternalAnchorMutationExecuteSnapshotsSealOutsideLedgerLock(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareInlineExternalAnchorExecuteFixture(t, "run-anchor-execute-lock-boundary", "req-anchor-prepare-lock-boundary", "sha256:"+strings.Repeat("1", 64))
	started, continueCh, resultCh := startBlockingExternalAnchorExecution(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-lock-boundary")
	waitForExternalAnchorRuntimeEntry(t, started)
	rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	if rec.LifecycleState != gitRemoteMutationLifecycleExecuting {
		t.Fatalf("lifecycle_state during runtime=%q, want executing", rec.LifecycleState)
	}
	continueCh <- struct{}{}
	if out := <-resultCh; out.err != nil {
		t.Fatalf("HandleExternalAnchorMutationExecute returned error: %+v", out.err)
	}
}

func TestExternalAnchorMutationExecuteSnapshotAllowsNoSealedSegmentForInlineMode(t *testing.T) {
	s, preparedID, approvalID, requestDigest, decisionDigest, leaseID := prepareNoSealedSegmentExternalAnchorFixture(t)
	resp := mustExecuteExternalAnchorMutation(t, s, preparedID, approvalID, requestDigest, decisionDigest, leaseID, "req-anchor-execute-no-seal", false)
	if resp.ExecutionState != gitRemoteMutationExecutionCompleted {
		t.Fatalf("execution_state=%q, want completed", resp.ExecutionState)
	}
	if got := strings.TrimSpace(resp.Prepared.LastExecuteSnapshotSegmentID); got != "" {
		t.Fatalf("last_execute_snapshot_segment_id=%q, want empty for no sealed segment", got)
	}
}

func assertExternalAnchorInlineCompletionResponse(t *testing.T, resp ExternalAnchorMutationExecuteResponse) {
	t.Helper()
	if resp.ExecutionState != gitRemoteMutationExecutionCompleted {
		t.Fatalf("execution_state=%q, want completed", resp.ExecutionState)
	}
	if resp.Prepared.AnchorPosture != "external_execute_completed" {
		t.Fatalf("prepared.anchor_posture=%q, want external_execute_completed", resp.Prepared.AnchorPosture)
	}
}

func assertExternalAnchorInlineCompletionRecord(t *testing.T, rec artifacts.ExternalAnchorPreparedMutationRecord) {
	t.Helper()
	if rec.LifecycleState != gitRemoteMutationLifecycleExecuted {
		t.Fatalf("lifecycle_state=%q, want executed", rec.LifecycleState)
	}
	if rec.ExecutionState != gitRemoteMutationExecutionCompleted {
		t.Fatalf("execution_state=%q, want completed", rec.ExecutionState)
	}
	for _, field := range []struct {
		value string
		name  string
	}{{rec.LastAnchorEvidenceDigest, "last_anchor_evidence_digest"}, {rec.LastAnchorProofDigest, "last_anchor_proof_digest"}, {rec.LastAnchorProviderReceipt, "last_anchor_provider_receipt_digest"}, {rec.LastAnchorTranscriptDigest, "last_anchor_transcript_digest"}, {rec.LastAnchorReceiptDigest, "last_anchor_receipt_digest"}} {
		if strings.TrimSpace(field.value) == "" {
			t.Fatalf("%s empty", field.name)
		}
	}
}

func assertExternalAnchorReceiptCopyExported(t *testing.T, s *Service, receiptDigest string) {
	t.Helper()
	for _, item := range s.List() {
		if item.Reference.DataClass != artifacts.DataClassAuditReceiptExportCopy {
			continue
		}
		if strings.TrimSpace(item.Reference.ProvenanceReceiptHash) == strings.TrimSpace(receiptDigest) {
			return
		}
	}
	t.Fatalf("expected exported receipt copy with provenance_receipt_hash=%q", receiptDigest)
}

func readExternalAnchorEvidencePayload(t *testing.T, s *Service, rec artifacts.ExternalAnchorPreparedMutationRecord) trustpolicy.ExternalAnchorEvidencePayload {
	t.Helper()
	if strings.TrimSpace(rec.LastAnchorEvidenceDigest) == "" {
		t.Fatal("last_anchor_evidence_digest empty")
	}
	evidencePath := filepath.Join(s.auditRoot, "sidecar", "external-anchor-evidence", strings.TrimPrefix(rec.LastAnchorEvidenceDigest, "sha256:")+".json")
	b, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("ReadFile(evidence sidecar) returned error: %v", err)
	}
	payload := trustpolicy.ExternalAnchorEvidencePayload{}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("decode external anchor evidence payload returned error: %v", err)
	}
	if err := trustpolicy.ValidateExternalAnchorEvidencePayload(payload); err != nil {
		t.Fatalf("ValidateExternalAnchorEvidencePayload returned error: %v", err)
	}
	return payload
}

func assertExternalAnchorEvidenceSidecarsPresent(t *testing.T, s *Service, rec artifacts.ExternalAnchorPreparedMutationRecord) {
	t.Helper()
	for _, digestID := range []string{rec.LastAnchorEvidenceDigest, rec.LastAnchorProofDigest, rec.LastAnchorProviderReceipt, rec.LastAnchorTranscriptDigest} {
		if strings.TrimSpace(digestID) == "" {
			t.Fatalf("required sidecar digest missing in prepared record: %#v", rec)
		}
		path := filepath.Join(s.auditRoot, "sidecar", "external-anchor-evidence", strings.TrimPrefix(digestID, "sha256:")+".json")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected sidecar at %q: %v", path, err)
		}
	}
	if strings.TrimSpace(rec.LastAnchorReceiptDigest) == "" {
		return
	}
	path := filepath.Join(s.auditRoot, "sidecar", "receipts", strings.TrimPrefix(rec.LastAnchorReceiptDigest, "sha256:")+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected receipt sidecar at %q: %v", path, err)
	}
}

func assertExternalAnchorEvidenceContextRefs(t *testing.T, payload trustpolicy.ExternalAnchorEvidencePayload) {
	t.Helper()
	var hasAttestationRef, hasProjectContextRef bool
	for i := range payload.SidecarRefs {
		ref := payload.SidecarRefs[i]
		if ref.EvidenceKind == trustpolicy.ExternalAnchorSidecarKindAttestationRef {
			hasAttestationRef = true
		}
		if ref.EvidenceKind == trustpolicy.ExternalAnchorSidecarKindProjectContextRef {
			hasProjectContextRef = true
			if got, _ := ref.Digest.Identity(); got == "" {
				t.Fatal("project_context_ref digest identity empty")
			}
		}
	}
	if !hasAttestationRef {
		t.Fatal("attestation_ref missing from external anchor evidence sidecar_refs")
	}
	if !hasProjectContextRef {
		t.Fatal("project_context_ref missing from external anchor evidence sidecar_refs")
	}
}

func prepareDeferredPollExternalAnchorExecuteFixture(t *testing.T) (*Service, string, string, trustpolicy.Digest, trustpolicy.Digest, string) {
	t.Helper()
	runID := "run-anchor-execute-deferred-poll"
	targetDigest := "sha256:" + strings.Repeat("3", 64)
	s := newBrokerAPIServiceForTests(t, APIConfig{ExternalAnchor: ExternalAnchorConfig{MaxParallelExecutions: 1}})
	s.SetNowFuncForTests(func() time.Time { return time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC) })
	putTrustedExternalAnchorGatewayContextForRun(t, s, runID, targetDigest)
	prepareResp := mustPrepareExternalAnchorMutation(t, s, runID, "req-anchor-prepare-deferred-poll", targetDigest, 2)
	preparedID := strings.TrimSpace(prepareResp.PreparedMutationID)
	approvalID := strings.TrimSpace(prepareResp.Prepared.RequiredApprovalID)
	requestDigest, decisionDigest := approveExternalAnchorForExecuteTests(t, s, preparedID, approvalID)
	leaseID := mustIssueExternalAnchorGatewayLease(t, s, preparedID)
	return s, preparedID, approvalID, requestDigest, decisionDigest, leaseID
}

func waitForExternalAnchorCompletion(t *testing.T, s *Service, preparedID string) artifacts.ExternalAnchorPreparedMutationRecord {
	t.Helper()
	for i := 0; i < 120; i++ {
		rec := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
		if rec.ExecutionState == gitRemoteMutationExecutionCompleted {
			return rec
		}
		time.Sleep(5 * time.Millisecond)
	}
	final := mustGetExternalAnchorPreparedRecord(t, s, preparedID)
	t.Fatalf("final execution_state=%q, want completed", final.ExecutionState)
	return artifacts.ExternalAnchorPreparedMutationRecord{}
}

func prepareNoSealedSegmentExternalAnchorFixture(t *testing.T) (*Service, string, string, trustpolicy.Digest, trustpolicy.Digest, string) {
	t.Helper()
	runID := "run-anchor-execute-no-seal"
	targetDigest := "sha256:" + strings.Repeat("0", 64)
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedExternalAnchorGatewayContextForRun(t, s, runID, targetDigest)
	prepareResp := mustPrepareExternalAnchorMutation(t, s, runID, "req-anchor-prepare-no-seal", targetDigest, 0)
	setExternalAnchorInlineCompletionRuntime(s)
	preparedID := strings.TrimSpace(prepareResp.PreparedMutationID)
	tamperExternalAnchorPreparedRecord(t, s, preparedID, func(rec *artifacts.ExternalAnchorPreparedMutationRecord) {
		rec.TypedRequest["seal_digest"] = digestObject("sha256:" + strings.Repeat("f", 64))
		rec.TypedRequestHash = mustCanonicalExternalAnchorTypedRequestHash(t, rec.TypedRequest)
	})
	approvalID := strings.TrimSpace(prepareResp.Prepared.RequiredApprovalID)
	requestDigest, decisionDigest := approveExternalAnchorForExecuteTests(t, s, preparedID, approvalID)
	leaseID := mustIssueExternalAnchorGatewayLease(t, s, preparedID)
	return s, preparedID, approvalID, requestDigest, decisionDigest, leaseID
}
