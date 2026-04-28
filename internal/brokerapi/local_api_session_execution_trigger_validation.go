package brokerapi

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/runplan"
)

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

func (s *Service) validateApprovedImplementationRouting(requestID string, routing *SessionWorkflowPackRouting) *ErrorResponse {
	inputSetDigest := ""
	for _, artifact := range routing.BoundInputArtifacts {
		if strings.TrimSpace(artifact.ArtifactRef) == "implementation_input_set" {
			inputSetDigest = strings.TrimSpace(artifact.ArtifactDigest)
			break
		}
	}
	if inputSetDigest == "" {
		return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing approved_change_implementation requires implementation_input_set artifact binding")
	}
	return s.validateApprovedImplementationIdentityTuple(requestID, inputSetDigest)
}

func (s *Service) validateApprovedImplementationIdentityTuple(requestID, inputSetDigest string) *ErrorResponse {
	decoded, errResp := s.decodeApprovedImplementationInputSet(requestID, inputSetDigest)
	if errResp != nil {
		return errResp
	}
	if !matchesBoundInputSetDigest(decoded, inputSetDigest) {
		return sessionExecutionTriggerValidationError(s, requestID, "implementation_input_set input_set_digest does not match bound artifact digest")
	}
	if errResp := validateApprovedImplementationCatalogBinding(s, requestID, decoded); errResp != nil {
		return errResp
	}
	project, errResp := s.requireSupportedProjectSubstrateForSessionExecution(requestID)
	if errResp != nil {
		return errResp
	}
	validatedDigest, _ := decoded["validated_project_substrate_digest"].(string)
	if strings.TrimSpace(validatedDigest) != strings.TrimSpace(sessionExecutionBoundDigest(project)) {
		return sessionExecutionTriggerValidationError(s, requestID, "implementation_input_set validated_project_substrate_digest drift detected")
	}
	return nil
}

func (s *Service) decodeApprovedImplementationInputSet(requestID, inputSetDigest string) (map[string]any, *ErrorResponse) {
	payload, err := s.readArtifactPayload(inputSetDigest)
	if err != nil {
		return nil, sessionExecutionTriggerValidationError(s, requestID, "workflow_routing implementation_input_set artifact is unreadable")
	}
	if err := artifacts.ValidateObjectPayloadAgainstSchema(payload, "objects/RuneContextApprovedImplementationInputSet.schema.json"); err != nil {
		return nil, sessionExecutionTriggerValidationError(s, requestID, "workflow_routing implementation_input_set payload is invalid")
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, sessionExecutionTriggerValidationError(s, requestID, "workflow_routing implementation_input_set payload decode failed")
	}
	return decoded, nil
}

func matchesBoundInputSetDigest(decoded map[string]any, inputSetDigest string) bool {
	inputSetField, _ := decoded["input_set_digest"].(string)
	return strings.TrimSpace(inputSetField) == strings.TrimSpace(inputSetDigest)
}

func validateApprovedImplementationCatalogBinding(s *Service, requestID string, decoded map[string]any) *ErrorResponse {
	workflowHash, _ := decoded["workflow_definition_hash"].(string)
	processHash, _ := decoded["process_definition_hash"].(string)
	catalogEntry := approvedImplementationCatalogEntry()
	if catalogEntry.WorkflowID == "" {
		return sessionExecutionTriggerValidationError(s, requestID, "approved implementation catalog entry is missing")
	}
	if strings.TrimSpace(workflowHash) != strings.TrimSpace(catalogEntry.WorkflowDefinitionHash) || strings.TrimSpace(processHash) != strings.TrimSpace(catalogEntry.ProcessDefinitionHash) {
		return sessionExecutionTriggerValidationError(s, requestID, "implementation_input_set workflow/process digests do not match approved implementation identity")
	}
	return nil
}

func approvedImplementationCatalogEntry() runplan.BuiltInWorkflowCatalogEntry {
	for _, entry := range runplan.BuiltInWorkflowCatalogV0() {
		if entry.WorkflowID == "builtin_rc_approved_implementation_v0" {
			return entry
		}
	}
	return runplan.BuiltInWorkflowCatalogEntry{}
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
	return requestedOperation == "continue" && strings.TrimSpace(routing.WorkflowFamily) == "" && strings.TrimSpace(routing.WorkflowOperation) == ""
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
