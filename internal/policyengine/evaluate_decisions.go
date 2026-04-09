package policyengine

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
