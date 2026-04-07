package brokerapi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
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
	if err := s.seedStubApprovals(); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
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
	if err := s.seedStubApprovals(); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ApprovalGetResponse{}, &errOut
	}
	rec, ok := s.getApproval(req.ApprovalID)
	if !ok {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q not found", req.ApprovalID))
		return ApprovalGetResponse{}, &errOut
	}
	resp := ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Approval: rec.Summary, SignedApprovalRequest: rec.RequestEnvelope, SignedApprovalDecision: rec.DecisionEnvelope}
	if err := s.validateResponse(resp, approvalGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleApprovalResolve(ctx context.Context, req ApprovalResolveRequest, meta RequestContext) (ApprovalResolveResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, approvalResolveRequestSchemaPath)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ApprovalResolveResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return ApprovalResolveResponse{}, &errOut
	}
	approvalID, decisionDigest, errResp := s.resolveApprovalDigests(requestID, req)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	head, errResp := s.promoteAndHeadResolvedArtifact(requestID, req)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	record := buildResolvedApprovalRecord(req, approvalID, decisionDigest)
	s.putApproval(record)
	resp := buildApprovalResolveResponse(requestID, record, head)
	if err := s.validateResponse(resp, approvalResolveResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalResolveResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) seedStubApprovals() error {
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	if s.approvals.seeded {
		return nil
	}
	if s.approvals.records == nil {
		s.approvals.records = map[string]approvalRecord{}
	}
	runs, err := s.runSummaries("updated_at_desc")
	if err != nil {
		return err
	}
	seedStubApprovalRecords(s.approvals.records, runs, time.Now().UTC())
	s.approvals.seeded = true
	return nil
}

func seedStubApprovalRecords(records map[string]approvalRecord, runs []RunSummary, now time.Time) {
	for i, run := range runs {
		if i >= 2 {
			return
		}
		records[stubApprovalID(run.RunID)] = stubApprovalRecord(run, i, now)
	}
}

func stubApprovalID(runID string) string {
	return shaDigestIdentity("stub-approval:" + runID)
}

func stubApprovalRecord(run RunSummary, idx int, now time.Time) approvalRecord {
	id := stubApprovalID(run.RunID)
	return approvalRecord{Summary: ApprovalSummary{
		SchemaID:               "runecode.protocol.v0.ApprovalSummary",
		SchemaVersion:          "0.1.0",
		ApprovalID:             id,
		Status:                 "pending",
		RequestedAt:            now.Add(-time.Duration(idx+1) * time.Minute).Format(time.RFC3339),
		ExpiresAt:              now.Add(20 * time.Minute).Format(time.RFC3339),
		ApprovalTriggerCode:    "stage_sign_off",
		ChangesIfApproved:      "Unblock stage progression for local workflow.",
		ApprovalAssuranceLevel: "session_authenticated",
		PresenceMode:           "os_confirmation",
		BoundScope: ApprovalBoundScope{
			SchemaID:      "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion: "0.1.0",
			WorkspaceID:   run.WorkspaceID,
			RunID:         run.RunID,
			StageID:       run.CurrentStageID,
			ActionKind:    "stage_transition",
		},
		PolicyDecisionHash: "sha256:" + strings.Repeat("1", 64),
		RequestDigest:      id,
	}}
}

func (s *Service) listApprovals() []ApprovalSummary {
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	out := make([]ApprovalSummary, 0, len(s.approvals.records))
	for _, record := range s.approvals.records {
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
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	rec, ok := s.approvals.records[id]
	return rec, ok
}

func (s *Service) putApproval(rec approvalRecord) {
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	if s.approvals.records == nil {
		s.approvals.records = map[string]approvalRecord{}
	}
	s.approvals.records[rec.Summary.ApprovalID] = rec
}
