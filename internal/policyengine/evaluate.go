package policyengine

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	policyDecisionSchemaID                    = "runecode.protocol.v0.PolicyDecision"
	policyDecisionSchemaVersion               = "0.3.0"
	policyEvaluationDetailsSchemaID           = "runecode.protocol.details.policy.evaluation.v0"
	requiredApprovalHardFloorSchemaID         = "runecode.protocol.details.policy.required_approval.hard_floor.v0"
	requiredApprovalModerateStageSchemaID     = "runecode.protocol.details.policy.required_approval.stage_sign_off.v0"
	requiredApprovalModerateGatewaySchemaID   = "runecode.protocol.details.policy.required_approval.gateway_egress_scope.v0"
	requiredApprovalModerateBackendSchemaID   = "runecode.protocol.details.policy.required_approval.reduced_assurance_backend.v0"
	requiredApprovalModerateGateSchemaID      = "runecode.protocol.details.policy.required_approval.gate_override.v0"
	requiredApprovalModerateWorkspaceSchemaID = "runecode.protocol.details.policy.required_approval.out_of_workspace_write.v0"
	requiredApprovalModerateSecretSchemaID    = "runecode.protocol.details.policy.required_approval.secret_access.v0"
)

type actionRegistries struct {
	actionKinds          map[string]struct{}
	payloadSchemaIDs     map[string]struct{}
	policyReasonCodes    map[string]struct{}
	approvalTriggerCodes map[string]struct{}
}

var (
	actionRegistriesOnce sync.Once
	actionRegistriesData actionRegistries
	actionRegistriesErr  error
)

func loadActionRegistries() (actionRegistries, error) {
	actionRegistriesOnce.Do(func() {
		actionKinds, err := loadRegistryCodes(actionKindRegistryPath)
		if err != nil {
			actionRegistriesErr = err
			return
		}
		payloadSchemaIDs, err := loadRegistryCodes(actionPayloadRegistryPath)
		if err != nil {
			actionRegistriesErr = err
			return
		}
		actionRegistriesData = actionRegistries{actionKinds: actionKinds, payloadSchemaIDs: payloadSchemaIDs}
		policyReasonCodes, err := loadRegistryCodes(policyReasonRegistryPath)
		if err != nil {
			actionRegistriesErr = err
			return
		}
		approvalTriggerCodes, err := loadRegistryCodes(approvalTriggerRegistryPath)
		if err != nil {
			actionRegistriesErr = err
			return
		}
		actionRegistriesData = actionRegistries{
			actionKinds:          actionKinds,
			payloadSchemaIDs:     payloadSchemaIDs,
			policyReasonCodes:    policyReasonCodes,
			approvalTriggerCodes: approvalTriggerCodes,
		}
	})
	if actionRegistriesErr != nil {
		return actionRegistries{}, actionRegistriesErr
	}
	return actionRegistriesData, nil
}

func ensureKnownPolicyReasonCode(code string) error {
	registries, err := loadActionRegistries()
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("load reason-code registries: %v", err)}
	}
	if _, ok := registries.policyReasonCodes[code]; !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown policy_reason_code %q (fail-closed)", code)}
	}
	return nil
}

func ensureKnownApprovalTriggerCode(code string) error {
	registries, err := loadActionRegistries()
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("load trigger-code registries: %v", err)}
	}
	if _, ok := registries.approvalTriggerCodes[code]; !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown approval_trigger_code %q (fail-closed)", code)}
	}
	return nil
}

func actionRelevantArtifactHashes(action ActionRequest) []string {
	out := make([]string, 0, len(action.RelevantArtifactHashes))
	for _, digest := range action.RelevantArtifactHashes {
		identity, err := digest.Identity()
		if err != nil {
			continue
		}
		out = append(out, identity)
	}
	return out
}

func loadRegistryCodes(path string) (map[string]struct{}, error) {
	abs, err := schemaAbsolutePath(path)
	if err != nil {
		return nil, err
	}
	doc := map[string]any{}
	if err := loadJSONFile(abs, &doc); err != nil {
		return nil, err
	}
	rawCodes, ok := doc["codes"].([]any)
	if !ok {
		return nil, fmt.Errorf("registry %q missing codes array", path)
	}
	codes := make(map[string]struct{}, len(rawCodes))
	for _, raw := range rawCodes {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("registry %q contains non-object code entry", path)
		}
		code, ok := entry["code"].(string)
		if !ok || code == "" {
			return nil, fmt.Errorf("registry %q contains code entry without non-empty code", path)
		}
		codes[code] = struct{}{}
	}
	return codes, nil
}

func Evaluate(compiled *CompiledContext, action ActionRequest) (PolicyDecision, error) {
	if compiled == nil {
		return PolicyDecision{}, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "compiled context is required"}
	}
	if err := validateActionRequest(action); err != nil {
		return PolicyDecision{}, err
	}
	actionHash, err := canonicalHashValue(action)
	if err != nil {
		return PolicyDecision{}, err
	}

	if denyDecision, denied := evaluateInvariantDeny(compiled, action, actionHash); denied {
		return finalizePolicyDecision(denyDecision)
	}

	if denyDecision, denied := evaluateAbsentCapabilityDeny(compiled, action, actionHash); denied {
		return finalizePolicyDecision(denyDecision)
	}

	if denyDecision, denied := evaluateAllowlistPresenceDeny(compiled, action, actionHash); denied {
		return finalizePolicyDecision(denyDecision)
	}

	if hardBoundaryDecision, matched := evaluateHardBoundaryInvariants(compiled, action, actionHash); matched {
		return finalizePolicyDecision(hardBoundaryDecision)
	}

	moderateApproval, needsModerateApproval := evaluateModerateProfileApproval(compiled, action, actionHash)

	if compiled.RuleSet != nil {
		if decision, matched := evaluateRuleSet(compiled, action, actionHash, *compiled.RuleSet); matched {
			if decision.DecisionOutcome == DecisionAllow && needsModerateApproval {
				return finalizePolicyDecision(moderateApproval)
			}
			return finalizePolicyDecision(decision)
		}
	}

	if needsModerateApproval {
		return finalizePolicyDecision(moderateApproval)
	}

	return finalizePolicyDecision(PolicyDecision{
		SchemaID:               policyDecisionSchemaID,
		SchemaVersion:          policyDecisionSchemaVersion,
		DecisionOutcome:        DecisionAllow,
		PolicyReasonCode:       "allow_manifest_opt_in",
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: actionRelevantArtifactHashes(action),
		DetailsSchemaID:        policyEvaluationDetailsSchemaID,
		Details: map[string]any{
			"precedence":            "invariants_role_run_stage_allowlists",
			"capability_id":         action.CapabilityID,
			"approval_profile":      string(compiled.Context.ApprovalProfile),
			"active_allowlist_refs": append([]string{}, compiled.Context.ActiveAllowlistRefs...),
			"secondary_factors":     []string{},
		},
	})
}

