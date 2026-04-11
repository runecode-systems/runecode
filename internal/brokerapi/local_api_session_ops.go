package brokerapi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) HandleSessionList(ctx context.Context, req SessionListRequest, meta RequestContext) (SessionListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, sessionListRequestSchemaPath)
	if errResp != nil {
		return SessionListResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionListResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return SessionListResponse{}, &errOut
	}
	order := req.Order
	if order == "" {
		order = "updated_at_desc"
	}
	summaries, err := s.sessionSummaries(order)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionListResponse{}, &errOut
	}
	limit := normalizeLimit(req.Limit, 50, 200)
	page, next, err := paginate(summaries, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return SessionListResponse{}, &errOut
	}
	resp := SessionListResponse{SchemaID: "runecode.protocol.v0.SessionListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Sessions: page, NextCursor: next}
	if err := s.validateResponse(resp, sessionListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionListResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleSessionGet(ctx context.Context, req SessionGetRequest, meta RequestContext) (SessionGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, sessionGetRequestSchemaPath)
	if errResp != nil {
		return SessionGetResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionGetResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return SessionGetResponse{}, &errOut
	}
	if strings.TrimSpace(req.SessionID) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "session_id is required")
		return SessionGetResponse{}, &errOut
	}
	detail, ok, err := s.sessionDetail(req.SessionID)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionGetResponse{}, &errOut
	}
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_session", "storage", false, fmt.Sprintf("session %q not found", req.SessionID))
		return SessionGetResponse{}, &errOut
	}
	resp := SessionGetResponse{SchemaID: "runecode.protocol.v0.SessionGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Session: detail}
	if err := s.validateResponse(resp, sessionGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleSessionSendMessage(ctx context.Context, req SessionSendMessageRequest, meta RequestContext) (SessionSendMessageResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, sessionSendMessageRequestSchemaPath)
	if errResp != nil {
		return SessionSendMessageResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionSendMessageResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return SessionSendMessageResponse{}, &errOut
	}
	if strings.TrimSpace(req.SessionID) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "session_id is required")
		return SessionSendMessageResponse{}, &errOut
	}
	if strings.TrimSpace(req.ContentText) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "content_text is required")
		return SessionSendMessageResponse{}, &errOut
	}
	if req.Role != "user" && req.Role != "assistant" && req.Role != "system" && req.Role != "tool" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "role is invalid")
		return SessionSendMessageResponse{}, &errOut
	}
	if prior, ok := s.sessionIdempotentInteractionResponse(req.SessionID, strings.TrimSpace(req.IdempotencyKey)); ok {
		prior.RequestID = requestID
		if err := s.validateResponse(prior, sessionSendMessageResponseSchemaPath); err != nil {
			errOut := s.errorFromValidation(requestID, err)
			return SessionSendMessageResponse{}, &errOut
		}
		return prior, nil
	}
	session, ok, err := s.sessionDetail(req.SessionID)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return SessionSendMessageResponse{}, &errOut
	}
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_session", "storage", false, fmt.Sprintf("session %q not found", req.SessionID))
		return SessionSendMessageResponse{}, &errOut
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	createdAt := now.Format(time.RFC3339)
	turnIndex := s.nextSessionInteractionTurnIndex(req.SessionID, session.Summary.TurnCount)
	turnID := fmt.Sprintf("%s.turn.%06d", req.SessionID, turnIndex)
	message := SessionTranscriptMessage{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptMessage",
		SchemaVersion: "0.1.0",
		MessageID:     fmt.Sprintf("%s.msg.%06d", turnID, 1),
		TurnID:        turnID,
		SessionID:     req.SessionID,
		MessageIndex:  1,
		Role:          req.Role,
		CreatedAt:     createdAt,
		ContentText:   req.ContentText,
		RelatedLinks:  normalizedSessionTranscriptLinks(req.RelatedLinks),
	}
	turn := SessionTranscriptTurn{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptTurn",
		SchemaVersion: "0.1.0",
		TurnID:        turnID,
		SessionID:     req.SessionID,
		TurnIndex:     turnIndex,
		StartedAt:     createdAt,
		CompletedAt:   createdAt,
		Status:        "completed",
		Messages:      []SessionTranscriptMessage{message},
	}
	seq := s.nextSessionInteractionSeq(req.SessionID)
	resp := SessionSendMessageResponse{
		SchemaID:      "runecode.protocol.v0.SessionSendMessageResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		SessionID:     req.SessionID,
		Turn:          turn,
		Message:       message,
		EventType:     "session_message_ack",
		StreamID:      sessionInteractionStreamID(req.SessionID),
		Seq:           seq,
	}
	if err := s.validateResponse(resp, sessionSendMessageResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return SessionSendMessageResponse{}, &errOut
	}
	s.storeSessionIdempotentInteractionResponse(req.SessionID, strings.TrimSpace(req.IdempotencyKey), resp)
	_ = s.AppendTrustedAuditEvent("session_message_recorded", "brokerapi", map[string]interface{}{"session_id": req.SessionID, "turn_id": turn.TurnID, "message_id": message.MessageID, "stream_id": resp.StreamID, "seq": resp.Seq, "request_id": requestID, "role": req.Role})
	return resp, nil
}

