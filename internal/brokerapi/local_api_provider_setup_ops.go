package brokerapi

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) HandleProviderSetupSessionBegin(ctx context.Context, req ProviderSetupSessionBeginRequest, meta RequestContext) (ProviderSetupSessionBeginResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderSetupRequest(ctx, req, req.RequestID, meta, providerSetupSessionBeginRequestSchemaPath)
	if errResp != nil {
		return ProviderSetupSessionBeginResponse{}, errResp
	}
	defer cleanup()
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

func (s *Service) HandleProviderSetupSecretIngressSubmit(ctx context.Context, req ProviderSetupSecretIngressSubmitRequest, secret []byte, meta RequestContext) (ProviderSetupSecretIngressSubmitResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderSetupRequest(ctx, req, req.RequestID, meta, providerSetupSecretIngressSubmitRequestSchemaPath)
	if errResp != nil {
		return ProviderSetupSecretIngressSubmitResponse{}, errResp
	}
	defer cleanup()
	if len(secret) == 0 {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "secret ingress payload is required")
		return ProviderSetupSecretIngressSubmitResponse{}, &errOut
	}
	session, _, err := s.providerSetup.consumeIngress(req.SecretIngressToken)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return ProviderSetupSecretIngressSubmitResponse{}, &errOut
	}
	updated, err := s.persistDirectCredential(session, secret)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSecretIngressSubmitResponse{}, &errOut
	}
	session, err = s.providerSetup.completeIngress(req.SecretIngressToken)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSecretIngressSubmitResponse{}, &errOut
	}
	resp := ProviderSetupSecretIngressSubmitResponse{SchemaID: "runecode.protocol.v0.ProviderSetupSecretIngressSubmitResponse", SchemaVersion: "0.1.0", RequestID: requestID, SetupSession: session, Profile: updated.projected()}
	if err := s.validateResponse(resp, providerSetupSecretIngressSubmitResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderSetupSecretIngressSubmitResponse{}, &errOut
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

func providerProfileFromSetupBegin(req ProviderSetupSessionBeginRequest) ProviderProfile {
	family := strings.TrimSpace(strings.ToLower(req.ProviderFamily))
	adapterKind := strings.TrimSpace(strings.ToLower(req.AdapterKind))
	if adapterKind == "" {
		adapterKind = defaultAdapterKindForProviderFamily(family)
	}
	host := strings.TrimSpace(strings.ToLower(req.CanonicalHost))
	pathPrefix := strings.TrimSpace(req.CanonicalPathPrefix)
	if pathPrefix == "" {
		pathPrefix = "/v1"
	}
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	return ProviderProfile{
		DisplayLabel:   strings.TrimSpace(req.DisplayLabel),
		ProviderFamily: family,
		AdapterKind:    adapterKind,
		DestinationIdentity: policyengine.DestinationDescriptor{
			SchemaID:               "runecode.protocol.v0.DestinationDescriptor",
			SchemaVersion:          "0.1.0",
			DescriptorKind:         "model_endpoint",
			CanonicalHost:          host,
			CanonicalPathPrefix:    pathPrefix,
			ProviderOrNamespace:    family,
			TLSRequired:            true,
			PrivateRangeBlocking:   "enforced",
			DNSRebindingProtection: "enforced",
		},
		SupportedAuthModes:   []string{"direct_credential"},
		CurrentAuthMode:      "direct_credential",
		AllowlistedModelIDs:  req.AllowlistedModelIDs,
		ModelCatalogPosture:  ProviderModelCatalogPosture{SelectionAuthority: "manual_allowlist_canonical", DiscoveryPosture: "advisory", CompatibilityProbePosture: "advisory"},
		CompatibilityPosture: "unverified",
		AuthMaterial:         ProviderAuthMaterial{MaterialKind: "direct_credential", MaterialState: "missing"},
		ReadinessPosture:     ProviderReadinessPosture{ConfigurationState: "configured", CredentialState: "missing", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready", ReasonCodes: []string{"secret_ingress_required"}},
	}
}

func (s *Service) persistDirectCredential(session ProviderSetupSession, secret []byte) (ProviderProfile, error) {
	secretRef := fmt.Sprintf("secrets/model-providers/%s/direct-credential", session.ProviderProfileID)
	if _, err := s.secretsSvc.ImportSecret(secretRef, bytes.NewReader(secret)); err != nil {
		return ProviderProfile{}, err
	}
	updated, err := s.providerSubstrate.setAuthMaterial(session.ProviderProfileID, ProviderAuthMaterial{MaterialKind: "direct_credential", MaterialState: "present", SecretRef: secretRef, LeasePolicyRef: "secretsd://lease-policy/model-provider-default", LastRotatedAt: session.UpdatedAt})
	if err != nil {
		return ProviderProfile{}, err
	}
	updated.ReadinessPosture.ConfigurationState = "configured"
	updated.ReadinessPosture.CredentialState = "present"
	updated.ReadinessPosture.ReasonCodes = providerReadinessReasonCodes(updated.ReadinessPosture)
	return s.providerSubstrate.upsertProfile(updated)
}

func (s *Service) providerProfileByID(profileID string) (ProviderProfile, bool) {
	profiles := s.providerSubstrate.snapshotProfiles()
	for _, profile := range profiles {
		if profile.ProviderProfileID == profileID {
			return profile, true
		}
	}
	return ProviderProfile{}, false
}

func providerReadinessReasonCodes(posture ProviderReadinessPosture) []string {
	reasons := []string{}
	if strings.TrimSpace(posture.ConfigurationState) != "configured" {
		reasons = append(reasons, "provider_configuration_required")
	}
	if strings.TrimSpace(posture.CredentialState) != "present" {
		reasons = append(reasons, "secret_ingress_required")
	}
	if strings.TrimSpace(posture.ConnectivityState) == "unknown" {
		reasons = append(reasons, "connectivity_validation_pending")
	}
	if strings.TrimSpace(posture.CompatibilityState) == "unknown" {
		reasons = append(reasons, "compatibility_probe_pending")
	}
	return normalizedStringSet(reasons)
}
