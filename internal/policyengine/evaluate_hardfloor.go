package policyengine

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

func hardFloorApprovalDecision(compiled *CompiledContext, action ActionRequest, actionHash string, classes []HardFloorOperationClass, floor ApprovalAssuranceLevel) PolicyDecision {
	requiredApproval := hardFloorRequiredApprovalPayload(compiled, action, actionHash, classes, floor)
	details := hardFloorDecisionDetails(compiled, action, classes, floor)
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
		Details:                  details,
	}
}

func hardFloorRequiredApprovalPayload(compiled *CompiledContext, action ActionRequest, actionHash string, classes []HardFloorOperationClass, floor ApprovalAssuranceLevel) map[string]any {
	return map[string]any{
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
}

func hardFloorDecisionDetails(compiled *CompiledContext, action ActionRequest, classes []HardFloorOperationClass, floor ApprovalAssuranceLevel) map[string]any {
	return map[string]any{
		"precedence":                   "invariants_hard_floor",
		"approval_profile":             string(compiled.Context.ApprovalProfile),
		"hard_floor_operation_classes": toStringSlice(classes),
		"required_assurance_floor":     string(floor),
		"strongest_floor_selected":     true,
		"approval_trigger_codes":       hardFloorApprovalTriggers(action),
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
