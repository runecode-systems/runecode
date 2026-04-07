package brokerapi

import (
	"context"
	"fmt"
	"sort"
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
	approvalID, decisionDigest, outcome, errResp := s.resolveApprovalDigestsAndOutcome(requestID, req)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	records := s.approvalRecordsByID()
	current, ok := records[approvalID]
	if !ok {
		for _, rec := range records {
			if rec.SourceDigest == req.UnapprovedDigest {
				current = rec
				ok = true
				break
			}
		}
	}
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_approval", "storage", false, fmt.Sprintf("approval %q not found", approvalID))
		return ApprovalResolveResponse{}, &errOut
	}
	if current.Summary.Status != "pending" {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q is already terminal with status %q", approvalID, current.Summary.Status))
		return ApprovalResolveResponse{}, &errOut
	}
	if current.SourceDigest != "" && current.SourceDigest != req.UnapprovedDigest {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "unapproved_digest does not match pending approval source")
		return ApprovalResolveResponse{}, &errOut
	}
	record, buildErr := buildResolvedApprovalRecordForOutcome(req, current, approvalID, decisionDigest, outcome)
	if buildErr != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, buildErr.Error())
		return ApprovalResolveResponse{}, &errOut
	}

	var approvedArtifact *ArtifactSummary
	if outcome == "approve" {
		head, promoteErr := s.promoteAndHeadResolvedArtifact(requestID, req)
		if promoteErr != nil {
			return ApprovalResolveResponse{}, promoteErr
		}
		approvedArtifact = ptrArtifactSummary(toArtifactSummary(head))
	}

	s.putApproval(record)
	resp := buildApprovalResolveResponseNoArtifact(requestID, record, approvedArtifact)
	_ = s.auditApprovalResolution(requestID, record.Summary.ApprovalID, record.Summary.Status, resp.ResolutionReasonCode)
	if err := s.validateResponse(resp, approvalResolveResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalResolveResponse{}, &errOut
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

func (s *Service) putApproval(rec approvalRecord) {
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	if s.approvals.records == nil {
		s.approvals.records = map[string]approvalRecord{}
	}
	s.approvals.records[rec.Summary.ApprovalID] = rec
}
