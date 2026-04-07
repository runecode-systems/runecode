package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) requestContextError(requestID string, requestCtx context.Context) *ErrorResponse {
	select {
	case <-requestCtx.Done():
		errResp := s.errorFromContext(requestID, requestCtx.Err())
		return &errResp
	default:
		return nil
	}
}

func (s *Service) errorFromValidation(requestID string, err error) ErrorResponse {
	code := "broker_validation_schema_invalid"
	if errors.Is(err, context.DeadlineExceeded) {
		return s.errorFromContext(requestID, err)
	}
	if contains(err, "message size") {
		code = "broker_limit_message_size_exceeded"
		return s.makeError(requestID, code, "transport", false, err.Error())
	}
	if contains(err, "message depth") || contains(err, "array length") || contains(err, "object property count") {
		code = "broker_limit_structural_complexity_exceeded"
		return s.makeError(requestID, code, "transport", false, err.Error())
	}
	return s.makeError(requestID, code, "validation", false, err.Error())
}

func (s *Service) errorFromLimit(requestID string, err error) ErrorResponse {
	if errors.Is(err, errInFlightLimitExceeded) {
		return s.makeError(requestID, "broker_limit_in_flight_exceeded", "transport", true, err.Error())
	}
	return s.makeError(requestID, "broker_limit_in_flight_exceeded", "transport", true, err.Error())
}

func (s *Service) errorFromContext(requestID string, err error) ErrorResponse {
	if errors.Is(err, context.DeadlineExceeded) {
		return s.makeError(requestID, "broker_timeout_request_deadline_exceeded", "timeout", true, err.Error())
	}
	if errors.Is(err, context.Canceled) {
		return s.makeError(requestID, "request_cancelled", "transport", true, err.Error())
	}
	return s.makeError(requestID, "broker_timeout_request_deadline_exceeded", "timeout", true, err.Error())
}

func (s *Service) errorFromStore(requestID string, err error) ErrorResponse {
	switch {
	case errors.Is(err, artifacts.ErrArtifactNotFound):
		return s.makeError(requestID, "broker_not_found_artifact", "storage", false, err.Error())
	case errors.Is(err, artifacts.ErrFlowDenied),
		errors.Is(err, artifacts.ErrFlowProducerRoleMismatch),
		errors.Is(err, artifacts.ErrUnapprovedEgressDenied),
		errors.Is(err, artifacts.ErrApprovedEgressRequiresManifest),
		errors.Is(err, artifacts.ErrApprovedExcerptRevoked),
		errors.Is(err, artifacts.ErrQuotaExceeded),
		errors.Is(err, artifacts.ErrPromotionRateLimited),
		errors.Is(err, artifacts.ErrPromotionTooLarge):
		return s.makeError(requestID, "broker_limit_policy_rejected", "policy", false, err.Error())
	case errors.Is(err, artifacts.ErrApprovalRequestArtifactRequired),
		errors.Is(err, artifacts.ErrApprovalArtifactRequired),
		errors.Is(err, artifacts.ErrVerifierNotFound),
		errors.Is(err, artifacts.ErrApprovalVerificationFailed):
		return s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
	default:
		return s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
	}
}

func contains(err error, needle string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(fmt.Sprintf("%v", err), needle)
}
