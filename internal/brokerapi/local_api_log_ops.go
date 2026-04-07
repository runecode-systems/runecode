package brokerapi

import (
	"context"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleLogStreamRequest(ctx context.Context, req LogStreamRequest, meta RequestContext) (LogStreamRequest, *ErrorResponse) {
	if req.StreamID == "" {
		req.StreamID = "log-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, logStreamRequestSchemaPath)
	if errResp != nil {
		return LogStreamRequest{}, errResp
	}
	ack := req
	ack.RequestID = requestID
	if ack.StreamID == "" {
		ack.StreamID = "log-" + requestID
	}
	return ack, nil
}

func (s *Service) StreamLogEvents(req LogStreamRequest) ([]LogStreamEvent, error) {
	events := make([]LogStreamEvent, 0, 8)
	events = append(events, logStreamStartEvent(req, 1))
	seq := int64(2)
	for _, record := range s.logRecordsForRequest(req) {
		events = append(events, logStreamChunkEvent(req, seq, record))
		seq++
	}
	events = append(events, logStreamTerminalEvent(req, seq))

	if err := validateLogStreamSemantics(events); err != nil {
		return nil, err
	}
	for i := range events {
		if err := s.validateResponse(events[i], logStreamEventSchemaPath); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func logStreamStartEvent(req LogStreamRequest, seq int64) LogStreamEvent {
	return LogStreamEvent{
		SchemaID:       "runecode.protocol.v0.LogStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "log_stream_start",
		RunID:          req.RunID,
		RoleInstanceID: req.RoleInstanceID,
	}
}

func logStreamChunkEvent(req LogStreamRequest, seq int64, record logStreamRecord) LogStreamEvent {
	return LogStreamEvent{
		SchemaID:       "runecode.protocol.v0.LogStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "log_stream_chunk",
		RunID:          record.RunID,
		RoleInstanceID: record.RoleInstanceID,
		Cursor:         record.Cursor,
		Timestamp:      record.Timestamp,
		Level:          record.Level,
		Message:        record.Message,
	}
}

func logStreamTerminalEvent(req LogStreamRequest, seq int64) LogStreamEvent {
	terminal := LogStreamEvent{
		SchemaID:       "runecode.protocol.v0.LogStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "log_stream_terminal",
		RunID:          req.RunID,
		RoleInstanceID: req.RoleInstanceID,
		Terminal:       true,
		TerminalStatus: "completed",
	}
	if req.StartCursor == "force_failure" {
		terminal.TerminalStatus = "failed"
		terminal.Error = &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "gateway_failure",
			Category:      "internal",
			Retryable:     false,
			Message:       "forced log stream failure",
		}
	}
	return terminal
}

func (s *Service) logRecordsForRequest(req LogStreamRequest) []logStreamRecord {
	now := time.Now().UTC()
	filtered := filterLogSeedRecords(logSeedRecords(now), req)
	filtered = appendFollowLogRecord(filtered, req, now)
	return maybeTrimLogBacklog(filtered, req.IncludeBacklog)
}

func logSeedRecords(now time.Time) []logStreamRecord {
	return []logStreamRecord{
		{RunID: "run-123", RoleInstanceID: "workspace-1", Cursor: "cursor-1", Timestamp: now.Add(-2 * time.Second).Format(time.RFC3339Nano), Level: "info", Message: "workflow started"},
		{RunID: "run-123", RoleInstanceID: "workspace-1", Cursor: "cursor-2", Timestamp: now.Add(-1 * time.Second).Format(time.RFC3339Nano), Level: "info", Message: "artifact prepared"},
		{RunID: "run-xyz", RoleInstanceID: "gateway-1", Cursor: "cursor-3", Timestamp: now.Format(time.RFC3339Nano), Level: "warn", Message: "waiting for approval"},
	}
}

func filterLogSeedRecords(seed []logStreamRecord, req LogStreamRequest) []logStreamRecord {
	filtered := make([]logStreamRecord, 0, len(seed)+1)
	for _, rec := range seed {
		if req.RunID != "" && rec.RunID != req.RunID {
			continue
		}
		if req.RoleInstanceID != "" && rec.RoleInstanceID != req.RoleInstanceID {
			continue
		}
		if req.StartCursor != "" && rec.Cursor <= req.StartCursor {
			continue
		}
		filtered = append(filtered, rec)
	}
	return filtered
}

func appendFollowLogRecord(records []logStreamRecord, req LogStreamRequest, now time.Time) []logStreamRecord {
	if !req.Follow {
		return records
	}
	return append(records, logStreamRecord{
		RunID:          coalesce(req.RunID, "run-123"),
		RoleInstanceID: coalesce(req.RoleInstanceID, "workspace-1"),
		Cursor:         "cursor-live",
		Timestamp:      now.Add(1 * time.Second).Format(time.RFC3339Nano),
		Level:          "info",
		Message:        "live log tail event",
	})
}

func maybeTrimLogBacklog(records []logStreamRecord, includeBacklog bool) []logStreamRecord {
	if includeBacklog {
		return records
	}
	if len(records) == 0 {
		return nil
	}
	return []logStreamRecord{records[len(records)-1]}
}

func coalesce(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func (s *Service) HandleAuditTimeline(ctx context.Context, req AuditTimelineRequest, meta RequestContext) (AuditTimelineResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditTimelineRequestSchemaPath)
	if errResp != nil {
		return AuditTimelineResponse{}, errResp
	}
	order := req.Order
	if order == "" {
		order = "operational_seq_asc"
	}
	surface, err := s.LatestAuditVerificationSurface(1000)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return AuditTimelineResponse{}, &errOut
	}
	views := append([]trustpolicy.AuditOperationalView{}, surface.Views...)
	if order == "operational_seq_desc" {
		reverseViews(views)
	}
	limit := normalizeLimit(req.Limit, 100, 500)
	page, next, err := paginate(views, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditTimelineResponse{}, &errOut
	}
	resp := AuditTimelineResponse{SchemaID: "runecode.protocol.v0.AuditTimelineResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Views: page, NextCursor: next}
	if err := s.validateResponse(resp, auditTimelineResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditTimelineResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleAuditVerificationGet(ctx context.Context, req AuditVerificationGetRequest, meta RequestContext) (AuditVerificationGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditVerificationGetRequestSchemaPath)
	if errResp != nil {
		return AuditVerificationGetResponse{}, errResp
	}
	limit := req.ViewLimit
	if limit <= 0 {
		limit = 50
	}
	surface, err := s.LatestAuditVerificationSurface(limit)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return AuditVerificationGetResponse{}, &errOut
	}
	resp := AuditVerificationGetResponse{SchemaID: "runecode.protocol.v0.AuditVerificationGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Summary: surface.Summary, Report: surface.Report, Views: surface.Views}
	if err := s.validateResponse(resp, auditVerificationGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditVerificationGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleReadinessGet(ctx context.Context, req ReadinessGetRequest, meta RequestContext) (ReadinessGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, readinessGetRequestSchemaPath)
	if errResp != nil {
		return ReadinessGetResponse{}, errResp
	}
	readiness, err := s.AuditReadiness()
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ReadinessGetResponse{}, &errOut
	}
	model := BrokerReadiness{
		SchemaID:                  "runecode.protocol.v0.BrokerReadiness",
		SchemaVersion:             "0.1.0",
		Ready:                     readiness.Ready,
		LocalOnly:                 readiness.LocalOnly,
		ConsumptionChannel:        readiness.ConsumptionChannel,
		RecoveryComplete:          readiness.RecoveryComplete,
		AppendPositionStable:      readiness.AppendPositionStable,
		CurrentSegmentWritable:    readiness.CurrentSegmentWritable,
		VerifierMaterialAvailable: readiness.VerifierMaterialAvailable,
		DerivedIndexCaughtUp:      readiness.DerivedIndexCaughtUp,
	}
	resp := ReadinessGetResponse{SchemaID: "runecode.protocol.v0.ReadinessGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Readiness: model}
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ReadinessGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleVersionInfoGet(ctx context.Context, req VersionInfoGetRequest, meta RequestContext) (VersionInfoGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, versionInfoGetRequestSchemaPath)
	if errResp != nil {
		return VersionInfoGetResponse{}, errResp
	}
	resp := VersionInfoGetResponse{SchemaID: "runecode.protocol.v0.VersionInfoGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, VersionInfo: s.versionInfo}
	if err := s.validateResponse(resp, versionInfoGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return VersionInfoGetResponse{}, &errOut
	}
	return resp, nil
}

func reverseViews(views []trustpolicy.AuditOperationalView) {
	for i, j := 0, len(views)-1; i < j; i, j = i+1, j-1 {
		views[i], views[j] = views[j], views[i]
	}
}
