package runplan

import (
	"fmt"
	"sort"
	"strings"
)

const builtInWorkflowCatalogVersion = "v0"

type BuiltInWorkflowCatalogEntry struct {
	WorkflowID                          string
	WorkflowFamily                      string
	WorkflowVersion                     string
	Provenance                          string
	SelectedProcessID                   string
	WorkflowDefinitionHash              string
	ProcessDefinitionHash               string
	ImplementationInputSetSchemaID      string
	ImplementationInputBindingFields    []string
	ExecutionAuthorityModel             string
	DependencyResolutionModel           string
	DependencyScopeApprovalModel        string
	SubstrateLifecyclePolicy            string
	ExecutionDriftBindingFields         []string
	WaitSemanticsModel                  string
	ContinuationCompatibility           string
	SeparatesApprovalAndAutonomy        bool
	DraftArtifactSchemaID               string
	DraftEvidenceLinkKinds              []string
	PromoteApplyWorkflowID              string
	WritableRuneContextPath             []string
	RequiresValidatedProjectSubstrate   bool
	FailClosedOnProjectSubstratePosture bool
	MutationPathModel                   string
}

var builtInWorkflowCatalogByID = mustBuildBuiltInWorkflowCatalog()

func BuiltInWorkflowCatalogV0() []BuiltInWorkflowCatalogEntry {
	entries := make([]BuiltInWorkflowCatalogEntry, 0, len(builtInWorkflowCatalogByID))
	for _, entry := range builtInWorkflowCatalogByID {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].WorkflowID < entries[j].WorkflowID })
	return entries
}

func validateBuiltInWorkflowReservation(workflow WorkflowDefinition, process ProcessDefinition, workflowHash, processHash string) error {
	workflowID := strings.TrimSpace(workflow.WorkflowID)
	entry, ok := builtInWorkflowCatalogByID[workflowID]
	if !ok {
		return nil
	}
	if strings.TrimSpace(workflow.WorkflowVersion) != entry.WorkflowVersion {
		return fmt.Errorf("workflow_id %q is reserved for built-in family %q version %q", entry.WorkflowID, entry.WorkflowFamily, entry.WorkflowVersion)
	}
	if strings.TrimSpace(process.ProcessID) != entry.SelectedProcessID {
		return fmt.Errorf("workflow_id %q is reserved for built-in process_id %q", entry.WorkflowID, entry.SelectedProcessID)
	}
	if strings.TrimSpace(workflowHash) != entry.WorkflowDefinitionHash || strings.TrimSpace(processHash) != entry.ProcessDefinitionHash {
		return fmt.Errorf("workflow_id %q is reserved for product-shipped built-in definitions in catalog %q and cannot be repository-overridden", entry.WorkflowID, builtInWorkflowCatalogVersion)
	}
	return nil
}

func mustBuildBuiltInWorkflowCatalog() map[string]BuiltInWorkflowCatalogEntry {
	entries := builtInWorkflowCatalogEntryDefinitions()
	result := make(map[string]BuiltInWorkflowCatalogEntry, len(entries))
	for _, raw := range entries {
		result[raw.workflowID] = mustBuildCatalogEntry(raw)
	}
	return result
}

func mustBuildCatalogEntry(raw builtInWorkflowCatalogEntryDefinition) BuiltInWorkflowCatalogEntry {
	_, processHash, err := decodeProcessDefinition([]byte(raw.processTemplate))
	if err != nil {
		panic(fmt.Sprintf("invalid built-in process definition for %s: %v", raw.workflowID, err))
	}
	workflowPayload := strings.ReplaceAll(raw.workflowTemplate, "{{PROCESS_HASH}}", processHash)
	_, workflowHash, err := decodeWorkflowDefinition([]byte(workflowPayload))
	if err != nil {
		panic(fmt.Sprintf("invalid built-in workflow definition for %s: %v", raw.workflowID, err))
	}
	return builtInWorkflowCatalogEntry(raw, workflowHash, processHash)
}

