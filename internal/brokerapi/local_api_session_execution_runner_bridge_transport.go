package brokerapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type stdioRunnerTransportRequest struct {
	MessageType string          `json:"message_type"`
	Payload     json.RawMessage `json:"payload"`
}

type stdioRunnerTransportResponse struct {
	MessageType string `json:"message_type"`
	Payload     any    `json:"payload"`
}

func (s *Service) proxyRunnerTransport(ctx context.Context, requestID, runID string, stdin io.WriteCloser, stdout io.Reader) error {
	defer stdin.Close()
	decoder := bufio.NewScanner(stdout)
	buffer := make([]byte, 0, 64*1024)
	decoder.Buffer(buffer, s.apiConfig.Limits.MaxMessageBytes)
	encoder := json.NewEncoder(stdin)
	messageIndex := 0
	for decoder.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		messageIndex++
		response, err := s.handleRunnerTransportLine(ctx, requestID, runID, decoder.Bytes(), messageIndex)
		if err != nil {
			return err
		}
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("write typed broker response: %w", err)
		}
	}
	if err := decoder.Err(); err != nil {
		return fmt.Errorf("read runner transport message: %w", err)
	}
	return nil
}

func (s *Service) handleRunnerTransportLine(ctx context.Context, requestID, runID string, line []byte, messageIndex int) (stdioRunnerTransportResponse, error) {
	message := stdioRunnerTransportRequest{}
	if err := json.Unmarshal(line, &message); err != nil {
		return stdioRunnerTransportResponse{}, fmt.Errorf("parse runner stdio message %d: %w", messageIndex, err)
	}
	switch strings.TrimSpace(message.MessageType) {
	case "dependency_cache_handoff_request":
		return s.handleDependencyCacheHandoffTransport(ctx, requestID, runID, message.Payload, messageIndex)
	case "runner_checkpoint_report_request":
		return s.handleRunnerCheckpointTransport(ctx, requestID, runID, message.Payload, messageIndex)
	case "runner_result_report_request":
		return s.handleRunnerResultTransport(ctx, requestID, runID, message.Payload, messageIndex)
	default:
		return stdioRunnerTransportResponse{}, fmt.Errorf("unsupported runner transport message_type %q", strings.TrimSpace(message.MessageType))
	}
}

func (s *Service) handleDependencyCacheHandoffTransport(ctx context.Context, requestID, runID string, payload json.RawMessage, messageIndex int) (stdioRunnerTransportResponse, error) {
	var req DependencyCacheHandoffRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return stdioRunnerTransportResponse{}, fmt.Errorf("decode dependency cache handoff request: %w", err)
	}
	resp, errResp := s.HandleDependencyCacheHandoff(ctx, req, RequestContext{RequestID: requestIDForRunnerTransport(requestID, runID, "dependency_cache_handoff", messageIndex)})
	if errResp != nil {
		return stdioRunnerTransportResponse{}, fmt.Errorf("dependency cache handoff rejected: %s", strings.TrimSpace(errResp.Error.Message))
	}
	return stdioRunnerTransportResponse{MessageType: "dependency_cache_handoff_response", Payload: resp}, nil
}

func (s *Service) handleRunnerCheckpointTransport(ctx context.Context, requestID, runID string, payload json.RawMessage, messageIndex int) (stdioRunnerTransportResponse, error) {
	var req RunnerCheckpointReportRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return stdioRunnerTransportResponse{}, fmt.Errorf("decode runner checkpoint request: %w", err)
	}
	resp, errResp := s.HandleRunnerCheckpointReport(ctx, req, RequestContext{RequestID: requestIDForRunnerTransport(requestID, runID, "checkpoint", messageIndex)})
	if errResp != nil {
		return stdioRunnerTransportResponse{}, fmt.Errorf("runner checkpoint report rejected: %s", strings.TrimSpace(errResp.Error.Message))
	}
	return stdioRunnerTransportResponse{MessageType: "runner_checkpoint_report_response", Payload: resp}, nil
}

func (s *Service) handleRunnerResultTransport(ctx context.Context, requestID, runID string, payload json.RawMessage, messageIndex int) (stdioRunnerTransportResponse, error) {
	var req RunnerResultReportRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return stdioRunnerTransportResponse{}, fmt.Errorf("decode runner result request: %w", err)
	}
	resp, errResp := s.HandleRunnerResultReport(ctx, req, RequestContext{RequestID: requestIDForRunnerTransport(requestID, runID, "result", messageIndex)})
	if errResp != nil {
		return stdioRunnerTransportResponse{}, fmt.Errorf("runner result report rejected: %s", strings.TrimSpace(errResp.Error.Message))
	}
	return stdioRunnerTransportResponse{MessageType: "runner_result_report_response", Payload: resp}, nil
}
