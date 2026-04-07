package brokerapi

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) HandleRunList(ctx context.Context, req RunListRequest, meta RequestContext) (RunListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, runListRequestSchemaPath)
	if errResp != nil {
		return RunListResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errResp := s.errorFromLimit(requestID, err)
		return RunListResponse{}, &errResp
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errResp := s.errorFromContext(requestID, err)
		return RunListResponse{}, &errResp
	}
	order := req.Order
	if order == "" {
		order = "updated_at_desc"
	}
	runs, err := s.runSummaries(order)
	if err != nil {
		errResp := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return RunListResponse{}, &errResp
	}
	limit := normalizeLimit(req.Limit, 50, 200)
	page, next, err := paginate(runs, req.Cursor, limit)
	if err != nil {
		errResp := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return RunListResponse{}, &errResp
	}
	resp := RunListResponse{SchemaID: "runecode.protocol.v0.RunListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Runs: page, NextCursor: next}
	if err := s.validateResponse(resp, runListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return RunListResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleRunGet(ctx context.Context, req RunGetRequest, meta RequestContext) (RunGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, runGetRequestSchemaPath)
	if errResp != nil {
		return RunGetResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errResp := s.errorFromLimit(requestID, err)
		return RunGetResponse{}, &errResp
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errResp := s.errorFromContext(requestID, err)
		return RunGetResponse{}, &errResp
	}
	if strings.TrimSpace(req.RunID) == "" {
		errResp := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "run_id is required")
		return RunGetResponse{}, &errResp
	}
	detail, ok, err := s.runDetail(req.RunID)
	if err != nil {
		errResp := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return RunGetResponse{}, &errResp
	}
	if !ok {
		errResp := s.makeError(requestID, "broker_not_found_artifact", "storage", false, fmt.Sprintf("run %q not found", req.RunID))
		return RunGetResponse{}, &errResp
	}
	resp := RunGetResponse{SchemaID: "runecode.protocol.v0.RunGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Run: detail}
	if err := s.validateResponse(resp, runGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return RunGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) prepareLocalRequest(reqID, fallbackReqID string, admissionErr error, req any, schemaPath string) (string, *ErrorResponse) {
	requestID := strings.TrimSpace(resolveRequestID(reqID, fallbackReqID))
	if admissionErr != nil {
		errID := requestID
		if errID == "" {
			errID = defaultRequestIDFallback
		}
		err := s.makeError(errID, "broker_api_auth_admission_denied", "auth", false, admissionErr.Error())
		return "", &err
	}
	if requestID == "" {
		err := s.makeError(defaultRequestIDFallback, "broker_validation_request_id_missing", "validation", false, "request_id is required")
		return "", &err
	}
	if err := s.validateRequest(req, schemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return "", &errOut
	}
	return requestID, nil
}
