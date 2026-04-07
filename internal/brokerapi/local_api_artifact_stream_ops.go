package brokerapi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

func (s *Service) StreamArtifactReadEvents(handle ArtifactReadHandle) ([]ArtifactStreamEvent, error) {
	defer finalizeArtifactReadHandle(handle)
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
		if err := artifactReadContextErr(handle.RequestCtx); err != nil {
			events = append(events, artifactStreamTerminalFromContextErr(handle, seq, err))
			_ = handle.Reader.Close()
			break
		}
		n, readErr := handle.Reader.Read(buffer)
		if n > 0 {
			total += n
			if total > s.apiConfig.Limits.MaxResponseStreamBytes {
				events = append(events, artifactStreamTerminalLimitExceeded(handle, seq))
				_ = handle.Reader.Close()
				break
			}
			chunk := append([]byte(nil), buffer[:n]...)
			events = append(events, artifactStreamChunkEvent(handle, seq, chunk))
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
			Code:          "broker_limit_response_stream_size_exceeded",
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
			Message:       "artifact stream read failed",
		},
	}
}

func artifactReadContextErr(requestCtx context.Context) error {
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

func artifactStreamTerminalFromContextErr(handle ArtifactReadHandle, seq int64, ctxErr error) ArtifactStreamEvent {
	terminal := ArtifactStreamEvent{
		SchemaID:      "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion: "0.1.0",
		StreamID:      handle.StreamID,
		RequestID:     handle.RequestID,
		Seq:           seq,
		EventType:     "artifact_stream_terminal",
		Digest:        handle.Digest,
		DataClass:     string(handle.DataClass),
		Terminal:      true,
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "request_cancelled",
			Category:      "transport",
			Retryable:     true,
			Message:       "artifact stream cancelled",
		},
	}
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		terminal.TerminalStatus = "failed"
		terminal.Error.Code = "broker_timeout_request_deadline_exceeded"
		terminal.Error.Category = "timeout"
		terminal.Error.Message = "artifact stream deadline exceeded"
		return terminal
	}
	terminal.TerminalStatus = "cancelled"
	terminal.Error = nil
	return terminal
}

func finalizeArtifactReadHandle(handle ArtifactReadHandle) {
	if handle.Release != nil {
		handle.Release()
	}
	if handle.Cancel != nil {
		handle.Cancel()
	}
}
