package brokerapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

type resolvedApprovalResult struct {
	record       approvalRecord
	resumeResult approvalResumeResult
}

func (s *Service) resolveApprovalForPersistence(requestID string, req ApprovalResolveRequest) (resolvedApprovalResult, *ErrorResponse) {
	resolvedInput, errResp := s.resolveApprovalInput(requestID, req)
	if errResp != nil {
		return resolvedApprovalResult{}, errResp
	}
	current, errResp := s.resolveCurrentPendingApproval(requestID, req, resolvedInput.approvalID)
	if errResp != nil {
		return resolvedApprovalResult{}, errResp
	}
	if err := validateApprovalRequestBindingToStoredRecord(current, resolvedInput.requestPayload); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return resolvedApprovalResult{}, &errOut
	}
	if errResp := s.enforcePendingApprovalFreshness(requestID, current); errResp != nil {
		return resolvedApprovalResult{}, errResp
	}
	resolvedStatus, _ := approvalStatusForDecisionOutcome(resolvedInput.outcome)
	resumeResult, errResp := s.dispatchApprovalResumeHandler(requestID, req, current, resolvedInput)
	if errResp != nil {
		return resolvedApprovalResult{}, errResp
	}
	record, errResp := s.buildResolvedApprovalRecord(requestID, req, current, resolvedInput, resolvedStatus, resumeResult)
	if errResp != nil {
		return resolvedApprovalResult{}, errResp
	}
	return resolvedApprovalResult{record: record, resumeResult: resumeResult}, nil
}

func (s *Service) buildResolvedApprovalRecord(requestID string, req ApprovalResolveRequest, current approvalRecord, input approvalResolutionInput, resolvedStatus string, resumeResult approvalResumeResult) (approvalRecord, *ErrorResponse) {
	if resumeResult.statusOverride != "" {
		resolvedStatus = resumeResult.statusOverride
	}
	resolvedAt := s.now().UTC()
	record, err := buildResolvedApprovalRecordForOutcome(req, current, input.approvalID, input.decisionDigest, resolvedStatus, resumeResult.supersededByID, resolvedAt)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return approvalRecord{}, &errOut
	}
	applyResolvedApprovalRecordSummary(&record, req, current, resolvedStatus, resolvedAt)
	return record, nil
}

func applyResolvedApprovalRecordSummary(record *approvalRecord, req ApprovalResolveRequest, current approvalRecord, resolvedStatus string, resolvedAt time.Time) {
	if resolvedStatus == "consumed" {
		record.Summary.ConsumedAt = resolvedAt.Format(time.RFC3339)
	}
	if current.Summary.BoundScope.ActionKind != policyengine.ActionKindBackendPosture {
		return
	}
	selection := req.normalizedResolutionDetails().BackendPostureSelection
	if selection != nil {
		record.Summary.BoundScope.InstanceID = selection.TargetInstanceID
	}
}

func (s *Service) enforcePendingApprovalFreshness(requestID string, current approvalRecord) *ErrorResponse {
	if strings.TrimSpace(current.Summary.ExpiresAt) == "" {
		return nil
	}
	expiresAt, ok := parseRFC3339(current.Summary.ExpiresAt)
	if !ok {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stored pending approval has invalid expires_at")
		return &errOut
	}
	if !s.now().UTC().Before(expiresAt) {
		expiredRecord, err := s.buildExpiredApprovalRecord(current)
		if err != nil {
			errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
			return &errOut
		}
		if persistErr := s.persistApprovalRecord(expiredRecord); persistErr != nil {
			errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, persistErr.Error())
			return &errOut
		}
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q is expired", current.Summary.ApprovalID))
		return &errOut
	}
	return nil
}

func (s *Service) buildExpiredApprovalRecord(current approvalRecord) (approvalRecord, error) {
	now := s.now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(current.Summary.RequestedAt) == "" {
		return approvalRecord{}, fmt.Errorf("stored pending approval missing requested_at")
	}
	updated := current
	updated.Summary.Status = "expired"
	updated.Summary.DecidedAt = now
	updated.DecisionEnvelope = nil
	updated.Summary.DecisionDigest = ""
	return updated, nil
}