func finalizePolicyDecision(decision PolicyDecision) (PolicyDecision, error) {
	if err := ensureKnownPolicyReasonCode(decision.PolicyReasonCode); err != nil {
		return PolicyDecision{}, err
	}
	if decision.DecisionOutcome == DecisionRequireHumanApproval {
		if decision.RequiredApproval == nil {
			return PolicyDecision{}, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: "required_approval missing for require_human_approval decision"}
		}
		trigger, ok := decision.RequiredApproval["approval_trigger_code"].(string)
		if !ok || strings.TrimSpace(trigger) == "" {
			return PolicyDecision{}, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: "required_approval.approval_trigger_code is required"}
		}
		if err := ensureKnownApprovalTriggerCode(trigger); err != nil {
			return PolicyDecision{}, err
		}
	}
	return decision, nil
}

func evaluateModerateProfileApproval(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if compiled.Context.ApprovalProfile != ApprovalProfileModerate {
		return PolicyDecision{}, false
	}

	switch action.ActionKind {
	case ActionKindStageSummarySign:
		reqSchemaID, reqPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
		return policyApprovalDecision(compiled, action, actionHash, "approval_required", reqSchemaID, reqPayload, map[string]any{
			"precedence":       "approval_profile_moderate",
			"approval_profile": string(compiled.Context.ApprovalProfile),
			"checkpoint_model": "stage_sign_off",
		}), true
	case ActionKindGatewayEgress, ActionKindDependencyFetch:
		reqSchemaID, reqPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
		if reqSchemaID != "" {
			return policyApprovalDecision(compiled, action, actionHash, "approval_required", reqSchemaID, reqPayload, map[string]any{
				"precedence":       "approval_profile_moderate",
				"approval_profile": string(compiled.Context.ApprovalProfile),
				"checkpoint_model": "scope_checkpoint",
			}), true
		}
	case ActionKindWorkspaceWrite:
		reqSchemaID, reqPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
		if reqSchemaID != "" {
			return policyApprovalDecision(compiled, action, actionHash, "approval_required", reqSchemaID, reqPayload, map[string]any{
				"precedence":       "approval_profile_moderate",
				"approval_profile": string(compiled.Context.ApprovalProfile),
			}), true
		}
	case ActionKindSecretAccess:
		reqSchemaID, reqPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
		if reqSchemaID != "" {
			return policyApprovalDecision(compiled, action, actionHash, "approval_required", reqSchemaID, reqPayload, map[string]any{
				"precedence":       "approval_profile_moderate",
				"approval_profile": string(compiled.Context.ApprovalProfile),
				"checkpoint_model": "secret_checkpoint",
			}), true
		}
	}

	return PolicyDecision{}, false
}

type executorRunPayload struct {
	SchemaID       string   `json:"schema_id"`
	SchemaVersion  string   `json:"schema_version"`
	ExecutorClass  string   `json:"executor_class"`
	ExecutorID     string   `json:"executor_id"`
	Argv           []string `json:"argv"`
	WorkingDir     string   `json:"working_directory,omitempty"`
	NetworkAccess  string   `json:"network_access,omitempty"`
	TimeoutSeconds *int     `json:"timeout_seconds,omitempty"`
}

type gatewayEgressPayload struct {
	SchemaID        string `json:"schema_id"`
	SchemaVersion   string `json:"schema_version"`
	GatewayRoleKind string `json:"gateway_role_kind"`
	DestinationKind string `json:"destination_kind"`
	DestinationRef  string `json:"destination_ref"`
	EgressDataClass string `json:"egress_data_class"`
	Operation       string `json:"operation,omitempty"`
	PayloadHash     string `json:"payload_hash,omitempty"`
}

func evaluateHardBoundaryInvariants(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if decision, denied := evaluateNoEscalationInPlace(compiled, action, actionHash); denied {
		return decision, true
	}

	if decision, matched := evaluateHardFloorApprovalRequirement(compiled, action, actionHash); matched {
		return decision, true
	}

	if compiled.Context.ActiveRoleFamily == "gateway" {
		if decision, denied := evaluateGatewayNoWorkspaceAccess(compiled, action, actionHash); denied {
			return decision, true
		}
	}

	switch action.ActionKind {
	case ActionKindGatewayEgress, ActionKindDependencyFetch:
		decision, matched := evaluateGatewayBoundary(compiled, action, actionHash)
		return decision, matched
	case ActionKindExecutorRun:
		decision, matched := evaluateExecutorBoundary(compiled, action, actionHash)
		return decision, matched
	case ActionKindBackendPosture:
		decision, matched := evaluateBackendSelectionRules(compiled, action, actionHash)
		return decision, matched
	default:
		return PolicyDecision{}, false
	}
}

type backendPosturePayload struct {
	SchemaID         string `json:"schema_id"`
	SchemaVersion    string `json:"schema_version"`
	BackendClass     string `json:"backend_class"`
	ChangeKind       string `json:"change_kind"`
	RequestedPosture string `json:"requested_posture"`
	RequiresOptIn    bool   `json:"requires_opt_in"`
}

type secretAccessPayload struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	SecretRef     string `json:"secret_ref"`
	AccessMode    string `json:"access_mode"`
}

type promotionPayload struct {
	SchemaID            string `json:"schema_id"`
	SchemaVersion       string `json:"schema_version"`
	PromotionKind       string `json:"promotion_kind"`
	TargetDataClass     string `json:"target_data_class"`
	AuthoritativeImport bool   `json:"authoritative_import,omitempty"`
}

