package brokerapi

import (
	"context"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (s *Service) HandleProjectSubstrateAdopt(ctx context.Context, req ProjectSubstrateAdoptRequest, meta RequestContext) (ProjectSubstrateAdoptResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareProjectSubstrateRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, projectSubstrateAdoptRequestSchemaPath, meta)
	if errResp != nil {
		return ProjectSubstrateAdoptResponse{}, errResp
	}
	defer cleanup()
	repoRoot, authority := s.projectSubstrateOperationInput()
	adoption, err := projectsubstrate.AdoptExisting(projectsubstrate.AdoptionInput{RepositoryRoot: repoRoot, Authority: authority})
	if err != nil {
		return ProjectSubstrateAdoptResponse{}, projectSubstrateGatewayError(s, requestID, err)
	}
	result, errResp := s.refreshProjectSubstrateDiscovery(requestID)
	if errResp != nil {
		return ProjectSubstrateAdoptResponse{}, errResp
	}
	adoption.RepositoryRoot = result.RepositoryRoot
	adoption.Snapshot = result.Snapshot
	resp := ProjectSubstrateAdoptResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateAdoptResponse", SchemaVersion: "0.1.0", RequestID: requestID, Adoption: adoption}
	return validateProjectSubstrateAdoptResponse(s, requestID, resp)
}

func (s *Service) HandleProjectSubstrateInitPreview(ctx context.Context, req ProjectSubstrateInitPreviewRequest, meta RequestContext) (ProjectSubstrateInitPreviewResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareProjectSubstrateRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, projectSubstrateInitPreviewRequestSchemaPath, meta)
	if errResp != nil {
		return ProjectSubstrateInitPreviewResponse{}, errResp
	}
	defer cleanup()
	preview, errResp := s.projectSubstrateInitPreview(requestID)
	if errResp != nil {
		return ProjectSubstrateInitPreviewResponse{}, errResp
	}
	resp := ProjectSubstrateInitPreviewResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateInitPreviewResponse", SchemaVersion: "0.1.0", RequestID: requestID, Preview: preview}
	return validateProjectSubstrateInitPreviewResponse(s, requestID, resp)
}

func (s *Service) HandleProjectSubstrateInitApply(ctx context.Context, req ProjectSubstrateInitApplyRequest, meta RequestContext) (ProjectSubstrateInitApplyResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareProjectSubstrateRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, projectSubstrateInitApplyRequestSchemaPath, meta)
	if errResp != nil {
		return ProjectSubstrateInitApplyResponse{}, errResp
	}
	defer cleanup()
	preview, errResp := s.projectSubstrateInitPreview(requestID)
	if errResp != nil {
		return ProjectSubstrateInitApplyResponse{}, errResp
	}
	applyResult, err := projectsubstrate.ApplyInitialize(projectsubstrate.InitApplyInput{Preview: preview, ExpectedPreviewToken: req.ExpectedPreviewToken})
	if err != nil {
		return ProjectSubstrateInitApplyResponse{}, projectSubstrateGatewayError(s, requestID, err)
	}
	if _, refreshErr := s.refreshProjectSubstrateDiscovery(requestID); refreshErr != nil {
		return ProjectSubstrateInitApplyResponse{}, refreshErr
	}
	resp := ProjectSubstrateInitApplyResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateInitApplyResponse", SchemaVersion: "0.1.0", RequestID: requestID, ApplyResult: applyResult}
	return validateProjectSubstrateInitApplyResponse(s, requestID, resp)
}

func (s *Service) HandleProjectSubstrateUpgradePreview(ctx context.Context, req ProjectSubstrateUpgradePreviewRequest, meta RequestContext) (ProjectSubstrateUpgradePreviewResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareProjectSubstrateRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, projectSubstrateUpgradePreviewRequestSchemaPath, meta)
	if errResp != nil {
		return ProjectSubstrateUpgradePreviewResponse{}, errResp
	}
	defer cleanup()
	preview, errResp := s.projectSubstrateUpgradePreview(requestID)
	if errResp != nil {
		return ProjectSubstrateUpgradePreviewResponse{}, errResp
	}
	resp := ProjectSubstrateUpgradePreviewResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradePreviewResponse", SchemaVersion: "0.1.0", RequestID: requestID, Preview: preview}
	return validateProjectSubstrateUpgradePreviewResponse(s, requestID, resp)
}