func (s *Service) dispatchApprovalResumeHandler(requestID string, req ApprovalResolveRequest, current approvalRecord, input approvalResolutionInput) (approvalResumeResult, *ErrorResponse) {
	if input.outcome != "approve" {
		return approvalResumeResult{}, nil
	}
	if current.Summary.BoundScope.ActionKind == policyengine.ActionKindBackendPosture {
		return s.resumeBackendPostureApproval(requestID, req, current)
	}
	switch current.Summary.BoundScope.ActionKind {
	case policyengine.ActionKindPromotion:
		artifact, errResp := s.resumePromotionApproval(requestID, req)
		if errResp != nil {
			return approvalResumeResult{}, errResp
		}
		return approvalResumeResult{statusOverride: "consumed", resolutionReason: "approval_consumed", approvedArtifact: artifact}, nil
	case policyengine.ActionKindStageSummarySign:
		return s.resumeStageSummarySignOff(requestID, current, input)
	default:
		return s.resumeExactActionApproval(requestID, current)
	}
}

func (s *Service) resumeBackendPostureApproval(requestID string, req ApprovalResolveRequest, current approvalRecord) (approvalResumeResult, *ErrorResponse) {
	if strings.TrimSpace(current.ActionRequestHash) == "" {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "backend posture approval missing bound action_request_hash")
		return approvalResumeResult{}, &errOut
	}
	if err := s.validateBackendPostureBinding(current, req); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return approvalResumeResult{}, &errOut
	}
	if err := s.applyResolvedBackendPosture(current, req); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return approvalResumeResult{}, &errOut
	}
	return approvalResumeResult{statusOverride: "consumed", resolutionReason: "approval_consumed"}, nil
}

func (s *Service) resumeExactActionApproval(requestID string, current approvalRecord) (approvalResumeResult, *ErrorResponse) {
	if strings.TrimSpace(current.ActionRequestHash) == "" {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "exact-action approval missing bound action_request_hash")
		return approvalResumeResult{}, &errOut
	}
	return approvalResumeResult{statusOverride: "consumed", resolutionReason: "approval_consumed"}, nil
}

func (s *Service) resumeStageSummarySignOff(requestID string, current approvalRecord, input approvalResolutionInput) (approvalResumeResult, *ErrorResponse) {
	bound, errResp := s.resolveStoredStageSignOffBinding(requestID, current)
	if errResp != nil {
		return approvalResumeResult{}, errResp
	}
	latestDigest, latestRevision, latestID, hasLatest := s.latestPendingStageSignOffBinding(current, input.approvalID)
	if !hasLatest {
		return approvalResumeResult{statusOverride: "superseded", resolutionReason: "approval_superseded", supersededByID: latestID}, nil
	}
	if bound.planID != "" {
		latestPlanID := s.latestPendingStageSignOffPlanID(current, latestID)
		if latestPlanID != "" && latestPlanID != bound.planID {
			return approvalResumeResult{statusOverride: "superseded", resolutionReason: "approval_superseded", supersededByID: latestID}, nil
		}
	}
	if latestDigest != bound.digest {
		return approvalResumeResult{statusOverride: "superseded", resolutionReason: "approval_superseded", supersededByID: latestID}, nil
	}
	if bound.hasRevision && latestRevision > bound.revision {
		return approvalResumeResult{statusOverride: "superseded", resolutionReason: "approval_superseded", supersededByID: latestID}, nil
	}
	return approvalResumeResult{statusOverride: "consumed", resolutionReason: "approval_consumed"}, nil
}

func (s *Service) resolveStoredStageSignOffBinding(requestID string, current approvalRecord) (latestStageBinding, *ErrorResponse) {
	if current.RequestEnvelope == nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stage sign-off approval has no stored signed request")
		return latestStageBinding{}, &errOut
	}
	payload, err := decodeApprovalRequestPayload(*current.RequestEnvelope)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return latestStageBinding{}, &errOut
	}
	digest, revision, hasRevision, err := stageSignOffBindingFromRequestPayload(payload)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return latestStageBinding{}, &errOut
	}
	return latestStageBinding{
		planID:      stagePlanIDFromRequestPayload(payload),
		digest:      digest,
		revision:    revision,
		hasRevision: hasRevision,
	}, nil
}
