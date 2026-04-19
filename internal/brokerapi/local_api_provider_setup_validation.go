package brokerapi

import (
	"context"
	"errors"
	"strings"
)

func (s *Service) HandleProviderValidationBegin(ctx context.Context, req ProviderValidationBeginRequest, meta RequestContext) (ProviderValidationBeginResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderSetupRequest(ctx, req, req.RequestID, meta, providerValidationBeginRequestSchemaPath)
	if errResp != nil {
		return ProviderValidationBeginResponse{}, errResp
	}
	defer cleanup()
	profile, errResp := s.requireProviderValidationBeginProfile(requestID, req.ProviderProfileID)
	if errResp != nil {
		return ProviderValidationBeginResponse{}, errResp
	}
	session, profile, errResp := s.beginProviderValidationAttempt(requestID, profile, req.ValidationAttemptID)
	if errResp != nil {
		return ProviderValidationBeginResponse{}, errResp
	}
	resp := ProviderValidationBeginResponse{SchemaID: "runecode.protocol.v0.ProviderValidationBeginResponse", SchemaVersion: "0.1.0", RequestID: requestID, ProviderProfileID: profile.ProviderProfileID, ValidationAttemptID: session.ValidationAttemptID, SetupSession: session, Profile: profile.projected()}
	if err := s.validateResponse(resp, providerValidationBeginResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderValidationBeginResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) requireProviderValidationBeginProfile(requestID, rawProfileID string) (ProviderProfile, *ErrorResponse) {
	profileID := strings.TrimSpace(rawProfileID)
	if profileID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider_profile_id is required")
		return ProviderProfile{}, &errOut
	}
	profile, ok := s.providerProfileByID(profileID)
	if !ok {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider profile not found")
		return ProviderProfile{}, &errOut
	}
	if strings.TrimSpace(profile.AuthMaterial.MaterialState) != "present" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider profile does not have committed credential material")
		return ProviderProfile{}, &errOut
	}
	return profile, nil
}

func (s *Service) beginProviderValidationAttempt(requestID string, profile ProviderProfile, attemptID string) (ProviderSetupSession, ProviderProfile, *ErrorResponse) {
	session, err := s.providerSetup.startValidationForProfile(profile.ProviderProfileID, attemptID)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return ProviderSetupSession{}, ProviderProfile{}, &errOut
	}
	updated, _, err := s.providerSubstrate.upsertProfile(withValidationInProgress(profile, session.ValidationAttemptID))
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSession{}, ProviderProfile{}, &errOut
	}
	return session, updated, nil
}

func withValidationInProgress(profile ProviderProfile, attemptID string) ProviderProfile {
	posture := profile.ReadinessPosture
	posture.ValidationAttemptID = attemptID
	if !stringSliceContains(posture.ReasonCodes, "validation_in_progress") {
		posture.ReasonCodes = append(posture.ReasonCodes, "validation_in_progress")
	}
	posture.ReasonCodes = normalizedStringSet(posture.ReasonCodes)
	return withUpdatedReadinessPosture(profile, posture)
}

func (s *Service) HandleProviderValidationCommit(ctx context.Context, req ProviderValidationCommitRequest, meta RequestContext) (ProviderValidationCommitResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderSetupRequest(ctx, req, req.RequestID, meta, providerValidationCommitRequestSchemaPath)
	if errResp != nil {
		return ProviderValidationCommitResponse{}, errResp
	}
	defer cleanup()
	profileID, attemptID, before, errResp := s.providerValidationCommitInputs(requestID, req)
	if errResp != nil {
		return ProviderValidationCommitResponse{}, errResp
	}
	nextPosture, outcome := providerValidationCommitOutcome(before.ReadinessPosture, req, attemptID)
	session, updated, errResp := s.commitProviderValidation(requestID, profileID, attemptID, outcome, nextPosture)
	if errResp != nil {
		return ProviderValidationCommitResponse{}, errResp
	}
	resp := ProviderValidationCommitResponse{SchemaID: "runecode.protocol.v0.ProviderValidationCommitResponse", SchemaVersion: "0.1.0", RequestID: requestID, ProviderProfileID: profileID, ValidationAttemptID: attemptID, ValidationOutcome: outcome, SetupSession: session, Profile: updated.projected()}
	if err := s.validateResponse(resp, providerValidationCommitResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderValidationCommitResponse{}, &errOut
	}
	if errResp := s.auditProviderValidationCommit(requestID, profileID, attemptID, outcome, before, updated); errResp != nil {
		return ProviderValidationCommitResponse{}, errResp
	}
	return resp, nil
}

func (s *Service) providerValidationCommitInputs(requestID string, req ProviderValidationCommitRequest) (string, string, ProviderProfile, *ErrorResponse) {
	profileID := strings.TrimSpace(req.ProviderProfileID)
	if profileID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider_profile_id is required")
		return "", "", ProviderProfile{}, &errOut
	}
	attemptID := strings.TrimSpace(req.ValidationAttemptID)
	if attemptID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "validation_attempt_id is required")
		return "", "", ProviderProfile{}, &errOut
	}
	before, ok := s.providerProfileByID(profileID)
	if !ok {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "provider profile not found")
		return "", "", ProviderProfile{}, &errOut
	}
	return profileID, attemptID, before, nil
}

func providerValidationCommitOutcome(previous ProviderReadinessPosture, req ProviderValidationCommitRequest, attemptID string) (ProviderReadinessPosture, string) {
	nextPosture := buildProviderValidationPosture(previous, req)
	nextPosture.ValidationAttemptID = attemptID
	if nextPosture.EffectiveReadiness == "ready" {
		return nextPosture, "succeeded"
	}
	return nextPosture, "failed"
}

func (s *Service) commitProviderValidation(requestID, profileID, attemptID, outcome string, nextPosture ProviderReadinessPosture) (ProviderSetupSession, ProviderProfile, *ErrorResponse) {
	session, err := s.providerSetup.commitValidationForProfile(profileID, attemptID, outcome)
	if err != nil {
		return ProviderSetupSession{}, ProviderProfile{}, providerValidationCommitError(s, requestID, err)
	}
	updated, err := s.providerSubstrate.recordValidation(profileID, nextPosture)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSession{}, ProviderProfile{}, &errOut
	}
	updated.CompatibilityPosture = compatibilityPostureFromReadiness(updated.ReadinessPosture.CompatibilityState)
	updated, _, err = s.providerSubstrate.upsertProfile(updated)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSession{}, ProviderProfile{}, &errOut
	}
	return session, updated, nil
}

func providerValidationCommitError(s *Service, requestID string, err error) *ErrorResponse {
	code := "gateway_failure"
	category := "internal"
	if errors.Is(err, errProviderValidationCommitPrecondition) {
		code = "broker_validation_schema_invalid"
		category = "validation"
	}
	errOut := s.makeError(requestID, code, category, false, err.Error())
	return &errOut
}

func (s *Service) auditProviderValidationCommit(requestID, profileID, attemptID, outcome string, before, updated ProviderProfile) *ErrorResponse {
	if err := s.auditProviderValidationResult(requestID, profileID, attemptID, outcome, updated.ReadinessPosture.ReasonCodes); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	if err := s.auditProviderReadinessTransition(requestID, profileID, before.ReadinessPosture.EffectiveReadiness, updated.ReadinessPosture.EffectiveReadiness, updated.ReadinessPosture.ReasonCodes); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	return nil
}
