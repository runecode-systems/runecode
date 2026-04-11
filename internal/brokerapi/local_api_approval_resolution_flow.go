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
	if resumeResult.statusOverride != "" {
		resolvedStatus = resumeResult.statusOverride
	}
	resolvedAt := s.now().UTC()
	record, buildErr := buildResolvedApprovalRecordForOutcome(req, current, resolvedInput.approvalID, resolvedInput.decisionDigest, resolvedStatus, resumeResult.supersededByID, resolvedAt)
	if buildErr != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, buildErr.Error())
		return resolvedApprovalResult{}, &errOut
	}
	if resolvedStatus == "consumed" {
		record.Summary.ConsumedAt = resolvedAt.Format(time.RFC3339)
	}
	return resolvedApprovalResult{record: record, resumeResult: resumeResult}, nil
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
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("resume handler not yet supported for action_kind %q", current.Summary.BoundScope.ActionKind))
		return approvalResumeResult{}, &errOut
	}
}

func (s *Service) resumeStageSummarySignOff(requestID string, current approvalRecord, input approvalResolutionInput) (approvalResumeResult, *ErrorResponse) {
	if current.RequestEnvelope == nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "stage sign-off approval has no stored signed request")
		return approvalResumeResult{}, &errOut
	}
	payload, err := decodeApprovalRequestPayload(*current.RequestEnvelope)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return approvalResumeResult{}, &errOut
	}
	boundDigest, revision, hasRevision, err := stageSignOffBindingFromRequestPayload(payload)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return approvalResumeResult{}, &errOut
	}
	latestDigest, latestRevision, latestID, hasLatest := s.latestPendingStageSignOffBinding(current, input.approvalID)
	if !hasLatest {
		return approvalResumeResult{statusOverride: "superseded", resolutionReason: "approval_superseded", supersededByID: latestID}, nil
	}
	if latestDigest != boundDigest {
		return approvalResumeResult{statusOverride: "superseded", resolutionReason: "approval_superseded", supersededByID: latestID}, nil
	}
	if hasRevision && latestRevision > revision {
		return approvalResumeResult{statusOverride: "superseded", resolutionReason: "approval_superseded", supersededByID: latestID}, nil
	}
	return approvalResumeResult{statusOverride: "consumed", resolutionReason: "approval_consumed"}, nil
}

func stageSignOffBindingFromRequestPayload(payload map[string]any) (string, int64, bool, error) {
	details, _ := payload["details"].(map[string]any)
	if len(details) == 0 {
		return "", 0, false, fmt.Errorf("stage sign-off approval request missing details payload")
	}
	stageSummaryHash, err := digestIdentityFromPayloadObject(details, "stage_summary_hash")
	if err != nil {
		return "", 0, false, fmt.Errorf("details.stage_summary_hash: %w", err)
	}
	revisionRaw, ok := details["summary_revision"]
	if !ok {
		return stageSummaryHash, 0, false, nil
	}
	switch value := revisionRaw.(type) {
	case float64:
		return stageSummaryHash, int64(value), true, nil
	case int64:
		return stageSummaryHash, value, true, nil
	case int:
		return stageSummaryHash, int64(value), true, nil
	default:
		return "", 0, false, fmt.Errorf("details.summary_revision has unsupported type %T", revisionRaw)
	}
}

func (s *Service) latestPendingStageSignOffBinding(current approvalRecord, currentApprovalID string) (string, int64, string, bool) {
	approvals := s.approvalRecordsByID()
	latest := latestStageBinding{}
	for _, rec := range approvals {
		candidate, ok := pendingStageBindingCandidate(rec, current)
		if !ok {
			continue
		}
		if prefersStageBindingCandidate(candidate, latest) {
			latest = candidate
		}
	}
	if latest.approvalID == "" {
		return "", 0, "", false
	}
	if latest.approvalID == currentApprovalID {
		return latest.digest, latest.revision, latest.approvalID, true
	}
	if !latest.hasRevision {
		return latest.digest, latest.revision, latest.approvalID, true
	}
	return latest.digest, latest.revision, latest.approvalID, true
}

type latestStageBinding struct {
	approvalID  string
	requestedAt time.Time
	digest      string
	revision    int64
	hasRevision bool
}

func pendingStageBindingCandidate(rec approvalRecord, current approvalRecord) (latestStageBinding, bool) {
	if rec.Summary.Status != "pending" {
		return latestStageBinding{}, false
	}
	if rec.Summary.BoundScope.ActionKind != policyengine.ActionKindStageSummarySign {
		return latestStageBinding{}, false
	}
	if rec.Summary.BoundScope.RunID != current.Summary.BoundScope.RunID || rec.Summary.BoundScope.StageID != current.Summary.BoundScope.StageID {
		return latestStageBinding{}, false
	}
	if rec.RequestEnvelope == nil {
		return latestStageBinding{}, false
	}
	payload, err := decodeApprovalRequestPayload(*rec.RequestEnvelope)
	if err != nil {
		return latestStageBinding{}, false
	}
	digest, rev, revOK, err := stageSignOffBindingFromRequestPayload(payload)
	if err != nil || strings.TrimSpace(digest) == "" {
		return latestStageBinding{}, false
	}
	return latestStageBinding{
		approvalID:  rec.Summary.ApprovalID,
		requestedAt: parseRequestedAt(rec.Summary.RequestedAt),
		digest:      digest,
		revision:    rev,
		hasRevision: revOK,
	}, true
}

func parseRequestedAt(value string) time.Time {
	ts, ok := parseRFC3339(strings.TrimSpace(value))
	if !ok {
		return time.Time{}
	}
	return ts
}

func prefersStageBindingCandidate(candidate, latest latestStageBinding) bool {
	if latest.approvalID == "" {
		return true
	}
	switch {
	case candidate.hasRevision && !latest.hasRevision:
		return true
	case candidate.hasRevision && latest.hasRevision && candidate.revision > latest.revision:
		return true
	case candidate.hasRevision == latest.hasRevision && candidate.revision == latest.revision:
		if candidate.requestedAt.After(latest.requestedAt) {
			return true
		}
		return candidate.requestedAt.Equal(latest.requestedAt) && candidate.approvalID > latest.approvalID
	default:
		return false
	}
}
