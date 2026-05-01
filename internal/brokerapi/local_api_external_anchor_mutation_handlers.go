package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleExternalAnchorMutationPrepare(ctx context.Context, req ExternalAnchorMutationPrepareRequest, meta RequestContext) (ExternalAnchorMutationPrepareResponse, *ErrorResponse) {
	requestID, requestCtx, cleanup, errResp := s.beginGitRemoteMutationRequest(ctx, req, req.RequestID, meta, externalAnchorMutationPrepareRequestSchemaPath)
	if errResp != nil {
		return ExternalAnchorMutationPrepareResponse{}, errResp
	}
	defer cleanup()

	resolved, errResp := s.resolveExternalAnchorPrepareInput(req, requestID)
	if errResp != nil {
		return ExternalAnchorMutationPrepareResponse{}, errResp
	}
	decision, approvalBinding, policyDecisionHash, errResp := s.evaluatePreparedExternalAnchorMutation(requestCtx, requestID, resolved, req.TypedRequest)
	if errResp != nil {
		return ExternalAnchorMutationPrepareResponse{}, errResp
	}
	record, errResp := s.buildPreparedExternalAnchorMutationRecord(req, requestID, resolved, decision, approvalBinding, policyDecisionHash)
	if errResp != nil {
		return ExternalAnchorMutationPrepareResponse{}, errResp
	}
	return s.persistPreparedExternalAnchorMutationResponse(requestID, record, resolved.typedRequestHash)
}

func (s *Service) HandleExternalAnchorMutationGet(ctx context.Context, req ExternalAnchorMutationGetRequest, meta RequestContext) (ExternalAnchorMutationGetResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.beginGitRemoteMutationRequest(ctx, req, req.RequestID, meta, externalAnchorMutationGetRequestSchemaPath)
	if errResp != nil {
		return ExternalAnchorMutationGetResponse{}, errResp
	}
	defer cleanup()

	preparedMutationID := strings.TrimSpace(req.PreparedMutationID)
	if preparedMutationID == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "prepared_mutation_id is required")
		return ExternalAnchorMutationGetResponse{}, &errOut
	}
	record, ok := s.ExternalAnchorPreparedGet(preparedMutationID)
	if !ok {
		errOut := s.makeError(requestID, "broker_not_found_prepared_mutation", "storage", false, fmt.Sprintf("prepared mutation %q not found", preparedMutationID))
		return ExternalAnchorMutationGetResponse{}, &errOut
	}
	record.LastGetRequestID = requestID
	if err := s.ExternalAnchorPreparedUpsert(record); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return ExternalAnchorMutationGetResponse{}, &errOut
	}
	preparedState, errResp := s.readPreparedExternalAnchorMutationState(requestID, preparedMutationID)
	if errResp != nil {
		return ExternalAnchorMutationGetResponse{}, errResp
	}
	resp := ExternalAnchorMutationGetResponse{
		SchemaID:      "runecode.protocol.v0.ExternalAnchorMutationGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Prepared:      preparedState,
	}
	if err := s.validateResponse(resp, externalAnchorMutationGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ExternalAnchorMutationGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) persistPreparedExternalAnchorMutationResponse(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord, typedRequestHash trustpolicy.Digest) (ExternalAnchorMutationPrepareResponse, *ErrorResponse) {
	if errResp := s.persistExternalAnchorPreparedRecord(requestID, record); errResp != nil {
		return ExternalAnchorMutationPrepareResponse{}, errResp
	}
	preparedState, errResp := s.readPreparedExternalAnchorMutationState(requestID, record.PreparedMutationID)
	if errResp != nil {
		return ExternalAnchorMutationPrepareResponse{}, errResp
	}
	resp := ExternalAnchorMutationPrepareResponse{
		SchemaID:           "runecode.protocol.v0.ExternalAnchorMutationPrepareResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          requestID,
		PreparedMutationID: record.PreparedMutationID,
		TypedRequestHash:   typedRequestHash,
		Prepared:           preparedState,
	}
	if err := s.validateResponse(resp, externalAnchorMutationPrepareResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ExternalAnchorMutationPrepareResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) externalAnchorMutationExecuteResponse(requestID, preparedMutationID string) (ExternalAnchorMutationExecuteResponse, *ErrorResponse) {
	preparedState, errResp := s.readPreparedExternalAnchorMutationState(requestID, preparedMutationID)
	if errResp != nil {
		return ExternalAnchorMutationExecuteResponse{}, errResp
	}
	resp := ExternalAnchorMutationExecuteResponse{
		SchemaID:           "runecode.protocol.v0.ExternalAnchorMutationExecuteResponse",
		SchemaVersion:      "0.1.0",
		RequestID:          requestID,
		PreparedMutationID: preparedMutationID,
		ExecutionState:     preparedState.ExecutionState,
		Prepared:           preparedState,
	}
	if err := s.validateResponse(resp, externalAnchorMutationExecuteResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ExternalAnchorMutationExecuteResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) persistExternalAnchorPreparedRecord(requestID string, record artifacts.ExternalAnchorPreparedMutationRecord) *ErrorResponse {
	if err := s.ExternalAnchorPreparedUpsert(record); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return &errOut
	}
	return nil
}

func (s *Service) readPreparedExternalAnchorMutationState(requestID, preparedMutationID string) (ExternalAnchorMutationPreparedState, *ErrorResponse) {
	record, ok := s.ExternalAnchorPreparedGet(preparedMutationID)
	if !ok {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "prepared mutation unavailable after persistence")
		return ExternalAnchorMutationPreparedState{}, &errOut
	}
	preparedState, err := externalAnchorPreparedStateFromRecord(record)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ExternalAnchorMutationPreparedState{}, &errOut
	}
	return preparedState, nil
}
