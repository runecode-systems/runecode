package brokerapi

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/runplan"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) validateApprovedImplementationRouting(requestID string, routing *SessionWorkflowPackRouting) *ErrorResponse {
	inputSetDigest := ""
	inputSetCount := 0
	for _, artifact := range routing.BoundInputArtifacts {
		if strings.TrimSpace(artifact.ArtifactRef) != "implementation_input_set" {
			return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing approved_change_implementation only accepts implementation_input_set artifact bindings")
		}
		inputSetCount++
		if inputSetCount > 1 {
			return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing approved_change_implementation allows exactly one implementation_input_set artifact binding")
		}
		inputSetDigest = strings.TrimSpace(artifact.ArtifactDigest)
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
	validatedDigest, ok := digestIdentityFromApprovedImplementationField(decoded, "validated_project_substrate_digest")
	if !ok || strings.TrimSpace(validatedDigest) != strings.TrimSpace(sessionExecutionBoundDigest(project)) {
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
	inputSetField, ok := digestIdentityFromApprovedImplementationField(decoded, "input_set_digest")
	if !ok {
		return false
	}
	return strings.TrimSpace(inputSetField) == strings.TrimSpace(inputSetDigest)
}

func validateApprovedImplementationCatalogBinding(s *Service, requestID string, decoded map[string]any) *ErrorResponse {
	workflowHash, workflowOK := digestIdentityFromApprovedImplementationField(decoded, "workflow_definition_hash")
	processHash, processOK := digestIdentityFromApprovedImplementationField(decoded, "process_definition_hash")
	catalogEntry := approvedImplementationCatalogEntry()
	if catalogEntry.WorkflowID == "" {
		return sessionExecutionTriggerValidationError(s, requestID, "approved implementation catalog entry is missing")
	}
	if !workflowOK || !processOK || strings.TrimSpace(workflowHash) != strings.TrimSpace(catalogEntry.WorkflowDefinitionHash) || strings.TrimSpace(processHash) != strings.TrimSpace(catalogEntry.ProcessDefinitionHash) {
		return sessionExecutionTriggerValidationError(s, requestID, "implementation_input_set workflow/process digests do not match approved implementation identity")
	}
	return nil
}

func digestIdentityFromApprovedImplementationField(decoded map[string]any, field string) (string, bool) {
	typed, ok := decoded[field].(map[string]any)
	if !ok {
		return "", false
	}
	hashAlg, _ := typed["hash_alg"].(string)
	hash, _ := typed["hash"].(string)
	identity, err := (trustpolicy.Digest{HashAlg: hashAlg, Hash: hash}).Identity()
	if err != nil {
		return "", false
	}
	return identity, true
}

func approvedImplementationCatalogEntry() runplan.BuiltInWorkflowCatalogEntry {
	for _, entry := range runplan.BuiltInWorkflowCatalogV0() {
		if entry.WorkflowID == "builtin_rc_approved_implementation_v0" {
			return entry
		}
	}
	return runplan.BuiltInWorkflowCatalogEntry{}
}
