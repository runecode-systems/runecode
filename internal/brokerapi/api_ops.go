package brokerapi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) HandleArtifactList(ctx context.Context, req ArtifactListRequest, meta RequestContext) (ArtifactListResponse, *ErrorResponse) {
	if meta.AdmissionErr != nil {
		err := s.makeError(resolveRequestID(req.RequestID, meta.RequestID), "broker_api_auth_admission_denied", "auth", false, meta.AdmissionErr.Error())
		return ArtifactListResponse{}, &err
	}
	requestID := resolveRequestID(req.RequestID, meta.RequestID)
	if requestID == "" {
		err := s.makeError(defaultRequestIDFallback, "broker_validation_request_id_missing", "validation", false, "request_id is required")
		return ArtifactListResponse{}, &err
	}
	req.RequestID = requestID
	if err := s.validateRequest(req, brokerArtifactListRequestSchemaPath); err != nil {
		errResp := s.errorFromValidation(requestID, err)
		return ArtifactListResponse{}, &errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errResp := s.errorFromLimit(requestID, err)
		return ArtifactListResponse{}, &errResp
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	select {
	case <-requestCtx.Done():
		errResp := s.errorFromContext(requestID, requestCtx.Err())
		return ArtifactListResponse{}, &errResp
	default:
	}
	resp := defaultArtifactListResponse(requestID, s.List())
	if err := s.validateResponse(resp, brokerArtifactListResponseSchemaPath); err != nil {
		errResp := s.errorFromValidation(requestID, err)
		return ArtifactListResponse{}, &errResp
	}
	return resp, nil
}

func (s *Service) HandleArtifactHead(ctx context.Context, req ArtifactHeadRequest, meta RequestContext) (ArtifactHeadResponse, *ErrorResponse) {
	if meta.AdmissionErr != nil {
		err := s.makeError(resolveRequestID(req.RequestID, meta.RequestID), "broker_api_auth_admission_denied", "auth", false, meta.AdmissionErr.Error())
		return ArtifactHeadResponse{}, &err
	}
	requestID := resolveRequestID(req.RequestID, meta.RequestID)
	if requestID == "" {
		err := s.makeError(defaultRequestIDFallback, "broker_validation_request_id_missing", "validation", false, "request_id is required")
		return ArtifactHeadResponse{}, &err
	}
	req.RequestID = requestID
	if err := s.validateRequest(req, brokerArtifactHeadRequestSchemaPath); err != nil {
		errResp := s.errorFromValidation(requestID, err)
		return ArtifactHeadResponse{}, &errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errResp := s.errorFromLimit(requestID, err)
		return ArtifactHeadResponse{}, &errResp
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	select {
	case <-requestCtx.Done():
		errResp := s.errorFromContext(requestID, requestCtx.Err())
		return ArtifactHeadResponse{}, &errResp
	default:
	}
	record, err := s.Head(req.Digest)
	if err != nil {
		errResp := s.errorFromStore(requestID, err)
		return ArtifactHeadResponse{}, &errResp
	}
	resp := defaultArtifactHeadResponse(requestID, record)
	if err := s.validateResponse(resp, brokerArtifactHeadResponseSchemaPath); err != nil {
		errResp := s.errorFromValidation(requestID, err)
		return ArtifactHeadResponse{}, &errResp
	}
	return resp, nil
}

func (s *Service) HandleArtifactPut(ctx context.Context, req ArtifactPutRequest, meta RequestContext) (ArtifactPutResponse, *ErrorResponse) {
	if meta.AdmissionErr != nil {
		err := s.makeError(resolveRequestID(req.RequestID, meta.RequestID), "broker_api_auth_admission_denied", "auth", false, meta.AdmissionErr.Error())
		return ArtifactPutResponse{}, &err
	}
	requestID := resolveRequestID(req.RequestID, meta.RequestID)
	if requestID == "" {
		err := s.makeError(defaultRequestIDFallback, "broker_validation_request_id_missing", "validation", false, "request_id is required")
		return ArtifactPutResponse{}, &err
	}
	req.RequestID = requestID
	if err := s.validateRequest(req, brokerArtifactPutRequestSchemaPath); err != nil {
		errResp := s.errorFromValidation(requestID, err)
		return ArtifactPutResponse{}, &errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errResp := s.errorFromLimit(requestID, err)
		return ArtifactPutResponse{}, &errResp
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	select {
	case <-requestCtx.Done():
		errResp := s.errorFromContext(requestID, requestCtx.Err())
		return ArtifactPutResponse{}, &errResp
	default:
	}
	putReq, errResp := s.artifactPutRequestToStore(requestID, req)
	if errResp != nil {
		return ArtifactPutResponse{}, errResp
	}
	ref, err := s.Put(putReq)
	if err != nil {
		errResp := s.errorFromStore(requestID, err)
		return ArtifactPutResponse{}, &errResp
	}
	resp := defaultArtifactPutResponse(requestID, ref)
	if err := s.validateResponse(resp, brokerArtifactPutResponseSchemaPath); err != nil {
		errResp := s.errorFromValidation(requestID, err)
		return ArtifactPutResponse{}, &errResp
	}
	return resp, nil
}

func (s *Service) artifactPutRequestToStore(requestID string, req ArtifactPutRequest) (artifacts.PutRequest, *ErrorResponse) {
	payload, errResp := s.decodeArtifactPutPayload(requestID, req.PayloadBase64)
	if errResp != nil {
		return artifacts.PutRequest{}, errResp
	}
	class, err := ParseDataClass(req.DataClass)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_data_class_invalid", "validation", false, err.Error())
		return artifacts.PutRequest{}, &errOut
	}
	return artifacts.PutRequest{
		Payload:               payload,
		ContentType:           req.ContentType,
		DataClass:             class,
		ProvenanceReceiptHash: req.ProvenanceReceiptHash,
		CreatedByRole:         req.CreatedByRole,
		RunID:                 req.RunID,
		StepID:                req.StepID,
	}, nil
}

func (s *Service) decodeArtifactPutPayload(requestID, payloadBase64 string) ([]byte, *ErrorResponse) {
	payload, err := base64.StdEncoding.DecodeString(payloadBase64)
	if err != nil {
		errResp := s.makeError(requestID, "broker_validation_payload_base64_invalid", "validation", false, "payload_base64 must be valid base64")
		return nil, &errResp
	}
	if len(payload) > s.apiConfig.Limits.MaxMessageBytes {
		errResp := s.makeError(requestID, "broker_limit_message_size_exceeded", "transport", false, "decoded payload exceeds max message size")
		return nil, &errResp
	}
	return payload, nil
}

func (s *Service) makeError(requestID string, code string, category string, retryable bool, message string) ErrorResponse {
	resp := toErrorResponse(requestID, code, category, retryable, message)
	if err := s.validateResponse(resp, brokerErrorResponseSchemaPath); err == nil {
		return resp
	}
	fallback := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, "broker error-envelope validation failed")
	return fallback
}

func resolveRequestID(primary string, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func (s *Service) validateRequest(value any, schemaPath string) error {
	if err := validateMessageLimits(value, s.apiConfig.Limits); err != nil {
		return err
	}
	return validateJSONEnvelope(value, schemaPath)
}

func (s *Service) validateResponse(value any, schemaPath string) error {
	if err := validateMessageLimits(value, s.apiConfig.Limits); err != nil {
		return err
	}
	return validateJSONEnvelope(value, schemaPath)
}

func (s *Service) acquireInFlight(meta RequestContext) (func(), error) {
	if s.apiInflight == nil {
		return func() {}, nil
	}
	return s.apiInflight.acquire(meta.ClientID, meta.LaneID)
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