func sessionInteractionStreamID(sessionID string) string {
	return "session-" + sessionID
}

func normalizedSessionTranscriptLinks(in *SessionTranscriptLinks) SessionTranscriptLinks {
	if in == nil {
		return SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: []string{}, ApprovalIDs: []string{}, ArtifactDigests: []string{}, AuditRecordDigests: []string{}}
	}
	return SessionTranscriptLinks{SchemaID: "runecode.protocol.v0.SessionTranscriptLinks", SchemaVersion: "0.1.0", RunIDs: append([]string{}, in.RunIDs...), ApprovalIDs: append([]string{}, in.ApprovalIDs...), ArtifactDigests: append([]string{}, in.ArtifactDigests...), AuditRecordDigests: append([]string{}, in.AuditRecordDigests...)}
}

func (s *Service) sessionSummaries(order string) ([]SessionSummary, error) {
	all := s.List()
	byID := map[string][]artifacts.ArtifactRecord{}
	approvalsBySession := map[string]map[string]struct{}{}
	auditBySession, err := s.auditRecordDigestsBySession()
	if err != nil {
		return nil, err
	}
	for _, approval := range s.listApprovals() {
		runID := strings.TrimSpace(approval.BoundScope.RunID)
		if runID == "" {
			continue
		}
		runtime := s.RuntimeFacts(runID)
		sessionID := strings.TrimSpace(runtime.LaunchReceipt.Normalized().SessionID)
		if sessionID == "" {
			continue
		}
		if _, ok := approvalsBySession[sessionID]; !ok {
			approvalsBySession[sessionID] = map[string]struct{}{}
		}
		approvalsBySession[sessionID][approval.ApprovalID] = struct{}{}
	}
	for _, rec := range all {
		if rec.RunID == "" {
			continue
		}
		runtime := s.RuntimeFacts(rec.RunID)
		sessionID := strings.TrimSpace(runtime.LaunchReceipt.Normalized().SessionID)
		if sessionID == "" {
			continue
		}
		byID[sessionID] = append(byID[sessionID], rec)
	}
	out := make([]SessionSummary, 0, len(byID))
	for sessionID, records := range byID {
		out = append(out, buildSessionSummary(sessionID, records, len(approvalsBySession[sessionID]), len(auditBySession[sessionID])))
	}
	sort.Slice(out, func(i, j int) bool {
		if order == "updated_at_asc" {
			if out[i].UpdatedAt == out[j].UpdatedAt {
				return out[i].Identity.SessionID < out[j].Identity.SessionID
			}
			return out[i].UpdatedAt < out[j].UpdatedAt
		}
		if out[i].UpdatedAt == out[j].UpdatedAt {
			return out[i].Identity.SessionID < out[j].Identity.SessionID
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out, nil
}

func (s *Service) sessionDetail(sessionID string) (SessionDetail, bool, error) {
	summaries, err := s.sessionSummaries("updated_at_desc")
	if err != nil {
		return SessionDetail{}, false, err
	}
	auditBySession, err := s.auditRecordDigestsBySession()
	if err != nil {
		return SessionDetail{}, false, err
	}
	for _, summary := range summaries {
		if summary.Identity.SessionID != sessionID {
			continue
		}
		all := s.List()
		runs := map[string]struct{}{}
		approvals := map[string]struct{}{}
		artifactsByDigest := map[string]struct{}{}
		for _, rec := range all {
			if rec.RunID == "" {
				continue
			}
			runtime := s.RuntimeFacts(rec.RunID)
			if strings.TrimSpace(runtime.LaunchReceipt.Normalized().SessionID) != sessionID {
				continue
			}
			runs[rec.RunID] = struct{}{}
			artifactsByDigest[rec.Reference.Digest] = struct{}{}
		}
		for _, approval := range s.listApprovals() {
			if approval.BoundScope.RunID == "" {
				continue
			}
			runtime := s.RuntimeFacts(approval.BoundScope.RunID)
			if strings.TrimSpace(runtime.LaunchReceipt.Normalized().SessionID) != sessionID {
				continue
			}
			approvals[approval.ApprovalID] = struct{}{}
		}
		linkedRuns := boundedStrings(sortedStringKeys(runs), 256)
		linkedApprovals := boundedStrings(sortedStringKeys(approvals), 512)
		linkedArtifacts := boundedStrings(sortedStringKeys(artifactsByDigest), 1024)
		linkedAudit := boundedStrings(sortedStringKeys(auditBySession[sessionID]), 1024)
		detail := SessionDetail{
			SchemaID:                 "runecode.protocol.v0.SessionDetail",
			SchemaVersion:            "0.1.0",
			Summary:                  summary,
			TranscriptTurns:          buildSessionTranscriptTurns(sessionID, summary, runs, approvals, artifactsByDigest, auditBySession[sessionID]),
			LinkedRunIDs:             linkedRuns,
			LinkedApprovalIDs:        linkedApprovals,
			LinkedArtifactDigests:    linkedArtifacts,
			LinkedAuditRecordDigests: linkedAudit,
		}
		return detail, true, nil
	}
	return SessionDetail{}, false, nil
}

func (s *Service) auditRecordDigestsBySession() (map[string]map[string]struct{}, error) {
	events, err := s.ReadAuditEvents()
	if err != nil {
		return nil, err
	}
	bySession := map[string]map[string]struct{}{}
	for _, event := range events {
		details := event.Details
		if details == nil {
			continue
		}
		sessionID, ok := detailString(details, "session_id")
		if !ok {
			continue
		}
		recordDigest, ok := detailString(details, "record_digest")
		if !ok {
			continue
		}
		if _, exists := bySession[sessionID]; !exists {
			bySession[sessionID] = map[string]struct{}{}
		}
		bySession[sessionID][recordDigest] = struct{}{}
	}
	return bySession, nil
}

func detailString(details map[string]interface{}, key string) (string, bool) {
	raw, ok := details[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func buildSessionTranscriptTurns(sessionID string, summary SessionSummary, runs, approvals, artifactsByDigest, auditRecordDigests map[string]struct{}) []SessionTranscriptTurn {
	if summary.TurnCount <= 0 {
		return []SessionTranscriptTurn{}
	}
	linkedRuns := boundedStrings(sortedStringKeys(runs), 256)
	linkedApprovals := boundedStrings(sortedStringKeys(approvals), 512)
	linkedArtifacts := boundedStrings(sortedStringKeys(artifactsByDigest), 1024)
	linkedAudit := boundedStrings(sortedStringKeys(auditRecordDigests), 1024)
	turnLimit := summary.TurnCount
	if turnLimit > 2048 {
		turnLimit = 2048
	}
	turns := make([]SessionTranscriptTurn, 0, turnLimit)
	for i := 0; i < turnLimit; i++ {
		turnIndex := i + 1
		turnID := fmt.Sprintf("%s.turn.%06d", sessionID, turnIndex)
		messageID := fmt.Sprintf("%s.msg.%06d", turnID, 1)
		turn := SessionTranscriptTurn{
			SchemaID:      "runecode.protocol.v0.SessionTranscriptTurn",
			SchemaVersion: "0.1.0",
			TurnID:        turnID,
			SessionID:     sessionID,
			TurnIndex:     turnIndex,
			StartedAt:     summary.UpdatedAt,
			Status:        "completed",
			Messages: []SessionTranscriptMessage{{
				SchemaID:      "runecode.protocol.v0.SessionTranscriptMessage",
				SchemaVersion: "0.1.0",
				MessageID:     messageID,
				TurnID:        turnID,
				SessionID:     sessionID,
				MessageIndex:  1,
				Role:          "system",
				CreatedAt:     summary.UpdatedAt,
				ContentText:   summary.LastActivityPreview,
				RelatedLinks: SessionTranscriptLinks{
					SchemaID:           "runecode.protocol.v0.SessionTranscriptLinks",
					SchemaVersion:      "0.1.0",
					RunIDs:             linkedRuns,
					ApprovalIDs:        linkedApprovals,
					ArtifactDigests:    linkedArtifacts,
					AuditRecordDigests: linkedAudit,
				},
			}},
		}
		turn.CompletedAt = turn.StartedAt
		turns = append(turns, turn)
	}
	return turns
}

func buildSessionSummary(sessionID string, records []artifacts.ArtifactRecord, linkedApprovalCount, linkedAuditEventCount int) SessionSummary {
	created := time.Unix(0, 0).UTC()
	updated := created
	if len(records) > 0 {
		created = records[0].CreatedAt.UTC()
		updated = records[0].CreatedAt.UTC()
	}
	runs := map[string]struct{}{}
	artifactsByDigest := map[string]struct{}{}
	for _, rec := range records {
		if rec.CreatedAt.Before(created) {
			created = rec.CreatedAt.UTC()
		}
		if rec.CreatedAt.After(updated) {
			updated = rec.CreatedAt.UTC()
		}
		if rec.RunID != "" {
			runs[rec.RunID] = struct{}{}
		}
		if rec.Reference.Digest != "" {
			artifactsByDigest[rec.Reference.Digest] = struct{}{}
		}
	}
	preview := ""
	if len(records) > 0 {
		preview = "session linked to run activity"
	}
	turnCount := len(runs)
	if turnCount == 0 && len(records) > 0 {
		turnCount = 1
	}
	return SessionSummary{
		SchemaID:              "runecode.protocol.v0.SessionSummary",
		SchemaVersion:         "0.1.0",
		Identity:              SessionIdentity{SchemaID: "runecode.protocol.v0.SessionIdentity", SchemaVersion: "0.1.0", SessionID: sessionID, WorkspaceID: "workspace-local", CreatedAt: created.Format(time.RFC3339), CreatedByRunID: firstSortedStringKey(runs)},
		UpdatedAt:             updated.Format(time.RFC3339),
		Status:                "active",
		LastActivityAt:        updated.Format(time.RFC3339),
		LastActivityKind:      "run_progress",
		LastActivityPreview:   preview,
		TurnCount:             turnCount,
		LinkedRunCount:        len(runs),
		LinkedApprovalCount:   linkedApprovalCount,
		LinkedArtifactCount:   len(artifactsByDigest),
		LinkedAuditEventCount: linkedAuditEventCount,
		HasIncompleteTurn:     false,
	}
}

func sortedStringKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func firstSortedStringKey(values map[string]struct{}) string {
	keys := sortedStringKeys(values)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func boundedStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}
