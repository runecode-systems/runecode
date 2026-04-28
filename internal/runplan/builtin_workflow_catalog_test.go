package runplan

import (
	"strings"
	"testing"
)

func TestBuiltInDraftingCatalogEntriesRequireValidatedProjectSubstrate(t *testing.T) {
	entries := BuiltInWorkflowCatalogV0()
	for _, entry := range entries {
		switch entry.WorkflowID {
		case "builtin_rc_change_draft_v0", "builtin_rc_spec_draft_v0", "builtin_rc_draft_promote_v0":
			if !entry.RequiresValidatedProjectSubstrate {
				t.Fatalf("%s missing validated substrate requirement", entry.WorkflowID)
			}
			if !entry.FailClosedOnProjectSubstratePosture {
				t.Fatalf("%s missing fail-closed substrate posture", entry.WorkflowID)
			}
		}
	}
}

func TestBuiltInWorkflowCatalogV0DefinesStableFamiliesAndProvenance(t *testing.T) {
	entries := BuiltInWorkflowCatalogV0()
	if len(entries) != 4 {
		t.Fatalf("catalog entries len = %d, want 4", len(entries))
	}
	for _, entry := range entries {
		assertBuiltInCatalogEntryIdentity(t, entry)
	}
}

func assertBuiltInCatalogEntryIdentity(t *testing.T, entry BuiltInWorkflowCatalogEntry) {
	t.Helper()
	if !strings.HasPrefix(entry.WorkflowID, "builtin_rc_") {
		t.Fatalf("workflow_id %q missing built-in prefix", entry.WorkflowID)
	}
	if entry.WorkflowVersion != "0.1.0" {
		t.Fatalf("workflow_version for %q = %q, want 0.1.0", entry.WorkflowID, entry.WorkflowVersion)
	}
	if entry.Provenance != "product-shipped-reviewed:first-party" {
		t.Fatalf("provenance for %q = %q", entry.WorkflowID, entry.Provenance)
	}
	if entry.WorkflowFamily == "" || entry.SelectedProcessID == "" {
		t.Fatalf("catalog entry missing family/process: %+v", entry)
	}
}

func TestBuiltInDraftingCatalogEntriesDefineTypedArtifactsAndPromotionBinding(t *testing.T) {
	changeDraft := requireBuiltInCatalogEntry(t, "builtin_rc_change_draft_v0")
	if changeDraft.DraftArtifactSchemaID != "runecode.protocol.v0.RuneContextChangeDraftArtifact" {
		t.Fatalf("change draft schema = %q", changeDraft.DraftArtifactSchemaID)
	}
	if changeDraft.PromoteApplyWorkflowID != "builtin_rc_draft_promote_v0" {
		t.Fatalf("change draft promote workflow = %q", changeDraft.PromoteApplyWorkflowID)
	}
	if len(changeDraft.DraftEvidenceLinkKinds) == 0 {
		t.Fatal("change draft evidence linkage must be non-empty")
	}

	specDraft := requireBuiltInCatalogEntry(t, "builtin_rc_spec_draft_v0")
	if specDraft.DraftArtifactSchemaID != "runecode.protocol.v0.RuneContextSpecDraftArtifact" {
		t.Fatalf("spec draft schema = %q", specDraft.DraftArtifactSchemaID)
	}
	if specDraft.PromoteApplyWorkflowID != "builtin_rc_draft_promote_v0" {
		t.Fatalf("spec draft promote workflow = %q", specDraft.PromoteApplyWorkflowID)
	}
}

func TestBuiltInDraftPromoteCatalogEntryDefinesNarrowMutationScope(t *testing.T) {
	promote := requireBuiltInCatalogEntry(t, "builtin_rc_draft_promote_v0")
	if promote.MutationPathModel != "shared_broker_mutation_approval_audit_verification" {
		t.Fatalf("mutation path model = %q", promote.MutationPathModel)
	}
	if len(promote.WritableRuneContextPath) != 2 {
		t.Fatalf("writable scope len = %d, want 2", len(promote.WritableRuneContextPath))
	}
	if promote.WritableRuneContextPath[0] != "runecontext/changes/" || promote.WritableRuneContextPath[1] != "runecontext/specs/" {
		t.Fatalf("writable scope = %#v", promote.WritableRuneContextPath)
	}
}

