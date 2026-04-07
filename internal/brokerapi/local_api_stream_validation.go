package brokerapi

import (
	"fmt"
	"strings"
)

func validateArtifactStreamSemantics(events []ArtifactStreamEvent) error {
	return validateStreamSemantics(
		events,
		"artifact",
		"artifact stream",
		"artifact_stream_terminal",
	)
}

func validateLogStreamSemantics(events []LogStreamEvent) error {
	return validateStreamSemantics(
		events,
		"log",
		"log stream",
		"log_stream_terminal",
	)
}

func validateStreamSemantics[T streamEvent](events []T, kind, label, terminalType string) error {
	streamID, requestID, err := validateStreamHeader(events, kind, label)
	if err != nil {
		return err
	}
	terminalCount, err := validateStreamEventSequence(events, kind, terminalType, streamID, requestID)
	if err != nil {
		return err
	}
	return validateTerminalPlacement(events, kind, label, terminalType, terminalCount)
}

func validateStreamHeader[T streamEvent](events []T, kind, label string) (string, string, error) {
	if len(events) == 0 {
		return "", "", fmt.Errorf("%s must emit at least one event", label)
	}
	streamID := events[0].GetStreamID()
	requestID := events[0].GetRequestID()
	if strings.TrimSpace(streamID) == "" {
		return "", "", fmt.Errorf("%s stream_id is required", kind)
	}
	if strings.TrimSpace(requestID) == "" {
		return "", "", fmt.Errorf("%s request_id is required", kind)
	}
	return streamID, requestID, nil
}

func validateStreamEventSequence[T streamEvent](events []T, kind, terminalType, streamID, requestID string) (int, error) {
	terminalCount := 0
	for i, event := range events {
		if err := validateStableStreamEventIDs(kind, event.GetStreamID(), streamID, event.GetRequestID(), requestID); err != nil {
			return 0, err
		}
		if err := validateStrictlyMonotonicSeq(kind, events, i); err != nil {
			return 0, err
		}
		if event.GetEventType() != terminalType {
			continue
		}
		terminalCount++
		if err := validateTerminalEvent(kind, event.IsTerminal(), event.GetTerminalStatus(), event.GetError() != nil); err != nil {
			return 0, err
		}
	}
	return terminalCount, nil
}

func validateTerminalPlacement[T streamEvent](events []T, kind, label, terminalType string, terminalCount int) error {
	if terminalCount != 1 {
		return fmt.Errorf("%s must include exactly one terminal event", label)
	}
	if events[len(events)-1].GetEventType() != terminalType {
		return fmt.Errorf("%s terminal event must be last event", kind)
	}
	return nil
}

type streamEvent interface {
	GetSeq() int64
	GetStreamID() string
	GetRequestID() string
	GetEventType() string
	IsTerminal() bool
	GetTerminalStatus() string
	GetError() *ProtocolError
}

func (e ArtifactStreamEvent) GetSeq() int64             { return e.Seq }
func (e ArtifactStreamEvent) GetStreamID() string       { return e.StreamID }
func (e ArtifactStreamEvent) GetRequestID() string      { return e.RequestID }
func (e ArtifactStreamEvent) GetEventType() string      { return e.EventType }
func (e ArtifactStreamEvent) IsTerminal() bool          { return e.Terminal }
func (e ArtifactStreamEvent) GetTerminalStatus() string { return e.TerminalStatus }
func (e ArtifactStreamEvent) GetError() *ProtocolError  { return e.Error }

func (e LogStreamEvent) GetSeq() int64             { return e.Seq }
func (e LogStreamEvent) GetStreamID() string       { return e.StreamID }
func (e LogStreamEvent) GetRequestID() string      { return e.RequestID }
func (e LogStreamEvent) GetEventType() string      { return e.EventType }
func (e LogStreamEvent) IsTerminal() bool          { return e.Terminal }
func (e LogStreamEvent) GetTerminalStatus() string { return e.TerminalStatus }
func (e LogStreamEvent) GetError() *ProtocolError  { return e.Error }

func validateStableStreamEventIDs(kind, streamID, expectedStreamID, requestID, expectedRequestID string) error {
	if streamID != expectedStreamID {
		return fmt.Errorf("%s stream_id must remain stable", kind)
	}
	if requestID != expectedRequestID {
		return fmt.Errorf("%s request_id must remain stable", kind)
	}
	return nil
}

func validateStrictlyMonotonicSeq[T interface{ GetSeq() int64 }](kind string, events []T, index int) error {
	if index == 0 {
		if events[0].GetSeq() < 1 {
			return fmt.Errorf("%s seq must start at >=1", kind)
		}
		return nil
	}
	if events[index].GetSeq() <= events[index-1].GetSeq() {
		return fmt.Errorf("%s seq must be strictly monotonic", kind)
	}
	return nil
}

func validateTerminalEvent(kind string, terminal bool, terminalStatus string, hasError bool) error {
	if !terminal {
		return fmt.Errorf("%s terminal event must set terminal=true", kind)
	}
	if strings.TrimSpace(terminalStatus) == "" {
		return fmt.Errorf("%s terminal event must set terminal_status", kind)
	}
	if terminalStatus != "completed" && terminalStatus != "failed" && terminalStatus != "cancelled" {
		return fmt.Errorf("%s terminal_status %q unsupported", kind, terminalStatus)
	}
	if terminalStatus == "failed" && !hasError {
		return fmt.Errorf("failed %s terminal event must include error envelope", kind)
	}
	if terminalStatus == "completed" && hasError {
		return fmt.Errorf("completed %s terminal event must not include error envelope", kind)
	}
	return nil
}