func evaluateBackendSelectionRules(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	payload := backendPosturePayload{}
	if err := decodeActionPayload(action.ActionPayload, &payload); err != nil {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "backend_selection_rules",
			"non_approvable":    true,
			"payload_parse_err": err.Error(),
		}), true
	}

	requestedPosture := strings.ToLower(payload.RequestedPosture)
	if payload.BackendClass == "microvm" {
		if strings.Contains(requestedPosture, "fallback") {
			return denyInvariantDecision(compiled, action, actionHash, map[string]any{
				"precedence":        "invariants_first",
				"invariant":         "backend_selection_rules",
				"non_approvable":    true,
				"backend_class":     payload.BackendClass,
				"requested_posture": payload.RequestedPosture,
				"reason":            "automatic_fallback_not_allowed",
				"secondary_factors": []string{"microvm_default_backend_when_available"},
			}), true
		}
		return PolicyDecision{
			SchemaID:               policyDecisionSchemaID,
			SchemaVersion:          policyDecisionSchemaVersion,
			DecisionOutcome:        DecisionAllow,
			PolicyReasonCode:       "allow_microvm_default",
			ManifestHash:           compiled.ManifestHash,
			PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
			ActionRequestHash:      actionHash,
			RelevantArtifactHashes: actionRelevantArtifactHashes(action),
			DetailsSchemaID:        policyEvaluationDetailsSchemaID,
			Details: map[string]any{
				"precedence":        "invariants_first",
				"invariant":         "backend_selection_rules",
				"backend_class":     payload.BackendClass,
				"change_kind":       payload.ChangeKind,
				"requested_posture": payload.RequestedPosture,
				"secondary_factors": []string{"microvm_default_backend_when_available"},
			},
		}, true
	}

	if payload.BackendClass == "container" {
		if strings.Contains(requestedPosture, "fallback") {
			return PolicyDecision{
				SchemaID:               policyDecisionSchemaID,
				SchemaVersion:          policyDecisionSchemaVersion,
				DecisionOutcome:        DecisionDeny,
				PolicyReasonCode:       "deny_container_automatic_fallback",
				ManifestHash:           compiled.ManifestHash,
				PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
				ActionRequestHash:      actionHash,
				RelevantArtifactHashes: actionRelevantArtifactHashes(action),
				DetailsSchemaID:        policyEvaluationDetailsSchemaID,
				Details: map[string]any{
					"precedence":        "invariants_first",
					"invariant":         "backend_selection_rules",
					"non_approvable":    true,
					"backend_class":     payload.BackendClass,
					"requested_posture": payload.RequestedPosture,
					"secondary_factors": []string{"no_automatic_microvm_to_container_fallback"},
				},
			}, true
		}
		if !payload.RequiresOptIn {
			return PolicyDecision{
				SchemaID:               policyDecisionSchemaID,
				SchemaVersion:          policyDecisionSchemaVersion,
				DecisionOutcome:        DecisionDeny,
				PolicyReasonCode:       "deny_container_opt_in_required",
				ManifestHash:           compiled.ManifestHash,
				PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
				ActionRequestHash:      actionHash,
				RelevantArtifactHashes: actionRelevantArtifactHashes(action),
				DetailsSchemaID:        policyEvaluationDetailsSchemaID,
				Details: map[string]any{
					"precedence":        "invariants_first",
					"invariant":         "backend_selection_rules",
					"non_approvable":    true,
					"backend_class":     payload.BackendClass,
					"requires_opt_in":   payload.RequiresOptIn,
					"secondary_factors": []string{"container_backend_requires_explicit_opt_in"},
				},
			}, true
		}
		reqSchemaID, reqPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
		if reqSchemaID == "" {
			return PolicyDecision{}, false
		}
		return policyApprovalDecision(compiled, action, actionHash, "approval_required", reqSchemaID, reqPayload, map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "backend_selection_rules",
			"backend_class":     payload.BackendClass,
			"change_kind":       payload.ChangeKind,
			"requested_posture": payload.RequestedPosture,
			"approval_profile":  string(compiled.Context.ApprovalProfile),
			"secondary_factors": []string{"container_backend_explicit_opt_in_requires_approval", "approval_must_be_audited"},
		}), true
	}

	return PolicyDecision{}, false
}

func evaluateHardFloorApprovalRequirement(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	classes, floor := classifyHardFloorOperation(action, nil)
	if len(classes) == 0 {
		return PolicyDecision{}, false
	}
	return hardFloorApprovalDecision(compiled, action, actionHash, classes, floor), true
}

func evaluateNoEscalationInPlace(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if action.RoleFamily != "" && action.RoleFamily != compiled.Context.ActiveRoleFamily {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":                 "invariants_first",
			"invariant":                  "no_escalation_in_place",
			"non_approvable":             true,
			"requested_role_family":      action.RoleFamily,
			"active_context_role_family": compiled.Context.ActiveRoleFamily,
		}), true
	}
	if action.RoleKind != "" && action.RoleKind != compiled.Context.ActiveRoleKind {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":               "invariants_first",
			"invariant":                "no_escalation_in_place",
			"non_approvable":           true,
			"requested_role_kind":      action.RoleKind,
			"active_context_role_kind": compiled.Context.ActiveRoleKind,
		}), true
	}
	return PolicyDecision{}, false
}

func evaluateGatewayNoWorkspaceAccess(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	switch action.ActionKind {
	case ActionKindWorkspaceWrite, ActionKindExecutorRun:
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "gateway_no_workspace_access",
			"non_approvable":   true,
			"active_role_kind": compiled.Context.ActiveRoleKind,
			"action_kind":      action.ActionKind,
		}), true
	default:
		return PolicyDecision{}, false
	}
}

