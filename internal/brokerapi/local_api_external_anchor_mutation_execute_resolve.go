package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/secretsd"
)

func (s *Service) resolveExternalAnchorExecuteRequest(req ExternalAnchorMutationExecuteRequest, requestID string) (artifacts.ExternalAnchorPreparedMutationRecord, string, string, string, string, *ErrorResponse) {
	record, errResp := s.loadExternalAnchorPreparedForExecute(requestID, req.PreparedMutationID)
	if errResp != nil {
		return artifacts.ExternalAnchorPreparedMutationRecord{}, "", "", "", "", errResp
	}
	approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, errResp := s.resolveExternalAnchorExecuteApprovalInput(requestID, req)
	if errResp != nil {
		return artifacts.ExternalAnchorPreparedMutationRecord{}, "", "", "", "", errResp
	}
	targetAuthLeaseID, errResp := s.resolveExternalAnchorTargetAuthLease(requestID, record, req.TargetAuthLeaseID)
	if errResp != nil {
		return artifacts.ExternalAnchorPreparedMutationRecord{}, "", "", "", "", errResp
	}
	if _, errResp = s.verifyExternalAnchorExecuteApprovalBindings(requestID, record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity); errResp != nil {
		return artifacts.ExternalAnchorPreparedMutationRecord{}, "", "", "", "", errResp
	}
	if errResp := s.verifyExternalAnchorExecuteTargetIdentityBinding(requestID, record); errResp != nil {
		return artifacts.ExternalAnchorPreparedMutationRecord{}, "", "", "", "", errResp
	}
	return record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, targetAuthLeaseID, nil
}

func (s *Service) loadExternalAnchorPreparedForExecute(requestID, preparedMutationID string) (artifacts.ExternalAnchorPreparedMutationRecord, *ErrorResponse) {
	trimmedID := strings.TrimSpace(preparedMutationID)
	if trimmedID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "prepared_mutation_id is required")
		return artifacts.ExternalAnchorPreparedMutationRecord{}, &errOut
	}
	record, ok := s.ExternalAnchorPreparedGet(trimmedID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_prepared_mutation", "storage", false, fmt.Sprintf("prepared mutation %q not found", trimmedID))
		return artifacts.ExternalAnchorPreparedMutationRecord{}, &errOut
	}
	if !externalAnchorPreparedLifecycleExecutable(record.LifecycleState) {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "prepared mutation is not in executable prepared state")
		return artifacts.ExternalAnchorPreparedMutationRecord{}, &errOut
	}
	return record, nil
}

func externalAnchorPreparedLifecycleExecutable(lifecycle string) bool {
	switch strings.TrimSpace(lifecycle) {
	case gitRemoteMutationLifecyclePrepared, gitRemoteMutationLifecycleExecuting, gitRemoteMutationLifecycleExecuted, gitRemoteMutationLifecycleFailed:
		return true
	default:
		return false
	}
}

func (s *Service) resolveExternalAnchorExecuteApprovalInput(requestID string, req ExternalAnchorMutationExecuteRequest) (string, string, string, *ErrorResponse) {
	approvalID := strings.TrimSpace(req.ApprovalID)
	if approvalID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "approval_id is required")
		return "", "", "", &errOut
	}
	approvalRequestHashIdentity, err := req.ApprovalRequestHash.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "approval_request_hash is invalid")
		return "", "", "", &errOut
	}
	approvalDecisionHashIdentity, err := req.ApprovalDecisionHash.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "approval_decision_hash is invalid")
		return "", "", "", &errOut
	}
	return approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity, nil
}

func (s *Service) resolveExternalAnchorTargetAuthLease(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord, leaseID string) (string, *ErrorResponse) {
	trimmedLeaseID := strings.TrimSpace(leaseID)
	if trimmedLeaseID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "target_auth_lease_id is required for external anchor execute")
		return "", &errOut
	}
	if s == nil || s.secretsSvc == nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "secrets service unavailable for external anchor execute")
		return "", &errOut
	}
	_, _, err := s.secretsSvc.Retrieve(secretsd.RetrieveRequest{
		LeaseID:      trimmedLeaseID,
		ConsumerID:   "principal:gateway:git:1",
		RoleKind:     "git-gateway",
		Scope:        "run:" + strings.TrimSpace(record.RunID),
		DeliveryKind: "git_gateway",
		GitUseContext: &secretsd.GitLeaseUseContext{
			RepositoryIdentity: strings.TrimSpace(record.DestinationRef),
			Operation:          "external_anchor_submit",
			ActionRequestHash:  strings.TrimSpace(record.ActionRequestHash),
			PolicyContextHash:  strings.TrimSpace(record.PolicyDecisionHash),
		},
	})
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "target auth lease retrieval failed for execute")
		return "", &errOut
	}
	return trimmedLeaseID, nil
}

