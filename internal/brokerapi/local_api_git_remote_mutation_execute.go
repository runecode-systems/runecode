package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) resolveGitRemoteExecuteRequest(req GitRemoteMutationExecuteRequest, requestID string) (artifacts.GitRemotePreparedMutationRecord, string, string, string, *ErrorResponse) {
	preparedMutationID := strings.TrimSpace(req.PreparedMutationID)
	if preparedMutationID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "prepared_mutation_id is required")
		return artifacts.GitRemotePreparedMutationRecord{}, "", "", "", &errOut
	}
	record, ok := s.GitRemotePreparedGet(preparedMutationID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_prepared_mutation", "storage", false, fmt.Sprintf("prepared mutation %q not found", preparedMutationID))
		return artifacts.GitRemotePreparedMutationRecord{}, "", "", "", &errOut
	}
	if strings.TrimSpace(record.LifecycleState) != gitRemoteMutationLifecyclePrepared {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "prepared mutation is not in executable prepared state")
		return artifacts.GitRemotePreparedMutationRecord{}, "", "", "", &errOut
	}
	approvalID := strings.TrimSpace(req.ApprovalID)
	if approvalID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "approval_id is required")
		return artifacts.GitRemotePreparedMutationRecord{}, "", "", "", &errOut
	}
	approvalRequestHashIdentity, approvalDecisionHashIdentity, errResp := s.resolveApprovalHashBindings(requestID, req)
	if errResp != nil {
		return artifacts.GitRemotePreparedMutationRecord{}, "", "", "", errResp
	}
	_, errResp = s.verifyGitRemoteExecuteApprovalBindings(requestID, record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity)
	if errResp != nil {
		return artifacts.GitRemotePreparedMutationRecord{}, "", "", "", errResp
	}
	return record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, nil
}

func (s *Service) resolveApprovalHashBindings(requestID string, req GitRemoteMutationExecuteRequest) (string, string, *ErrorResponse) {
	approvalRequestHashIdentity, err := req.ApprovalRequestHash.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "approval_request_hash is invalid")
		return "", "", &errOut
	}
	approvalDecisionHashIdentity, err := req.ApprovalDecisionHash.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "approval_decision_hash is invalid")
		return "", "", &errOut
	}
	return approvalRequestHashIdentity, approvalDecisionHashIdentity, nil
}

func (s *Service) verifyGitRemoteExecuteApprovalBindings(requestID string, record artifacts.GitRemotePreparedMutationRecord, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity string) (artifacts.ApprovalRecord, *ErrorResponse) {
	if errResp := s.verifyPreparedApprovalReference(requestID, record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity); errResp != nil {
		return artifacts.ApprovalRecord{}, errResp
	}
	approval, ok := s.ApprovalGet(approvalID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_approval", "storage", false, fmt.Sprintf("approval %q not found", approvalID))
		return artifacts.ApprovalRecord{}, &errOut
	}
	if errResp := s.verifyApprovalRecordBindings(requestID, record, approval, approvalRequestHashIdentity, approvalDecisionHashIdentity); errResp != nil {
		return artifacts.ApprovalRecord{}, errResp
	}
	if errResp := s.verifyStoredTypedRequestHash(requestID, record); errResp != nil {
		return artifacts.ApprovalRecord{}, errResp
	}
	return approval, nil
}

func (s *Service) verifyPreparedApprovalReference(requestID string, record artifacts.GitRemotePreparedMutationRecord, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity string) *ErrorResponse {
	if record.RequiredApprovalID != "" && strings.TrimSpace(record.RequiredApprovalID) != approvalID {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_id does not match prepared mutation required approval")
		return &errOut
	}
	if record.RequiredApprovalReqHash != "" && strings.TrimSpace(record.RequiredApprovalReqHash) != approvalRequestHashIdentity {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_request_hash does not match prepared mutation required approval request hash")
		return &errOut
	}
	if record.RequiredApprovalDecHash != "" && strings.TrimSpace(record.RequiredApprovalDecHash) != approvalDecisionHashIdentity {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_decision_hash does not match prepared mutation required approval decision hash")
		return &errOut
	}
	return nil
}

func (s *Service) verifyApprovalRecordBindings(requestID string, record artifacts.GitRemotePreparedMutationRecord, approval artifacts.ApprovalRecord, approvalRequestHashIdentity, approvalDecisionHashIdentity string) *ErrorResponse {
	if approval.Status != "approved" && approval.Status != "consumed" {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q is not approved", approval.ApprovalID))
		return &errOut
	}
	if strings.TrimSpace(approval.ActionRequestHash) == "" || strings.TrimSpace(approval.ActionRequestHash) != strings.TrimSpace(record.ActionRequestHash) {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval action_request_hash does not match prepared mutation binding")
		return &errOut
	}
	if strings.TrimSpace(approval.PolicyDecisionHash) == "" || strings.TrimSpace(approval.PolicyDecisionHash) != strings.TrimSpace(record.PolicyDecisionHash) {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval policy_decision_hash does not match prepared mutation binding")
		return &errOut
	}
	if strings.TrimSpace(approval.RequestDigest) == "" || strings.TrimSpace(approval.RequestDigest) != approvalRequestHashIdentity {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_request_hash does not match stored approval request digest")
		return &errOut
	}
	if strings.TrimSpace(approval.DecisionDigest) == "" || strings.TrimSpace(approval.DecisionDigest) != approvalDecisionHashIdentity {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_decision_hash does not match stored approval decision digest")
		return &errOut
	}
	return nil
}