func evaluateGatewayBoundary(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if compiled.Context.ActiveRoleFamily != "gateway" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":             "invariants_first",
			"invariant":              "network_egress_hard_boundary",
			"non_approvable":         true,
			"action_kind":            action.ActionKind,
			"required_role_family":   "gateway",
			"active_role_family":     compiled.Context.ActiveRoleFamily,
			"workspace_offline_only": true,
		}), true
	}
	if compiled.Context.ActiveRoleKind == "workspace-read" || compiled.Context.ActiveRoleKind == "workspace-edit" || compiled.Context.ActiveRoleKind == "workspace-test" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":          "invariants_first",
			"invariant":           "network_egress_hard_boundary",
			"non_approvable":      true,
			"workspace_role_kind": compiled.Context.ActiveRoleKind,
		}), true
	}
	payload := gatewayEgressPayload{}
	if err := decodeActionPayload(action.ActionPayload, &payload); err != nil {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "deny_by_default_network",
			"non_approvable":    true,
			"payload_parse_err": err.Error(),
		}), true
	}
	if payload.GatewayRoleKind != compiled.Context.ActiveRoleKind {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":                "invariants_first",
			"invariant":                 "no_escalation_in_place",
			"non_approvable":            true,
			"payload_gateway_role_kind": payload.GatewayRoleKind,
			"active_context_role_kind":  compiled.Context.ActiveRoleKind,
		}), true
	}

	requiredRoleForDestination := map[string]string{
		"model_endpoint":   "model-gateway",
		"auth_provider":    "auth-gateway",
		"git_remote":       "git-gateway",
		"web_origin":       "web-research",
		"package_registry": "dependency-fetch",
	}
	if action.ActionKind == ActionKindDependencyFetch && payload.DestinationKind != "package_registry" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "dependency_behavior_split",
			"non_approvable":   true,
			"action_kind":      ActionKindDependencyFetch,
			"destination_kind": payload.DestinationKind,
		}), true
	}
	if action.ActionKind == ActionKindGatewayEgress && payload.DestinationKind == "package_registry" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":           "invariants_first",
			"invariant":            "dependency_behavior_split",
			"non_approvable":       true,
			"action_kind":          ActionKindGatewayEgress,
			"required_action_kind": ActionKindDependencyFetch,
		}), true
	}
	requiredRole, ok := requiredRoleForDestination[payload.DestinationKind]
	if !ok {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "network_egress_hard_boundary",
			"non_approvable":   true,
			"destination_kind": payload.DestinationKind,
			"reason":           "unknown_destination_kind_fail_closed",
		}), true
	}
	if payload.GatewayRoleKind != requiredRole {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":            "invariants_first",
			"invariant":             "network_egress_hard_boundary",
			"non_approvable":        true,
			"destination_kind":      payload.DestinationKind,
			"required_gateway_role": requiredRole,
			"gateway_role_kind":     payload.GatewayRoleKind,
		}), true
	}

	if !gatewayDestinationAllowedBySignedAllowlists(compiled, action, payload) {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "allowlist_active_manifest_set",
			"invariant":        "network_egress_hard_boundary",
			"non_approvable":   true,
			"destination_kind": payload.DestinationKind,
			"destination_ref":  payload.DestinationRef,
			"reason":           "destination_not_allowlisted",
		}), true
	}

	return PolicyDecision{}, false
}

func gatewayDestinationAllowedBySignedAllowlists(compiled *CompiledContext, action ActionRequest, payload gatewayEgressPayload) bool {
	refs := action.AllowlistRefs
	if len(refs) == 0 {
		refs = compiled.Context.ActiveAllowlistRefs
	}
	for _, ref := range refs {
		allowlist, ok := compiled.AllowlistsByHash[ref]
		if !ok {
			continue
		}
		for _, entry := range allowlist.Entries {
			if entry.ScopeKind != "gateway_destination" {
				continue
			}
			if entry.GatewayRoleKind != "" && entry.GatewayRoleKind != payload.GatewayRoleKind {
				continue
			}
			if entry.Destination.DescriptorKind != payload.DestinationKind {
				continue
			}
			if !destinationRefMatches(entry.Destination, payload.DestinationRef) {
				continue
			}
			if payload.Operation != "" && !containsString(entry.PermittedOperations, payload.Operation) {
				continue
			}
			if !containsString(entry.AllowedEgressDataClasses, payload.EgressDataClass) {
				continue
			}
			return true
		}
	}
	return false
}

func destinationRefMatches(descriptor DestinationDescriptor, destinationRef string) bool {
	if strings.TrimSpace(destinationRef) == "" {
		return false
	}

	host, port, path := parseDestinationRef(destinationRef)
	if host == "" || host != descriptor.CanonicalHost {
		return false
	}

	expectedPort := 443
	if descriptor.CanonicalPort != nil {
		expectedPort = *descriptor.CanonicalPort
	}
	if port != nil && *port != expectedPort {
		return false
	}

	if descriptor.CanonicalPathPrefix != "" {
		if !strings.HasPrefix(path, descriptor.CanonicalPathPrefix) {
			return false
		}
	}

	return true
}

func parseDestinationRef(ref string) (string, *int, string) {
	value := strings.TrimSpace(ref)
	if value == "" {
		return "", nil, ""
	}

	hostPort := value
	path := ""
	if slash := strings.Index(hostPort, "/"); slash >= 0 {
		path = hostPort[slash:]
		hostPort = hostPort[:slash]
	}

	host := hostPort
	var port *int
	if colon := strings.LastIndex(hostPort, ":"); colon > 0 && colon < len(hostPort)-1 {
		if parsed, err := strconv.Atoi(hostPort[colon+1:]); err == nil && parsed > 0 && parsed <= 65535 {
			h := hostPort[:colon]
			host = h
			port = &parsed
		}
	}

	if host == "" {
		return "", nil, ""
	}
	if path == "" {
		path = "/"
	}

	return host, port, path
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func evaluateExecutorBoundary(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	payload := executorRunPayload{}
	if err := decodeActionPayload(action.ActionPayload, &payload); err != nil {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "deny_by_default_shell",
			"non_approvable":    true,
			"payload_parse_err": err.Error(),
		}), true
	}

	if payload.NetworkAccess != "" && payload.NetworkAccess != "none" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "network_egress_hard_boundary",
			"non_approvable":   true,
			"network_access":   payload.NetworkAccess,
			"required_network": "none",
			"action_kind":      ActionKindExecutorRun,
		}), true
	}

	if payload.ExecutorClass == "workspace_ordinary" {
		if isSystemModifyingArgv(payload.Argv) {
			return denyInvariantDecision(compiled, action, actionHash, map[string]any{
				"precedence":     "invariants_first",
				"invariant":      "ordinary_workspace_executor_constraints",
				"non_approvable": true,
				"reason":         "system_modifying_execution_not_ordinary",
			}), true
		}
		if payload.WorkingDir != "" {
			if !isWorkspaceRelativePath(payload.WorkingDir) {
				return denyInvariantDecision(compiled, action, actionHash, map[string]any{
					"precedence":        "invariants_first",
					"invariant":         "ordinary_workspace_executor_constraints",
					"non_approvable":    true,
					"working_directory": payload.WorkingDir,
					"workspace_scoped":  false,
				}), true
			}
		}
		if isRawShellInvocation(payload) {
			return denyInvariantDecision(compiled, action, actionHash, map[string]any{
				"precedence":     "invariants_first",
				"invariant":      "ordinary_workspace_executor_constraints",
				"non_approvable": true,
				"reason":         "raw_shell_not_implicitly_ordinary",
				"executor_id":    payload.ExecutorID,
			}), true
		}
		return PolicyDecision{}, false
	}

	classes, floor := classifyHardFloorOperation(action, &payload)
	if len(classes) > 0 {
		return hardFloorApprovalDecision(compiled, action, actionHash, classes, floor), true
	}

	return PolicyDecision{}, false
}

