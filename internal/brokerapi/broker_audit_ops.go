package brokerapi

import "fmt"

func (s *Service) auditErrorResponse(resp ErrorResponse) ErrorResponse {
	if s == nil || s.auditor == nil || s.store == nil {
		resp.Error.Code = "gateway_failure"
		resp.Error.Category = "internal"
		resp.Error.Retryable = false
		resp.Error.Message = "broker rejection audit path unavailable"
		return resp
	}
	if !shouldAuditErrorCode(resp.Error.Code) {
		return resp
	}
	if err := s.auditor.emitRejection(s.store, resp); err != nil {
		resp.Error.Code = "gateway_failure"
		resp.Error.Category = "internal"
		resp.Error.Retryable = false
		resp.Error.Message = "failed to persist broker rejection audit event"
	}
	return resp
}

func (s *Service) auditApprovalResolution(requestID, approvalID, status, reasonCode string) error {
	if s == nil || s.auditor == nil || s.store == nil {
		return fmt.Errorf("broker approval audit path unavailable for request %q", requestID)
	}
	if err := s.auditor.emitApprovalResolution(s.store, requestID, approvalID, status, reasonCode); err != nil {
		return fmt.Errorf("failed to persist broker approval audit event for request %q: %w", requestID, err)
	}
	return nil
}
