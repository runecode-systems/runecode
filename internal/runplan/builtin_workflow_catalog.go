package runplan

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/workflowpackassets"
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

type builtInWorkflowManifest struct {
	CatalogVersion string                         `json:"catalog_version"`
	Entries        []builtInWorkflowManifestEntry `json:"entries"`
}

type builtInWorkflowManifestEntry struct {
	WorkflowID                          string   `json:"workflow_id"`
	WorkflowFamily                      string   `json:"workflow_family"`
	WorkflowVersion                     string   `json:"workflow_version"`
	Provenance                          string   `json:"provenance"`
	WorkflowAssetPath                   string   `json:"workflow_asset_path"`
	ProcessAssetPath                    string   `json:"process_asset_path"`
	WorkflowDefinitionHash              string   `json:"workflow_definition_hash"`
	ProcessDefinitionHash               string   `json:"process_definition_hash"`
	ImplementationInputSetSchemaID      string   `json:"implementation_input_set_schema_id"`
	ImplementationInputBindingFields    []string `json:"implementation_input_binding_fields"`
	ExecutionAuthorityModel             string   `json:"execution_authority_model"`
	DependencyResolutionModel           string   `json:"dependency_resolution_model"`
	DependencyScopeApprovalModel        string   `json:"dependency_scope_approval_model"`
	SubstrateLifecyclePolicy            string   `json:"substrate_lifecycle_policy"`
	ExecutionDriftBindingFields         []string `json:"execution_drift_binding_fields"`
	WaitSemanticsModel                  string   `json:"wait_semantics_model"`
	ContinuationCompatibility           string   `json:"continuation_compatibility"`
	SeparatesApprovalAndAutonomy        bool     `json:"separates_approval_and_autonomy"`
	DraftArtifactSchemaID               string   `json:"draft_artifact_schema_id"`
	DraftEvidenceLinkKinds              []string `json:"draft_evidence_link_kinds"`
	PromoteApplyWorkflowID              string   `json:"promote_apply_workflow_id"`
	WritableRuneContextPath             []string `json:"writable_runecontext_path"`
	RequiresValidatedProjectSubstrate   bool     `json:"requires_validated_project_substrate"`
	FailClosedOnProjectSubstratePosture bool     `json:"fail_closed_on_project_substrate_posture"`
	MutationPathModel                   string   `json:"mutation_path_model"`
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
	manifest, err := loadBuiltInWorkflowManifest(workflowpackassets.BuiltInFS())
	if err != nil {
		panic(fmt.Sprintf("load built-in workflow catalog: %v", err))
	}
	result, err := buildBuiltInWorkflowCatalogFromManifest(manifest, workflowpackassets.BuiltInFS())
	if err != nil {
		panic(fmt.Sprintf("build built-in workflow catalog: %v", err))
	}
	return result
}

func buildBuiltInWorkflowCatalogFromManifest(manifest builtInWorkflowManifest, assetFS fs.FS) (map[string]BuiltInWorkflowCatalogEntry, error) {
	result := make(map[string]BuiltInWorkflowCatalogEntry, len(manifest.Entries))
	for _, raw := range manifest.Entries {
		if _, exists := result[raw.WorkflowID]; exists {
			return nil, fmt.Errorf("duplicate built-in workflow_id %q", raw.WorkflowID)
		}
		entry, err := buildCatalogEntryFromManifest(raw, assetFS)
		if err != nil {
			return nil, fmt.Errorf("invalid built-in catalog entry %q: %w", raw.WorkflowID, err)
		}
		result[entry.WorkflowID] = entry
	}
	return result, nil
}

func loadBuiltInWorkflowManifest(assetFS fs.FS) (builtInWorkflowManifest, error) {
	payload, err := fs.ReadFile(assetFS, workflowpackassets.BuiltInManifestPath)
	if err != nil {
		return builtInWorkflowManifest{}, err
	}
	var manifest builtInWorkflowManifest
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return builtInWorkflowManifest{}, err
	}
	if manifest.CatalogVersion != builtInWorkflowCatalogVersion {
		return builtInWorkflowManifest{}, fmt.Errorf("catalog_version %q does not match %q", manifest.CatalogVersion, builtInWorkflowCatalogVersion)
	}
	return manifest, nil
}

