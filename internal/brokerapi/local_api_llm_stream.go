package brokerapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

type llmStreamExecutionState struct {
	Translated     map[string]any              `json:"translated"`
	ProviderID     string                      `json:"provider_id"`
	ProviderFamily string                      `json:"provider_family"`
	AdapterKind    string                      `json:"adapter_kind"`
	Endpoint       string                      `json:"endpoint"`
	LeaseID        string                      `json:"lease_id"`
	InputRef       artifacts.ArtifactReference `json:"input_ref"`
}

func (s *Service) StreamLLMEvents(req LLMStreamRequest, binding llmExecutionBinding, inputRef artifacts.ArtifactReference) (LLMStreamEnvelope, error) {
	defer finalizeLLMStreamRequest(req)
	streamState, err := decodeLLMStreamExecutionState(inputRef)
	if err != nil {
		return LLMStreamEnvelope{}, err
	}
	defer s.revokeLLMLease(req.RunID, streamState.LeaseID)

	decision, execCtx, err := s.prepareLLMStreamExecution(req, binding, streamState)
	if err != nil {
		return LLMStreamEnvelope{}, err
	}
	text, outboundBytes, startedAt, completedAt, err := s.runLLMStreamExecution(req, execCtx, streamState)
	if err != nil {
		return LLMStreamEnvelope{}, err
	}
	binding.StartedAt = startedAt
	binding.CompletedAt = completedAt
	binding.OutboundBytes = outboundBytes

	return s.finalizeLLMStreamExecution(req, binding, decision, text)
}

func decodeLLMStreamExecutionState(inputRef artifacts.ArtifactReference) (llmStreamExecutionState, error) {
	streamState := llmStreamExecutionState{}
	if err := json.Unmarshal([]byte(inputRef.Digest), &streamState); err != nil {
		return llmStreamExecutionState{}, err
	}
	return streamState, nil
}

func (s *Service) prepareLLMStreamExecution(req LLMStreamRequest, binding llmExecutionBinding, streamState llmStreamExecutionState) (policyengine.PolicyDecision, llmExecutionContext, error) {
	decision, err := s.admitLLMStreamExecution(req, binding)
	if err != nil {
		return policyengine.PolicyDecision{}, llmExecutionContext{}, err
	}
	execCtx, err := s.llmExecutionContextFromStreamState(req, binding, streamState)
	if err != nil {
		return policyengine.PolicyDecision{}, llmExecutionContext{}, err
	}
	return decision, execCtx, nil
}

func (s *Service) admitLLMStreamExecution(req LLMStreamRequest, binding llmExecutionBinding) (policyengine.PolicyDecision, error) {
	admissionBinding := binding
	admissionBinding.StartedAt = time.Now().UTC()
	admissionBinding.CompletedAt = admissionBinding.StartedAt.Add(time.Millisecond)
	admissionBinding.OutboundBytes = 1
	decision, errResp := s.evaluateModelGatewayInvokeAdmission(req.RequestID, req.RunID, admissionBinding)
	if errResp != nil {
		return policyengine.PolicyDecision{}, errors.New(errResp.Error.Message)
	}
	return decision, nil
}

func (s *Service) llmExecutionContextFromStreamState(req LLMStreamRequest, binding llmExecutionBinding, streamState llmStreamExecutionState) (llmExecutionContext, error) {
	profile, ok := s.providerProfileByID(streamState.ProviderID)
	if !ok {
		return llmExecutionContext{}, fmt.Errorf("provider profile unavailable for llm stream execution")
	}
	adapter, err := adapterForProviderProfile(profile)
	if err != nil {
		return llmExecutionContext{}, err
	}
	endpoint, err := url.Parse(streamState.Endpoint)
	if err != nil {
		return llmExecutionContext{}, err
	}
	return llmExecutionContext{Adapter: adapter, Endpoint: endpoint, ProviderID: streamState.ProviderID, ProviderFamily: streamState.ProviderFamily, AdapterKind: streamState.AdapterKind, LeaseID: streamState.LeaseID, RequestID: req.RequestID, RunID: req.RunID, RequestHash: binding.RequestHash, RequestDigest: binding.RequestDigest}, nil
}