func decodeActionPayload(payload map[string]any, target any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func isRawShellInvocation(payload executorRunPayload) bool {
	rawShellNames := map[string]struct{}{
		"sh": {}, "bash": {}, "zsh": {}, "fish": {}, "pwsh": {}, "powershell": {}, "cmd": {}, "cmd.exe": {},
	}
	if _, ok := rawShellNames[strings.ToLower(payload.ExecutorID)]; ok {
		return true
	}
	if len(payload.Argv) == 0 {
		return false
	}
	argv := unwrapLauncherArgv(payload.Argv)
	if len(argv) == 0 {
		return false
	}
	base := strings.ToLower(filepath.Base(argv[0]))
	_, ok := rawShellNames[base]
	return ok
}

func unwrapLauncherArgv(argv []string) []string {
	idx := 0
	for idx < len(argv) {
		tok := strings.ToLower(filepath.Base(argv[idx]))
		switch tok {
		case "env":
			idx++
			for idx < len(argv) && strings.Contains(argv[idx], "=") {
				idx++
			}
			continue
		case "command", "nohup", "sudo":
			idx++
			continue
		default:
			return argv[idx:]
		}
	}
	return argv[idx:]
}

func isWorkspaceRelativePath(raw string) bool {
	path := strings.TrimSpace(raw)
	if path == "" {
		return false
	}
	if isCrossPlatformAbsolutePath(path) {
		return false
	}

	clean := filepath.Clean(path)
	normalized := strings.ReplaceAll(clean, "\\", "/")
	return normalized != ".." && !strings.HasPrefix(normalized, "../")
}

func isCrossPlatformAbsolutePath(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}
	if strings.HasPrefix(path, "\\\\") || strings.HasPrefix(path, "\\") {
		return true
	}
	if len(path) >= 2 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' {
		return true
	}
	return false
}

func isSystemModifyingArgv(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	first := strings.ToLower(filepath.Base(argv[0]))
	systemTools := map[string]struct{}{
		"apt": {}, "apt-get": {}, "yum": {}, "dnf": {}, "apk": {}, "pacman": {}, "brew": {},
		"systemctl": {}, "service": {}, "modprobe": {}, "sysctl": {}, "mount": {}, "umount": {},
		"iptables": {}, "ufw": {}, "nft": {}, "netsh": {}, "sc": {},
		"docker": {}, "podman": {}, "kubectl": {}, "helm": {},
	}
	if _, ok := systemTools[first]; ok {
		return true
	}
	for _, arg := range argv {
		lower := strings.ToLower(arg)
		if strings.Contains(lower, "/etc/") || strings.Contains(lower, "c:\\windows") {
			return true
		}
	}
	return false
}

func classifyHardFloorOperation(action ActionRequest, exec *executorRunPayload) ([]HardFloorOperationClass, ApprovalAssuranceLevel) {
	classes := []HardFloorOperationClass{}

	if action.ActionKind == ActionKindGateOverride {
		classes = append(classes, HardFloorSecurityPostureWeakening, HardFloorTrustRootChange)
	}
	if action.ActionKind == ActionKindExecutorRun && exec != nil && exec.ExecutorClass == "system_modifying" {
		classes = append(classes, HardFloorSecurityPostureWeakening)
	}
	if action.ActionKind == ActionKindPromotion {
		payload := promotionPayload{}
		if decodeActionPayload(action.ActionPayload, &payload) == nil {
			if payload.AuthoritativeImport {
				classes = append(classes, HardFloorAuthoritativeStateReconciliation)
			}
		}
	}
	classes = uniqueHardFloorClasses(classes)

	floor := strongestAssuranceFloor(classes)
	return classes, floor
}

func uniqueHardFloorClasses(classes []HardFloorOperationClass) []HardFloorOperationClass {
	if len(classes) == 0 {
		return []HardFloorOperationClass{}
	}
	seen := map[HardFloorOperationClass]struct{}{}
	out := make([]HardFloorOperationClass, 0, len(classes))
	for _, class := range classes {
		if _, ok := seen[class]; ok {
			continue
		}
		seen[class] = struct{}{}
		out = append(out, class)
	}
	return out
}

func strongestAssuranceFloor(classes []HardFloorOperationClass) ApprovalAssuranceLevel {
	floorByClass := map[HardFloorOperationClass]ApprovalAssuranceLevel{
		HardFloorTrustRootChange:                  ApprovalAssuranceHardwareBacked,
		HardFloorSecurityPostureWeakening:         ApprovalAssuranceReauthenticated,
		HardFloorAuthoritativeStateReconciliation: ApprovalAssuranceReauthenticated,
		HardFloorDeploymentBootstrapAuthority:     ApprovalAssuranceHardwareBacked,
	}
	strongest := ApprovalAssuranceNone
	for _, class := range classes {
		candidate, ok := floorByClass[class]
		if !ok {
			continue
		}
		if assuranceRank(candidate) > assuranceRank(strongest) {
			strongest = candidate
		}
	}
	return strongest
}

