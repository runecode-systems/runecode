package brokerapi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) HandleApprovalList(ctx context.Context, req ApprovalListRequest, meta RequestContext) (ApprovalListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, approvalListRequestSchemaPath)
	if errResp != nil {
		return ApprovalListResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ApprovalListResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return ApprovalListResponse{}, &errOut
	}
	order := approvalListOrder(req.Order)
	filtered := filterApprovalSummaries(s.listApprovals(), req)
	sortApprovals(filtered)
	limit := normalizeLimit(req.Limit, 50, 200)
	page, next, err := paginate(filtered, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return ApprovalListResponse{}, &errOut
	}
	resp := ApprovalListResponse{SchemaID: "runecode.protocol.v0.ApprovalListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Approvals: page, NextCursor: next}
	if err := s.validateResponse(resp, approvalListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalListResponse{}, &errOut
	}
	return resp, nil
}

func approvalListOrder(order string) string {
	if order == "" {
		return "pending_first_newest_within_status"
	}
	return order
}

func filterApprovalSummaries(records []ApprovalSummary, req ApprovalListRequest) []ApprovalSummary {
	filtered := make([]ApprovalSummary, 0, len(records))
	for _, rec := range records {
		if req.Status != "" && rec.Status != req.Status {
			continue
		}
		if req.RunID != "" && rec.BoundScope.RunID != req.RunID {
			continue
		}
		filtered = append(filtered, rec)
	}
	return filtered
}

func (s *Service) HandleApprovalGet(ctx context.Context, req ApprovalGetRequest, meta RequestContext) (ApprovalGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, approvalGetRequestSchemaPath)
	if errResp != nil {
		return ApprovalGetResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ApprovalGetResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return ApprovalGetResponse{}, &errOut
	}
	rec, ok := s.getApproval(req.ApprovalID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_approval", "storage", false, fmt.Sprintf("approval %q not found", req.ApprovalID))
		return ApprovalGetResponse{}, &errOut
	}
	resp := ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Approval: rec.Summary, SignedApprovalRequest: rec.RequestEnvelope, SignedApprovalDecision: rec.DecisionEnvelope}
	if err := s.validateResponse(resp, approvalGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) listApprovals() []ApprovalSummary {
	records := s.approvalRecordsByID()
	out := make([]ApprovalSummary, 0, len(records))
	for _, record := range records {
		out = append(out, record.Summary)
	}
	return out
}

func sortApprovals(items []ApprovalSummary) {
	statusRank := map[string]int{"pending": 0, "approved": 1, "denied": 2, "expired": 3, "cancelled": 4, "superseded": 5, "consumed": 6}
	sort.SliceStable(items, func(i, j int) bool {
		ri := statusRank[items[i].Status]
		rj := statusRank[items[j].Status]
		if ri != rj {
			return ri < rj
		}
		if items[i].RequestedAt == items[j].RequestedAt {
			return items[i].ApprovalID < items[j].ApprovalID
		}
		return items[i].RequestedAt > items[j].RequestedAt
	})
}

func (s *Service) getApproval(id string) (approvalRecord, bool) {
	rec, ok := s.approvalRecordsByID()[id]
	return rec, ok
}