func (s *Service) runLLMStreamExecution(req LLMStreamRequest, execCtx llmExecutionContext, streamState llmStreamExecutionState) (string, int64, time.Time, time.Time, error) {
	ctx := req.RequestCtx
	if ctx == nil {
		ctx = context.Background()
	}
	text, outboundBytes, startedAt, completedAt, errResp := s.executeProviderRequest(ctx, execCtx, streamState.Translated)
	if errResp != nil {
		return "", 0, time.Time{}, time.Time{}, errors.New(errResp.Error.Message)
	}
	return text, outboundBytes, startedAt, completedAt, nil
}

func (s *Service) finalizeLLMStreamExecution(req LLMStreamRequest, binding llmExecutionBinding, decision policyengine.PolicyDecision, text string) (LLMStreamEnvelope, error) {
	outputRef, errResp := s.storeLLMOutputArtifact(req.RequestID, req.RunID, text)
	if errResp != nil {
		return LLMStreamEnvelope{}, errors.New(errResp.Error.Message)
	}
	_, responseHash, errResp := s.buildCanonicalLLMResponseFromOutput(req.RequestID, binding.RequestHash, outputRef)
	if errResp != nil {
		return LLMStreamEnvelope{}, errors.New(errResp.Error.Message)
	}
	binding.ResponseHash = responseHash
	if err := s.emitModelGatewayTerminalAudit(req.RunID, decision, llmOutcomeSucceeded, binding); err != nil {
		return LLMStreamEnvelope{}, err
	}
	events := llmStreamEventsFromText(req.StreamID, req.RunID, binding, text)
	if err := validateLLMStreamEvents(events); err != nil {
		return LLMStreamEnvelope{}, err
	}
	envelope := llmStreamEnvelope(req.RequestID, req.RunID, binding.RequestDigest, events)
	if err := s.validateResponse(envelope, llmStreamEnvelopeSchemaPath); err != nil {
		return LLMStreamEnvelope{}, err
	}
	return envelope, nil
}

func llmStreamEventsFromText(streamID, runID string, binding llmExecutionBinding, text string) []LLMStreamAny {
	responseHash := digestObjectMap(binding.ResponseHash)
	requestHash := digestObjectMap(binding.RequestHash)
	emitter := llmStreamEmitter(runID)
	events := []LLMStreamAny{{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": streamID, "request_hash": requestHash, "seq": 1, "emitter": emitter, "event_type": "response_start"}}
	if text != "" {
		events = append(events, LLMStreamAny{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": streamID, "request_hash": requestHash, "seq": 2, "emitter": emitter, "event_type": "output_delta", "content_delta": text})
		events = append(events, LLMStreamAny{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": streamID, "request_hash": requestHash, "seq": 3, "emitter": emitter, "event_type": "response_terminal", "terminal_status": "success", "final_response_hash": responseHash})
		return events
	}
	events = append(events, LLMStreamAny{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": streamID, "request_hash": requestHash, "seq": 2, "emitter": emitter, "event_type": "response_terminal", "terminal_status": "success", "final_response_hash": responseHash})
	return events
}

func validateLLMStreamEvents(events []LLMStreamAny) error {
	if err := validateLLMStreamEventSchemas(events); err != nil {
		return err
	}
	if len(events) == 0 {
		return fmt.Errorf("llm stream must emit at least one event")
	}
	streamID, err := requireLLMStreamID(events[0])
	if err != nil {
		return err
	}
	terminalCount, err := validateLLMStreamEventSequence(events, streamID)
	if err != nil {
		return err
	}
	if terminalCount != 1 {
		return fmt.Errorf("llm stream must include exactly one response_terminal event")
	}
	if stringField(events[len(events)-1], "event_type") != "response_terminal" {
		return fmt.Errorf("llm stream response_terminal must be final event")
	}
	return nil
}

func finalizeLLMStreamRequest(req LLMStreamRequest) {
	if req.Release != nil {
		req.Release()
	}
	if req.Cancel != nil {
		req.Cancel()
	}
}
