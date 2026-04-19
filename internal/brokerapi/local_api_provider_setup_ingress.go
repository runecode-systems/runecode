package brokerapi

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) HandleProviderSetupSecretIngressSubmit(ctx context.Context, req ProviderSetupSecretIngressSubmitRequest, secret []byte, meta RequestContext) (ProviderSetupSecretIngressSubmitResponse, *ErrorResponse) {
	requestID, cleanup, errResp := s.beginProviderSetupRequest(ctx, req, req.RequestID, meta, providerSetupSecretIngressSubmitRequestSchemaPath)
	if errResp != nil {
		return ProviderSetupSecretIngressSubmitResponse{}, errResp
	}
	defer cleanup()
	if errResp := validateProviderSecretIngressPayload(s, requestID, secret); errResp != nil {
		return ProviderSetupSecretIngressSubmitResponse{}, errResp
	}
	session, updated, changeKind, readinessChanged, previousReadiness, errResp := s.commitProviderSecretIngress(requestID, req.SecretIngressToken, secret)
	if errResp != nil {
		return ProviderSetupSecretIngressSubmitResponse{}, errResp
	}
	resp, errResp := s.providerSetupSecretIngressSubmitResponse(requestID, session, updated)
	if errResp != nil {
		return ProviderSetupSecretIngressSubmitResponse{}, errResp
	}
	if errResp := s.auditProviderSecretIngressSubmit(requestID, updated, changeKind, readinessChanged, previousReadiness); errResp != nil {
		return ProviderSetupSecretIngressSubmitResponse{}, errResp
	}
	return resp, nil
}

func validateProviderSecretIngressPayload(s *Service, requestID string, secret []byte) *ErrorResponse {
	if len(secret) != 0 {
		return nil
	}
	errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "secret ingress payload is required")
	return &errOut
}

func (s *Service) commitProviderSecretIngress(requestID, ingressToken string, secret []byte) (ProviderSetupSession, ProviderProfile, string, bool, string, *ErrorResponse) {
	session, _, err := s.providerSetup.consumeIngress(ingressToken)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return ProviderSetupSession{}, ProviderProfile{}, "", false, "", &errOut
	}
	updated, changeKind, readinessChanged, previousReadiness, err := s.persistDirectCredential(session, secret)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSession{}, ProviderProfile{}, "", false, "", &errOut
	}
	session, err = s.providerSetup.completeIngress(ingressToken)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ProviderSetupSession{}, ProviderProfile{}, "", false, "", &errOut
	}
	return session, updated, changeKind, readinessChanged, previousReadiness, nil
}

func (s *Service) providerSetupSecretIngressSubmitResponse(requestID string, session ProviderSetupSession, updated ProviderProfile) (ProviderSetupSecretIngressSubmitResponse, *ErrorResponse) {
	resp := ProviderSetupSecretIngressSubmitResponse{SchemaID: "runecode.protocol.v0.ProviderSetupSecretIngressSubmitResponse", SchemaVersion: "0.1.0", RequestID: requestID, SetupSession: session, Profile: updated.projected()}
	if err := s.validateResponse(resp, providerSetupSecretIngressSubmitResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProviderSetupSecretIngressSubmitResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) auditProviderSecretIngressSubmit(requestID string, updated ProviderProfile, changeKind string, readinessChanged bool, previousReadiness string) *ErrorResponse {
	if err := s.auditProviderCredentialChange(requestID, updated, changeKind); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	if !readinessChanged {
		return nil
	}
	if err := s.auditProviderReadinessTransition(requestID, updated.ProviderProfileID, previousReadiness, updated.ReadinessPosture.EffectiveReadiness, updated.ReadinessPosture.ReasonCodes); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return &errOut
	}
	return nil
}

func (s *Service) persistDirectCredential(session ProviderSetupSession, secret []byte) (ProviderProfile, string, bool, string, error) {
	secretRef := fmt.Sprintf("secrets/model-providers/%s/direct-credential", session.ProviderProfileID)
	rotatedAt := s.now().UTC().Format(time.RFC3339)
	if _, err := s.secretsSvc.ImportSecret(secretRef, bytes.NewReader(secret)); err != nil {
		return ProviderProfile{}, "", false, "", err
	}
	before, ok := s.providerProfileByID(session.ProviderProfileID)
	if !ok {
		return ProviderProfile{}, "", false, "", fmt.Errorf("provider profile not found")
	}
	changeKind := "committed"
	if strings.TrimSpace(before.AuthMaterial.MaterialState) == "present" {
		changeKind = "rotated"
	}
	updated, err := s.providerSubstrate.setAuthMaterial(session.ProviderProfileID, ProviderAuthMaterial{MaterialKind: "direct_credential", MaterialState: "present", SecretRef: secretRef, LeasePolicyRef: "secretsd://lease-policy/model-provider-default", LastRotatedAt: rotatedAt})
	if err != nil {
		return ProviderProfile{}, "", false, "", err
	}
	updated.ReadinessPosture.ConfigurationState = "configured"
	updated.ReadinessPosture.CredentialState = "present"
	updated.ReadinessPosture.EffectiveReadiness = providerEffectiveReadiness(updated.ReadinessPosture)
	updated.ReadinessPosture.ReasonCodes = providerReadinessReasonCodes(updated.ReadinessPosture)
	updated, _, err = s.providerSubstrate.upsertProfile(updated)
	if err != nil {
		return ProviderProfile{}, "", false, "", err
	}
	return updated, changeKind, before.ReadinessPosture.EffectiveReadiness != updated.ReadinessPosture.EffectiveReadiness, before.ReadinessPosture.EffectiveReadiness, nil
}
