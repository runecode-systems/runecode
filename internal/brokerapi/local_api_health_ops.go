package brokerapi

import (
	"context"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleAuditTimeline(ctx context.Context, req AuditTimelineRequest, meta RequestContext) (AuditTimelineResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditTimelineRequestSchemaPath)
	if errResp != nil {
		return AuditTimelineResponse{}, errResp
	}
	order := req.Order
	if order == "" {
		order = "operational_seq_asc"
	}
	surface, err := s.LatestAuditVerificationSurface(1000)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return AuditTimelineResponse{}, &errOut
	}
	views := append([]trustpolicy.AuditOperationalView{}, surface.Views...)
	if order == "operational_seq_desc" {
		reverseViews(views)
	}
	limit := normalizeLimit(req.Limit, 100, 500)
	page, next, err := paginate(views, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditTimelineResponse{}, &errOut
	}
	resp := AuditTimelineResponse{SchemaID: "runecode.protocol.v0.AuditTimelineResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Views: page, NextCursor: next}
	if err := s.validateResponse(resp, auditTimelineResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditTimelineResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleAuditVerificationGet(ctx context.Context, req AuditVerificationGetRequest, meta RequestContext) (AuditVerificationGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditVerificationGetRequestSchemaPath)
	if errResp != nil {
		return AuditVerificationGetResponse{}, errResp
	}
	limit := req.ViewLimit
	if limit <= 0 {
		limit = 50
	}
	surface, err := s.LatestAuditVerificationSurface(limit)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return AuditVerificationGetResponse{}, &errOut
	}
	resp := AuditVerificationGetResponse{SchemaID: "runecode.protocol.v0.AuditVerificationGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Summary: surface.Summary, Report: surface.Report, Views: surface.Views}
	if err := s.validateResponse(resp, auditVerificationGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditVerificationGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleReadinessGet(ctx context.Context, req ReadinessGetRequest, meta RequestContext) (ReadinessGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, readinessGetRequestSchemaPath)
	if errResp != nil {
		return ReadinessGetResponse{}, errResp
	}
	readiness, err := s.AuditReadiness()
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ReadinessGetResponse{}, &errOut
	}
	model := BrokerReadiness{
		SchemaID:                  "runecode.protocol.v0.BrokerReadiness",
		SchemaVersion:             "0.1.0",
		Ready:                     readiness.Ready,
		LocalOnly:                 readiness.LocalOnly,
		ConsumptionChannel:        readiness.ConsumptionChannel,
		RecoveryComplete:          readiness.RecoveryComplete,
		AppendPositionStable:      readiness.AppendPositionStable,
		CurrentSegmentWritable:    readiness.CurrentSegmentWritable,
		VerifierMaterialAvailable: readiness.VerifierMaterialAvailable,
		DerivedIndexCaughtUp:      readiness.DerivedIndexCaughtUp,
	}
	resp := ReadinessGetResponse{SchemaID: "runecode.protocol.v0.ReadinessGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Readiness: model}
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ReadinessGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleVersionInfoGet(ctx context.Context, req VersionInfoGetRequest, meta RequestContext) (VersionInfoGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, versionInfoGetRequestSchemaPath)
	if errResp != nil {
		return VersionInfoGetResponse{}, errResp
	}
	resp := VersionInfoGetResponse{SchemaID: "runecode.protocol.v0.VersionInfoGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, VersionInfo: s.versionInfo}
	if err := s.validateResponse(resp, versionInfoGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return VersionInfoGetResponse{}, &errOut
	}
	return resp, nil
}

func reverseViews(views []trustpolicy.AuditOperationalView) {
	for i, j := 0, len(views)-1; i < j; i, j = i+1, j-1 {
		views[i], views[j] = views[j], views[i]
	}
}
