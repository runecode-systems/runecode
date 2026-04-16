package brokerapi

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

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
	revision, err := parsePositiveSummaryRevision(revisionRaw)
	if err != nil {
		return "", 0, false, err
	}
	return stageSummaryHash, revision, true, nil
}

func parsePositiveSummaryRevision(value any) (int64, error) {
	const maxSafeInteger = 9007199254740991

	switch typed := value.(type) {
	case float64:
		if typed < 1 || typed > maxSafeInteger || math.Trunc(typed) != typed {
			return 0, fmt.Errorf("details.summary_revision must be a positive integer")
		}
		return int64(typed), nil
	case int64:
		if typed < 1 || typed > maxSafeInteger {
			return 0, fmt.Errorf("details.summary_revision must be a positive integer")
		}
		return typed, nil
	case int:
		if typed < 1 || typed > maxSafeInteger {
			return 0, fmt.Errorf("details.summary_revision must be a positive integer")
		}
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("details.summary_revision has unsupported type %T", value)
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
	return latest.digest, latest.revision, latest.approvalID, true
}

func (s *Service) latestPendingStageSignOffPlanID(current approvalRecord, approvalID string) string {
	if strings.TrimSpace(approvalID) == "" {
		return ""
	}
	records := s.approvalRecordsByID()
	rec, ok := records[approvalID]
	if !ok {
		return ""
	}
	candidate, ok := pendingStageBindingCandidate(rec, current)
	if !ok {
		return ""
	}
	return candidate.planID
}

type latestStageBinding struct {
	approvalID  string
	requestedAt time.Time
	planID      string
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
		planID:      stagePlanIDFromRequestPayload(payload),
		digest:      digest,
		revision:    rev,
		hasRevision: revOK,
	}, true
}

func stagePlanIDFromRequestPayload(payload map[string]any) string {
	details, _ := payload["details"].(map[string]any)
	if len(details) == 0 {
		return ""
	}
	planID, _ := details["plan_id"].(string)
	return strings.TrimSpace(planID)
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
	if preferred, decided := compareStageBindingPlan(candidate, latest); decided {
		return preferred
	}
	switch {
	case candidate.hasRevision && !latest.hasRevision:
		return true
	case !candidate.hasRevision && latest.hasRevision:
		return false
	case candidate.hasRevision && latest.hasRevision && candidate.revision != latest.revision:
		return candidate.revision > latest.revision
	default:
		return prefersMoreRecentStageBinding(candidate, latest)
	}
}

func compareStageBindingPlan(candidate, latest latestStageBinding) (bool, bool) {
	switch {
	case candidate.planID != "" && latest.planID == "":
		return true, true
	case candidate.planID == "" && latest.planID != "":
		return false, true
	case candidate.planID != "" && latest.planID != "" && candidate.planID != latest.planID:
		return prefersMoreRecentStageBinding(candidate, latest), true
	default:
		return false, false
	}
}

func prefersMoreRecentStageBinding(candidate, latest latestStageBinding) bool {
	if candidate.requestedAt.After(latest.requestedAt) {
		return true
	}
	return candidate.requestedAt.Equal(latest.requestedAt) && candidate.approvalID > latest.approvalID
}
