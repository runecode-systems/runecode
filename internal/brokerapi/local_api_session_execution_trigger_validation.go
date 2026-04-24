package brokerapi

import "strings"

func (s *Service) validateSessionExecutionTriggerRequest(requestID string, req SessionExecutionTriggerRequest) *ErrorResponse {
	if strings.TrimSpace(req.SessionID) == "" {
		return sessionExecutionTriggerValidationError(s, requestID, "session_id is required")
	}
	if !validSessionTriggerSource(req.TriggerSource) {
		return sessionExecutionTriggerValidationError(s, requestID, "trigger_source is invalid")
	}
	if !validSessionRequestedOperation(req.RequestedOperation) {
		return sessionExecutionTriggerValidationError(s, requestID, "requested_operation is invalid")
	}
	if missingInteractiveSessionTriggerMessage(req) {
		return sessionExecutionTriggerValidationError(s, requestID, "user_message_content_text is required for interactive_user trigger_source")
	}
	if !validSessionApprovalProfile(req.ApprovalProfile) {
		return sessionExecutionTriggerValidationError(s, requestID, "approval_profile is invalid")
	}
	if !validSessionAutonomyPosture(req.AutonomyPosture) {
		return sessionExecutionTriggerValidationError(s, requestID, "autonomy_posture is invalid")
	}
	return nil
}

func sessionExecutionTriggerValidationError(s *Service, requestID, message string) *ErrorResponse {
	errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, message)
	return &errOut
}

func validSessionTriggerSource(source string) bool {
	switch source {
	case "interactive_user", "autonomous_background", "resume_follow_up":
		return true
	default:
		return false
	}
}

func validSessionRequestedOperation(operation string) bool {
	return operation == "start" || operation == "continue"
}

func missingInteractiveSessionTriggerMessage(req SessionExecutionTriggerRequest) bool {
	return req.TriggerSource == "interactive_user" && strings.TrimSpace(req.UserMessageContentText) == ""
}

func validSessionApprovalProfile(profile string) bool {
	trimmed := strings.TrimSpace(profile)
	return trimmed == "" || trimmed == "moderate"
}

func validSessionAutonomyPosture(posture string) bool {
	switch strings.TrimSpace(posture) {
	case "", "operator_guided", "balanced", "autonomous_preferred":
		return true
	default:
		return false
	}
}

func normalizeSessionTriggerApprovalProfile(in string) string {
	if strings.TrimSpace(in) == "" {
		return "moderate"
	}
	return strings.TrimSpace(in)
}

func normalizeSessionTriggerAutonomyPosture(in string) string {
	if strings.TrimSpace(in) == "" {
		return "operator_guided"
	}
	return strings.TrimSpace(in)
}