func assuranceRank(level ApprovalAssuranceLevel) int {
	switch level {
	case ApprovalAssuranceNone:
		return 0
	case ApprovalAssuranceSessionAuthenticated:
		return 1
	case ApprovalAssuranceReauthenticated:
		return 2
	case ApprovalAssuranceHardwareBacked:
		return 3
	default:
		return -1
	}
}

func toStringSlice(classes []HardFloorOperationClass) []string {
	if len(classes) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(classes))
	for _, class := range classes {
		out = append(out, string(class))
	}
	return out
}

func firstTrigger(triggers []string, fallback string) string {
	if len(triggers) == 0 {
		return fallback
	}
	return triggers[0]
}

func approvalScopeForAction(action ActionRequest) map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.ApprovalBoundScope",
		"schema_version": "0.1.0",
		"action_kind":    action.ActionKind,
	}
}

func hardFloorChangesIfApproved(action ActionRequest) string {
	switch action.ActionKind {
	case ActionKindGateOverride:
		return "Requested gate override is permitted for one bounded action execution."
	case ActionKindBackendPosture:
		return "Requested backend posture change is permitted for bounded execution context."
	case ActionKindExecutorRun:
		return "Requested system-modifying execution is permitted for one bounded action."
	default:
		return "Requested high-impact action is permitted for one bounded action."
	}
}

func policyApprovalDecision(compiled *CompiledContext, action ActionRequest, actionHash, reasonCode, requiredSchemaID string, requiredPayload map[string]any, details map[string]any) PolicyDecision {
	outDetails := map[string]any{
		"approval_profile": string(compiled.Context.ApprovalProfile),
	}
	for k, v := range details {
		outDetails[k] = v
	}
	return PolicyDecision{
		SchemaID:                 policyDecisionSchemaID,
		SchemaVersion:            policyDecisionSchemaVersion,
		DecisionOutcome:          DecisionRequireHumanApproval,
		PolicyReasonCode:         reasonCode,
		ManifestHash:             compiled.ManifestHash,
		PolicyInputHashes:        append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:        actionHash,
		RelevantArtifactHashes:   actionRelevantArtifactHashes(action),
		DetailsSchemaID:          policyEvaluationDetailsSchemaID,
		Details:                  outDetails,
		RequiredApprovalSchemaID: requiredSchemaID,
		RequiredApproval:         requiredPayload,
	}
}

