package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
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
	postures := deriveRecordVerificationPosturesFromFindings(surface.Report.Findings)
	resp := AuditTimelineResponse{SchemaID: "runecode.protocol.v0.AuditTimelineResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Views: s.projectAuditTimelineEntries(page, postures), NextCursor: next}
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
	limit = normalizeLimit(limit, 50, 500)
	surface, err := s.LatestAuditVerificationSurface(limit)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return AuditVerificationGetResponse{}, &errOut
	}
	resp := AuditVerificationGetResponse{SchemaID: "runecode.protocol.v0.AuditVerificationGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Summary: surface.Summary, Report: surface.Report, Views: surface.Views}
	resp.ProjectContextID = strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
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
	readiness, err := s.readinessBase(requestID)
	if err != nil {
		return ReadinessGetResponse{}, err
	}
	model := s.buildReadinessModel(readiness)
	resp := ReadinessGetResponse{SchemaID: "runecode.protocol.v0.ReadinessGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Readiness: model}
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ReadinessGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) readinessBase(requestID string) (trustpolicy.AuditdReadiness, *ErrorResponse) {
	if _, err := s.discoverProjectSubstrate(); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return trustpolicy.AuditdReadiness{}, &errOut
	}
	readiness, err := s.AuditReadiness()
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return trustpolicy.AuditdReadiness{}, &errOut
	}
	return readiness, nil
}

func (s *Service) buildReadinessModel(readiness trustpolicy.AuditdReadiness) BrokerReadiness {
	model := BrokerReadiness{SchemaID: "runecode.protocol.v0.BrokerReadiness", SchemaVersion: "0.1.0", Ready: readiness.Ready, LocalOnly: readiness.LocalOnly, ConsumptionChannel: readiness.ConsumptionChannel, RecoveryComplete: readiness.RecoveryComplete, AppendPositionStable: readiness.AppendPositionStable, CurrentSegmentWritable: readiness.CurrentSegmentWritable, VerifierMaterialAvailable: readiness.VerifierMaterialAvailable, DerivedIndexCaughtUp: readiness.DerivedIndexCaughtUp}
	s.applySecretsReadiness(&model)
	s.applyModelGatewayReadiness(&model)
	model.ProviderProfiles = s.projectProviderProfilesForReadiness()
	model.ProjectSubstrateSummary = s.projectSubstrateSummaryForReadiness()
	if !s.projectSubstrate.Compatibility.NormalOperationAllowed {
		model.Ready = false
	}
	return model
}

func (s *Service) applySecretsReadiness(model *BrokerReadiness) {
	if model == nil {
		return
	}
	model.SecretsReady, model.SecretsHealthState, model.SecretsOperationalMetrics, model.SecretsStoragePosture = projectSecretsReadinessFromLocalState()
	if !model.SecretsReady {
		model.Ready = false
	}
}

func (s *Service) applyModelGatewayReadiness(model *BrokerReadiness) {
	if model == nil {
		return
	}
	model.ModelGatewayReady, model.ModelGatewayHealthState, model.ModelGatewayPosture = s.projectModelGatewayPostureForReadiness()
	if !model.ModelGatewayReady || model.ModelGatewayHealthState == "failed" || model.ModelGatewayHealthState == "degraded" {
		model.Ready = false
	}
}

func (s *Service) projectSubstrateSummaryForReadiness() *ProjectSubstratePostureSummary {
	snapshot := s.projectSubstrate.Snapshot
	if snapshot.SchemaID == "" {
		return nil
	}
	summary := s.buildProjectSubstrateSummary(s.projectSubstrate)
	return &summary
}