func (s *Service) verifyStoredTypedRequestHash(requestID string, record artifacts.GitRemotePreparedMutationRecord) *ErrorResponse {
	storedTypedRequestHash, err := canonicalGitTypedRequestHash(record.TypedRequest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("stored typed request hash invalid: %v", err))
		return &errOut
	}
	if strings.TrimSpace(record.TypedRequestHash) != storedTypedRequestHash {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request hash no longer matches authoritative typed request")
		return &errOut
	}
	return nil
}

func gitPreparedMutationID(runID, provider, destinationRef, typedRequestHash, actionRequestHash, policyDecisionHash string) (string, error) {
	payload := map[string]any{
		"run_id":               strings.TrimSpace(runID),
		"provider":             strings.TrimSpace(provider),
		"destination_ref":      strings.TrimSpace(destinationRef),
		"typed_request_hash":   strings.TrimSpace(typedRequestHash),
		"action_request_hash":  strings.TrimSpace(actionRequestHash),
		"policy_decision_hash": strings.TrimSpace(policyDecisionHash),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return policyengine.CanonicalHashBytes(b)
}

func gitPreparedStateFromRecord(record artifacts.GitRemotePreparedMutationRecord) (GitRemoteMutationPreparedState, error) {
	typedRequestHash, actionRequestHash, policyDecisionHash, approvalReqHash, approvalDecHash, derivedSummary, err := decodeGitPreparedStateRecord(record)
	if err != nil {
		return GitRemoteMutationPreparedState{}, err
	}
	return GitRemoteMutationPreparedState{
		SchemaID:                     "runecode.protocol.v0.GitRemoteMutationPreparedState",
		SchemaVersion:                "0.1.0",
		PreparedMutationID:           record.PreparedMutationID,
		RunID:                        record.RunID,
		Provider:                     record.Provider,
		DestinationRef:               record.DestinationRef,
		RequestKind:                  record.RequestKind,
		TypedRequestSchemaID:         record.TypedRequestSchemaID,
		TypedRequestSchemaVersion:    record.TypedRequestSchemaVer,
		TypedRequest:                 cloneStringAnyMap(record.TypedRequest),
		TypedRequestHash:             typedRequestHash,
		ActionRequestHash:            actionRequestHash,
		PolicyDecisionHash:           policyDecisionHash,
		RequiredApprovalID:           record.RequiredApprovalID,
		RequiredApprovalRequestHash:  approvalReqHash,
		RequiredApprovalDecisionHash: approvalDecHash,
		LifecycleState:               record.LifecycleState,
		LifecycleReasonCode:          record.LifecycleReasonCode,
		ExecutionState:               record.ExecutionState,
		ExecutionReasonCode:          record.ExecutionReasonCode,
		CreatedAt:                    record.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                    record.UpdatedAt.UTC().Format(time.RFC3339),
		DerivedSummary:               derivedSummary,
		LastPrepareRequestID:         record.LastPrepareRequestID,
		LastGetRequestID:             record.LastGetRequestID,
		LastExecuteRequestID:         record.LastExecuteRequestID,
	}, nil
}

func decodeGitPreparedStateRecord(record artifacts.GitRemotePreparedMutationRecord) (trustpolicy.Digest, trustpolicy.Digest, trustpolicy.Digest, *trustpolicy.Digest, *trustpolicy.Digest, GitRemoteMutationDerivedSummary, error) {
	typedRequestHash, err := digestFromIdentity(record.TypedRequestHash)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, nil, GitRemoteMutationDerivedSummary{}, fmt.Errorf("typed_request_hash invalid: %w", err)
	}
	actionRequestHash, err := digestFromIdentity(record.ActionRequestHash)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, nil, GitRemoteMutationDerivedSummary{}, fmt.Errorf("action_request_hash invalid: %w", err)
	}
	policyDecisionHash, err := digestFromIdentity(record.PolicyDecisionHash)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, nil, GitRemoteMutationDerivedSummary{}, fmt.Errorf("policy_decision_hash invalid: %w", err)
	}
	approvalReqHash, err := optionalDigestFromIdentity(record.RequiredApprovalReqHash, "required_approval_request_hash")
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, nil, GitRemoteMutationDerivedSummary{}, err
	}
	approvalDecHash, err := optionalDigestFromIdentity(record.RequiredApprovalDecHash, "required_approval_decision_hash")
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, nil, GitRemoteMutationDerivedSummary{}, err
	}
	derivedSummary := GitRemoteMutationDerivedSummary{}
	if err := remarshalValue(record.DerivedSummary, &derivedSummary); err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, nil, nil, GitRemoteMutationDerivedSummary{}, fmt.Errorf("derived_summary invalid: %w", err)
	}
	if derivedSummary.SchemaID == "" {
		derivedSummary.SchemaID = "runecode.protocol.v0.GitRemoteMutationDerivedSummary"
	}
	if derivedSummary.SchemaVersion == "" {
		derivedSummary.SchemaVersion = "0.1.0"
	}
	return typedRequestHash, actionRequestHash, policyDecisionHash, approvalReqHash, approvalDecHash, derivedSummary, nil
}

func optionalDigestFromIdentity(identity, field string) (*trustpolicy.Digest, error) {
	if strings.TrimSpace(identity) == "" {
		return nil, nil
	}
	d, err := digestFromIdentity(identity)
	if err != nil {
		return nil, fmt.Errorf("%s invalid: %w", field, err)
	}
	return &d, nil
}
