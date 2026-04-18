package brokerapi

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) HandleGitSetupGet(ctx context.Context, req GitSetupGetRequest, meta RequestContext) (GitSetupGetResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginGitSetupRequest(ctx, req, req.RequestID, meta, gitSetupGetRequestSchemaPath)
	if errResp != nil {
		return GitSetupGetResponse{}, errResp
	}
	defer cleanup()
	provider := normalizeGitProvider(req.Provider)
	account, profiles, auth, control := s.gitSetup.snapshot(provider)
	resp := GitSetupGetResponse{
		SchemaID:          "runecode.protocol.v0.GitSetupGetResponse",
		SchemaVersion:     "0.1.0",
		RequestID:         requestID,
		ProviderAccount:   account,
		IdentityProfiles:  profiles,
		AuthPosture:       auth,
		ControlPlaneState: control,
		PolicySurface: GitPolicySurfaceState{
			ArtifactManagedOnly:   true,
			InspectionSupported:   true,
			PrepareChangesSupport: true,
			DirectMutationSupport: false,
		},
	}
	if err := s.validateResponse(resp, gitSetupGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return GitSetupGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleGitSetupAuthBootstrap(ctx context.Context, req GitSetupAuthBootstrapRequest, meta RequestContext) (GitSetupAuthBootstrapResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginGitSetupRequest(ctx, req, req.RequestID, meta, gitSetupAuthBootstrapRequestSchemaPath)
	if errResp != nil {
		return GitSetupAuthBootstrapResponse{}, errResp
	}
	defer cleanup()
	provider := normalizeGitProvider(req.Provider)
	mode, modeErr := validateGitBootstrapMode(strings.TrimSpace(req.Mode))
	if modeErr != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, modeErr.Error())
		return GitSetupAuthBootstrapResponse{}, &errOut
	}
	account, auth, status, deviceURI, nextPoll, deviceCode := s.gitSetup.applyAuthBootstrap(provider, mode)
	resp := GitSetupAuthBootstrapResponse{
		SchemaID:              "runecode.protocol.v0.GitSetupAuthBootstrapResponse",
		SchemaVersion:         "0.1.0",
		RequestID:             requestID,
		Provider:              provider,
		Mode:                  mode,
		Status:                status,
		DeviceVerificationURI: deviceURI,
		DeviceUserCode:        deviceCode,
		NextPollAfterSeconds:  nextPoll,
		AccountState:          account,
		AuthPosture:           auth,
	}
	if err := s.validateResponse(resp, gitSetupAuthBootstrapResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return GitSetupAuthBootstrapResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleGitSetupIdentityUpsert(ctx context.Context, req GitSetupIdentityUpsertRequest, meta RequestContext) (GitSetupIdentityUpsertResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginGitSetupRequest(ctx, req, req.RequestID, meta, gitSetupIdentityUpsertRequestSchemaPath)
	if errResp != nil {
		return GitSetupIdentityUpsertResponse{}, errResp
	}
	defer cleanup()
	provider := normalizeGitProvider(req.Provider)
	profile, profileErr := normalizeGitCommitIdentityProfile(req.Profile)
	if profileErr != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, profileErr.Error())
		return GitSetupIdentityUpsertResponse{}, &errOut
	}
	updated, control := s.gitSetup.upsertProfile(provider, profile)
	resp := GitSetupIdentityUpsertResponse{SchemaID: "runecode.protocol.v0.GitSetupIdentityUpsertResponse", SchemaVersion: "0.1.0", RequestID: requestID, Provider: provider, Profile: updated, ControlPlaneState: control}
	if err := s.validateResponse(resp, gitSetupIdentityUpsertResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return GitSetupIdentityUpsertResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) beginGitSetupRequest(ctx context.Context, req any, requestID string, meta RequestContext, schemaPath string) (string, func(), *ErrorResponse) {
	if err := s.ensureGitSetupState(); err != nil {
		return "", nil, gitSetupUnavailableError(err)
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

func gitSetupUnavailableError(err error) *ErrorResponse {
	errOut := ErrorResponse{SchemaID: "runecode.protocol.v0.BrokerErrorResponse", SchemaVersion: "0.1.0", RequestID: defaultRequestIDFallback, Error: ProtocolError{SchemaID: "runecode.protocol.v0.Error", SchemaVersion: "0.3.0", Code: "gateway_failure", Category: "internal", Retryable: false, Message: err.Error()}}
	return &errOut
}

func validateGitBootstrapMode(mode string) (string, error) {
	switch mode {
	case "browser", "device_code", "interactive_token_prompt":
		return mode, nil
	default:
		return "", fmt.Errorf("mode must be browser, device_code, or interactive_token_prompt")
	}
}

func normalizeGitCommitIdentityProfile(profile GitCommitIdentityProfile) (GitCommitIdentityProfile, error) {
	profile.SchemaID = "runecode.protocol.v0.GitCommitIdentityProfile"
	profile.SchemaVersion = "0.1.0"
	if strings.TrimSpace(profile.ProfileID) == "" {
		return GitCommitIdentityProfile{}, fmt.Errorf("profile.profile_id is required")
	}
	if strings.TrimSpace(profile.DisplayName) == "" {
		profile.DisplayName = profile.ProfileID
	}
	if missingGitIdentityFields(profile) {
		return GitCommitIdentityProfile{}, fmt.Errorf("profile author/committer/signoff identity fields are required")
	}
	return profile, nil
}

func missingGitIdentityFields(profile GitCommitIdentityProfile) bool {
	return strings.TrimSpace(profile.AuthorName) == "" || strings.TrimSpace(profile.AuthorEmail) == "" || strings.TrimSpace(profile.CommitterName) == "" || strings.TrimSpace(profile.CommitterEmail) == "" || strings.TrimSpace(profile.SignoffName) == "" || strings.TrimSpace(profile.SignoffEmail) == ""
}

func (s *Service) ensureGitSetupState() error {
	if s == nil {
		return fmt.Errorf("service unavailable")
	}
	if s.gitSetup == nil {
		s.gitSetup = newGitSetupState()
	}
	return nil
}
