package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const llmLeaseIDUnavailableSentinel = "lease-unavailable-until-authoritative-wiring"

func (s *Service) HandleLLMInvoke(ctx context.Context, req LLMInvokeRequest, meta RequestContext) (LLMInvokeResponse, *ErrorResponse) {
	requestID, _, release, cancel, errResp := s.prepareLLMRequestContext(ctx, req.RequestID, meta, req, llmInvokeRequestSchemaPath)
	if errResp != nil {
		return LLMInvokeResponse{}, errResp
	}
	defer release()
	defer cancel()
	return LLMInvokeResponse{}, s.llmExecutionUnavailable(requestID)
}

func (s *Service) HandleLLMStreamRequest(ctx context.Context, req LLMStreamRequest, meta RequestContext) (LLMStreamRequest, llmExecutionBinding, artifacts.ArtifactReference, *ErrorResponse) {
	if strings.TrimSpace(req.StreamID) == "" {
		req.StreamID = "llm-stream-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, _, release, cancel, errResp := s.prepareLLMRequestContext(ctx, req.RequestID, meta, req, llmStreamRequestSchemaPath)
	if errResp != nil {
		return LLMStreamRequest{}, llmExecutionBinding{}, artifacts.ArtifactReference{}, errResp
	}
	defer release()
	defer cancel()
	return LLMStreamRequest{}, llmExecutionBinding{}, artifacts.ArtifactReference{}, s.llmExecutionUnavailable(requestID)
}

func (s *Service) StreamLLMEvents(req LLMStreamRequest, binding llmExecutionBinding, inputRef artifacts.ArtifactReference) (LLMStreamEnvelope, error) {
	_ = req
	_ = binding
	_ = inputRef
	return LLMStreamEnvelope{}, fmt.Errorf(llmExecutionUnavailableMessage)
}

func (s *Service) bindLLMRequestToArtifacts(requestID, runID string, expectedDigest *trustpolicy.Digest, llmReq any) (llmExecutionBinding, artifacts.ArtifactReference, *ErrorResponse) {
	runID, reqDigest, errResp := s.resolveLLMRequestArtifact(requestID, runID, expectedDigest, llmReq)
	if errResp != nil {
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, errResp
	}
	inputRefs, decodeErr := decodeLLMInputArtifactRefs(llmReq)
	if decodeErr != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, decodeErr.Error())
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, &errOut
	}
	if len(inputRefs) == 0 {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "llm_request input_artifacts must be non-empty")
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, &errOut
	}
	primaryInputRecord, errResp := s.ensureInputArtifactsExist(requestID, runID, inputRefs)
	if errResp != nil {
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, errResp
	}
	binding := llmExecutionBinding{
		RequestDigest: reqDigest,
		RequestHash:   reqDigest,
		LeaseID:       llmLeaseIDUnavailableSentinel,
	}
	return binding, primaryInputRecord.Reference, nil
}

func (s *Service) ensureInputArtifactsExist(requestID, runID string, refs []artifacts.ArtifactReference) (artifacts.ArtifactRecord, *ErrorResponse) {
	var primary artifacts.ArtifactRecord
	for _, ref := range refs {
		record, err := s.Head(trimArtifactRefDigest(ref))
		if err != nil {
			if errors.Is(err, artifacts.ErrArtifactNotFound) {
				errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "llm_request input_artifact digest must exist")
				return artifacts.ArtifactRecord{}, &errOut
			}
			errOut := s.errorFromStore(requestID, err)
			return artifacts.ArtifactRecord{}, &errOut
		}
		if strings.TrimSpace(record.RunID) != runID {
			errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "llm_request input_artifact run binding mismatch")
			return artifacts.ArtifactRecord{}, &errOut
		}
		if primary.Reference.Digest == "" {
			primary = record
		}
	}
	return primary, nil
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
