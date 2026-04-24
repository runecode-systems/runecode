package brokerapi

func (s *Service) buildRunnerCheckpointResponseFromCanonical(requestID, runID, idempotencyKey string, accepted bool) (RunnerCheckpointReportResponse, *ErrorResponse) {
	canonical, _, err := s.currentCanonicalLifecycleForRun(runID)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerCheckpointReportResponse{}, &errOut
	}
	return s.buildRunnerCheckpointReportResponse(requestID, runID, canonical, idempotencyKey, accepted)
}

func (s *Service) buildRunnerResultResponseFromCanonical(requestID, runID, idempotencyKey string, accepted bool) (RunnerResultReportResponse, *ErrorResponse) {
	canonical, _, err := s.currentCanonicalLifecycleForRun(runID)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return RunnerResultReportResponse{}, &errOut
	}
	return s.buildRunnerResultReportResponse(requestID, runID, canonical, idempotencyKey, accepted)
}