func buildCatalogEntryFromManifest(raw builtInWorkflowManifestEntry, assetFS fs.FS) (BuiltInWorkflowCatalogEntry, error) {
	process, workflowHash, processHash, err := loadAndValidateBuiltInWorkflowAssets(raw, assetFS)
	if err != nil {
		return BuiltInWorkflowCatalogEntry{}, err
	}
	return catalogEntryFromManifestMetadata(raw, process.ProcessID, workflowHash, processHash), nil
}

func loadAndValidateBuiltInWorkflowAssets(raw builtInWorkflowManifestEntry, assetFS fs.FS) (ProcessDefinition, string, string, error) {
	processPayload, err := fs.ReadFile(assetFS, raw.ProcessAssetPath)
	if err != nil {
		return ProcessDefinition{}, "", "", err
	}
	process, processHash, err := decodeProcessDefinition(processPayload)
	if err != nil {
		return ProcessDefinition{}, "", "", err
	}
	if strings.TrimSpace(processHash) != strings.TrimSpace(raw.ProcessDefinitionHash) {
		return ProcessDefinition{}, "", "", fmt.Errorf("process digest mismatch for %q", raw.WorkflowID)
	}

	workflowPayloadRaw, err := fs.ReadFile(assetFS, raw.WorkflowAssetPath)
	if err != nil {
		return ProcessDefinition{}, "", "", err
	}
	workflowPayload := strings.ReplaceAll(string(workflowPayloadRaw), "{{PROCESS_HASH}}", processHash)
	workflow, workflowHash, err := decodeWorkflowDefinition([]byte(workflowPayload))
	if err != nil {
		return ProcessDefinition{}, "", "", err
	}
	if strings.TrimSpace(workflowHash) != strings.TrimSpace(raw.WorkflowDefinitionHash) {
		return ProcessDefinition{}, "", "", fmt.Errorf("workflow digest mismatch for %q", raw.WorkflowID)
	}
	if strings.TrimSpace(workflow.WorkflowID) != strings.TrimSpace(raw.WorkflowID) {
		return ProcessDefinition{}, "", "", fmt.Errorf("workflow_id mismatch for %q", raw.WorkflowID)
	}
	if strings.TrimSpace(workflow.SelectedProcessID) != strings.TrimSpace(process.ProcessID) {
		return ProcessDefinition{}, "", "", fmt.Errorf("selected_process_id mismatch for %q", raw.WorkflowID)
	}
	return process, workflowHash, processHash, nil
}

func catalogEntryFromManifestMetadata(raw builtInWorkflowManifestEntry, processID, workflowHash, processHash string) BuiltInWorkflowCatalogEntry {
	return BuiltInWorkflowCatalogEntry{
		WorkflowID:                          raw.WorkflowID,
		WorkflowFamily:                      raw.WorkflowFamily,
		WorkflowVersion:                     raw.WorkflowVersion,
		Provenance:                          raw.Provenance,
		SelectedProcessID:                   processID,
		WorkflowDefinitionHash:              workflowHash,
		ProcessDefinitionHash:               processHash,
		ImplementationInputSetSchemaID:      raw.ImplementationInputSetSchemaID,
		ImplementationInputBindingFields:    append([]string(nil), raw.ImplementationInputBindingFields...),
		ExecutionAuthorityModel:             raw.ExecutionAuthorityModel,
		DependencyResolutionModel:           raw.DependencyResolutionModel,
		DependencyScopeApprovalModel:        raw.DependencyScopeApprovalModel,
		SubstrateLifecyclePolicy:            raw.SubstrateLifecyclePolicy,
		ExecutionDriftBindingFields:         append([]string(nil), raw.ExecutionDriftBindingFields...),
		WaitSemanticsModel:                  raw.WaitSemanticsModel,
		ContinuationCompatibility:           raw.ContinuationCompatibility,
		SeparatesApprovalAndAutonomy:        raw.SeparatesApprovalAndAutonomy,
		DraftArtifactSchemaID:               raw.DraftArtifactSchemaID,
		DraftEvidenceLinkKinds:              append([]string(nil), raw.DraftEvidenceLinkKinds...),
		PromoteApplyWorkflowID:              raw.PromoteApplyWorkflowID,
		WritableRuneContextPath:             append([]string(nil), raw.WritableRuneContextPath...),
		RequiresValidatedProjectSubstrate:   raw.RequiresValidatedProjectSubstrate,
		FailClosedOnProjectSubstratePosture: raw.FailClosedOnProjectSubstratePosture,
		MutationPathModel:                   raw.MutationPathModel,
	}
}
