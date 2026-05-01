package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

type externalAnchorAttemptBinding struct {
	AttemptID            string
	SealDigestIdentity   string
	TargetDigestIdentity string
	TypedRequestHash     string
	DeferredPolls        int
}

type externalAnchorExecutionSnapshot struct {
	SegmentID    string
	SealIdentity string
}

func (s *Service) beginExternalAnchorPreparedExecution(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, targetAuthLeaseID string, attemptBinding externalAnchorAttemptBinding, snapshot externalAnchorExecutionSnapshot) (artifacts.ExternalAnchorPreparedMutationRecord, bool, *ErrorResponse) {
	updated, err := s.ExternalAnchorPreparedTransitionLifecycle(record.PreparedMutationID, gitRemoteMutationLifecyclePrepared, func(current artifacts.ExternalAnchorPreparedMutationRecord) artifacts.ExternalAnchorPreparedMutationRecord {
		if strings.TrimSpace(current.LastExecuteAttemptID) == strings.TrimSpace(attemptBinding.AttemptID) && strings.TrimSpace(current.LastExecuteAttemptID) != "" {
			return current
		}
		return prepareExternalAnchorMutationForExecution(current, requestID, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, targetAuthLeaseID, attemptBinding, snapshot)
	})
	if err == nil {
		return updated, externalAnchorExecutionStarted(updated, attemptBinding, requestID), nil
	}
	if current, ok := s.ExternalAnchorPreparedGet(record.PreparedMutationID); ok && strings.TrimSpace(current.LifecycleState) != gitRemoteMutationLifecyclePrepared {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "prepared mutation is not in executable prepared state")
		return artifacts.ExternalAnchorPreparedMutationRecord{}, false, &errOut
	}
	errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
	return artifacts.ExternalAnchorPreparedMutationRecord{}, false, &errOut
}

func externalAnchorExecutionStarted(updated artifacts.ExternalAnchorPreparedMutationRecord, attemptBinding externalAnchorAttemptBinding, requestID string) bool {
	if strings.TrimSpace(updated.LastExecuteAttemptID) != strings.TrimSpace(attemptBinding.AttemptID) {
		return false
	}
	return strings.TrimSpace(updated.LastExecuteRequestID) == strings.TrimSpace(requestID)
}

func (s *Service) persistExternalAnchorExecutionOutcome(requestID, preparedMutationID, attemptID string, pollRemaining int, outcome externalAnchorExecutionOutcome) (artifacts.ExternalAnchorPreparedMutationRecord, *ErrorResponse) {
	updated, err := s.ExternalAnchorPreparedTransitionLifecycle(preparedMutationID, gitRemoteMutationLifecycleExecuting, func(current artifacts.ExternalAnchorPreparedMutationRecord) artifacts.ExternalAnchorPreparedMutationRecord {
		if strings.TrimSpace(current.LastExecuteAttemptID) != strings.TrimSpace(attemptID) {
			return current
		}
		setExternalAnchorExecutionOutcome(&current, outcome)
		if strings.TrimSpace(current.ExecutionState) == gitRemoteMutationExecutionDeferred {
			setExternalAnchorDeferredPollRemaining(&current, pollRemaining)
		}
		return current
	})
	if err == nil {
		return updated, nil
	}
	current, ok := s.ExternalAnchorPreparedGet(preparedMutationID)
	if !ok {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "prepared mutation unavailable after execution outcome")
		return artifacts.ExternalAnchorPreparedMutationRecord{}, &errOut
	}
	if strings.TrimSpace(current.LastExecuteAttemptID) == strings.TrimSpace(attemptID) && strings.TrimSpace(current.LifecycleState) != gitRemoteMutationLifecycleExecuting {
		return current, nil
	}
	errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
	return artifacts.ExternalAnchorPreparedMutationRecord{}, &errOut
}

func (s *Service) externalAnchorExecuteAttemptBinding(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorAttemptBinding, *ErrorResponse) {
	sealDigest, sealDigestIdentity, err := externalAnchorSealDigest(record.TypedRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request seal digest is invalid")
		return externalAnchorAttemptBinding{}, &errOut
	}
	if _, err := sealDigest.Identity(); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request seal digest is invalid")
		return externalAnchorAttemptBinding{}, &errOut
	}
	primaryTarget, err := externalAnchorResolvedPrimaryTargetFromPreparedRecord(record)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request target descriptor digest is invalid")
		return externalAnchorAttemptBinding{}, &errOut
	}
	targetDigestIdentity := primaryTarget.TargetDescriptorIdentity
	typedRequestHash, err := canonicalExternalAnchorTypedRequestHash(record.TypedRequest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("stored typed request hash invalid: %v", err))
		return externalAnchorAttemptBinding{}, &errOut
	}
	attemptID, err := externalAnchorExecuteAttemptID(record.PreparedMutationID, sealDigestIdentity, targetDigestIdentity, typedRequestHash)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("attempt id generation failed: %v", err))
		return externalAnchorAttemptBinding{}, &errOut
	}
	deferredPolls := int(intField(record.TypedRequest, "deferred_poll_count"))
	if deferredPolls < 1 {
		deferredPolls = 2
	}
	return externalAnchorAttemptBinding{
		AttemptID:            attemptID,
		SealDigestIdentity:   sealDigestIdentity,
		TargetDigestIdentity: targetDigestIdentity,
		TypedRequestHash:     typedRequestHash,
		DeferredPolls:        deferredPolls,
	}, nil
}