func (s *Service) HandleProjectSubstrateUpgradeApply(ctx context.Context, req ProjectSubstrateUpgradeApplyRequest, meta RequestContext) (ProjectSubstrateUpgradeApplyResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareProjectSubstrateRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, projectSubstrateUpgradeApplyRequestSchemaPath, meta)
	if errResp != nil {
		return ProjectSubstrateUpgradeApplyResponse{}, errResp
	}
	defer cleanup()
	preview, errResp := s.projectSubstrateUpgradePreview(requestID)
	if errResp != nil {
		return ProjectSubstrateUpgradeApplyResponse{}, errResp
	}
	applyResult, err := projectsubstrate.ApplyUpgrade(projectsubstrate.UpgradeApplyInput{Preview: preview, ExpectedPreviewHash: req.ExpectedPreviewDigest, AuditAppender: s})
	if err != nil {
		return ProjectSubstrateUpgradeApplyResponse{}, projectSubstrateGatewayError(s, requestID, err)
	}
	if _, refreshErr := s.refreshProjectSubstrateDiscovery(requestID); refreshErr != nil {
		return ProjectSubstrateUpgradeApplyResponse{}, refreshErr
	}
	resp := ProjectSubstrateUpgradeApplyResponse{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradeApplyResponse", SchemaVersion: "0.1.0", RequestID: requestID, ApplyResult: applyResult}
	return validateProjectSubstrateUpgradeApplyResponse(s, requestID, resp)
}

func (s *Service) projectSubstrateInitPreview(requestID string) (projectsubstrate.InitPreview, *ErrorResponse) {
	repoRoot, authority := s.projectSubstrateOperationInput()
	preview, err := projectsubstrate.PreviewInitialize(projectsubstrate.InitPreviewInput{RepositoryRoot: repoRoot, Authority: authority})
	if err != nil {
		return projectsubstrate.InitPreview{}, projectSubstrateGatewayError(s, requestID, err)
	}
	return preview, nil
}

func (s *Service) projectSubstrateUpgradePreview(requestID string) (projectsubstrate.UpgradePreview, *ErrorResponse) {
	repoRoot, authority := s.projectSubstrateOperationInput()
	preview, err := projectsubstrate.PreviewUpgrade(projectsubstrate.UpgradePreviewInput{RepositoryRoot: repoRoot, Authority: authority})
	if err != nil {
		return projectsubstrate.UpgradePreview{}, projectSubstrateGatewayError(s, requestID, err)
	}
	return preview, nil
}

func validateProjectSubstrateAdoptResponse(s *Service, requestID string, resp ProjectSubstrateAdoptResponse) (ProjectSubstrateAdoptResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, projectSubstrateAdoptResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProjectSubstrateAdoptResponse{}, &errOut
	}
	return resp, nil
}

func validateProjectSubstrateInitPreviewResponse(s *Service, requestID string, resp ProjectSubstrateInitPreviewResponse) (ProjectSubstrateInitPreviewResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, projectSubstrateInitPreviewResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProjectSubstrateInitPreviewResponse{}, &errOut
	}
	return resp, nil
}

func validateProjectSubstrateInitApplyResponse(s *Service, requestID string, resp ProjectSubstrateInitApplyResponse) (ProjectSubstrateInitApplyResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, projectSubstrateInitApplyResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProjectSubstrateInitApplyResponse{}, &errOut
	}
	return resp, nil
}

func validateProjectSubstrateUpgradePreviewResponse(s *Service, requestID string, resp ProjectSubstrateUpgradePreviewResponse) (ProjectSubstrateUpgradePreviewResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, projectSubstrateUpgradePreviewResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProjectSubstrateUpgradePreviewResponse{}, &errOut
	}
	return resp, nil
}

func validateProjectSubstrateUpgradeApplyResponse(s *Service, requestID string, resp ProjectSubstrateUpgradeApplyResponse) (ProjectSubstrateUpgradeApplyResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, projectSubstrateUpgradeApplyResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ProjectSubstrateUpgradeApplyResponse{}, &errOut
	}
	return resp, nil
}
