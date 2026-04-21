package brokerapi

import (
	"context"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (s *Service) HandleProjectSubstrateGet(ctx context.Context, req ProjectSubstrateGetRequest, meta RequestContext) (ProjectSubstrateGetResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareProjectSubstrateRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, projectSubstrateGetRequestSchemaPath, meta)
	if errResp != nil {
		return ProjectSubstrateGetResponse{}, errResp
	}
	defer cleanup()
	result, errResp := s.refreshProjectSubstrateDiscovery(requestID)
	if errResp != nil {
		return ProjectSubstrateGetResponse{}, errResp
	}
	resp := ProjectSubstrateGetResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, RepositoryRoot: result.RepositoryRoot, Contract: result.Contract, Snapshot: result.Snapshot}
	return validateProjectSubstrateGetResponse(s, requestID, resp)
}

func (s *Service) HandleProjectSubstratePostureGet(ctx context.Context, req ProjectSubstratePostureGetRequest, meta RequestContext) (ProjectSubstratePostureGetResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareProjectSubstrateRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, projectSubstratePostureGetRequestSchemaPath, meta)
	if errResp != nil {
		return ProjectSubstratePostureGetResponse{}, errResp
	}
	defer cleanup()
	result, errResp := s.refreshProjectSubstrateDiscovery(requestID)
	if errResp != nil {
		return ProjectSubstratePostureGetResponse{}, errResp
	}
	lifecycle, errResp := s.projectSubstrateLifecycleProjection(requestID)
	if errResp != nil {
		return ProjectSubstratePostureGetResponse{}, errResp
	}
	summary := s.buildProjectSubstrateSummary(result)
	blockedExplanation, remediation := projectSubstrateBlockedProjection(summary, lifecycle.initPreview, lifecycle.upgradePreview)
	resp := ProjectSubstratePostureGetResponse{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, RepositoryRoot: result.RepositoryRoot, Contract: result.Contract, Snapshot: result.Snapshot, PostureSummary: summary, Adoption: lifecycle.adoption, InitPreview: lifecycle.initPreview, UpgradePreview: lifecycle.upgradePreview, BlockedExplanation: blockedExplanation, RemediationGuidance: remediation}
	return validateProjectSubstratePostureResponse(s, requestID, resp)
}

type projectSubstrateLifecycleState struct {
	adoption       projectsubstrate.AdoptionResult
	initPreview    projectsubstrate.InitPreview
	upgradePreview projectsubstrate.UpgradePreview
}

func (s *Service) projectSubstrateLifecycleProjection(requestID string) (projectSubstrateLifecycleState, *ErrorResponse) {
	repoRoot, authority := s.projectSubstrateOperationInput()
	adoption, err := projectsubstrate.AdoptExisting(projectsubstrate.AdoptionInput{RepositoryRoot: repoRoot, Authority: authority})
	if err != nil {
		return projectSubstrateLifecycleState{}, projectSubstrateGatewayError(s, requestID, err)
	}
	initPreview, err := projectsubstrate.PreviewInitialize(projectsubstrate.InitPreviewInput{RepositoryRoot: repoRoot, Authority: authority})
	if err != nil {
		return projectSubstrateLifecycleState{}, projectSubstrateGatewayError(s, requestID, err)
	}
	upgradePreview, err := projectsubstrate.PreviewUpgrade(projectsubstrate.UpgradePreviewInput{RepositoryRoot: repoRoot, Authority: authority})
	if err != nil {
		return projectSubstrateLifecycleState{}, projectSubstrateGatewayError(s, requestID, err)
	}
	return projectSubstrateLifecycleState{adoption: adoption, initPreview: initPreview, upgradePreview: upgradePreview}, nil
}

func validateProjectSubstrateGetResponse(s *Service, requestID string, resp ProjectSubstrateGetResponse) (ProjectSubstrateGetResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, projectSubstrateGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProjectSubstrateGetResponse{}, &errOut
	}
	return resp, nil
}

func validateProjectSubstratePostureResponse(s *Service, requestID string, resp ProjectSubstratePostureGetResponse) (ProjectSubstratePostureGetResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, projectSubstratePostureGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProjectSubstratePostureGetResponse{}, &errOut
	}
	return resp, nil
}

func projectSubstrateGatewayError(s *Service, requestID string, err error) *ErrorResponse {
	errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
	return &errOut
}
