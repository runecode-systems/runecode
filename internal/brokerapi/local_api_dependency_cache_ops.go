package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) HandleDependencyCacheEnsure(ctx context.Context, req DependencyCacheEnsureRequest, meta RequestContext) (DependencyCacheEnsureResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, dependencyCacheEnsureRequestSchemaPath)
	if errResp != nil {
		return DependencyCacheEnsureResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return DependencyCacheEnsureResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return DependencyCacheEnsureResponse{}, errResp
	}
	resp, err := s.dependencyFetchService.EnsureBatch(requestCtx, requestID, req)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return DependencyCacheEnsureResponse{}, &errOut
	}
	if err := s.validateResponse(resp, dependencyCacheEnsureResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return DependencyCacheEnsureResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleDependencyFetchRegistry(ctx context.Context, req DependencyFetchRegistryRequest, meta RequestContext) (DependencyFetchRegistryResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, dependencyFetchRegistryRequestSchemaPath)
	if errResp != nil {
		return DependencyFetchRegistryResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return DependencyFetchRegistryResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return DependencyFetchRegistryResponse{}, errResp
	}
	resp, err := s.dependencyFetchService.FetchSingle(requestCtx, requestID, req)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return DependencyFetchRegistryResponse{}, &errOut
	}
	if err := s.validateResponse(resp, dependencyFetchRegistryResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return DependencyFetchRegistryResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleDependencyCacheHandoff(ctx context.Context, req DependencyCacheHandoffRequest, meta RequestContext) (DependencyCacheHandoffResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, dependencyCacheHandoffRequestSchemaPath)
	if errResp != nil {
		return DependencyCacheHandoffResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return DependencyCacheHandoffResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return DependencyCacheHandoffResponse{}, errResp
	}
	consumerRole, errResp := s.normalizeDependencyCacheHandoffConsumerRole(requestID, req.ConsumerRole)
	if errResp != nil {
		return DependencyCacheHandoffResponse{}, errResp
	}
	handoff, ok, err := s.DependencyCacheHandoffByRequest(artifacts.DependencyCacheHandoffRequest{
		RequestDigest: mustDigestIdentity(req.RequestDigest),
		ConsumerRole:  consumerRole,
	})
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return DependencyCacheHandoffResponse{}, &errOut
	}
	resp, payloadDigests, err := buildDependencyCacheHandoffResponse(s, requestID, handoff, ok)
	if err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return DependencyCacheHandoffResponse{}, &errOut
	}
	if err := s.validateResponse(resp, dependencyCacheHandoffResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return DependencyCacheHandoffResponse{}, &errOut
	}
	appendDependencyCacheHandoffAudit(s, requestID, req, consumerRole, handoff, ok, payloadDigests)
	return resp, nil
}

func (s *Service) normalizeDependencyCacheHandoffConsumerRole(requestID, consumerRole string) (string, *ErrorResponse) {
	switch strings.TrimSpace(consumerRole) {
	case "workspace", "workspace-read", "workspace-edit", "workspace-test":
		return "workspace", nil
	default:
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "consumer_role must be one of workspace, workspace-read, workspace-edit, or workspace-test")
		return "", &errOut
	}
}

func buildDependencyCacheHandoffResponse(s *Service, requestID string, handoff artifacts.DependencyCacheHandoff, found bool) (DependencyCacheHandoffResponse, []string, error) {
	resp := DependencyCacheHandoffResponse{
		SchemaID:      "runecode.protocol.v0.DependencyCacheHandoffResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Found:         found,
	}
	if !found {
		return resp, nil, nil
	}
	payloadDigests := append([]string{}, handoff.PayloadDigests...)
	resp.Handoff = &DependencyCacheHandoffMetadata{
		SchemaID:            "runecode.protocol.v0.DependencyCacheHandoffMetadata",
		SchemaVersion:       "0.1.0",
		RequestDigest:       mustDigestObjectFromIdentity(handoff.RequestDigest),
		ResolvedUnitDigest:  mustDigestObjectFromIdentity(handoff.ResolvedUnitDigest),
		ManifestDigest:      mustDigestObjectFromIdentity(handoff.ManifestDigest),
		PayloadDigests:      mapDigestIdentities(payloadDigests),
		MaterializationMode: handoff.MaterializationMode,
		HandoffMode:         handoff.HandoffMode,
	}
	if err := s.validateResponse(*resp.Handoff, dependencyCacheHandoffMetadataSchemaPath); err != nil {
		return DependencyCacheHandoffResponse{}, nil, err
	}
	return resp, payloadDigests, nil
}

func appendDependencyCacheHandoffAudit(s *Service, requestID string, req DependencyCacheHandoffRequest, effectiveConsumerRole string, handoff artifacts.DependencyCacheHandoff, found bool, payloadDigests []string) {
	details := map[string]any{
		"request_id":              requestID,
		"gateway_role_kind":       "dependency-fetch",
		"operation":               "dependency_cache_handoff",
		"consumer_role":           effectiveConsumerRole,
		"requested_consumer_role": strings.TrimSpace(req.ConsumerRole),
		"request_payload_bound":   true,
	}
	if found {
		details["audit_outcome"] = "succeeded"
		details["request_digest"] = handoff.RequestDigest
		details["resolved_unit_digest"] = handoff.ResolvedUnitDigest
		details["manifest_digest"] = handoff.ManifestDigest
		details["payload_digests"] = payloadDigests
		details["materialization_mode"] = handoff.MaterializationMode
		details["handoff_mode"] = handoff.HandoffMode
	} else {
		details["audit_outcome"] = "not_found"
		details["request_digest"] = mustDigestIdentity(req.RequestDigest)
	}
	_ = s.AppendTrustedAuditEvent("dependency_cache_handoff", "brokerapi", details)
}