func builtInWorkflowCatalogEntry(raw builtInWorkflowCatalogEntryDefinition, workflowHash, processHash string) BuiltInWorkflowCatalogEntry {
	return BuiltInWorkflowCatalogEntry{
		WorkflowID:                          raw.workflowID,
		WorkflowFamily:                      raw.family,
		WorkflowVersion:                     raw.version,
		Provenance:                          raw.provenance,
		SelectedProcessID:                   raw.selectedProcess,
		WorkflowDefinitionHash:              workflowHash,
		ProcessDefinitionHash:               processHash,
		ImplementationInputSetSchemaID:      raw.implementationInput,
		ImplementationInputBindingFields:    append([]string(nil), raw.inputBindingFields...),
		ExecutionAuthorityModel:             raw.execAuthority,
		DependencyResolutionModel:           raw.dependencyModel,
		DependencyScopeApprovalModel:        raw.dependencyApproval,
		SubstrateLifecyclePolicy:            raw.substratePolicy,
		ExecutionDriftBindingFields:         append([]string(nil), raw.driftBindings...),
		WaitSemanticsModel:                  raw.waitModel,
		ContinuationCompatibility:           raw.continuationModel,
		SeparatesApprovalAndAutonomy:        raw.separateControls,
		DraftArtifactSchemaID:               raw.draftSchemaID,
		DraftEvidenceLinkKinds:              append([]string(nil), raw.draftEvidence...),
		PromoteApplyWorkflowID:              raw.promoteWorkflow,
		WritableRuneContextPath:             append([]string(nil), raw.writableScope...),
		RequiresValidatedProjectSubstrate:   raw.requiresSubstrate,
		FailClosedOnProjectSubstratePosture: raw.failClosedSubstrate,
		MutationPathModel:                   raw.mutationPathModel,
	}
}

type builtInWorkflowCatalogEntryDefinition struct {
	workflowID          string
	family              string
	version             string
	provenance          string
	selectedProcess     string
	workflowTemplate    string
	processTemplate     string
	draftSchemaID       string
	draftEvidence       []string
	promoteWorkflow     string
	writableScope       []string
	requiresSubstrate   bool
	failClosedSubstrate bool
	mutationPathModel   string
	implementationInput string
	inputBindingFields  []string
	execAuthority       string
	dependencyModel     string
	dependencyApproval  string
	substratePolicy     string
	driftBindings       []string
	waitModel           string
	continuationModel   string
	separateControls    bool
}

func builtInWorkflowCatalogEntryDefinitions() []builtInWorkflowCatalogEntryDefinition {
	return []builtInWorkflowCatalogEntryDefinition{
		builtInChangeDraftDefinition(),
		builtInSpecDraftDefinition(),
		builtInDraftPromoteDefinition(),
		builtInApprovedImplementationDefinition(),
	}
}

func builtInChangeDraftDefinition() builtInWorkflowCatalogEntryDefinition {
	return builtInWorkflowCatalogEntryDefinition{
		workflowID:          "builtin_rc_change_draft_v0",
		family:              "runecontext.change_draft",
		version:             "0.1.0",
		provenance:          "product-shipped-reviewed:first-party",
		selectedProcess:     "builtin_rc_change_draft_process_v0",
		workflowTemplate:    `{"schema_id":"runecode.protocol.v0.WorkflowDefinition","schema_version":"0.5.0","workflow_id":"builtin_rc_change_draft_v0","workflow_version":"0.1.0","selected_process_id":"builtin_rc_change_draft_process_v0","selected_process_definition_hash":"{{PROCESS_HASH}}","reviewed_process_artifacts":[{"process_id":"builtin_rc_change_draft_process_v0","process_definition_hash":"{{PROCESS_HASH}}"}],"approval_profile":"moderate","autonomy_posture":"operator_guided"}`,
		processTemplate:     `{"schema_id":"runecode.protocol.v0.ProcessDefinition","schema_version":"0.4.0","process_id":"builtin_rc_change_draft_process_v0","executor_bindings":[{"binding_id":"binding_workspace_runner","executor_id":"workspace-runner","executor_class":"workspace_ordinary","allowed_role_kinds":["workspace-edit"]}],"gate_definitions":[{"schema_id":"runecode.protocol.v0.GateDefinition","schema_version":"0.2.0","gate":{"schema_id":"runecode.protocol.v0.GateContract","schema_version":"0.1.0","gate_id":"build_gate","gate_kind":"build","gate_version":"1.0.0","normalized_inputs":[{"input_id":"source_tree","input_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"}],"plan_binding":{"checkpoint_code":"step_validation_started","order_index":0},"retry_semantics":{"retry_mode":"new_attempt_required","max_attempts":3},"override_semantics":{"override_mode":"policy_action_required","action_kind":"action_gate_override","approval_trigger_code":"gate_override"}},"checkpoint_code":"step_validation_started","order_index":0,"stage_id":"validation","step_id":"change_draft_step","role_instance_id":"workspace_editor_1","executor_binding_id":"binding_workspace_runner"}],"dependency_edges":[]}`,
		draftSchemaID:       "runecode.protocol.v0.RuneContextChangeDraftArtifact",
		draftEvidence:       []string{"gate_evidence", "source_prompt_identity", "validated_project_substrate_digest"},
		promoteWorkflow:     "builtin_rc_draft_promote_v0",
		requiresSubstrate:   true,
		failClosedSubstrate: true,
		mutationPathModel:   "artifact_only_generation",
	}
}

