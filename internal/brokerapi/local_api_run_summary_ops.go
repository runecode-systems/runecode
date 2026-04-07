package brokerapi

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) runSummaries(order string) ([]RunSummary, error) {
	runStatus := s.RunStatuses()
	verification := s.runAuditVerificationOrFallback()
	byRun := buildRunRecordIndex(s.List(), runStatus)
	pendingByRun := pendingApprovalCountByRun(s.listApprovals())
	summaries := make([]RunSummary, 0, len(byRun))
	for runID, records := range byRun {
		summaries = append(summaries, buildRunSummary(runID, records, runStatus[runID], pendingByRun[runID], verification))
	}
	sortRunSummaries(summaries, order)
	return summaries, nil
}

func (s *Service) runAuditVerificationOrFallback() AuditVerificationSurface {
	verification, err := s.LatestAuditVerificationSurface(20)
	if err == nil {
		return verification
	}
	return AuditVerificationSurface{Summary: trustpolicy.DerivedRunAuditVerificationSummary{
		CryptographicallyValid: false,
		HistoricallyAdmissible: false,
		CurrentlyDegraded:      true,
		IntegrityStatus:        "failed",
		AnchoringStatus:        "failed",
		StoragePostureStatus:   "failed",
		SegmentLifecycleStatus: "failed",
		HardFailures:           []string{"audit_surface_unavailable"},
	}}
}

func buildRunRecordIndex(all []artifacts.ArtifactRecord, runStatus map[string]string) map[string][]artifacts.ArtifactRecord {
	byRun := map[string][]artifacts.ArtifactRecord{}
	for _, rec := range all {
		if rec.RunID == "" {
			continue
		}
		byRun[rec.RunID] = append(byRun[rec.RunID], rec)
	}
	for runID := range runStatus {
		if _, ok := byRun[runID]; !ok {
			byRun[runID] = nil
		}
	}
	return byRun
}

func buildRunSummary(runID string, records []artifacts.ArtifactRecord, status string, pending int, verification AuditVerificationSurface) RunSummary {
	created, updated := runRecordTiming(records)
	state := runLifecycleFromStore(status, pending, len(records) > 0)
	workflowKind := inferWorkflowKind(records)
	backendKind := inferBackendKind(records)
	assuranceLevel := inferAssuranceLevel(verification)
	summary := RunSummary{
		SchemaID:               "runecode.protocol.v0.RunSummary",
		SchemaVersion:          "0.1.0",
		RunID:                  runID,
		WorkspaceID:            workspaceIDForRun(runID),
		WorkflowKind:           workflowKind,
		WorkflowDefinitionHash: shaDigestIdentity("workflow:" + runID + ":" + workflowKind),
		CreatedAt:              created.UTC().Format(time.RFC3339),
		StartedAt:              created.UTC().Format(time.RFC3339),
		UpdatedAt:              updated.UTC().Format(time.RFC3339),
		LifecycleState:         state,
		CurrentStageID:         stageIDForRun(runID),
		PendingApprovalCount:   pending,
		ApprovalProfile:        "moderate",
		BackendKind:            backendKind,
		AssuranceLevel:         assuranceLevel,
		AuditIntegrityStatus:   verification.Summary.IntegrityStatus,
		AuditAnchoringStatus:   verification.Summary.AnchoringStatus,
		AuditCurrentlyDegraded: verification.Summary.CurrentlyDegraded,
	}
	if state == "blocked" {
		summary.BlockingReasonCode = "pending_approval"
	}
	if state == "completed" || state == "failed" || state == "cancelled" {
		summary.FinishedAt = updated.UTC().Format(time.RFC3339)
	}
	return summary
}

func runRecordTiming(records []artifacts.ArtifactRecord) (time.Time, time.Time) {
	emptyRunTime := time.Unix(0, 0).UTC()
	created := emptyRunTime
	updated := emptyRunTime
	if len(records) == 0 {
		return created, updated
	}
	created = records[0].CreatedAt
	updated = records[0].CreatedAt
	for _, rec := range records {
		if rec.CreatedAt.Before(created) {
			created = rec.CreatedAt
		}
		if rec.CreatedAt.After(updated) {
			updated = rec.CreatedAt
		}
	}
	return created, updated
}

func sortRunSummaries(summaries []RunSummary, order string) {
	sort.Slice(summaries, func(i, j int) bool {
		if order == "updated_at_asc" {
			return summaries[i].UpdatedAt < summaries[j].UpdatedAt
		}
		if summaries[i].UpdatedAt == summaries[j].UpdatedAt {
			return summaries[i].RunID < summaries[j].RunID
		}
		return summaries[i].UpdatedAt > summaries[j].UpdatedAt
	})
}

func runLifecycleFromStore(status string, pendingApprovals int, hasArtifacts bool) string {
	if pendingApprovals > 0 {
		return "blocked"
	}
	switch status {
	case "pending", "starting", "active", "blocked", "recovering", "completed", "failed", "cancelled":
		if status == "active" && !hasArtifacts {
			return "starting"
		}
		return status
	case "retained", "closed":
		return "completed"
	default:
		if !hasArtifacts {
			return "pending"
		}
		return "active"
	}
}

func pendingApprovalCountByRun(approvals []ApprovalSummary) map[string]int {
	counts := map[string]int{}
	for _, approval := range approvals {
		if approval.Status != "pending" || approval.BoundScope.RunID == "" {
			continue
		}
		counts[approval.BoundScope.RunID]++
	}
	return counts
}

func workspaceIDForRun(runID string) string {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return "workspace-local"
	}
	return "workspace-" + trimmed
}

func stageIDForRun(runID string) string {
	if strings.TrimSpace(runID) == "" {
		return "artifact_flow"
	}
	return "artifact_flow"
}

func inferWorkflowKind(records []artifacts.ArtifactRecord) string {
	hasDiff := false
	hasBuildLogs := false
	hasUnapproved := false
	for _, record := range records {
		switch record.Reference.DataClass {
		case artifacts.DataClassDiffs:
			hasDiff = true
		case artifacts.DataClassBuildLogs:
			hasBuildLogs = true
		case artifacts.DataClassUnapprovedFileExcerpts, artifacts.DataClassApprovedFileExcerpts:
			hasUnapproved = true
		}
	}
	switch {
	case hasUnapproved:
		return "excerpt_promotion"
	case hasDiff && hasBuildLogs:
		return "edit_build_gate"
	case hasDiff:
		return "edit_diff"
	default:
		return "artifact_flow"
	}
}

func inferBackendKind(records []artifacts.ArtifactRecord) string {
	for _, record := range records {
		if record.CreatedByRole == "auditd" {
			return "local_broker"
		}
	}
	return "local_broker"
}

func inferAssuranceLevel(verification AuditVerificationSurface) string {
	if verification.Summary.CurrentlyDegraded {
		return "degraded"
	}
	if verification.Summary.CryptographicallyValid {
		return "verified"
	}
	return "session_authenticated"
}

func runRoleInstanceID(role string) string {
	if strings.TrimSpace(role) == "" {
		return "role-unknown-1"
	}
	return fmt.Sprintf("%s-1", role)
}
