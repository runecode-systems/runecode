package brokerapi

import "context"

func (s *Service) HandleDependencyCacheEnsure(ctx context.Context, req DependencyCacheEnsureRequest, meta RequestContext) (DependencyCacheEnsureResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, dependencyCacheEnsureRequestSchemaPath)
	if errResp != nil {
		return DependencyCacheEnsureResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return DependencyCacheEnsureResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return DependencyCacheEnsureResponse{}, errResp
	}
	resp, err := s.dependencyFetchService.EnsureBatch(requestCtx, requestID, req)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return DependencyCacheEnsureResponse{}, &errOut
	}
	if err := s.validateResponse(resp, dependencyCacheEnsureResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return DependencyCacheEnsureResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleDependencyFetchRegistry(ctx context.Context, req DependencyFetchRegistryRequest, meta RequestContext) (DependencyFetchRegistryResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, dependencyFetchRegistryRequestSchemaPath)
	if errResp != nil {
		return DependencyFetchRegistryResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return DependencyFetchRegistryResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		return DependencyFetchRegistryResponse{}, errResp
	}
	resp, err := s.dependencyFetchService.FetchSingle(requestCtx, requestID, req)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return DependencyFetchRegistryResponse{}, &errOut
	}
	if err := s.validateResponse(resp, dependencyFetchRegistryResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return DependencyFetchRegistryResponse{}, &errOut
	}
	return resp, nil
}
