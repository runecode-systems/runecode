package brokerapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) HandleArtifactListV0(ctx context.Context, req LocalArtifactListRequest, meta RequestContext) (LocalArtifactListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactListRequestSchemaPath)
	if errResp != nil {
		return LocalArtifactListResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return LocalArtifactListResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return LocalArtifactListResponse{}, errResp
	}
	order := artifactListOrder(req.Order)
	summaries := filterArtifactSummaries(s.List(), req)
	sortArtifactSummariesNewestFirst(summaries)
	limit := normalizeLimit(req.Limit, 100, 500)
	page, next, err := paginate(summaries, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return LocalArtifactListResponse{}, &errOut
	}
	resp := LocalArtifactListResponse{SchemaID: "runecode.protocol.v0.ArtifactListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Artifacts: page, NextCursor: next}
	if err := s.validateResponse(resp, artifactListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return LocalArtifactListResponse{}, &errOut
	}
	return resp, nil
}

func artifactListOrder(order string) string {
	if order == "" {
		return "created_at_desc"
	}
	return order
}

func filterArtifactSummaries(all []artifacts.ArtifactRecord, req LocalArtifactListRequest) []ArtifactSummary {
	summaries := make([]ArtifactSummary, 0, len(all))
	for _, record := range all {
		summary := toArtifactSummary(record)
		if req.RunID != "" && summary.RunID != req.RunID {
			continue
		}
		if req.StepID != "" && summary.StepID != req.StepID {
			continue
		}
		if req.DataClass != "" && string(summary.Reference.DataClass) != req.DataClass {
			continue
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func (s *Service) HandleArtifactHeadV0(ctx context.Context, req LocalArtifactHeadRequest, meta RequestContext) (LocalArtifactHeadResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactHeadRequestSchemaPath)
	if errResp != nil {
		return LocalArtifactHeadResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return LocalArtifactHeadResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return LocalArtifactHeadResponse{}, errResp
	}
	record, err := s.Head(req.Digest)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return LocalArtifactHeadResponse{}, &errOut
	}
	resp := LocalArtifactHeadResponse{SchemaID: "runecode.protocol.v0.ArtifactHeadResponse", SchemaVersion: "0.1.0", RequestID: requestID, Artifact: toArtifactSummary(record)}
	if err := s.validateResponse(resp, artifactHeadResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return LocalArtifactHeadResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleArtifactRead(ctx context.Context, req ArtifactReadRequest, meta RequestContext) (ArtifactReadHandle, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactReadRequestSchemaPath)
	if errResp != nil {
		return ArtifactReadHandle{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ArtifactReadHandle{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return ArtifactReadHandle{}, errResp
	}
	if strings.TrimSpace(req.ProducerRole) == "" || strings.TrimSpace(req.ConsumerRole) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "producer_role and consumer_role are required")
		return ArtifactReadHandle{}, &errOut
	}
	if req.RangeStart != nil || req.RangeEnd != nil {
		errOut := s.makeError(requestID, "broker_validation_range_not_supported", "validation", false, "range_start/range_end are not supported for MVP artifact reads")
		return ArtifactReadHandle{}, &errOut
	}
	if req.StreamID == "" {
		req.StreamID = "artifact-read-" + requestID
	}
	if req.ChunkBytes <= 0 || req.ChunkBytes > s.apiConfig.Limits.MaxStreamChunkBytes {
		req.ChunkBytes = s.apiConfig.Limits.MaxStreamChunkBytes
	}
	class := artifacts.DataClass(req.DataClass)
	r, record, err := s.GetForFlow(artifacts.ArtifactReadRequest{
		Digest:        req.Digest,
		ProducerRole:  req.ProducerRole,
		ConsumerRole:  req.ConsumerRole,
		DataClass:     class,
		IsEgress:      true,
		ManifestOptIn: req.ManifestOptIn,
	})
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return ArtifactReadHandle{}, &errOut
	}
	return ArtifactReadHandle{RequestID: requestID, Digest: req.Digest, DataClass: record.Reference.DataClass, StreamID: req.StreamID, ChunkBytes: req.ChunkBytes, Reader: r}, nil
}

func (s *Service) StreamArtifactReadEvents(handle ArtifactReadHandle) ([]ArtifactStreamEvent, error) {
	if handle.Reader == nil {
		return nil, fmt.Errorf("artifact read handle reader is required")
	}
	if handle.StreamID == "" {
		return nil, fmt.Errorf("artifact read handle stream_id is required")
	}
	chunkSize := handle.ChunkBytes
	if chunkSize <= 0 || chunkSize > s.apiConfig.Limits.MaxStreamChunkBytes {
		chunkSize = s.apiConfig.Limits.MaxStreamChunkBytes
	}
	events, err := s.collectArtifactReadEvents(handle, chunkSize)
	if err != nil {
		return nil, err
	}
	if err := validateArtifactStreamSemantics(events); err != nil {
		return nil, err
	}
	for i := range events {
		if err := s.validateResponse(events[i], artifactStreamEventSchemaPath); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (s *Service) collectArtifactReadEvents(handle ArtifactReadHandle, chunkSize int) ([]ArtifactStreamEvent, error) {
	buffer := make([]byte, chunkSize)
	events := []ArtifactStreamEvent{artifactStreamStartEvent(handle, 1)}
	seq := int64(2)
	total := 0
	for {
		n, readErr := handle.Reader.Read(buffer)
		if n > 0 {
			total += n
			if total > s.apiConfig.Limits.MaxResponseStreamBytes {
				events = append(events, artifactStreamTerminalLimitExceeded(handle, seq))
				_ = handle.Reader.Close()
				break
			}
			events = append(events, artifactStreamChunkEvent(handle, seq, buffer[:n]))
			seq++
		}
		if readErr == nil {
			continue
		}
		events = append(events, artifactStreamTerminalFromReadErr(handle, seq, readErr))
		_ = handle.Reader.Close()
		break
	}
	return events, nil
}

func artifactStreamStartEvent(handle ArtifactReadHandle, seq int64) ArtifactStreamEvent {
	return ArtifactStreamEvent{
		SchemaID:      "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion: "0.1.0",
		StreamID:      handle.StreamID,
		RequestID:     handle.RequestID,
		Seq:           seq,
		EventType:     "artifact_stream_start",
		Digest:        handle.Digest,
		DataClass:     string(handle.DataClass),
	}
}

func artifactStreamChunkEvent(handle ArtifactReadHandle, seq int64, chunk []byte) ArtifactStreamEvent {
	return ArtifactStreamEvent{
		SchemaID:      "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion: "0.1.0",
		StreamID:      handle.StreamID,
		RequestID:     handle.RequestID,
		Seq:           seq,
		EventType:     "artifact_stream_chunk",
		Digest:        handle.Digest,
		DataClass:     string(handle.DataClass),
		ChunkBase64:   base64.StdEncoding.EncodeToString(chunk),
		ChunkBytes:    len(chunk),
	}
}

func artifactStreamTerminalLimitExceeded(handle ArtifactReadHandle, seq int64) ArtifactStreamEvent {
	return ArtifactStreamEvent{
		SchemaID:       "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       handle.StreamID,
		RequestID:      handle.RequestID,
		Seq:            seq,
		EventType:      "artifact_stream_terminal",
		Digest:         handle.Digest,
		DataClass:      string(handle.DataClass),
		Terminal:       true,
		TerminalStatus: "failed",
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "broker_limit_message_size_exceeded",
			Category:      "transport",
			Retryable:     false,
			Message:       "artifact stream exceeded broker max response stream bytes",
		},
	}
}

func artifactStreamTerminalFromReadErr(handle ArtifactReadHandle, seq int64, readErr error) ArtifactStreamEvent {
	if readErr == io.EOF {
		return ArtifactStreamEvent{
			SchemaID:       "runecode.protocol.v0.ArtifactStreamEvent",
			SchemaVersion:  "0.1.0",
			StreamID:       handle.StreamID,
			RequestID:      handle.RequestID,
			Seq:            seq,
			EventType:      "artifact_stream_terminal",
			Digest:         handle.Digest,
			DataClass:      string(handle.DataClass),
			EOF:            true,
			Terminal:       true,
			TerminalStatus: "completed",
		}
	}
	return ArtifactStreamEvent{
		SchemaID:       "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       handle.StreamID,
		RequestID:      handle.RequestID,
		Seq:            seq,
		EventType:      "artifact_stream_terminal",
		Digest:         handle.Digest,
		DataClass:      string(handle.DataClass),
		Terminal:       true,
		TerminalStatus: "failed",
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "gateway_failure",
			Category:      "internal",
			Retryable:     false,
			Message:       readErr.Error(),
		},
	}
}
