package brokerapi

import (
	"context"
	"strings"
)

func (s *Service) HandleProviderSetupSessionBegin(ctx context.Context, req ProviderSetupSessionBeginRequest, meta RequestContext) (ProviderSetupSessionBeginResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderSetupRequest(ctx, req, req.RequestID, meta, providerSetupSessionBeginRequestSchemaPath)
	if errResp != nil {
		return ProviderSetupSessionBeginResponse{}, errResp
	}
	defer cleanup()
	_, existed := s.providerProfileByID(stableProviderProfileID(strings.TrimSpace(strings.ToLower(req.ProviderFamily)), destinationRefFromHostAndPath(req.CanonicalHost, req.CanonicalPathPrefix)))
	profile, err := s.providerSubstrate.upsertProfile(providerProfileFromSetupBegin(req))
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return ProviderSetupSessionBeginResponse{}, &errOut
	}
	session, err := s.providerSetup.begin(profile)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSessionBeginResponse{}, &errOut
	}
	resp := ProviderSetupSessionBeginResponse{
		SchemaID:      "runecode.protocol.v0.ProviderSetupSessionBeginResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		SetupSession:  session,
		Profile:       profile.projected(),
	}
	if err := s.validateResponse(resp, providerSetupSessionBeginResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderSetupSessionBeginResponse{}, &errOut
	}
	changeKind := "created"
	if existed {
		changeKind = "updated"
	}
	if err := s.auditProviderProfileChange(requestID, profile, changeKind); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSessionBeginResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleProviderSetupSecretIngressPrepare(ctx context.Context, req ProviderSetupSecretIngressPrepareRequest, meta RequestContext) (ProviderSetupSecretIngressPrepareResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderSetupRequest(ctx, req, req.RequestID, meta, providerSetupSecretIngressPrepareRequestSchemaPath)
	if errResp != nil {
		return ProviderSetupSecretIngressPrepareResponse{}, errResp
	}
	defer cleanup()
	session, ingress, err := s.providerSetup.prepareIngress(req.SetupSessionID, req.IngressChannel, req.CredentialField, 0)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return ProviderSetupSecretIngressPrepareResponse{}, &errOut
	}
	resp := ProviderSetupSecretIngressPrepareResponse{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSecretIngressPrepareResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          requestID,
		SetupSession:       session,
		SecretIngressToken: ingress.Token,
		ExpiresAt:          ingress.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if err := s.validateResponse(resp, providerSetupSecretIngressPrepareResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderSetupSecretIngressPrepareResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleProviderProfileList(ctx context.Context, req ProviderProfileListRequest, meta RequestContext) (ProviderProfileListResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderReadRequest(ctx, req, req.RequestID, meta, providerProfileListRequestSchemaPath)
	if errResp != nil {
		return ProviderProfileListResponse{}, errResp
	}
	defer cleanup()
	profiles := s.providerSubstrate.snapshotProfiles()
	for i := range profiles {
		profiles[i] = profiles[i].projected()
	}
	resp := ProviderProfileListResponse{
		SchemaID:      "runecode.protocol.v0.ProviderProfileListResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Profiles:      profiles,
	}
	if err := s.validateResponse(resp, providerProfileListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderProfileListResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleProviderProfileGet(ctx context.Context, req ProviderProfileGetRequest, meta RequestContext) (ProviderProfileGetResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderReadRequest(ctx, req, req.RequestID, meta, providerProfileGetRequestSchemaPath)
	if errResp != nil {
		return ProviderProfileGetResponse{}, errResp
	}
	defer cleanup()
	profileID := strings.TrimSpace(req.ProviderProfileID)
	if profileID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider_profile_id is required")
		return ProviderProfileGetResponse{}, &errOut
	}
	profile, ok := s.providerProfileByID(profileID)
	if !ok {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider profile not found")
		return ProviderProfileGetResponse{}, &errOut
	}
	resp := ProviderProfileGetResponse{
		SchemaID:      "runecode.protocol.v0.ProviderProfileGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Profile:       profile.projected(),
	}
	if err := s.validateResponse(resp, providerProfileGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderProfileGetResponse{}, &errOut
	}
	return resp, nil
}
