package brokerapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const llmOutcomeSucceeded = "succeeded"
const llmExecutionUnavailableMessage = "broker-owned llm execution unavailable until authoritative gateway metering and lease wiring are implemented"

func (s *Service) prepareLLMRequestContext(ctx context.Context, requestID string, meta RequestContext, req any, schemaPath string) (string, context.Context, func(), context.CancelFunc, *ErrorResponse) {
	resolvedRequestID, errResp := s.prepareLocalRequest(requestID, meta.RequestID, meta.AdmissionErr, req, schemaPath)
	if errResp != nil {
		return "", nil, nil, nil, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(resolvedRequestID, err)
		return "", nil, nil, nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	if errResp := s.requestContextError(resolvedRequestID, requestCtx); errResp != nil {
		release()
		cancel()
		return "", nil, nil, nil, errResp
	}
	return resolvedRequestID, requestCtx, release, cancel, nil
}

func (s *Service) buildLLMResponseObject(requestID string, binding llmExecutionBinding, inputRef artifacts.ArtifactReference) (llmExecutionBinding, map[string]any, *ErrorResponse) {
	responseObj := llmResponseObject(binding.RequestHash, inputRef)
	if err := validateJSONEnvelope(responseObj, llmResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return llmExecutionBinding{}, nil, &errOut
	}
	responseDigest, err := canonicalDigestForValue(responseObj)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return llmExecutionBinding{}, nil, &errOut
	}
	binding.ResponseHash = responseDigest
	return binding, responseObj, nil
}

func (s *Service) resolveLLMRequestArtifact(requestID, runID string, expectedDigest *trustpolicy.Digest, llmReq any) (string, trustpolicy.Digest, *ErrorResponse) {
	trimmedRunID, errResp := s.validateAndNormalizeLLMRunID(requestID, runID)
	if errResp != nil {
		return "", trustpolicy.Digest{}, errResp
	}
	reqDigest, reqIdentity, errResp := s.resolveLLMRequestDigest(requestID, expectedDigest, llmReq)
	if errResp != nil {
		return "", trustpolicy.Digest{}, errResp
	}
	reqRecord, err := s.Head(reqIdentity)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "llm_request must be artifact-backed by canonical digest")
		return "", trustpolicy.Digest{}, &errOut
	}
	if strings.TrimSpace(reqRecord.RunID) != trimmedRunID {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "llm_request artifact run binding mismatch")
		return "", trustpolicy.Digest{}, &errOut
	}
	return trimmedRunID, reqDigest, nil
}

func (s *Service) validateAndNormalizeLLMRunID(requestID, runID string) (string, *ErrorResponse) {
	trimmed := strings.TrimSpace(runID)
	if trimmed != "" {
		return trimmed, nil
	}
	errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "run_id is required")
	return "", &errOut
}

func (s *Service) resolveLLMRequestDigest(requestID string, expectedDigest *trustpolicy.Digest, llmReq any) (trustpolicy.Digest, string, *ErrorResponse) {
	if err := validateJSONEnvelope(llmReq, llmRequestSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return trustpolicy.Digest{}, "", &errOut
	}
	reqDigest, err := canonicalDigestForValue(llmReq)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return trustpolicy.Digest{}, "", &errOut
	}
	reqIdentity, err := digestIdentityStrict(reqDigest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return trustpolicy.Digest{}, "", &errOut
	}
	if expectedDigest == nil {
		return reqDigest, reqIdentity, nil
	}
	expectedIdentity, err := expectedDigest.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "request_digest is invalid")
		return trustpolicy.Digest{}, "", &errOut
	}
	if expectedIdentity != reqIdentity {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "request_digest does not match canonical llm_request hash")
		return trustpolicy.Digest{}, "", &errOut
	}
	return reqDigest, reqIdentity, nil
}

func llmInvokeResponse(requestID, runID string, requestDigest trustpolicy.Digest, responseObj any) LLMInvokeResponse {
	return LLMInvokeResponse{
		SchemaID:      "runecode.protocol.v0.LLMInvokeResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		RunID:         runID,
		RequestDigest: requestDigest,
		Response:      responseObj,
	}
}

