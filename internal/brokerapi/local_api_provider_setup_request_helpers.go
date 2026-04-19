package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/secretsd"
)

func (s *Service) HandleProviderCredentialLeaseIssue(ctx context.Context, req ProviderCredentialLeaseIssueRequest, meta RequestContext) (ProviderCredentialLeaseIssueResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderSetupRequest(ctx, req, req.RequestID, meta, providerCredentialLeaseIssueRequestSchemaPath)
	if errResp != nil {
		return ProviderCredentialLeaseIssueResponse{}, errResp
	}
	defer cleanup()
	profile, runID, errResp := s.providerCredentialLeaseRequestProfile(requestID, req)
	if errResp != nil {
		return ProviderCredentialLeaseIssueResponse{}, errResp
	}
	lease, err := s.secretsSvc.IssueLease(secretsd.IssueLeaseRequest{
		SecretRef:    profile.AuthMaterial.SecretRef,
		ConsumerID:   "principal:gateway:model:" + runID,
		RoleKind:     "model-gateway",
		Scope:        "run:" + runID,
		DeliveryKind: "model_gateway",
		TTLSeconds:   req.TTLSeconds,
	})
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderCredentialLeaseIssueResponse{}, &errOut
	}
	resp := ProviderCredentialLeaseIssueResponse{SchemaID: "runecode.protocol.v0.ProviderCredentialLeaseIssueResponse", SchemaVersion: "0.1.0", RequestID: requestID, ProviderProfileID: profile.ProviderProfileID, ProviderAuthLeaseID: lease.LeaseID, Lease: lease}
	if err := s.validateResponse(resp, providerCredentialLeaseIssueResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderCredentialLeaseIssueResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) providerCredentialLeaseRequestProfile(requestID string, req ProviderCredentialLeaseIssueRequest) (ProviderProfile, string, *ErrorResponse) {
	profileID := strings.TrimSpace(req.ProviderProfileID)
	if profileID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider_profile_id is required")
		return ProviderProfile{}, "", &errOut
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "run_id is required")
		return ProviderProfile{}, "", &errOut
	}
	profile, ok := s.providerProfileByID(profileID)
	if !ok || strings.TrimSpace(profile.AuthMaterial.SecretRef) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider_profile_id does not have direct credential material")
		return ProviderProfile{}, "", &errOut
	}
	return profile, runID, nil
}

func (s *Service) beginProviderSetupRequest(ctx context.Context, req any, requestID string, meta RequestContext, schemaPath string) (string, func(), *ErrorResponse) {
	return s.beginProviderRequest(ctx, req, requestID, meta, schemaPath, true)
}

func (s *Service) beginProviderReadRequest(ctx context.Context, req any, requestID string, meta RequestContext, schemaPath string) (string, func(), *ErrorResponse) {
	return s.beginProviderRequest(ctx, req, requestID, meta, schemaPath, false)
}

func (s *Service) beginProviderRequest(ctx context.Context, req any, requestID string, meta RequestContext, schemaPath string, requireSecrets bool) (string, func(), *ErrorResponse) {
	if s == nil || s.providerSetup == nil || s.providerSubstrate == nil || (requireSecrets && s.secretsSvc == nil) {
		errOut := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, "provider setup unavailable")
		return "", nil, &errOut
	}
	resolvedRequestID, errResp := s.prepareLocalRequest(requestID, meta.RequestID, meta.AdmissionErr, req, schemaPath)
	if errResp != nil {
		return "", nil, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(resolvedRequestID, err)
		return "", nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	cleanup := func() {
		cancel()
		release()
	}
	if err := requestCtx.Err(); err != nil {
		cleanup()
		errOut := s.errorFromContext(resolvedRequestID, err)
		return "", nil, &errOut
	}
	return resolvedRequestID, cleanup, nil
}