func TestBuiltInApprovedImplementationCatalogEntryDefinesReviewedInputSetAndBindings(t *testing.T) {
	impl := requireBuiltInCatalogEntry(t, "builtin_rc_approved_implementation_v0")
	if impl.ImplementationInputSetSchemaID != "runecode.protocol.v0.RuneContextApprovedImplementationInputSet" {
		t.Fatalf("implementation_input_set_schema_id = %q", impl.ImplementationInputSetSchemaID)
	}
	if len(impl.ImplementationInputBindingFields) == 0 {
		t.Fatal("implementation input binding fields must be non-empty")
	}
	required := map[string]bool{
		"approved_input_digests":             false,
		"workflow_definition_hash":           false,
		"process_definition_hash":            false,
		"approval_profile":                   false,
		"autonomy_posture":                   false,
		"validated_project_substrate_digest": false,
		"project_substrate_snapshot_digest":  false,
		"control_input_digest":               false,
		"repo_identity_digest":               false,
		"repo_state_identity_digest":         false,
	}
	for _, field := range impl.ImplementationInputBindingFields {
		if _, ok := required[field]; ok {
			required[field] = true
		}
	}
	for field, seen := range required {
		if !seen {
			t.Fatalf("missing required implementation binding field %q in %#v", field, impl.ImplementationInputBindingFields)
		}
	}
}

func TestBuiltInApprovedImplementationCatalogEntryBindsSharedRuntimeMutationDependencyAndWaitModels(t *testing.T) {
	impl := requireBuiltInCatalogEntry(t, "builtin_rc_approved_implementation_v0")
	if impl.ExecutionAuthorityModel != "broker_compiled_immutable_run_plan" {
		t.Fatalf("execution authority model = %q", impl.ExecutionAuthorityModel)
	}
	if impl.MutationPathModel != "shared_broker_mutation_approval_audit_verification" {
		t.Fatalf("mutation path model = %q", impl.MutationPathModel)
	}
	if !strings.Contains(impl.DependencyResolutionModel, "broker_owned_dependency_fetch_offline_cache") {
		t.Fatalf("dependency model = %q", impl.DependencyResolutionModel)
	}
	if !strings.Contains(impl.DependencyResolutionModel, "public_registry_first") {
		t.Fatalf("dependency model = %q, want public_registry_first posture", impl.DependencyResolutionModel)
	}
	if impl.DependencyScopeApprovalModel != "dependency_scope_enablement_or_expansion_requires_separate_approval_cache_miss_does_not" {
		t.Fatalf("dependency scope approval model = %q", impl.DependencyScopeApprovalModel)
	}
	if impl.SubstrateLifecyclePolicy != "no_implicit_substrate_init_upgrade_or_rewrite" {
		t.Fatalf("substrate lifecycle policy = %q", impl.SubstrateLifecyclePolicy)
	}
	if impl.WaitSemanticsModel != "shared_waiting_operator_input_and_waiting_approval" {
		t.Fatalf("wait semantics model = %q", impl.WaitSemanticsModel)
	}
	if impl.ContinuationCompatibility != "dependency_aware_scoped_blocking_chg_050_compatible" {
		t.Fatalf("continuation compatibility = %q", impl.ContinuationCompatibility)
	}
	if !impl.SeparatesApprovalAndAutonomy {
		t.Fatal("approved implementation entry must keep approval_profile and autonomy_posture separate")
	}
}

func requireBuiltInCatalogEntry(t *testing.T, workflowID string) BuiltInWorkflowCatalogEntry {
	t.Helper()
	for _, entry := range BuiltInWorkflowCatalogV0() {
		if entry.WorkflowID == workflowID {
			return entry
		}
	}
	t.Fatalf("missing %s", workflowID)
	return BuiltInWorkflowCatalogEntry{}
}
