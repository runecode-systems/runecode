package brokerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) prepareLLMExecution(requestID, runID string, expectedDigest *trustpolicy.Digest, llmReq any, streamAsFinal bool) (llmExecutionBinding, artifacts.ArtifactReference, map[string]any, llmExecutionContext, *ErrorResponse) {
	binding, inputRef, errResp := s.bindLLMRequestToArtifacts(requestID, runID, expectedDigest, llmReq)
	if errResp != nil {
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, nil, llmExecutionContext{}, errResp
	}
	profile, errResp := s.requireReadyProviderProfile(requestID, binding.ProviderID)
	if errResp != nil {
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, nil, llmExecutionContext{}, errResp
	}
	translated, errResp := s.translatePreparedLLMRequest(requestID, binding, llmReq, inputRef, streamAsFinal)
	if errResp != nil {
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, nil, llmExecutionContext{}, errResp
	}
	execCtx, destinationRef, errResp := s.buildLLMExecutionContext(requestID, runID, binding, profile)
	if errResp != nil {
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, nil, llmExecutionContext{}, errResp
	}
	binding.LeaseID = execCtx.LeaseID
	binding.DestinationRef = destinationRef
	return binding, inputRef, translated, execCtx, nil
}

func (s *Service) requireReadyProviderProfile(requestID, providerID string) (ProviderProfile, *ErrorResponse) {
	profile, ok := s.providerProfileByID(providerID)
	if !ok {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider profile unavailable for llm request")
		return ProviderProfile{}, &errOut
	}
	if strings.TrimSpace(profile.ReadinessPosture.EffectiveReadiness) != "ready" {
		errOut := s.makeError(requestID, "broker_validation_operation_invalid", "validation", false, "provider profile effective_readiness must be ready")
		return ProviderProfile{}, &errOut
	}
	if strings.TrimSpace(profile.AuthMaterial.MaterialState) != "present" || strings.TrimSpace(profile.AuthMaterial.SecretRef) == "" {
		errOut := s.makeError(requestID, "broker_validation_operation_invalid", "validation", false, "provider profile does not have direct credential material")
		return ProviderProfile{}, &errOut
	}
	return profile, nil
}

func (s *Service) translatePreparedLLMRequest(requestID string, binding llmExecutionBinding, llmReq any, inputRef artifacts.ArtifactReference, streamAsFinal bool) (map[string]any, *ErrorResponse) {
	translated, errResp := s.translateCanonicalLLMRequestForProfile(requestID, binding, llmReq, inputRef)
	if errResp != nil {
		return nil, errResp
	}
	if streamAsFinal {
		translated["stream"] = false
	}
	return translated, nil
}

func (s *Service) buildLLMExecutionContext(requestID, runID string, binding llmExecutionBinding, profile ProviderProfile) (llmExecutionContext, string, *ErrorResponse) {
	adapter, err := adapterForProviderProfile(profile)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return llmExecutionContext{}, "", &errOut
	}
	endpoint, destinationRef, err := hardenedProviderExecutionDestination(profile, adapter)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_operation_invalid", "validation", false, err.Error())
		return llmExecutionContext{}, "", &errOut
	}
	leaseID, err := s.issueProviderExecutionLease(runID, profile)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return llmExecutionContext{}, "", &errOut
	}
	return llmExecutionContext{Adapter: adapter, Endpoint: endpoint, ProviderID: binding.ProviderID, ProviderFamily: binding.ProviderFamily, AdapterKind: binding.AdapterKind, LeaseID: leaseID, RequestID: requestID, RunID: runID, RequestHash: binding.RequestHash, RequestDigest: binding.RequestDigest}, destinationRef, nil
}

func (s *Service) retrieveLLMCredential(runID, leaseID string) (string, *ErrorResponse) {
	material, _, err := s.secretsSvc.Retrieve(secretsd.RetrieveRequest{LeaseID: leaseID, ConsumerID: "principal:gateway:model:" + runID, RoleKind: "model-gateway", Scope: "run:" + runID, DeliveryKind: "model_gateway"})
	if err != nil {
		errOut := s.makeError("invalid_request", "gateway_failure", "internal", false, err.Error())
		return "", &errOut
	}
	credential := strings.TrimSpace(string(material))
	if credential == "" {
		errOut := s.makeError("invalid_request", "gateway_failure", "internal", false, "provider credential material is empty")
		return "", &errOut
	}
	return credential, nil
}

func (s *Service) revokeLLMLease(runID, leaseID string) {
	if strings.TrimSpace(leaseID) == "" || s == nil || s.secretsSvc == nil {
		return
	}
	lease, err := s.secretsSvc.RevokeLease(secretsd.RevokeLeaseRequest{LeaseID: leaseID, ConsumerID: "principal:gateway:model:" + runID, RoleKind: "model-gateway", Scope: "run:" + runID, Reason: "llm_execution_complete"})
	if err != nil {
		return
	}
	_ = lease
}