func llmStreamEnvelope(requestID, runID string, requestDigest trustpolicy.Digest, events []LLMStreamAny) LLMStreamEnvelope {
	return LLMStreamEnvelope{
		SchemaID:      "runecode.protocol.v0.LLMStreamEnvelope",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		RunID:         runID,
		RequestDigest: requestDigest,
		Events:        events,
	}
}

func llmStreamEvents(streamID, runID string, binding llmExecutionBinding) []LLMStreamAny {
	responseHash := digestObjectMap(binding.ResponseHash)
	requestHash := digestObjectMap(binding.RequestHash)
	emitter := llmStreamEmitter(runID)
	return []LLMStreamAny{
		{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": streamID, "request_hash": requestHash, "seq": 1, "emitter": emitter, "event_type": "response_start"},
		{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": streamID, "request_hash": requestHash, "seq": 2, "emitter": emitter, "event_type": "output_delta", "content_delta": "ok"},
		{"schema_id": "runecode.protocol.v0.LLMStreamEvent", "schema_version": "0.3.0", "stream_id": streamID, "request_hash": requestHash, "seq": 3, "emitter": emitter, "event_type": "response_terminal", "terminal_status": "success", "final_response_hash": responseHash},
	}
}

func llmStreamEmitter(runID string) map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
		"schema_version": "0.2.0",
		"actor_kind":     "role_instance",
		"principal_id":   "brokerapi",
		"instance_id":    "brokerapi-1",
		"role_family":    "gateway",
		"role_kind":      "model-gateway",
		"run_id":         runID,
	}
}

func validateLLMStreamEventSchemas(events []LLMStreamAny) error {
	for i := range events {
		if err := validateJSONEnvelope(events[i], llmStreamEventSchemaPath); err != nil {
			return err
		}
	}
	return nil
}

func requireLLMStreamID(event LLMStreamAny) (string, error) {
	streamID := strings.TrimSpace(stringField(event, "stream_id"))
	if streamID == "" {
		return "", fmt.Errorf("llm stream stream_id is required")
	}
	return streamID, nil
}

func validateLLMStreamEventSequence(events []LLMStreamAny, streamID string) (int, error) {
	terminalCount := 0
	for i := range events {
		if stringField(events[i], "stream_id") != streamID {
			return 0, fmt.Errorf("llm stream stream_id must remain stable")
		}
		if i > 0 && intField(events[i], "seq") <= intField(events[i-1], "seq") {
			return 0, fmt.Errorf("llm stream seq must be strictly monotonic")
		}
		if stringField(events[i], "event_type") == "response_terminal" {
			terminalCount++
		}
	}
	return terminalCount, nil
}

func trimArtifactRefDigest(ref artifacts.ArtifactReference) string {
	return strings.TrimSpace(ref.Digest)
}

func canonicalDigestForValue(value any) (trustpolicy.Digest, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("canonical digest marshal failed: %w", err)
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("canonical digest canonicalization failed: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func digestObjectMap(d trustpolicy.Digest) map[string]any {
	return map[string]any{"hash_alg": d.HashAlg, "hash": d.Hash}
}

func digestIdentityStrict(d trustpolicy.Digest) (string, error) {
	id, err := d.Identity()
	if err != nil {
		return "", fmt.Errorf("digest identity invalid: %w", err)
	}
	return id, nil
}

func digestFromIdentityOrNil(identity string) *trustpolicy.Digest {
	if strings.TrimSpace(identity) == "" {
		return nil
	}
	d, err := digestFromIdentity(identity)
	if err != nil {
		return nil
	}
	return &d
}

func digestFromIdentityOrPanic(identity string) trustpolicy.Digest {
	d, err := digestFromIdentity(identity)
	if err != nil {
		panic(err)
	}
	return d
}

func (s *Service) llmExecutionUnavailable(requestID string) *ErrorResponse {
	errOut := s.makeError(requestID, "gateway_failure", "internal", false, llmExecutionUnavailableMessage)
	return &errOut
}

func stringField(value map[string]any, key string) string {
	raw, _ := value[key].(string)
	return raw
}

func intField(value map[string]any, key string) int64 {
	raw := value[key]
	switch typed := raw.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}