func (s *Service) buildProjectSubstrateSummary(result projectsubstrate.DiscoveryResult) ProjectSubstratePostureSummary {
	snapshot := result.Snapshot
	return ProjectSubstratePostureSummary{
		SchemaID:                     "runecode.protocol.v0.ProjectSubstratePostureSummary",
		SchemaVersion:                "0.1.0",
		ActiveContractID:             snapshot.Contract.ContractID,
		ActiveContractVersion:        snapshot.Contract.ContractVersion,
		ActiveRuneContextVersion:     snapshot.RuneContextVersion,
		ContractID:                   snapshot.Contract.ContractID,
		ContractVersion:              snapshot.Contract.ContractVersion,
		ValidationState:              snapshot.ValidationState,
		CompatibilityPosture:         result.Compatibility.Posture,
		NormalOperationAllowed:       result.Compatibility.NormalOperationAllowed,
		SupportedContractVersionMin:  result.Compatibility.Policy.SupportedContractVersionMin,
		SupportedContractVersionMax:  result.Compatibility.Policy.SupportedContractVersionMax,
		RecommendedContractVersion:   result.Compatibility.Policy.RecommendedContractVersion,
		SupportedRuneContextMin:      result.Compatibility.Policy.SupportedRuneContextVersionMin,
		SupportedRuneContextMax:      result.Compatibility.Policy.SupportedRuneContextVersionMax,
		RecommendedRuneContextTarget: result.Compatibility.Policy.RecommendedRuneContextVersion,
		ReasonCodes:                  append([]string{}, result.Compatibility.ReasonCodes...),
		BlockedReasonCodes:           append([]string{}, result.Compatibility.BlockedReasonCodes...),
		ValidatedSnapshotDigest:      snapshot.ValidatedSnapshotDigest,
		ProjectContextIdentityDigest: snapshot.ProjectContextIdentityDigest,
	}
}

func (s *Service) projectProviderProfilesForReadiness() []ProviderProfile {
	profiles := s.providerSubstrate.snapshotProfiles()
	for i := range profiles {
		profiles[i] = profiles[i].projected()
	}
	return profiles
}

func (s *Service) HandleVersionInfoGet(ctx context.Context, req VersionInfoGetRequest, meta RequestContext) (VersionInfoGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, versionInfoGetRequestSchemaPath)
	if errResp != nil {
		return VersionInfoGetResponse{}, errResp
	}
	if _, err := s.discoverProjectSubstrate(); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return VersionInfoGetResponse{}, &errOut
	}
	resp := VersionInfoGetResponse{SchemaID: "runecode.protocol.v0.VersionInfoGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, VersionInfo: s.versionInfo}
	resp.VersionInfo.ProjectSubstrateContractID = s.projectSubstrate.Contract.ContractID
	resp.VersionInfo.ProjectSubstrateContractVersion = s.projectSubstrate.Contract.ContractVersion
	resp.VersionInfo.ProjectSubstrateVersion = s.projectSubstrate.Snapshot.RuneContextVersion
	resp.VersionInfo.ProjectSubstrateValidationState = s.projectSubstrate.Snapshot.ValidationState
	resp.VersionInfo.ProjectSubstratePosture = s.projectSubstrate.Compatibility.Posture
	resp.VersionInfo.ProjectSubstrateBlockedReasons = append([]string{}, s.projectSubstrate.Compatibility.BlockedReasonCodes...)
	resp.VersionInfo.ProjectSubstrateSupportedMin = s.projectSubstrate.Compatibility.Policy.SupportedRuneContextVersionMin
	resp.VersionInfo.ProjectSubstrateSupportedMax = s.projectSubstrate.Compatibility.Policy.SupportedRuneContextVersionMax
	resp.VersionInfo.ProjectSubstrateRecommended = s.projectSubstrate.Compatibility.Policy.RecommendedRuneContextVersion
	resp.VersionInfo.ProjectContextIdentityDigest = s.projectSubstrate.Snapshot.ProjectContextIdentityDigest
	summary := s.buildProjectSubstrateSummary(s.projectSubstrate)
	resp.VersionInfo.ProjectSubstratePostureSummary = &summary
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