func hardenedProviderExecutionDestination(profile ProviderProfile, adapter llmProviderAdapter) (*url.URL, string, error) {
	host := strings.TrimSpace(strings.ToLower(profile.DestinationIdentity.CanonicalHost))
	if host == "" {
		return nil, "", fmt.Errorf("provider destination canonical_host is required")
	}
	if !profile.DestinationIdentity.TLSRequired || profile.DestinationIdentity.PrivateRangeBlocking != "enforced" || profile.DestinationIdentity.DNSRebindingProtection != "enforced" {
		return nil, "", fmt.Errorf("provider destination hardening posture invalid")
	}
	base := normalizeDestinationPathPrefix(profile.DestinationIdentity.CanonicalPathPrefix)
	segment := path.Clean(adapter.endpointPath())
	if !strings.HasPrefix(segment, "/") {
		segment = "/" + segment
	}
	full := path.Clean(path.Join(base, segment))
	if !strings.HasPrefix(full, base) {
		return nil, "", fmt.Errorf("adapter endpoint path escapes provider canonical_path_prefix")
	}
	u := &url.URL{Scheme: "https", Host: host, Path: full}
	return u, host + full, nil
}

func (s *Service) executeProviderRequest(ctx context.Context, execCtx llmExecutionContext, translated map[string]any) (string, int64, time.Time, time.Time, *ErrorResponse) {
	body, started, errResp := s.prepareProviderRequestBody(execCtx.RequestID, translated)
	if errResp != nil {
		return "", 0, time.Time{}, time.Time{}, errResp
	}
	req, errResp := s.buildProviderHTTPRequest(ctx, execCtx, body, started)
	if errResp != nil {
		return "", int64(len(body)), started, normalizeExecutionCompletedAt(started, time.Now().UTC()), errResp
	}
	respBody, completed, errResp := s.doProviderHTTPRequest(execCtx, req)
	completed = normalizeExecutionCompletedAt(started, completed)
	if errResp != nil {
		return "", int64(len(body)), started, completed, errResp
	}
	text, errResp := s.parseProviderResponse(execCtx, respBody)
	if errResp != nil {
		return "", int64(len(body)), started, completed, errResp
	}
	return text, int64(len(body)), started, completed, nil
}

func normalizeExecutionCompletedAt(started, completed time.Time) time.Time {
	if completed.After(started) {
		return completed
	}
	return started.Add(time.Millisecond)
}

func (s *Service) prepareProviderRequestBody(requestID string, translated map[string]any) ([]byte, time.Time, *ErrorResponse) {
	body, err := json.Marshal(translated)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return nil, time.Time{}, &errOut
	}
	return body, time.Now().UTC(), nil
}

func (s *Service) buildProviderHTTPRequest(ctx context.Context, execCtx llmExecutionContext, body []byte, started time.Time) (*http.Request, *ErrorResponse) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, execCtx.Endpoint.String(), bytes.NewReader(body))
	if err != nil {
		errOut := s.makeError(execCtx.RequestID, "gateway_failure", "internal", false, err.Error())
		return nil, &errOut
	}
	credential, errResp := s.retrieveLLMCredential(execCtx.RunID, execCtx.LeaseID)
	if errResp != nil {
		return nil, errResp
	}
	headers := map[string]string{"content-type": "application/json", "accept": "application/json"}
	execCtx.Adapter.applyAuthHeaders(headers, credential)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	_ = started
	return req, nil
}

func (s *Service) doProviderHTTPRequest(execCtx llmExecutionContext, req *http.Request) ([]byte, time.Time, *ErrorResponse) {
	resp, err := newLLMHTTPClient().Do(req)
	completed := time.Now().UTC()
	if err != nil {
		errOut := s.makeError(execCtx.RequestID, "gateway_failure", "internal", false, err.Error())
		return nil, completed, &errOut
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, int64(gatewayRuntimeMaxResponseBytes)))
	if err != nil {
		errOut := s.makeError(execCtx.RequestID, "gateway_failure", "internal", false, err.Error())
		return nil, completed, &errOut
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errOut := s.makeError(execCtx.RequestID, "gateway_failure", "internal", false, fmt.Sprintf("provider request failed with status %d", resp.StatusCode))
		return nil, completed, &errOut
	}
	return respBody, completed, nil
}

func (s *Service) parseProviderResponse(execCtx llmExecutionContext, respBody []byte) (string, *ErrorResponse) {
	text, err := execCtx.Adapter.parseFinalResponse(respBody)
	if err != nil {
		errOut := s.makeError(execCtx.RequestID, "gateway_failure", "internal", false, err.Error())
		return "", &errOut
	}
	return text, nil
}
