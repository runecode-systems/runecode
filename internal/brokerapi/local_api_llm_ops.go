package brokerapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const llmHTTPTimeout = 30 * time.Second
const llmLeaseIDUnavailableSentinel = "lease-unavailable-until-authoritative-wiring"

type llmHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

var newLLMHTTPClient = func() llmHTTPClient {
	return &http.Client{Timeout: llmHTTPTimeout}
}

type llmExecutionContext struct {
	Adapter        llmProviderAdapter
	Endpoint       *url.URL
	ProviderID     string
	ProviderFamily string
	AdapterKind    string
	LeaseID        string
	RequestID      string
	RunID          string
	RequestHash    trustpolicy.Digest
	RequestDigest  trustpolicy.Digest
}

func (s *Service) HandleLLMInvoke(ctx context.Context, req LLMInvokeRequest, meta RequestContext) (LLMInvokeResponse, *ErrorResponse) {
	requestID, requestCtx, release, cancel, errResp := s.prepareLLMRequestContext(ctx, req.RequestID, meta, req, llmInvokeRequestSchemaPath)
	if errResp != nil {
		return LLMInvokeResponse{}, errResp
	}
	defer release()
	defer cancel()

	binding, inputRef, translated, execCtx, errResp := s.prepareLLMExecution(requestID, req.RunID, req.RequestDigest, req.LLMRequest, false)
	if errResp != nil {
		return LLMInvokeResponse{}, errResp
	}
	defer s.revokeLLMLease(execCtx.RunID, execCtx.LeaseID)

	decision, errResp := s.evaluateLLMInvokeAdmission(requestID, req.RunID, binding)
	if errResp != nil {
		return LLMInvokeResponse{}, errResp
	}
	binding, text, errResp := s.performLLMInvokeExecution(requestCtx, binding, execCtx, translated)
	if errResp != nil {
		return LLMInvokeResponse{}, errResp
	}
	resp, errResp := s.finalizeLLMInvoke(requestID, req.RunID, binding, decision, text)
	if errResp != nil {
		return LLMInvokeResponse{}, errResp
	}
	_ = inputRef
	return resp, nil
}

func (s *Service) HandleLLMStreamRequest(ctx context.Context, req LLMStreamRequest, meta RequestContext) (LLMStreamRequest, llmExecutionBinding, artifacts.ArtifactReference, *ErrorResponse) {
	if strings.TrimSpace(req.StreamID) == "" {
		req.StreamID = "llm-stream-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, requestCtx, release, cancel, errResp := s.prepareLLMRequestContext(ctx, req.RequestID, meta, req, llmStreamRequestSchemaPath)
	if errResp != nil {
		return LLMStreamRequest{}, llmExecutionBinding{}, artifacts.ArtifactReference{}, errResp
	}
	binding, inputRef, translated, execCtx, errResp := s.prepareLLMExecution(requestID, req.RunID, req.RequestDigest, req.LLMRequest, true)
	if errResp != nil {
		release()
		cancel()
		return LLMStreamRequest{}, llmExecutionBinding{}, artifacts.ArtifactReference{}, errResp
	}
	req.RequestID = requestID
	req.RequestCtx = requestCtx
	req.Release = release
	req.Cancel = cancel
	state := llmStreamExecutionState{Translated: translated, ProviderID: execCtx.ProviderID, ProviderFamily: execCtx.ProviderFamily, AdapterKind: execCtx.AdapterKind, Endpoint: execCtx.Endpoint.String(), LeaseID: execCtx.LeaseID, InputRef: inputRef}
	b, err := json.Marshal(state)
	if err != nil {
		s.revokeLLMLease(req.RunID, execCtx.LeaseID)
		release()
		cancel()
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return LLMStreamRequest{}, llmExecutionBinding{}, artifacts.ArtifactReference{}, &errOut
	}
	return req, binding, artifacts.ArtifactReference{Digest: string(b)}, nil
}

func (s *Service) evaluateLLMInvokeAdmission(requestID, runID string, binding llmExecutionBinding) (policyengine.PolicyDecision, *ErrorResponse) {
	admissionBinding := binding
	admissionBinding.StartedAt = time.Now().UTC()
	admissionBinding.CompletedAt = admissionBinding.StartedAt.Add(time.Millisecond)
	admissionBinding.OutboundBytes = 1
	decision, errResp := s.evaluateModelGatewayInvokeAdmission(requestID, runID, admissionBinding)
	if errResp != nil {
		return policyengine.PolicyDecision{}, errResp
	}
	return decision, nil
}

func (s *Service) performLLMInvokeExecution(ctx context.Context, binding llmExecutionBinding, execCtx llmExecutionContext, translated map[string]any) (llmExecutionBinding, string, *ErrorResponse) {
	text, outboundBytes, startedAt, completedAt, errResp := s.executeProviderRequest(ctx, execCtx, translated)
	if errResp != nil {
		return llmExecutionBinding{}, "", errResp
	}
	binding.StartedAt = startedAt
	binding.CompletedAt = completedAt
	binding.OutboundBytes = outboundBytes
	return binding, text, nil
}

func (s *Service) finalizeLLMInvoke(requestID, runID string, binding llmExecutionBinding, decision policyengine.PolicyDecision, text string) (LLMInvokeResponse, *ErrorResponse) {
	outputRef, errResp := s.storeLLMOutputArtifact(requestID, runID, text)
	if errResp != nil {
		return LLMInvokeResponse{}, errResp
	}
	responseObj, responseHash, errResp := s.buildCanonicalLLMResponseFromOutput(requestID, binding.RequestHash, outputRef)
	if errResp != nil {
		return LLMInvokeResponse{}, errResp
	}
	binding.ResponseHash = responseHash
	if err := s.emitModelGatewayTerminalAudit(runID, decision, llmOutcomeSucceeded, binding); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return LLMInvokeResponse{}, &errOut
	}
	resp := llmInvokeResponse(requestID, runID, binding.RequestDigest, responseObj)
	if err := s.validateResponse(resp, llmInvokeResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return LLMInvokeResponse{}, &errOut
	}
	return resp, nil
}