func requiredApprovalForModerateProfile(compiled *CompiledContext, action ActionRequest, actionHash string) (string, map[string]any) {
	ttl := 1800
	base := map[string]any{
		"approval_assurance_level":          string(ApprovalAssuranceSessionAuthenticated),
		"scope":                             approvalScopeForAction(action),
		"effects_if_denied_or_deferred":     "Action remains blocked until an approval decision is provided.",
		"blocked_work":                      []string{"action_execution"},
		"approval_ttl_seconds":              ttl,
		"approval_assertion_hash_supported": true,
		"related_hashes": map[string]any{
			"manifest_hash":            compiled.ManifestHash,
			"action_request_hash":      actionHash,
			"policy_input_hashes":      append([]string{}, compiled.PolicyInputHashes...),
			"relevant_artifact_hashes": actionRelevantArtifactHashes(action),
		},
	}

	switch action.ActionKind {
	case ActionKindStageSummarySign:
		payload := cloneMap(base)
		payload["approval_trigger_code"] = "stage_sign_off"
		payload["why_required"] = "Moderate profile requires stage checkpoint sign-off before proceeding."
		payload["changes_if_approved"] = "Stage summary is signed off for this exact summary hash."
		payload["security_posture_impact"] = "moderate"
		payload["stage_summary_staleness_posture"] = "invalidate_on_bound_input_change"
		return requiredApprovalModerateStageSchemaID, payload
	case ActionKindGateOverride:
		payload := cloneMap(base)
		payload["approval_trigger_code"] = "gate_override"
		payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
		payload["why_required"] = "Moderate profile requires explicit approval for gate overrides."
		payload["changes_if_approved"] = "Gate override can be consumed once for this exact action request hash."
		payload["security_posture_impact"] = "high"
		return requiredApprovalModerateGateSchemaID, payload
	case ActionKindBackendPosture:
		payload := cloneMap(base)
		payload["approval_trigger_code"] = "reduced_assurance_backend"
		payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
		payload["why_required"] = "Moderate profile requires explicit approval for reduced-assurance backend opt-ins."
		payload["changes_if_approved"] = "Reduced-assurance backend posture change may be applied."
		payload["security_posture_impact"] = "high"
		return requiredApprovalModerateBackendSchemaID, payload
	case ActionKindGatewayEgress, ActionKindDependencyFetch:
		if !isModerateGatewayCheckpointAction(action) {
			return "", nil
		}
		payload := cloneMap(base)
		payload["approval_trigger_code"] = "gateway_egress_scope_change"
		if action.ActionKind == ActionKindDependencyFetch {
			payload["approval_trigger_code"] = "dependency_network_fetch"
		}
		payload["checkpoint_scope"] = "gateway_or_dependency_scope_change"
		payload["why_required"] = "Moderate profile requires checkpoint approval only when enabling or expanding gateway/dependency scope."
		payload["changes_if_approved"] = "Gateway egress action can proceed for the bound request and manifest context."
		payload["security_posture_impact"] = "high"
		return requiredApprovalModerateGatewaySchemaID, payload
	case ActionKindWorkspaceWrite:
		if targetPath, ok := action.ActionPayload["target_path"].(string); ok {
			if !isWorkspaceRelativePath(targetPath) {
				payload := cloneMap(base)
				payload["approval_trigger_code"] = "out_of_workspace_write"
				payload["why_required"] = "Moderate profile requires approval for writes outside workspace allowlist."
				payload["changes_if_approved"] = "Out-of-workspace write can proceed for this exact action request hash."
				payload["security_posture_impact"] = "high"
				return requiredApprovalModerateWorkspaceSchemaID, payload
			}
		}
	case ActionKindSecretAccess:
		payload := cloneMap(base)
		payload["approval_trigger_code"] = "secret_access_lease"
		payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
		payload["why_required"] = "Moderate profile requires explicit approval for secret lease issue/renew operations."
		payload["changes_if_approved"] = "Secret lease operation can proceed once for this exact action request hash."
		payload["security_posture_impact"] = "high"
		return requiredApprovalModerateSecretSchemaID, payload
	}

	return "", nil
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func isModerateGatewayCheckpointAction(action ActionRequest) bool {
	operation, _ := action.ActionPayload["operation"].(string)
	scopeCheckpointOps := map[string]struct{}{
		"enable_gateway":          {},
		"expand_scope":            {},
		"change_allowlist":        {},
		"enable_dependency_fetch": {},
	}
	_, ok := scopeCheckpointOps[operation]
	return ok
}

func denyInvariantDecision(compiled *CompiledContext, action ActionRequest, actionHash string, details map[string]any) PolicyDecision {
	return PolicyDecision{
		SchemaID:               policyDecisionSchemaID,
		SchemaVersion:          policyDecisionSchemaVersion,
		DecisionOutcome:        DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: actionRelevantArtifactHashes(action),
		DetailsSchemaID:        policyEvaluationDetailsSchemaID,
		Details:                details,
	}
}

func hardFloorApprovalDecision(compiled *CompiledContext, action ActionRequest, actionHash string, classes []HardFloorOperationClass, floor ApprovalAssuranceLevel) PolicyDecision {
	requiredApproval := map[string]any{
		"approval_trigger_code":             firstTrigger(hardFloorApprovalTriggers(action), "system_command_execution"),
		"approval_assurance_level":          string(floor),
		"scope":                             approvalScopeForAction(action),
		"why_required":                      "Hard-floor operation class requires explicit human approval at or above fixed assurance floor.",
		"changes_if_approved":               hardFloorChangesIfApproved(action),
		"effects_if_denied_or_deferred":     "Action remains blocked until a conforming approval decision is provided.",
		"security_posture_impact":           "high",
		"blocked_work":                      []string{"action_execution"},
		"approval_ttl_seconds":              1800,
		"approval_assertion_hash_supported": true,
		"related_hashes": map[string]any{
			"manifest_hash":            compiled.ManifestHash,
			"action_request_hash":      actionHash,
			"policy_input_hashes":      append([]string{}, compiled.PolicyInputHashes...),
			"relevant_artifact_hashes": actionRelevantArtifactHashes(action),
		},
		"hard_floor_operation_classes": toStringSlice(classes),
	}
	return PolicyDecision{
		SchemaID:                 policyDecisionSchemaID,
		SchemaVersion:            policyDecisionSchemaVersion,
		DecisionOutcome:          DecisionRequireHumanApproval,
		PolicyReasonCode:         "approval_required",
		ManifestHash:             compiled.ManifestHash,
		PolicyInputHashes:        append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:        actionHash,
		RelevantArtifactHashes:   actionRelevantArtifactHashes(action),
		DetailsSchemaID:          policyEvaluationDetailsSchemaID,
		RequiredApprovalSchemaID: requiredApprovalHardFloorSchemaID,
		RequiredApproval:         requiredApproval,
		Details: map[string]any{
			"precedence":                   "invariants_hard_floor",
			"approval_profile":             string(compiled.Context.ApprovalProfile),
			"hard_floor_operation_classes": toStringSlice(classes),
			"required_assurance_floor":     string(floor),
			"strongest_floor_selected":     true,
			"approval_trigger_codes":       hardFloorApprovalTriggers(action),
		},
	}
}

func hardFloorApprovalTriggers(action ActionRequest) []string {
	switch action.ActionKind {
	case ActionKindGateOverride:
		return []string{"gate_override"}
	case ActionKindBackendPosture:
		return []string{"reduced_assurance_backend"}
	case ActionKindPromotion:
		return []string{"excerpt_promotion"}
	case ActionKindSecretAccess:
		return []string{"secret_access_lease"}
	case ActionKindExecutorRun:
		return []string{"system_command_execution"}
	default:
		return []string{"system_command_execution"}
	}
}

func validateActionRequest(action ActionRequest) error {
	if action.SchemaID != actionRequestSchemaID {
		return schemaIDError(action.SchemaID, actionRequestSchemaID)
	}
	if action.SchemaVersion != actionRequestSchemaVersion {
		return schemaVersionError(action.SchemaID, action.SchemaVersion, actionRequestSchemaVersion)
	}
	actionPayload, err := json.Marshal(action)
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("marshal action request: %v", err)}
	}
	if err := validateObjectPayloadAgainstSchema(actionPayload, actionRequestSchemaPath); err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	registries, err := loadActionRegistries()
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("load action registries: %v", err)}
	}
	if _, ok := registries.actionKinds[action.ActionKind]; !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown action_kind %q (fail-closed)", action.ActionKind)}
	}
	descriptor, ok := actionPayloadByKind[action.ActionKind]
	if !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("action_kind %q missing payload descriptor (fail-closed)", action.ActionKind)}
	}
	if _, ok := registries.payloadSchemaIDs[action.ActionPayloadSchemaID]; !ok {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("unknown action_payload_schema_id %q (fail-closed)", action.ActionPayloadSchemaID)}
	}
	if action.ActionPayloadSchemaID != descriptor.schemaID {
		return &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("action_kind %q requires action_payload_schema_id %q, got %q", action.ActionKind, descriptor.schemaID, action.ActionPayloadSchemaID)}
	}
	typedPayload, err := json.Marshal(action.ActionPayload)
	if err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("marshal action payload: %v", err)}
	}
	if err := validateObjectPayloadAgainstSchema(typedPayload, descriptor.schemaPath); err != nil {
		return &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	return nil
}

