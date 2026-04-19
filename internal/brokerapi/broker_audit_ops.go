package brokerapi

import "fmt"

func (s *Service) auditErrorResponse(resp ErrorResponse) ErrorResponse {
	if s == nil || s.auditor == nil || s.store == nil {
		resp.Error.Code = "gateway_failure"
		resp.Error.Category = "internal"
		resp.Error.Retryable = false
		resp.Error.Message = "broker rejection audit path unavailable"
		return resp
	}
	if !shouldAuditErrorCode(resp.Error.Code) {
		return resp
	}
	if err := s.auditor.emitRejection(s.store, resp); err != nil {
		resp.Error.Code = "gateway_failure"
		resp.Error.Category = "internal"
		resp.Error.Retryable = false
		resp.Error.Message = "failed to persist broker rejection audit event"
	}
	return resp
}

func (s *Service) auditApprovalResolution(requestID, approvalID, status, reasonCode string) error {
	if s == nil || s.auditor == nil || s.store == nil {
		return fmt.Errorf("broker approval audit path unavailable for request %q", requestID)
	}
	if err := s.auditor.emitApprovalResolution(s.store, requestID, approvalID, status, reasonCode); err != nil {
		return fmt.Errorf("failed to persist broker approval audit event for request %q: %w", requestID, err)
	}
	return nil
}

func (s *Service) auditProviderProfileChange(requestID string, profile ProviderProfile, changeKind string) error {
	if s == nil || s.auditor == nil || s.store == nil {
		return fmt.Errorf("broker provider profile audit path unavailable for request %q", requestID)
	}
	details := map[string]interface{}{
		"request_id":          requestID,
		"provider_profile_id": profile.ProviderProfileID,
		"provider_family":     profile.ProviderFamily,
		"destination_ref":     profile.DestinationRef,
		"change_kind":         changeKind,
	}
	if err := s.auditor.emitProviderProfileEvent(s.store, details); err != nil {
		return fmt.Errorf("failed to persist broker provider profile audit event for request %q: %w", requestID, err)
	}
	return nil
}

func (s *Service) auditProviderCredentialChange(requestID string, profile ProviderProfile, changeKind string) error {
	if s == nil || s.auditor == nil || s.store == nil {
		return fmt.Errorf("broker provider credential audit path unavailable for request %q", requestID)
	}
	details := map[string]interface{}{
		"request_id":          requestID,
		"provider_profile_id": profile.ProviderProfileID,
		"change_kind":         changeKind,
		"auth_material_kind":  profile.AuthMaterial.MaterialKind,
		"credential_state":    profile.ReadinessPosture.CredentialState,
	}
	if err := s.auditor.emitProviderCredentialEvent(s.store, details); err != nil {
		return fmt.Errorf("failed to persist broker provider credential audit event for request %q: %w", requestID, err)
	}
	return nil
}

func (s *Service) auditProviderValidationResult(requestID, profileID, attemptID, outcome string, reasonCodes []string) error {
	if s == nil || s.auditor == nil || s.store == nil {
		return fmt.Errorf("broker provider validation audit path unavailable for request %q", requestID)
	}
	details := map[string]interface{}{
		"request_id":            requestID,
		"provider_profile_id":   profileID,
		"validation_attempt_id": attemptID,
		"validation_outcome":    outcome,
		"reason_codes":          append([]string{}, reasonCodes...),
	}
	if err := s.auditor.emitProviderValidationEvent(s.store, details); err != nil {
		return fmt.Errorf("failed to persist broker provider validation audit event for request %q: %w", requestID, err)
	}
	return nil
}

func (s *Service) auditProviderReadinessTransition(requestID, profileID, from, to string, reasonCodes []string) error {
	if s == nil || s.auditor == nil || s.store == nil {
		return fmt.Errorf("broker provider readiness audit path unavailable for request %q", requestID)
	}
	details := map[string]interface{}{
		"request_id":          requestID,
		"provider_profile_id": profileID,
		"from_readiness":      from,
		"to_readiness":        to,
		"reason_codes":        append([]string{}, reasonCodes...),
	}
	if err := s.auditor.emitProviderReadinessEvent(s.store, details); err != nil {
		return fmt.Errorf("failed to persist broker provider readiness audit event for request %q: %w", requestID, err)
	}
	return nil
}
