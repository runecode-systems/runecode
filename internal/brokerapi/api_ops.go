package brokerapi

import (
	"context"
	"encoding/base64"
	"time"

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
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return ArtifactListResponse{}, errResp
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
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return ArtifactHeadResponse{}, errResp
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
	requestID, errResp := s.prepareArtifactPutRequest(req, meta)
	if errResp != nil {
		return ArtifactPutResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errResp := s.errorFromLimit(requestID, err)
		return ArtifactPutResponse{}, &errResp
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return ArtifactPutResponse{}, errResp
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

func (s *Service) prepareArtifactPutRequest(req ArtifactPutRequest, meta RequestContext) (string, *ErrorResponse) {
	if meta.AdmissionErr != nil {
		err := s.makeError(resolveRequestID(req.RequestID, meta.RequestID), "broker_api_auth_admission_denied", "auth", false, meta.AdmissionErr.Error())
		return "", &err
	}
	requestID := resolveRequestID(req.RequestID, meta.RequestID)
	if requestID == "" {
		err := s.makeError(defaultRequestIDFallback, "broker_validation_request_id_missing", "validation", false, "request_id is required")
		return "", &err
	}
	req.RequestID = requestID
	if err := s.validateRequest(req, brokerArtifactPutRequestSchemaPath); err != nil {
		errResp := s.errorFromValidation(requestID, err)
		return "", &errResp
	}
	return requestID, nil
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
		return s.auditErrorResponse(resp)
	}
	fallback := toErrorResponse(defaultRequestIDFallback, "gateway_failure", "internal", false, "broker error-envelope validation failed")
	return s.auditErrorResponse(fallback)
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
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	return s.apiInflight.acquireAt(meta.ClientID, meta.LaneID, now)
}