func builtInSpecDraftDefinition() builtInWorkflowCatalogEntryDefinition {
	return builtInWorkflowCatalogEntryDefinition{
		workflowID:          "builtin_rc_spec_draft_v0",
		family:              "runecontext.spec_draft",
		version:             "0.1.0",
		provenance:          "product-shipped-reviewed:first-party",
		selectedProcess:     "builtin_rc_spec_draft_process_v0",
		workflowTemplate:    `{"schema_id":"runecode.protocol.v0.WorkflowDefinition","schema_version":"0.5.0","workflow_id":"builtin_rc_spec_draft_v0","workflow_version":"0.1.0","selected_process_id":"builtin_rc_spec_draft_process_v0","selected_process_definition_hash":"{{PROCESS_HASH}}","reviewed_process_artifacts":[{"process_id":"builtin_rc_spec_draft_process_v0","process_definition_hash":"{{PROCESS_HASH}}"}],"approval_profile":"moderate","autonomy_posture":"operator_guided"}`,
		processTemplate:     `{"schema_id":"runecode.protocol.v0.ProcessDefinition","schema_version":"0.4.0","process_id":"builtin_rc_spec_draft_process_v0","executor_bindings":[{"binding_id":"binding_workspace_runner","executor_id":"workspace-runner","executor_class":"workspace_ordinary","allowed_role_kinds":["workspace-edit"]}],"gate_definitions":[{"schema_id":"runecode.protocol.v0.GateDefinition","schema_version":"0.2.0","gate":{"schema_id":"runecode.protocol.v0.GateContract","schema_version":"0.1.0","gate_id":"build_gate","gate_kind":"build","gate_version":"1.0.0","normalized_inputs":[{"input_id":"source_tree","input_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"}],"plan_binding":{"checkpoint_code":"step_validation_started","order_index":0},"retry_semantics":{"retry_mode":"new_attempt_required","max_attempts":3},"override_semantics":{"override_mode":"policy_action_required","action_kind":"action_gate_override","approval_trigger_code":"gate_override"}},"checkpoint_code":"step_validation_started","order_index":0,"stage_id":"validation","step_id":"spec_draft_step","role_instance_id":"workspace_editor_1","executor_binding_id":"binding_workspace_runner"}],"dependency_edges":[]}`,
		draftSchemaID:       "runecode.protocol.v0.RuneContextSpecDraftArtifact",
		draftEvidence:       []string{"gate_evidence", "source_prompt_identity", "validated_project_substrate_digest"},
		promoteWorkflow:     "builtin_rc_draft_promote_v0",
		requiresSubstrate:   true,
		failClosedSubstrate: true,
		mutationPathModel:   "artifact_only_generation",
	}
}

func builtInDraftPromoteDefinition() builtInWorkflowCatalogEntryDefinition {
	return builtInWorkflowCatalogEntryDefinition{
		workflowID:          "builtin_rc_draft_promote_v0",
		family:              "runecontext.draft_promote_apply",
		version:             "0.1.0",
		provenance:          "product-shipped-reviewed:first-party",
		selectedProcess:     "builtin_rc_draft_promote_process_v0",
		workflowTemplate:    `{"schema_id":"runecode.protocol.v0.WorkflowDefinition","schema_version":"0.5.0","workflow_id":"builtin_rc_draft_promote_v0","workflow_version":"0.1.0","selected_process_id":"builtin_rc_draft_promote_process_v0","selected_process_definition_hash":"{{PROCESS_HASH}}","reviewed_process_artifacts":[{"process_id":"builtin_rc_draft_promote_process_v0","process_definition_hash":"{{PROCESS_HASH}}"}],"approval_profile":"moderate","autonomy_posture":"operator_guided"}`,
		processTemplate:     `{"schema_id":"runecode.protocol.v0.ProcessDefinition","schema_version":"0.4.0","process_id":"builtin_rc_draft_promote_process_v0","executor_bindings":[{"binding_id":"binding_workspace_runner","executor_id":"workspace-runner","executor_class":"workspace_ordinary","allowed_role_kinds":["workspace-edit"]}],"gate_definitions":[{"schema_id":"runecode.protocol.v0.GateDefinition","schema_version":"0.2.0","gate":{"schema_id":"runecode.protocol.v0.GateContract","schema_version":"0.1.0","gate_id":"build_gate","gate_kind":"build","gate_version":"1.0.0","normalized_inputs":[{"input_id":"source_tree","input_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"}],"plan_binding":{"checkpoint_code":"step_validation_started","order_index":0},"retry_semantics":{"retry_mode":"new_attempt_required","max_attempts":3},"override_semantics":{"override_mode":"policy_action_required","action_kind":"action_gate_override","approval_trigger_code":"gate_override"}},"checkpoint_code":"step_validation_started","order_index":0,"stage_id":"validation","step_id":"draft_promote_step","role_instance_id":"workspace_editor_1","executor_binding_id":"binding_workspace_runner"}],"dependency_edges":[]}`,
		writableScope:       []string{"runecontext/changes/", "runecontext/specs/"},
		requiresSubstrate:   true,
		failClosedSubstrate: true,
		mutationPathModel:   "shared_broker_mutation_approval_audit_verification",
	}
}

