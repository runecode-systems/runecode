package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/launcherbackend"
	"github.com/runecode-systems/runecode/internal/runplan"
)

type runSummaryProjection struct {
	workflowKind           string
	workflowDefinitionHash string
	currentStageID         string
	approvalProfile        string
	projectionReason       string
}

type runSummaryPlanAuthoritativeProjection struct {
	workflowKind           string
	workflowDefinitionHash string
	currentStageID         string
	approvalProfile        string
	authoritative          bool
}

func (s *Service) resolveRunSummaryProjection(runID string, records []artifacts.ArtifactRecord, pending int) runSummaryProjection {
	planProjection := s.runSummaryPlanProjection(runID)
	workflowKind, workflowDefinitionHash, inferredByArtifacts := s.inferWorkflowIdentity(runID, records)
	currentStageID := currentStageIDFromArtifacts(records, pending)
	reason := "plan_authoritative"
	if planProjection.authoritative {
		workflowKind = coalesceTrimmed(planProjection.workflowKind, workflowKind)
		workflowDefinitionHash = coalesceTrimmed(planProjection.workflowDefinitionHash, workflowDefinitionHash)
		currentStageID = coalesceTrimmed(planProjection.currentStageID, currentStageID)
	} else if inferredByArtifacts {
		reason = "missing_active_run_plan_authority"
		workflowKind = ""
		workflowDefinitionHash = ""
		currentStageID = ""
	} else {
		reason = "projection_unknown"
		currentStageID = ""
	}

	return runSummaryProjection{
		workflowKind:           workflowKind,
		workflowDefinitionHash: workflowDefinitionHash,
		currentStageID:         currentStageID,
		approvalProfile:        defaultTrimmed(planProjection.approvalProfile, "unknown"),
		projectionReason:       reason,
	}
}

func newRunSummary(runID, projectContextIdentityDigest string, created, updated time.Time, state string, pending int, projection runSummaryProjection, verification AuditVerificationSurface, runtimeFacts launcherbackend.RuntimeFactsSnapshot) RunSummary {
	backendKind, isolationAssuranceLevel, provisioningPosture := normalizedRunSummaryPosture(runtimeFacts)
	createdAt := created.UTC().Format(time.RFC3339)
	updatedAt := updated.UTC().Format(time.RFC3339)

	return RunSummary{
		SchemaID:                "runecode.protocol.v0.RunSummary",
		SchemaVersion:           "0.2.0",
		RunID:                   runID,
		WorkspaceID:             workspaceIDForProjectContext(projectContextIdentityDigest),
		ProjectContextIdentity:  strings.TrimSpace(projectContextIdentityDigest),
		WorkflowKind:            projection.workflowKind,
		WorkflowDefinitionHash:  projection.workflowDefinitionHash,
		CreatedAt:               createdAt,
		StartedAt:               createdAt,
		UpdatedAt:               updatedAt,
		LifecycleState:          state,
		CurrentStageID:          projection.currentStageID,
		PendingApprovalCount:    pending,
		ApprovalProfile:         projection.approvalProfile,
		BackendKind:             backendKind,
		IsolationAssuranceLevel: isolationAssuranceLevel,
		ProvisioningPosture:     provisioningPosture,
		RuntimePostureDegraded:  runtimePostureDegraded(backendKind, isolationAssuranceLevel),
		AssuranceLevel:          isolationAssuranceLevel,
		AuditIntegrityStatus:    verification.Summary.IntegrityStatus,
		AuditAnchoringStatus:    verification.Summary.AnchoringStatus,
		AuditCurrentlyDegraded:  verification.Summary.CurrentlyDegraded,
	}
}

func (s *Service) runSummaryPlanProjection(runID string) runSummaryPlanAuthoritativeProjection {
	authority, ok, err := s.ActiveRunPlanAuthority(runID)
	if err != nil || !ok {
		return runSummaryPlanAuthoritativeProjection{}
	}
	projection := runSummaryPlanAuthoritativeProjection{
		workflowDefinitionHash: strings.TrimSpace(authority.WorkflowDefinitionHash),
		authoritative:          true,
	}
	if selectedEntry, err := selectSessionExecutionPlanEntry(authority.Entries); err == nil {
		projection.currentStageID = strings.TrimSpace(selectedEntry.StageID)
	}
	if plan, err := s.decodeTrustedRunPlan(authority.RunPlanDigest); err == nil {
		projection.approvalProfile = strings.TrimSpace(plan.ApprovalProfile)
		if len(plan.Entries) > 0 {
			projection.currentStageID = strings.TrimSpace(plan.Entries[len(plan.Entries)-1].StageID)
		}
	}
	projection.workflowKind = workflowIDForWorkflowDefinitionHash(projection.workflowDefinitionHash)
	return projection
}