func (s *Service) persistApprovalRecord(rec approvalRecord) error {
	stored := artifacts.ApprovalRecord{
		ApprovalID:             rec.Summary.ApprovalID,
		Status:                 rec.Summary.Status,
		WorkspaceID:            rec.Summary.BoundScope.WorkspaceID,
		RunID:                  rec.Summary.BoundScope.RunID,
		StageID:                rec.Summary.BoundScope.StageID,
		StepID:                 rec.Summary.BoundScope.StepID,
		RoleInstanceID:         rec.Summary.BoundScope.RoleInstanceID,
		ActionKind:             rec.Summary.BoundScope.ActionKind,
		ApprovalTriggerCode:    rec.Summary.ApprovalTriggerCode,
		ChangesIfApproved:      rec.Summary.ChangesIfApproved,
		ApprovalAssuranceLevel: rec.Summary.ApprovalAssuranceLevel,
		PresenceMode:           rec.Summary.PresenceMode,
		PolicyDecisionHash:     rec.Summary.PolicyDecisionHash,
		SupersededByApprovalID: rec.Summary.SupersededByApprovalID,
		RequestDigest:          rec.Summary.RequestDigest,
		DecisionDigest:         rec.Summary.DecisionDigest,
		SourceDigest:           rec.SourceDigest,
		RequestEnvelope:        rec.RequestEnvelope,
		DecisionEnvelope:       rec.DecisionEnvelope,
		ManifestHash:           rec.ManifestHash,
		ActionRequestHash:      rec.ActionRequestHash,
		RelevantArtifactHashes: append([]string{}, rec.RelevantArtifactHashes...),
	}
	applyApprovalSummaryTimes(&stored, rec.Summary)
	if err := s.RecordApproval(stored); err != nil {
		return err
	}
	return s.recordRunnerApprovalWaitFromCanonical(stored)
}

func applyApprovalSummaryTimes(stored *artifacts.ApprovalRecord, summary ApprovalSummary) {
	if ts, ok := parseRFC3339(summary.RequestedAt); ok {
		stored.RequestedAt = ts
	}
	if ts, ok := parseRFC3339(summary.ExpiresAt); ok {
		stored.ExpiresAt = &ts
	}
	if ts, ok := parseRFC3339(summary.DecidedAt); ok {
		stored.DecidedAt = &ts
	}
	if ts, ok := parseRFC3339(summary.ConsumedAt); ok {
		stored.ConsumedAt = &ts
	}
}

func parseRFC3339(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return ts.UTC(), true
}

func (s *Service) recordRunnerApprovalWaitFromCanonical(stored artifacts.ApprovalRecord) error {
	if strings.TrimSpace(stored.RunID) == "" {
		return nil
	}
	approvalType, actionHash, stageHash := approvalBindingForRunnerHint(stored)
	occurredAt := stored.RequestedAt
	if occurredAt.IsZero() {
		occurredAt = s.now().UTC()
	}
	resolvedAt := stored.DecidedAt
	if strings.TrimSpace(stored.Status) == "consumed" && stored.ConsumedAt != nil {
		resolvedAt = stored.ConsumedAt
	}
	var resolvedCopy *time.Time
	if resolvedAt != nil {
		t := resolvedAt.UTC()
		resolvedCopy = &t
	}
	return s.store.RecordRunnerApprovalWait(artifacts.RunnerApproval{
		ApprovalID:            stored.ApprovalID,
		RunID:                 stored.RunID,
		StageID:               stored.StageID,
		StepID:                stored.StepID,
		RoleInstanceID:        stored.RoleInstanceID,
		Status:                stored.Status,
		ApprovalType:          approvalType,
		BoundActionHash:       actionHash,
		BoundStageSummaryHash: stageHash,
		OccurredAt:            occurredAt.UTC(),
		ResolvedAt:            resolvedCopy,
		SupersededByApproval:  stored.SupersededByApprovalID,
	})
}

func approvalBindingForRunnerHint(stored artifacts.ApprovalRecord) (string, string, string) {
	if strings.TrimSpace(stored.ActionKind) == "stage_summary_sign_off" {
		if stored.RequestEnvelope != nil {
			payload, err := decodeApprovalRequestPayload(*stored.RequestEnvelope)
			if err == nil {
				details, _ := payload["details"].(map[string]any)
				if details != nil {
					if digest, err := digestIdentityFromPayloadObject(details, "stage_summary_hash"); err == nil {
						return "stage_sign_off", "", digest
					}
				}
			}
		}
		return "stage_sign_off", "", stored.ManifestHash
	}
	return "exact_action", stored.ActionRequestHash, ""
}