func evaluateInvariantDeny(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	for _, deniedAction := range compiled.Context.FixedInvariants.DeniedActionKinds {
		if action.ActionKind == deniedAction {
			return PolicyDecision{
				SchemaID:               policyDecisionSchemaID,
				SchemaVersion:          policyDecisionSchemaVersion,
				DecisionOutcome:        DecisionDeny,
				PolicyReasonCode:       "deny_by_default",
				ManifestHash:           compiled.ManifestHash,
				PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
				ActionRequestHash:      actionHash,
				RelevantArtifactHashes: actionRelevantArtifactHashes(action),
				DetailsSchemaID:        policyEvaluationDetailsSchemaID,
				Details: map[string]any{
					"precedence":        "invariants_first",
					"denied_action":     action.ActionKind,
					"cannot_be_relaxed": true,
					"secondary_factors": []string{"fixed_invariants"},
				},
			}, true
		}
	}
	return PolicyDecision{}, false
}

func evaluateAbsentCapabilityDeny(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	for _, capID := range compiled.Context.EffectiveCapabilities {
		if capID == action.CapabilityID {
			return PolicyDecision{}, false
		}
	}
	return PolicyDecision{
		SchemaID:               policyDecisionSchemaID,
		SchemaVersion:          policyDecisionSchemaVersion,
		DecisionOutcome:        DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: actionRelevantArtifactHashes(action),
		DetailsSchemaID:        policyEvaluationDetailsSchemaID,
		Details: map[string]any{
			"precedence":                 "role_run_stage",
			"missing_capability":         action.CapabilityID,
			"fail_closed_active_context": true,
			"secondary_factors":          []string{"capability_not_in_effective_context"},
		},
	}, true
}

func evaluateAllowlistPresenceDeny(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	active := map[string]struct{}{}
	for _, ref := range compiled.Context.ActiveAllowlistRefs {
		active[ref] = struct{}{}
	}
	for _, ref := range action.AllowlistRefs {
		if _, ok := active[ref]; !ok {
			return PolicyDecision{
				SchemaID:               policyDecisionSchemaID,
				SchemaVersion:          policyDecisionSchemaVersion,
				DecisionOutcome:        DecisionDeny,
				PolicyReasonCode:       "deny_by_default",
				ManifestHash:           compiled.ManifestHash,
				PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
				ActionRequestHash:      actionHash,
				RelevantArtifactHashes: actionRelevantArtifactHashes(action),
				DetailsSchemaID:        policyEvaluationDetailsSchemaID,
				Details: map[string]any{
					"precedence":                 "allowlist_active_manifest_set",
					"missing_allowlist_ref":      ref,
					"fail_closed_active_context": true,
					"secondary_factors":          []string{"allowlist_not_in_active_manifest_set"},
				},
			}, true
		}
	}
	return PolicyDecision{}, false
}

func evaluateRuleSet(compiled *CompiledContext, action ActionRequest, actionHash string, ruleSet PolicyRuleSet) (PolicyDecision, bool) {
	matchedRule, selected, found := selectHighestPrecedenceRule(ruleSet.Rules, action)
	if !found {
		return PolicyDecision{}, false
	}
	decision := PolicyDecision{
		SchemaID:               policyDecisionSchemaID,
		SchemaVersion:          policyDecisionSchemaVersion,
		DecisionOutcome:        selected,
		PolicyReasonCode:       matchedRule.ReasonCode,
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: actionRelevantArtifactHashes(action),
		DetailsSchemaID:        matchedRule.DetailsSchemaID,
		Details: map[string]any{
			"precedence": "deny_gt_require_human_approval_gt_allow",
			"source":     policyRuleSetSchemaID,
			"rule_id":    matchedRule.RuleID,
		},
	}
	if selected == DecisionRequireHumanApproval {
		requiredSchemaID, requiredPayload := requiredApprovalForModerateProfile(compiled, action, actionHash)
		if requiredSchemaID == "" {
			requiredSchemaID = "runecode.protocol.details.policy.required_approval.rule.v0"
			requiredPayload = map[string]any{
				"approval_trigger_code":         "stage_sign_off",
				"approval_assurance_level":      string(ApprovalAssuranceSessionAuthenticated),
				"scope":                         approvalScopeForAction(action),
				"why_required":                  "Rule-set selected require_human_approval effect.",
				"changes_if_approved":           "Action may proceed once approval is granted.",
				"effects_if_denied_or_deferred": "Action remains blocked.",
				"security_posture_impact":       "moderate",
				"blocked_work":                  []string{"action_execution"},
				"approval_ttl_seconds":          1800,
				"related_hashes": map[string]any{
					"manifest_hash":            compiled.ManifestHash,
					"action_request_hash":      actionHash,
					"policy_input_hashes":      append([]string{}, compiled.PolicyInputHashes...),
					"relevant_artifact_hashes": actionRelevantArtifactHashes(action),
				},
			}
		}
		decision.RequiredApprovalSchemaID = requiredSchemaID
		decision.RequiredApproval = requiredPayload
	}
	return decision, true
}

func selectHighestPrecedenceRule(rules []PolicyRule, action ActionRequest) (PolicyRule, DecisionOutcome, bool) {
	hasAllow := false
	hasApproval := false
	hasDeny := false
	var allowRule PolicyRule
	var approvalRule PolicyRule
	var denyRule PolicyRule
	for _, rule := range rules {
		if rule.ActionKind != action.ActionKind {
			continue
		}
		if rule.CapabilityID != "" && rule.CapabilityID != action.CapabilityID {
			continue
		}
		switch rule.Effect {
		case string(DecisionDeny):
			hasDeny = true
			denyRule = rule
		case string(DecisionRequireHumanApproval):
			hasApproval = true
			approvalRule = rule
		case string(DecisionAllow):
			hasAllow = true
			allowRule = rule
		}
	}
	if hasDeny {
		return denyRule, DecisionDeny, true
	}
	if hasApproval {
		return approvalRule, DecisionRequireHumanApproval, true
	}
	if hasAllow {
		return allowRule, DecisionAllow, true
	}
	return PolicyRule{}, "", false
}

func matchedReasonCode(outcome DecisionOutcome) string {
	switch outcome {
	case DecisionDeny:
		return "deny_by_default"
	case DecisionRequireHumanApproval:
		return "approval_required"
	case DecisionAllow:
		return "allow_manifest_opt_in"
	default:
		panic(fmt.Sprintf("unknown outcome %q", outcome))
	}
}