func builtInApprovedImplementationDefinition() builtInWorkflowCatalogEntryDefinition {
	return builtInWorkflowCatalogEntryDefinition{
		workflowID:          "builtin_rc_approved_implementation_v0",
		family:              "runecontext.approved_implementation",
		version:             "0.1.0",
		provenance:          "product-shipped-reviewed:first-party",
		selectedProcess:     "builtin_rc_approved_implementation_process_v0",
		workflowTemplate:    `{"schema_id":"runecode.protocol.v0.WorkflowDefinition","schema_version":"0.5.0","workflow_id":"builtin_rc_approved_implementation_v0","workflow_version":"0.1.0","selected_process_id":"builtin_rc_approved_implementation_process_v0","selected_process_definition_hash":"{{PROCESS_HASH}}","reviewed_process_artifacts":[{"process_id":"builtin_rc_approved_implementation_process_v0","process_definition_hash":"{{PROCESS_HASH}}"}],"approval_profile":"moderate","autonomy_posture":"operator_guided"}`,
		processTemplate:     `{"schema_id":"runecode.protocol.v0.ProcessDefinition","schema_version":"0.4.0","process_id":"builtin_rc_approved_implementation_process_v0","executor_bindings":[{"binding_id":"binding_workspace_runner","executor_id":"workspace-runner","executor_class":"workspace_ordinary","allowed_role_kinds":["workspace-edit"]}],"gate_definitions":[{"schema_id":"runecode.protocol.v0.GateDefinition","schema_version":"0.2.0","gate":{"schema_id":"runecode.protocol.v0.GateContract","schema_version":"0.1.0","gate_id":"build_gate","gate_kind":"build","gate_version":"1.0.0","normalized_inputs":[{"input_id":"source_tree","input_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111"}],"plan_binding":{"checkpoint_code":"step_validation_started","order_index":0},"retry_semantics":{"retry_mode":"new_attempt_required","max_attempts":3},"override_semantics":{"override_mode":"policy_action_required","action_kind":"action_gate_override","approval_trigger_code":"gate_override"}},"checkpoint_code":"step_validation_started","order_index":0,"stage_id":"validation","step_id":"approved_implementation_step","role_instance_id":"workspace_editor_1","executor_binding_id":"binding_workspace_runner"}],"dependency_edges":[]}`,
		implementationInput: "runecode.protocol.v0.RuneContextApprovedImplementationInputSet",
		inputBindingFields:  builtInImplementationInputBindingFields(),
		execAuthority:       "broker_compiled_immutable_run_plan",
		dependencyModel:     "broker_owned_dependency_fetch_offline_cache_and_artifact_handoff_public_registry_first",
		dependencyApproval:  "dependency_scope_enablement_or_expansion_requires_separate_approval_cache_miss_does_not",
		substratePolicy:     "no_implicit_substrate_init_upgrade_or_rewrite",
		driftBindings:       builtInImplementationDriftBindings(),
		waitModel:           "shared_waiting_operator_input_and_waiting_approval",
		continuationModel:   "dependency_aware_scoped_blocking_chg_050_compatible",
		separateControls:    true,
		requiresSubstrate:   true,
		failClosedSubstrate: true,
		mutationPathModel:   "shared_broker_mutation_approval_audit_verification",
	}
}

func builtInImplementationInputBindingFields() []string {
	return []string{"input_set_digest", "approved_input_digests", "workflow_definition_hash", "process_definition_hash", "approval_profile", "autonomy_posture", "validated_project_substrate_digest", "project_substrate_snapshot_digest", "control_input_digest", "repo_identity_digest", "repo_state_identity_digest"}
}

func builtInImplementationDriftBindings() []string {
	return []string{"approved_input_digests", "workflow_definition_hash", "process_definition_hash", "control_input_digest", "repo_state_identity_digest", "validated_project_substrate_digest"}
}
