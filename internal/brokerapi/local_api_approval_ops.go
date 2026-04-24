package brokerapi

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) HandleApprovalResolve(ctx context.Context, req ApprovalResolveRequest, meta RequestContext) (ApprovalResolveResponse, *ErrorResponse) {
	req = normalizeApprovalResolveRequest(req)
	requestID, requestCtx, done, errResp := s.prepareApprovalResolveExecution(ctx, req, meta)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	defer done()
	return s.resolveApprovalResponse(requestCtx, requestID, req)
}

func (s *Service) prepareApprovalResolveExecution(ctx context.Context, req ApprovalResolveRequest, meta RequestContext) (string, context.Context, func(), *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, approvalResolveRequestSchemaPath)
	if errResp != nil {
		return "", nil, nil, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return "", nil, nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	if err := requestCtx.Err(); err != nil {
		release()
		cancel()
		errOut := s.errorFromContext(requestID, err)
		return "", nil, nil, &errOut
	}
	return requestID, requestCtx, func() {
		release()
		cancel()
	}, nil
}

func (s *Service) resolveApprovalResponse(ctx context.Context, requestID string, req ApprovalResolveRequest) (ApprovalResolveResponse, *ErrorResponse) {
	if errResp := s.requestContextError(requestID, ctx); errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	resolved, errResp := s.resolveApprovalForPersistence(requestID, req)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	resp := buildApprovalResolveResponseNoArtifact(requestID, resolved.record, resolved.resumeResult.approvedArtifact, resolved.resumeResult.resolutionReason)
	if err := s.validateResponse(resp, approvalResolveResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalResolveResponse{}, &errOut
	}
	if errResp := s.requestContextError(requestID, ctx); errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	if err := s.persistApprovalRecord(resolved.record); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return ApprovalResolveResponse{}, &errOut
	}
	if errResp := s.persistApprovalFollowUps(ctx, requestID, resolved.record, resp.ResolutionReasonCode); errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	return resp, nil
}

func (s *Service) persistApprovalFollowUps(ctx context.Context, requestID string, record approvalRecord, resolutionReason string) *ErrorResponse {
	if errResp := s.requestContextError(requestID, ctx); errResp != nil {
		return errResp
	}
	if err := s.auditApprovalResolution(requestID, record.Summary.ApprovalID, record.Summary.Status, resolutionReason); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return &errOut
	}
	runID := strings.TrimSpace(record.Summary.BoundScope.RunID)
	if runID == "" {
		return nil
	}
	if err := s.syncSessionExecutionForRun(runID, s.now().UTC()); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return &errOut
	}
	if err := s.appendApprovalResolutionExecutionCheckpoint(runID, record.Summary.Status, record.Summary.ApprovalID); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return &errOut
	}
	return nil
}

func (s *Service) resolveCurrentPendingApproval(requestID string, req ApprovalResolveRequest, approvalID string) (approvalRecord, *ErrorResponse) {
	records := s.approvalRecordsByID()
	current, ok := records[approvalID]
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_approval", "storage", false, fmt.Sprintf("approval %q not found", approvalID))
		return approvalRecord{}, &errOut
	}
	if current.Summary.Status != "pending" {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q is already terminal with status %q", approvalID, current.Summary.Status))
		return approvalRecord{}, &errOut
	}
	if current.RequestEnvelope == nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q has no stored signed approval request", approvalID))
		return approvalRecord{}, &errOut
	}
	storedDigest, err := approvalIDFromRequest(*current.RequestEnvelope)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q stored request digest invalid: %v", approvalID, err))
		return approvalRecord{}, &errOut
	}
	if storedDigest != approvalID {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q stored request digest does not match approval id", approvalID))
		return approvalRecord{}, &errOut
	}
	promotion := req.promotionResolveDetails()
	if current.SourceDigest != "" && current.SourceDigest != promotion.UnapprovedDigest {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "unapproved_digest does not match pending approval source")
		return approvalRecord{}, &errOut
	}
	if errResp := validateBoundScopeMatchesStored(requestID, current.Summary.BoundScope, req.BoundScope, s); errResp != nil {
		return approvalRecord{}, errResp
	}
	return current, nil
}

func validateBoundScopeMatchesStored(requestID string, stored, requested ApprovalBoundScope, s *Service) *ErrorResponse {
	checks := []struct {
		name      string
		stored    string
		requested string
	}{
		{name: "action_kind", stored: stored.ActionKind, requested: requested.ActionKind},
		{name: "instance_id", stored: stored.InstanceID, requested: requested.InstanceID},
		{name: "workspace_id", stored: stored.WorkspaceID, requested: requested.WorkspaceID},
		{name: "run_id", stored: stored.RunID, requested: requested.RunID},
		{name: "stage_id", stored: stored.StageID, requested: requested.StageID},
		{name: "step_id", stored: stored.StepID, requested: requested.StepID},
		{name: "role_instance_id", stored: stored.RoleInstanceID, requested: requested.RoleInstanceID},
		{name: "policy_decision_hash", stored: stored.PolicyDecisionHash, requested: requested.PolicyDecisionHash},
	}
	for _, check := range checks {
		if mismatch := approvalBoundScopeFieldMismatch(check.requested, check.stored); mismatch {
			errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "bound_scope."+check.name+" does not match pending approval")
			return &errOut
		}
	}
	return nil
}

func approvalBoundScopeFieldMismatch(requested, stored string) bool {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return strings.TrimSpace(requested) != ""
	}
	return strings.TrimSpace(requested) != stored
}

func (s *Service) resumePromotionApproval(requestID string, req ApprovalResolveRequest) (*ArtifactSummary, *ErrorResponse) {
	head, promoteErr := s.promoteAndHeadResolvedArtifact(requestID, req)
	if promoteErr != nil {
		return nil, promoteErr
	}
	artifact := ptrArtifactSummary(toArtifactSummary(head))
	return artifact, nil
}
