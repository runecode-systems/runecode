package brokerapi

import "strings"

const (
	sessionWorkflowOperationChangeDraft            = "change_draft"
	sessionWorkflowOperationSpecDraft              = "spec_draft"
	sessionWorkflowOperationDraftPromoteApply      = "draft_promote_apply"
	sessionWorkflowOperationApprovedImplementation = "approved_change_implementation"
)

func (s *Service) validateSessionExecutionTriggerRequest(requestID string, req SessionExecutionTriggerRequest) *ErrorResponse {
	if strings.TrimSpace(req.SessionID) == "" {
		return sessionExecutionTriggerValidationError(s, requestID, "session_id is required")
	}
	if strings.TrimSpace(req.RequestedOperation) != "continue" && strings.TrimSpace(req.TurnID) != "" {
		return sessionExecutionTriggerValidationError(s, requestID, "turn_id is only allowed for continue requests")
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
	if !validSessionWorkflowRouting(req.RequestedOperation, req.WorkflowRouting) {
		return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing is invalid")
	}
	if errResp := s.validateSessionWorkflowRoutingSemantics(requestID, req); errResp != nil {
		return errResp
	}
	return nil
}

func (s *Service) validateSessionWorkflowRoutingSemantics(requestID string, req SessionExecutionTriggerRequest) *ErrorResponse {
	routing := req.WorkflowRouting
	if routing == nil {
		return nil
	}
	if allowEmptyContinueRouting(strings.TrimSpace(req.RequestedOperation), routing) {
		return nil
	}
	switch strings.TrimSpace(routing.WorkflowOperation) {
	case sessionWorkflowOperationChangeDraft, sessionWorkflowOperationSpecDraft:
		return validateArtifactOnlyDraftRouting(s, requestID, routing)
	case sessionWorkflowOperationDraftPromoteApply:
		return validateDraftPromoteRouting(s, requestID, routing)
	case sessionWorkflowOperationApprovedImplementation:
		return s.validateApprovedImplementationRouting(requestID, routing)
	default:
		return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing.workflow_operation is unsupported")
	}
}

func validateArtifactOnlyDraftRouting(s *Service, requestID string, routing *SessionWorkflowPackRouting) *ErrorResponse {
	if len(routing.BoundInputArtifacts) != 0 {
		return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing bound_input_artifacts are not allowed for artifact-only draft operations")
	}
	return nil
}

func validateDraftPromoteRouting(s *Service, requestID string, routing *SessionWorkflowPackRouting) *ErrorResponse {
	for _, artifact := range routing.BoundInputArtifacts {
		ref := strings.TrimSpace(artifact.ArtifactRef)
		if ref != "change_draft_artifact" && ref != "spec_draft_artifact" {
			return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing draft_promote_apply only accepts change_draft_artifact/spec_draft_artifact bindings")
		}
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

func validSessionWorkflowRouting(requestedOperation string, routing *SessionWorkflowPackRouting) bool {
	op := strings.TrimSpace(requestedOperation)
	if routing == nil {
		return op == "continue"
	}
	if allowEmptyContinueRouting(op, routing) {
		return true
	}
	if !hasValidWorkflowRoutingHeader(routing) {
		return false
	}
	if !validSessionWorkflowFamily(routing.WorkflowFamily) || !validSessionWorkflowOperation(routing.WorkflowOperation) {
		return false
	}
	return hasValidWorkflowRoutingArtifacts(routing.BoundInputArtifacts)
}

func allowEmptyContinueRouting(requestedOperation string, routing *SessionWorkflowPackRouting) bool {
	return requestedOperation == "continue" && hasValidWorkflowRoutingHeader(routing) && strings.TrimSpace(routing.WorkflowFamily) == "" && strings.TrimSpace(routing.WorkflowOperation) == "" && len(routing.BoundInputArtifacts) == 0
}

func hasValidWorkflowRoutingHeader(routing *SessionWorkflowPackRouting) bool {
	if strings.TrimSpace(routing.SchemaID) != "runecode.protocol.v0.SessionWorkflowPackRouting" {
		return false
	}
	return strings.TrimSpace(routing.SchemaVersion) == "0.1.0"
}

func hasValidWorkflowRoutingArtifacts(artifactsIn []SessionWorkflowPackBoundInputArtifact) bool {
	for _, artifact := range artifactsIn {
		if strings.TrimSpace(artifact.ArtifactRef) == "" || strings.TrimSpace(artifact.ArtifactDigest) == "" {
			return false
		}
	}
	return true
}

func validSessionWorkflowFamily(family string) bool {
	return strings.TrimSpace(family) == "runecontext"
}

func validSessionWorkflowOperation(operation string) bool {
	switch strings.TrimSpace(operation) {
	case sessionWorkflowOperationChangeDraft, sessionWorkflowOperationSpecDraft, sessionWorkflowOperationDraftPromoteApply, sessionWorkflowOperationApprovedImplementation:
		return true
	default:
		return false
	}
}
