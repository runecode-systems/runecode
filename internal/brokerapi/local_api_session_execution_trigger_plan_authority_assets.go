package brokerapi

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/runplan"
	"github.com/runecode-systems/runecode/internal/workflowpackassets"
)

func builtInCatalogEntryForWorkflowOperation(operation string) (runplan.BuiltInWorkflowCatalogEntry, error) {
	workflowID, err := builtInWorkflowIDForOperation(operation)
	if err != nil {
		return runplan.BuiltInWorkflowCatalogEntry{}, err
	}
	for _, entry := range runplan.BuiltInWorkflowCatalogV0() {
		if strings.TrimSpace(entry.WorkflowID) == workflowID {
			return entry, nil
		}
	}
	return runplan.BuiltInWorkflowCatalogEntry{}, fmt.Errorf("built-in workflow catalog entry missing for workflow operation %q", strings.TrimSpace(operation))
}

func builtInWorkflowIDForOperation(operation string) (string, error) {
	switch strings.TrimSpace(operation) {
	case sessionWorkflowOperationChangeDraft:
		return "builtin_rc_change_draft_v0", nil
	case sessionWorkflowOperationSpecDraft:
		return "builtin_rc_spec_draft_v0", nil
	case sessionWorkflowOperationDraftPromoteApply:
		return "builtin_rc_draft_promote_v0", nil
	case sessionWorkflowOperationApprovedImplementation:
		return "builtin_rc_approved_implementation_v0", nil
	default:
		return "", fmt.Errorf("unsupported workflow operation %q", strings.TrimSpace(operation))
	}
}

func builtInWorkflowAssetPayloads(workflowID string) ([]byte, []byte, error) {
	workflowPath, processPath, err := builtInAssetPathsForWorkflow(workflowID)
	if err != nil {
		return nil, nil, err
	}
	assetFS := workflowpackassets.BuiltInFS()
	processPayload, err := fs.ReadFile(assetFS, processPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read built-in process asset %q: %w", processPath, err)
	}
	processCanonical, err := artifacts.CanonicalizeJSONBytes(processPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("canonicalize built-in process asset %q: %w", processPath, err)
	}
	processDigest := artifacts.DigestBytes(processCanonical)
	workflowTemplate, err := fs.ReadFile(assetFS, workflowPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read built-in workflow asset %q: %w", workflowPath, err)
	}
	workflowResolved := strings.ReplaceAll(string(workflowTemplate), "{{PROCESS_HASH}}", processDigest)
	workflowCanonical, err := artifacts.CanonicalizeJSONBytes([]byte(workflowResolved))
	if err != nil {
		return nil, nil, fmt.Errorf("canonicalize built-in workflow asset %q: %w", workflowPath, err)
	}
	return workflowCanonical, processCanonical, nil
}

func builtInAssetPathsForWorkflow(workflowID string) (string, string, error) {
	switch strings.TrimSpace(workflowID) {
	case "builtin_rc_change_draft_v0":
		return "builtins/v0/change_draft.workflow.json", "builtins/v0/change_draft.process.json", nil
	case "builtin_rc_spec_draft_v0":
		return "builtins/v0/spec_draft.workflow.json", "builtins/v0/spec_draft.process.json", nil
	case "builtin_rc_draft_promote_v0":
		return "builtins/v0/draft_promote.workflow.json", "builtins/v0/draft_promote.process.json", nil
	case "builtin_rc_approved_implementation_v0":
		return "builtins/v0/approved_implementation.workflow.json", "builtins/v0/approved_implementation.process.json", nil
	default:
		return "", "", fmt.Errorf("built-in workflow asset paths missing for workflow id %q", strings.TrimSpace(workflowID))
	}
}
