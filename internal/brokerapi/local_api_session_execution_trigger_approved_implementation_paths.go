package brokerapi

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func validateApprovedImplementationTargetPath(authority sessionExecutionPlanAuthority, targetRelativePath string, lifecycleMetadata bool) error {
	target, err := normalizeBrokerOwnedRelativeTargetPath(targetRelativePath)
	if err != nil {
		return err
	}
	if target == "" {
		return fmt.Errorf("approved implementation target path is required")
	}
	if lifecycleMetadata {
		return validateApprovedImplementationLifecycleMetadataTargetPath(target)
	}
	return validateApprovedImplementationWorkspaceTargetPath(authority, target)
}

func validateApprovedImplementationLifecycleMetadataTargetPath(target string) error {
	if approvedImplementationLifecycleMetadataPathAllowed(target) {
		return nil
	}
	return fmt.Errorf("approved implementation lifecycle metadata target path %q is outside narrow allowed scope", target)
}

func validateApprovedImplementationWorkspaceTargetPath(authority sessionExecutionPlanAuthority, target string) error {
	allowed, err := approvedImplementationCatalogPathAllowed(authority, target)
	if err != nil {
		return err
	}
	if allowed || approvedImplementationWorkspacePathAllowed(target) {
		return nil
	}
	return fmt.Errorf("approved implementation target path %q is outside narrow broker-owned workspace mutation scope", target)
}

func approvedImplementationCatalogPathAllowed(authority sessionExecutionPlanAuthority, target string) (bool, error) {
	entry, err := builtInCatalogEntryForWorkflowOperation(authority.workflowOperation)
	if err != nil {
		return false, err
	}
	for _, allowed := range entry.WritableRuneContextPath {
		prefix, err := normalizeBrokerOwnedRelativeTargetPath(allowed)
		if err != nil {
			return false, err
		}
		if pathWithinAllowedPrefix(target, prefix) {
			return true, nil
		}
	}
	return false, nil
}

func approvedImplementationWorkspacePathAllowed(target string) bool {
	target = strings.TrimSpace(target)
	if pathWithinAllowedPrefix(target, projectsubstrate.CanonicalChangesPath) {
		name := filepath.Base(target)
		return name == projectsubstrate.CanonicalChangeProposalName || name == projectsubstrate.CanonicalChangeTasksName || name == projectsubstrate.CanonicalChangeStatusName
	}
	if pathWithinAllowedPrefix(target, projectsubstrate.CanonicalSpecsPath) {
		return strings.HasSuffix(target, ".md")
	}
	return false
}

func approvedImplementationLifecycleMetadataPathAllowed(target string) bool {
	target = strings.TrimSpace(target)
	if target == projectsubstrate.CanonicalConfigPath {
		return true
	}
	if target == "runecontext/project/roadmap.md" {
		return true
	}
	if pathWithinAllowedPrefix(target, projectsubstrate.CanonicalChangesPath) {
		name := filepath.Base(target)
		return name == projectsubstrate.CanonicalChangeTasksName || name == projectsubstrate.CanonicalChangeStatusName || name == "verification.md"
	}
	return false
}

func approvedImplementationMutationStepID(targetRelativePath string, lifecycleMetadata bool) string {
	class := "workspace_mutation"
	if lifecycleMetadata {
		class = "lifecycle_metadata_mutation"
	}
	token := sessionExecutionIdentifierToken(strings.ReplaceAll(filepath.ToSlash(strings.TrimSpace(targetRelativePath)), "/", "_"))
	return "session_execution/approved_implementation_" + class + "_" + token
}

func approvedImplementationActionHash(targetRelativePath, contentDigest, sourceDigest string) string {
	return shaDigestIdentity(strings.TrimSpace(targetRelativePath) + "\n" + strings.TrimSpace(contentDigest) + "\n" + strings.TrimSpace(sourceDigest))
}

func approvedImplementationApprovalID(targetRelativePath, sourceDigest string) string {
	return shaDigestIdentity(strings.TrimSpace(targetRelativePath) + "\n" + strings.TrimSpace(sourceDigest))
}