func (s *Service) snapshotExternalAnchorExecutionInputs(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorExecutionSnapshot, *ErrorResponse) {
	if s == nil || s.auditLedger == nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit ledger unavailable")
		return externalAnchorExecutionSnapshot{}, &errOut
	}
	wantSealDigest, wantSealIdentity, err := externalAnchorSealDigest(record.TypedRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request seal digest is invalid")
		return externalAnchorExecutionSnapshot{}, &errOut
	}
	if _, err := wantSealDigest.Identity(); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request seal digest is invalid")
		return externalAnchorExecutionSnapshot{}, &errOut
	}
	segmentID, digest, err := s.auditLedger.LatestAnchorableSeal()
	if err != nil {
		if err == auditd.ErrNoSealedSegment {
			return externalAnchorExecutionSnapshot{SegmentID: "", SealIdentity: wantSealIdentity}, nil
		}
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("latest anchorable seal unavailable: %v", err))
		return externalAnchorExecutionSnapshot{}, &errOut
	}
	identity, err := digest.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "latest anchorable seal digest is invalid")
		return externalAnchorExecutionSnapshot{}, &errOut
	}
	if got, _ := wantSealDigest.Identity(); got != identity || strings.TrimSpace(wantSealIdentity) != identity {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "latest anchorable seal no longer matches prepared typed request seal digest")
		return externalAnchorExecutionSnapshot{}, &errOut
	}
	return externalAnchorExecutionSnapshot{SegmentID: segmentID, SealIdentity: identity}, nil
}

func externalAnchorExecuteAttemptID(preparedMutationID, sealDigestIdentity, targetDigestIdentity, typedRequestHash string) (string, error) {
	payload := map[string]any{
		"prepared_mutation_id":                strings.TrimSpace(preparedMutationID),
		"seal_digest":                         strings.TrimSpace(sealDigestIdentity),
		"target_descriptor_digest":            strings.TrimSpace(targetDigestIdentity),
		"typed_request_hash":                  strings.TrimSpace(typedRequestHash),
		"attempt_identity_schema_version":     "0.1.0",
		"attempt_identity_source":             "external_anchor_execute",
		"attempt_identity_binding_invariants": []string{"seal_digest", "target_descriptor_digest", "typed_request_hash"},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return policyengine.CanonicalHashBytes(b)
}

func prepareExternalAnchorMutationForExecution(record artifacts.ExternalAnchorPreparedMutationRecord, requestID, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, targetAuthLeaseID string, attemptBinding externalAnchorAttemptBinding, snapshot externalAnchorExecutionSnapshot) artifacts.ExternalAnchorPreparedMutationRecord {
	record.LifecycleState = gitRemoteMutationLifecycleExecuting
	record.ExecutionState = gitRemoteMutationExecutionNotStarted
	record.ExecutionReasonCode = ""
	record.LifecycleReasonCode = ""
	record.RequiredApprovalReqHash = approvalRequestHashIdentity
	record.RequiredApprovalDecHash = approvalDecisionHashIdentity
	record.LastExecuteTargetAuthLeaseID = strings.TrimSpace(targetAuthLeaseID)
	record.LastExecuteApprovalID = approvalID
	record.LastExecuteApprovalReqID = approvalRequestHashIdentity
	record.LastExecuteApprovalDecID = approvalDecisionHashIdentity
	record.LastExecuteRequestID = requestID
	record.LastExecuteAttemptID = strings.TrimSpace(attemptBinding.AttemptID)
	record.LastExecuteAttemptSealDigest = strings.TrimSpace(attemptBinding.SealDigestIdentity)
	record.LastExecuteAttemptTargetID = strings.TrimSpace(attemptBinding.TargetDigestIdentity)
	record.LastExecuteAttemptRequestID = strings.TrimSpace(attemptBinding.TypedRequestHash)
	record.LastExecuteSnapshotSegmentID = strings.TrimSpace(snapshot.SegmentID)
	record.LastExecuteSnapshotSealID = strings.TrimSpace(snapshot.SealIdentity)
	record.LastExecuteDeferredPolls = attemptBinding.DeferredPolls
	record.LastExecuteDeferredClaimID = ""
	record.LastExecuteDeferredClaimedAt = nil
	record.LastAnchorReceiptDigest = ""
	record.LastAnchorEvidenceDigest = ""
	record.LastAnchorVerificationDigest = ""
	record.LastAnchorProofDigest = ""
	record.LastAnchorProviderReceipt = ""
	record.LastAnchorTranscriptDigest = ""
	return record
}