func (s *Service) decodeTrustedRunPlan(digest string) (runplan.RunPlan, error) {
	payload, err := s.readArtifactPayload(strings.TrimSpace(digest))
	if err != nil {
		return runplan.RunPlan{}, err
	}
	var planned runplan.RunPlan
	if err := json.Unmarshal(payload, &planned); err != nil {
		return runplan.RunPlan{}, err
	}
	return planned, nil
}

func workspaceIDForRun(runID string) string {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return "workspace-local"
	}
	return "workspace-" + trimmed
}

func workspaceIDForProjectContext(projectContextIdentityDigest string) string {
	identity := strings.TrimSpace(projectContextIdentityDigest)
	if identity == "" {
		return "workspace-local"
	}
	return "workspace-" + strings.TrimPrefix(identity, "sha256:")
}

func stageIDForRun(runID string) string {
	if strings.TrimSpace(runID) == "" {
		return "artifact_flow"
	}
	return "artifact_flow"
}

func currentStageIDFromArtifacts(records []artifacts.ArtifactRecord, pending int) string {
	if len(records) == 0 && pending == 0 {
		return ""
	}
	return "artifact_flow"
}

func (s *Service) inferWorkflowIdentity(runID string, records []artifacts.ArtifactRecord) (string, string, bool) {
	if authorityWorkflowID, workflowHash := s.inferWorkflowIdentityFromActivePlanAuthority(runID); authorityWorkflowID != "" || workflowHash != "" {
		return authorityWorkflowID, workflowHash, false
	}
	for _, entry := range runplan.BuiltInWorkflowCatalogV0() {
		if strings.TrimSpace(entry.WorkflowDefinitionHash) == "" {
			continue
		}
		if runHasTrustedWorkflowDefinitionHash(runID, records, entry.WorkflowDefinitionHash) {
			return strings.TrimSpace(entry.WorkflowID), strings.TrimSpace(entry.WorkflowDefinitionHash), true
		}
	}
	workflowDefinitionHash := ""
	manifestDigests := uniqueSortedDigests(runProvenanceDigests(records))
	if len(manifestDigests) == 1 {
		workflowDefinitionHash = manifestDigests[0]
	}
	return "", workflowDefinitionHash, false
}

func (s *Service) inferWorkflowIdentityFromActivePlanAuthority(runID string) (string, string) {
	authority, ok, err := s.ActiveRunPlanAuthority(runID)
	if err != nil || !ok {
		return "", ""
	}
	workflowHash := strings.TrimSpace(authority.WorkflowDefinitionHash)
	if workflowHash == "" {
		return "", ""
	}
	if workflowID := workflowIDForWorkflowDefinitionHash(workflowHash); workflowID != "" {
		return workflowID, workflowHash
	}
	return "", workflowHash
}

func workflowIDForWorkflowDefinitionHash(workflowHash string) string {
	workflowHash = strings.TrimSpace(workflowHash)
	if workflowHash == "" {
		return ""
	}
	for _, entry := range runplan.BuiltInWorkflowCatalogV0() {
		if strings.TrimSpace(entry.WorkflowDefinitionHash) == workflowHash {
			return strings.TrimSpace(entry.WorkflowID)
		}
	}
	return ""
}

func runHasTrustedWorkflowDefinitionHash(runID string, records []artifacts.ArtifactRecord, trustedHash string) bool {
	trustedHash = strings.TrimSpace(trustedHash)
	if trustedHash == "" {
		return false
	}
	for _, record := range records {
		if strings.TrimSpace(record.RunID) != strings.TrimSpace(runID) {
			continue
		}
		if strings.TrimSpace(record.StepID) != "session_execution/workflow_definition" {
			continue
		}
		if strings.TrimSpace(record.Reference.ProvenanceReceiptHash) == trustedHash {
			return true
		}
	}
	return false
}

func runProvenanceDigests(records []artifacts.ArtifactRecord) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, record.Reference.ProvenanceReceiptHash)
	}
	return out
}

func runRoleInstanceID(role string) string {
	if strings.TrimSpace(role) == "" {
		return "role-unknown-1"
	}
	return fmt.Sprintf("%s-1", role)
}

func coalesceTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func defaultTrimmed(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}
