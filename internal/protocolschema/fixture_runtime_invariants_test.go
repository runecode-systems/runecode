package protocolschema

import "fmt"

func requireSessionSendMessageAckAlignment(value map[string]any) error {
	sessionID, err := stringField(value, "session_id")
	if err != nil {
		return err
	}
	if err := requireSessionAckEventType(value); err != nil {
		return err
	}
	if err := requireSessionAckStreamID(value, sessionID); err != nil {
		return err
	}
	message, turn, err := sessionAckMessageAndTurn(value)
	if err != nil {
		return err
	}
	if err := requireSessionAckObjectAlignment(message, turn, sessionID); err != nil {
		return err
	}
	return requireSessionAckSeq(value)
}

func requireSessionAckEventType(value map[string]any) error {
	eventType, err := stringField(value, "event_type")
	if err != nil {
		return err
	}
	if eventType != "session_message_ack" {
		return fmt.Errorf("event_type must be session_message_ack")
	}
	return nil
}

func requireSessionAckStreamID(value map[string]any, sessionID string) error {
	streamID, err := stringField(value, "stream_id")
	if err != nil {
		return err
	}
	if streamID != "session-"+sessionID {
		return fmt.Errorf("stream_id %q must equal session-%s", streamID, sessionID)
	}
	return nil
}

func sessionAckMessageAndTurn(value map[string]any) (map[string]any, map[string]any, error) {
	message, err := requiredObjectField(value, "message")
	if err != nil {
		return nil, nil, err
	}
	turn, err := requiredObjectField(value, "turn")
	if err != nil {
		return nil, nil, err
	}
	return message, turn, nil
}

func requireSessionAckObjectAlignment(message, turn map[string]any, sessionID string) error {
	messageSessionID, err := stringField(message, "session_id")
	if err != nil {
		return err
	}
	if messageSessionID != sessionID {
		return fmt.Errorf("message.session_id %q must match session_id %q", messageSessionID, sessionID)
	}
	turnSessionID, err := stringField(turn, "session_id")
	if err != nil {
		return err
	}
	if turnSessionID != sessionID {
		return fmt.Errorf("turn.session_id %q must match session_id %q", turnSessionID, sessionID)
	}
	turnID, err := stringField(turn, "turn_id")
	if err != nil {
		return err
	}
	messageTurnID, err := stringField(message, "turn_id")
	if err != nil {
		return err
	}
	if messageTurnID != turnID {
		return fmt.Errorf("message.turn_id %q must match turn.turn_id %q", messageTurnID, turnID)
	}
	return nil
}

func requireSessionAckSeq(value map[string]any) error {
	seq, err := integerField(value, "seq")
	if err != nil {
		return err
	}
	if seq < 1 {
		return fmt.Errorf("seq must be >= 1")
	}
	return nil
}

func validateStreamSequence(events []any) error {
	if len(events) == 0 {
		return fmt.Errorf("stream sequence must contain at least one event")
	}
	parsedEvents, err := parseStreamEvents(events)
	if err != nil {
		return err
	}
	return validateParsedStreamEvents(parsedEvents)
}

type streamEventView struct {
	streamID    string
	requestHash string
	eventType   string
	seq         int64
}

func parseStreamEvents(events []any) ([]streamEventView, error) {
	parsed := make([]streamEventView, 0, len(events))
	for index, item := range events {
		event, err := objectFromFixtureValue(item, fmt.Sprintf("events[%d]", index))
		if err != nil {
			return nil, err
		}
		parsedEvent, err := parseStreamEvent(event)
		if err != nil {
			return nil, fmt.Errorf("events[%d]: %w", index, err)
		}
		parsed = append(parsed, parsedEvent)
	}
	return parsed, nil
}

func parseStreamEvent(event map[string]any) (streamEventView, error) {
	streamID, err := stringField(event, "stream_id")
	if err != nil {
		return streamEventView{}, err
	}
	requestHash, err := digestIdentityField(event, "request_hash")
	if err != nil {
		return streamEventView{}, err
	}
	eventType, err := stringField(event, "event_type")
	if err != nil {
		return streamEventView{}, err
	}
	seq, err := integerField(event, "seq")
	if err != nil {
		return streamEventView{}, err
	}
	return streamEventView{streamID: streamID, requestHash: requestHash, eventType: eventType, seq: seq}, nil
}

func validateParsedStreamEvents(events []streamEventView) error {
	if err := requireStreamStartsAtSeqOne(events[0]); err != nil {
		return fmt.Errorf("first stream event: %w", err)
	}
	if err := requireFinalStreamEventTerminal(events[len(events)-1]); err != nil {
		return err
	}
	streamID := events[0].streamID
	requestHash := events[0].requestHash
	lastSeq := int64(0)
	for index, event := range events {
		if err := requireMatchingStreamIdentity(event, streamID, requestHash); err != nil {
			return err
		}
		if err := requireStrictlyMonotonicSeq(event.seq, lastSeq); err != nil {
			return err
		}
		if index < len(events)-1 && event.eventType == "response_terminal" {
			return fmt.Errorf("response_terminal must be the final event in the stream")
		}
		lastSeq = event.seq
	}
	return nil
}

func requireStreamStartsAtSeqOne(first streamEventView) error {
	if first.seq != 1 {
		return fmt.Errorf("first stream event must use seq=1")
	}
	return nil
}

func requireFinalStreamEventTerminal(last streamEventView) error {
	if last.eventType != "response_terminal" {
		return fmt.Errorf("stream must contain exactly one terminal event")
	}
	return nil
}

func requireMatchingStreamIdentity(event streamEventView, streamID string, requestHash string) error {
	if event.streamID != streamID {
		return fmt.Errorf("stream_id must remain constant across a stream")
	}
	if event.requestHash != requestHash {
		return fmt.Errorf("request_hash must remain constant across a stream")
	}
	return nil
}

func requireStrictlyMonotonicSeq(seq int64, lastSeq int64) error {
	if seq <= lastSeq {
		return fmt.Errorf("stream sequence numbers must be strictly monotonic")
	}
	return nil
}
