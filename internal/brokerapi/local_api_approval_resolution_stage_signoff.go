package brokerapi

import (
	"fmt"
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