func (s *Service) verifyExternalAnchorExecuteApprovalBindings(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity string) (artifacts.ApprovalRecord, *ErrorResponse) {
	if errResp := s.verifyPreparedExternalAnchorApprovalBinding(requestID, record, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity); errResp != nil {
		return artifacts.ApprovalRecord{}, errResp
	}
	approval, ok := s.ApprovalGet(approvalID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_approval", "storage", false, fmt.Sprintf("approval %q not found", approvalID))
		return artifacts.ApprovalRecord{}, &errOut
	}
	if errResp := s.verifyStoredExternalAnchorApprovalRecord(requestID, record, approval, approvalRequestHashIdentity, approvalDecisionHashIdentity); errResp != nil {
		return artifacts.ApprovalRecord{}, errResp
	}
	return approval, nil
}

func (s *Service) verifyPreparedExternalAnchorApprovalBinding(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord, approvalID, approvalRequestHashIdentity, approvalDecisionHashIdentity string) *ErrorResponse {
	for _, binding := range []struct {
		current string
		want    string
		message string
	}{{current: record.RequiredApprovalID, want: approvalID, message: "approval_id does not match prepared mutation required approval"}, {current: record.RequiredApprovalReqHash, want: approvalRequestHashIdentity, message: "approval_request_hash does not match prepared mutation required approval request hash"}, {current: record.RequiredApprovalDecHash, want: approvalDecisionHashIdentity, message: "approval_decision_hash does not match prepared mutation required approval decision hash"}} {
		if binding.current == "" || strings.TrimSpace(binding.current) == binding.want {
			continue
		}
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, binding.message)
		return &errOut
	}
	return nil
}

func (s *Service) verifyStoredExternalAnchorApprovalRecord(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord, approval artifacts.ApprovalRecord, approvalRequestHashIdentity, approvalDecisionHashIdentity string) *ErrorResponse {
	if errResp := s.verifyExternalAnchorApprovalStatus(requestID, approval); errResp != nil {
		return errResp
	}
	if errResp := s.verifyExternalAnchorApprovalDigests(requestID, record, approval, approvalRequestHashIdentity, approvalDecisionHashIdentity); errResp != nil {
		return errResp
	}
	return s.verifyExternalAnchorStoredTypedRequestHash(requestID, record)
}

func (s *Service) verifyExternalAnchorApprovalStatus(requestID string, approval artifacts.ApprovalRecord) *ErrorResponse {
	if approval.Status == "approved" || approval.Status == "consumed" {
		return nil
	}
	errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q is not approved", approval.ApprovalID))
	return &errOut
}

func (s *Service) verifyExternalAnchorApprovalDigests(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord, approval artifacts.ApprovalRecord, approvalRequestHashIdentity, approvalDecisionHashIdentity string) *ErrorResponse {
	for _, binding := range []struct {
		current string
		want    string
		message string
	}{{current: approval.ActionRequestHash, want: strings.TrimSpace(record.ActionRequestHash), message: "approval action_request_hash does not match prepared mutation binding"}, {current: approval.PolicyDecisionHash, want: strings.TrimSpace(record.PolicyDecisionHash), message: "approval policy_decision_hash does not match prepared mutation binding"}, {current: approval.RequestDigest, want: approvalRequestHashIdentity, message: "approval_request_hash does not match stored approval request digest"}, {current: approval.DecisionDigest, want: approvalDecisionHashIdentity, message: "approval_decision_hash does not match stored approval decision digest"}} {
		if strings.TrimSpace(binding.current) != "" && strings.TrimSpace(binding.current) == binding.want {
			continue
		}
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, binding.message)
		return &errOut
	}
	return nil
}

func (s *Service) verifyExternalAnchorStoredTypedRequestHash(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord) *ErrorResponse {
	storedTypedRequestHash, err := canonicalExternalAnchorTypedRequestHash(record.TypedRequest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("stored typed request hash invalid: %v", err))
		return &errOut
	}
	if strings.TrimSpace(record.TypedRequestHash) == storedTypedRequestHash {
		return nil
	}
	errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request hash no longer matches authoritative typed request")
	return &errOut
}

func (s *Service) verifyExternalAnchorExecuteTargetIdentityBinding(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord) *ErrorResponse {
	primaryTarget, err := externalAnchorResolvedPrimaryTargetFromPreparedRecord(record)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request target descriptor digest is invalid")
		return &errOut
	}
	wantDestinationRef := externalAnchorDestinationRefFromTargetDescriptorDigest(primaryTarget.TargetDescriptorIdentity)
	if strings.TrimSpace(wantDestinationRef) == "" || strings.TrimSpace(wantDestinationRef) != strings.TrimSpace(record.DestinationRef) {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored typed request target descriptor identity no longer matches prepared destination binding")
		return &errOut
	}
	return nil
}
