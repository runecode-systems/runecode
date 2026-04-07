package brokerapi

import (
	"context"
	"errors"
	"strings"
	"time"
)

func (s *Service) HandleLogStreamRequest(ctx context.Context, req LogStreamRequest, meta RequestContext) (LogStreamRequest, *ErrorResponse) {
	if req.StreamID == "" {
		req.StreamID = "log-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, logStreamRequestSchemaPath)
	if errResp != nil {
		return LogStreamRequest{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return LogStreamRequest{}, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		release()
		cancel()
		return LogStreamRequest{}, errResp
	}
	ack := req
	ack.RequestID = requestID
	if ack.StreamID == "" {
		ack.StreamID = "log-" + requestID
	}
	ack.RequestCtx = requestCtx
	ack.Cancel = cancel
	ack.Release = release
	return ack, nil
}

func (s *Service) StreamLogEvents(req LogStreamRequest) ([]LogStreamEvent, error) {
	defer finalizeLogStreamRequest(req)
	events := make([]LogStreamEvent, 0, 8)
	events = append(events, logStreamStartEvent(req, 1))
	seq := int64(2)
	for _, record := range s.logRecordsForRequest(req) {
		if err := reqContextErr(req.RequestCtx); err != nil {
			events = append(events, logStreamTerminalFromContextErr(req, seq, err))
			seq++
			break
		}
		events = append(events, logStreamChunkEvent(req, seq, record))
		seq++
	}
	if len(events) == 0 || events[len(events)-1].EventType != "log_stream_terminal" {
		events = append(events, logStreamTerminalEvent(req, seq))
	}

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

func reqContextErr(requestCtx context.Context) error {
	if requestCtx == nil {
		return nil
	}
	select {
	case <-requestCtx.Done():
		return requestCtx.Err()
	default:
		return nil
	}
}

func logStreamTerminalFromContextErr(req LogStreamRequest, seq int64, ctxErr error) LogStreamEvent {
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
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "request_cancelled",
			Category:      "transport",
			Retryable:     true,
			Message:       "log stream cancelled",
		},
	}
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		terminal.TerminalStatus = "failed"
		terminal.Error.Code = "broker_timeout_request_deadline_exceeded"
		terminal.Error.Category = "timeout"
		terminal.Error.Message = "log stream deadline exceeded"
		return terminal
	}
	terminal.TerminalStatus = "cancelled"
	terminal.Error = nil
	return terminal
}

func finalizeLogStreamRequest(req LogStreamRequest) {
	if req.Release != nil {
		req.Release()
	}
	if req.Cancel != nil {
		req.Cancel()
	}
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
