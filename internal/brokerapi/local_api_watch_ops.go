package brokerapi

import "context"

func (s *Service) HandleRunWatchRequest(ctx context.Context, req RunWatchRequest, meta RequestContext) (RunWatchRequest, *ErrorResponse) {
	if req.StreamID == "" {
		req.StreamID = "run-watch-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, runWatchRequestSchemaPath)
	if errResp != nil {
		return RunWatchRequest{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return RunWatchRequest{}, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		release()
		cancel()
		return RunWatchRequest{}, errResp
	}
	ack := req
	ack.RequestID = requestID
	if ack.StreamID == "" {
		ack.StreamID = "run-watch-" + requestID
	}
	ack.RequestCtx = requestCtx
	ack.Cancel = cancel
	ack.Release = release
	return ack, nil
}

func (s *Service) HandleApprovalWatchRequest(ctx context.Context, req ApprovalWatchRequest, meta RequestContext) (ApprovalWatchRequest, *ErrorResponse) {
	if req.StreamID == "" {
		req.StreamID = "approval-watch-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, approvalWatchRequestSchemaPath)
	if errResp != nil {
		return ApprovalWatchRequest{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ApprovalWatchRequest{}, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		release()
		cancel()
		return ApprovalWatchRequest{}, errResp
	}
	ack := req
	ack.RequestID = requestID
	if ack.StreamID == "" {
		ack.StreamID = "approval-watch-" + requestID
	}
	ack.RequestCtx = requestCtx
	ack.Cancel = cancel
	ack.Release = release
	return ack, nil
}

func (s *Service) HandleSessionWatchRequest(ctx context.Context, req SessionWatchRequest, meta RequestContext) (SessionWatchRequest, *ErrorResponse) {
	if req.StreamID == "" {
		req.StreamID = "session-watch-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, sessionWatchRequestSchemaPath)
	if errResp != nil {
		return SessionWatchRequest{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionWatchRequest{}, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		release()
		cancel()
		return SessionWatchRequest{}, errResp
	}
	ack := req
	ack.RequestID = requestID
	if ack.StreamID == "" {
		ack.StreamID = "session-watch-" + requestID
	}
	ack.RequestCtx = requestCtx
	ack.Cancel = cancel
	ack.Release = release
	return ack, nil
}

func (s *Service) HandleSessionTurnExecutionWatchRequest(ctx context.Context, req SessionTurnExecutionWatchRequest, meta RequestContext) (SessionTurnExecutionWatchRequest, *ErrorResponse) {
	if req.StreamID == "" {
		req.StreamID = "session-turn-execution-watch-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, sessionTurnExecutionWatchRequestSchemaPath)
	if errResp != nil {
		return SessionTurnExecutionWatchRequest{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return SessionTurnExecutionWatchRequest{}, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	if errResp := s.requestContextError(requestID, requestCtx); errResp != nil {
		release()
		cancel()
		return SessionTurnExecutionWatchRequest{}, errResp
	}
	ack := req
	ack.RequestID = requestID
	if ack.StreamID == "" {
		ack.StreamID = "session-turn-execution-watch-" + requestID
	}
	ack.RequestCtx = requestCtx
	ack.Cancel = cancel
	ack.Release = release
	return ack, nil
}
