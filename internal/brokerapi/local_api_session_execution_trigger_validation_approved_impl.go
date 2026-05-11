package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/runplan"
	"github.com/runecode-systems/runecode/internal/trustpolicy"
)

type approvedImplementationInputSetState struct {
	decoded                map[string]any
	inputSetArtifactDigest string
	inputSetDigest         string
}

func (s *Service) validateApprovedImplementationRouting(requestID string, routing *SessionWorkflowPackRouting) *ErrorResponse {
	inputSetArtifactDigest := ""
	inputSetCount := 0
	for _, artifact := range routing.BoundInputArtifacts {
		if strings.TrimSpace(artifact.ArtifactRef) != "implementation_input_set" {
			return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing approved_change_implementation only accepts implementation_input_set artifact bindings")
		}
		inputSetCount++
		if inputSetCount > 1 {
			return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing approved_change_implementation allows exactly one implementation_input_set artifact binding")
		}
		inputSetArtifactDigest = strings.TrimSpace(artifact.ArtifactDigest)
	}
	if inputSetArtifactDigest == "" {
		return sessionExecutionTriggerValidationError(s, requestID, "workflow_routing approved_change_implementation requires implementation_input_set artifact binding")
	}
	return s.validateApprovedImplementationIdentityTuple(requestID, inputSetArtifactDigest)
}

func (s *Service) validateApprovedImplementationIdentityTuple(requestID, inputSetArtifactDigest string) *ErrorResponse {
	inputSet, errResp := s.decodeApprovedImplementationInputSet(requestID, inputSetArtifactDigest)
	if errResp != nil {
		return errResp
	}
	if errResp := validateApprovedImplementationCatalogBinding(s, requestID, inputSet.decoded); errResp != nil {
		return errResp
	}
	project, errResp := s.requireSupportedProjectSubstrateForSessionExecution(requestID)
	if errResp != nil {
		return errResp
	}
	validatedDigest, ok := digestIdentityFromApprovedImplementationField(inputSet.decoded, "validated_project_substrate_digest")
	if !ok || strings.TrimSpace(validatedDigest) != strings.TrimSpace(sessionExecutionBoundDigest(project)) {
		return sessionExecutionTriggerValidationError(s, requestID, "implementation_input_set validated_project_substrate_digest drift detected")
	}
	return nil
}

func (s *Service) decodeApprovedImplementationInputSet(requestID, inputSetArtifactDigest string) (approvedImplementationInputSetState, *ErrorResponse) {
	payload, err := s.readArtifactPayloadVerified(inputSetArtifactDigest)
	if err != nil {
		return approvedImplementationInputSetState{}, sessionExecutionTriggerValidationError(s, requestID, "workflow_routing implementation_input_set artifact is unreadable")
	}
	if err := artifacts.ValidateObjectPayloadAgainstSchema(payload, "objects/RuneContextApprovedImplementationInputSet.schema.json"); err != nil {
		return approvedImplementationInputSetState{}, sessionExecutionTriggerValidationError(s, requestID, "workflow_routing implementation_input_set payload is invalid")
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return approvedImplementationInputSetState{}, sessionExecutionTriggerValidationError(s, requestID, "workflow_routing implementation_input_set payload decode failed")
	}
	inputSetDigest, ok := approvedImplementationInputSetDigest(decoded)
	if !ok {
		return approvedImplementationInputSetState{}, sessionExecutionTriggerValidationError(s, requestID, "implementation_input_set input_set_digest is invalid")
	}
	recomputedInputSetDigest, err := recomputeApprovedImplementationInputSetDigest(decoded)
	if err != nil {
		return approvedImplementationInputSetState{}, sessionExecutionTriggerValidationError(s, requestID, "implementation_input_set input_set_digest recompute failed")
	}
	if strings.TrimSpace(inputSetDigest) != strings.TrimSpace(recomputedInputSetDigest) {
		return approvedImplementationInputSetState{}, sessionExecutionTriggerValidationError(s, requestID, "implementation_input_set input_set_digest drift detected")
	}
	return approvedImplementationInputSetState{
		decoded:                decoded,
		inputSetArtifactDigest: strings.TrimSpace(inputSetArtifactDigest),
		inputSetDigest:         strings.TrimSpace(recomputedInputSetDigest),
	}, nil
}

func approvedImplementationInputSetDigest(decoded map[string]any) (string, bool) {
	inputSetField, ok := digestIdentityFromApprovedImplementationField(decoded, "input_set_digest")
	if !ok {
		return "", false
	}
	return strings.TrimSpace(inputSetField), true
}

func recomputeApprovedImplementationInputSetDigest(decoded map[string]any) (string, error) {
	if decoded == nil {
		return "", fmt.Errorf("payload must be an object")
	}
	payloadWithoutDigest := make(map[string]any, len(decoded))
	for key, value := range decoded {
		payloadWithoutDigest[key] = value
	}
	delete(payloadWithoutDigest, "input_set_digest")
	raw, err := json.Marshal(payloadWithoutDigest)
	if err != nil {
		return "", fmt.Errorf("marshal canonical input set body: %w", err)
	}
	canonical, err := artifacts.CanonicalizeJSONBytes(raw)
	if err != nil {
		return "", fmt.Errorf("canonicalize input set body: %w", err)
	}
	return artifacts.DigestBytes(canonical), nil
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
