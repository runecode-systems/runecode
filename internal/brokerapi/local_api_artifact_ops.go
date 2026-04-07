package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) HandleArtifactListV0(ctx context.Context, req LocalArtifactListRequest, meta RequestContext) (LocalArtifactListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactListRequestSchemaPath)
	if errResp != nil {
		return LocalArtifactListResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return LocalArtifactListResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return LocalArtifactListResponse{}, errResp
	}
	order := artifactListOrder(req.Order)
	summaries := filterArtifactSummaries(s.List(), req)
	sortArtifactSummariesByOrder(summaries, order)
	limit := normalizeLimit(req.Limit, 100, 500)
	page, next, err := paginate(summaries, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return LocalArtifactListResponse{}, &errOut
	}
	resp := LocalArtifactListResponse{SchemaID: "runecode.protocol.v0.ArtifactListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Artifacts: page, NextCursor: next}
	if err := s.validateResponse(resp, artifactListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return LocalArtifactListResponse{}, &errOut
	}
	return resp, nil
}

func artifactListOrder(order string) string {
	if order == "" {
		return "created_at_desc"
	}
	return order
}

func sortArtifactSummariesByOrder(items []ArtifactSummary, order string) {
	if order == "created_at_asc" {
		sortArtifactSummariesOldestFirst(items)
		return
	}
	sortArtifactSummariesNewestFirst(items)
}

func filterArtifactSummaries(all []artifacts.ArtifactRecord, req LocalArtifactListRequest) []ArtifactSummary {
	summaries := make([]ArtifactSummary, 0, len(all))
	for _, record := range all {
		summary := toArtifactSummary(record)
		if req.RunID != "" && summary.RunID != req.RunID {
			continue
		}
		if req.StepID != "" && summary.StepID != req.StepID {
			continue
		}
		if req.DataClass != "" && string(summary.Reference.DataClass) != req.DataClass {
			continue
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func (s *Service) HandleArtifactHeadV0(ctx context.Context, req LocalArtifactHeadRequest, meta RequestContext) (LocalArtifactHeadResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactHeadRequestSchemaPath)
	if errResp != nil {
		return LocalArtifactHeadResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return LocalArtifactHeadResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return LocalArtifactHeadResponse{}, errResp
	}
	record, err := s.Head(req.Digest)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return LocalArtifactHeadResponse{}, &errOut
	}
	resp := LocalArtifactHeadResponse{SchemaID: "runecode.protocol.v0.ArtifactHeadResponse", SchemaVersion: "0.1.0", RequestID: requestID, Artifact: toArtifactSummary(record)}
	if err := s.validateResponse(resp, artifactHeadResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return LocalArtifactHeadResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleArtifactRead(ctx context.Context, req ArtifactReadRequest, meta RequestContext) (ArtifactReadHandle, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactReadRequestSchemaPath)
	if errResp != nil {
		return ArtifactReadHandle{}, errResp
	}
	release, requestCtx, cancel, errResp := s.prepareArtifactReadExecution(ctx, requestID, meta)
	if errResp != nil {
		return ArtifactReadHandle{}, errResp
	}
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		release()
		cancel()
		return ArtifactReadHandle{}, errResp
	}
	normalizedReq, errResp := s.normalizeArtifactReadRequest(requestID, req)
	if errResp != nil {
		release()
		cancel()
		return ArtifactReadHandle{}, errResp
	}
	r, record, err := s.GetForFlow(s.artifactReadRequestToStore(normalizedReq))
	if err != nil {
		release()
		cancel()
		errOut := s.errorFromStore(requestID, err)
		return ArtifactReadHandle{}, &errOut
	}
	return ArtifactReadHandle{RequestID: requestID, Digest: normalizedReq.Digest, DataClass: record.Reference.DataClass, StreamID: normalizedReq.StreamID, ChunkBytes: normalizedReq.ChunkBytes, Reader: r, RequestCtx: requestCtx, Cancel: cancel, Release: release}, nil
}

func (s *Service) prepareArtifactReadExecution(ctx context.Context, requestID string, meta RequestContext) (func(), context.Context, context.CancelFunc, *ErrorResponse) {
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return nil, nil, nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	return release, requestCtx, cancel, nil
}

func (s *Service) normalizeArtifactReadRequest(requestID string, req ArtifactReadRequest) (ArtifactReadRequest, *ErrorResponse) {
	if strings.TrimSpace(req.ProducerRole) == "" || strings.TrimSpace(req.ConsumerRole) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "producer_role and consumer_role are required")
		return ArtifactReadRequest{}, &errOut
	}
	if req.RangeStart != nil || req.RangeEnd != nil {
		errOut := s.makeError(requestID, "broker_validation_range_not_supported", "validation", false, "range_start/range_end are not supported for MVP artifact reads")
		return ArtifactReadRequest{}, &errOut
	}
	if req.StreamID == "" {
		req.StreamID = "artifact-read-" + requestID
	}
	if req.ChunkBytes <= 0 || req.ChunkBytes > s.apiConfig.Limits.MaxStreamChunkBytes {
		req.ChunkBytes = s.apiConfig.Limits.MaxStreamChunkBytes
	}
	return req, nil
}

func (s *Service) artifactReadRequestToStore(req ArtifactReadRequest) artifacts.ArtifactReadRequest {
	return artifacts.ArtifactReadRequest{
		Digest:        req.Digest,
		ProducerRole:  req.ProducerRole,
		ConsumerRole:  req.ConsumerRole,
		DataClass:     artifacts.DataClass(req.DataClass),
		IsEgress:      true,
		ManifestOptIn: req.ManifestOptIn,
	}
}
