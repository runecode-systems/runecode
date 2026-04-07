package brokerapi

import (
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
	summaries := make([]RunSummary, 0, len(byRun))
	for runID, records := range byRun {
		summaries = append(summaries, buildRunSummary(runID, records, runStatus[runID], verification))
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

func buildRunSummary(runID string, records []artifacts.ArtifactRecord, status string, verification AuditVerificationSurface) RunSummary {
	created, updated, pending := runRecordTimingAndPending(records)
	state := runLifecycleFromStore(status, pending)
	summary := RunSummary{
		SchemaID:               "runecode.protocol.v0.RunSummary",
		SchemaVersion:          "0.1.0",
		RunID:                  runID,
		WorkspaceID:            "local-workspace",
		WorkflowKind:           "broker_local_mvp",
		WorkflowDefinitionHash: "sha256:" + strings.Repeat("0", 64),
		CreatedAt:              created.UTC().Format(time.RFC3339),
		StartedAt:              created.UTC().Format(time.RFC3339),
		UpdatedAt:              updated.UTC().Format(time.RFC3339),
		LifecycleState:         state,
		CurrentStageID:         "artifact_flow",
		PendingApprovalCount:   pending,
		ApprovalProfile:        "moderate",
		BackendKind:            "local",
		AssuranceLevel:         "session_authenticated",
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

func runRecordTimingAndPending(records []artifacts.ArtifactRecord) (time.Time, time.Time, int) {
	emptyRunTime := time.Unix(0, 0).UTC()
	created := emptyRunTime
	updated := emptyRunTime
	pending := 0
	if len(records) == 0 {
		return created, updated, pending
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
		if rec.Reference.DataClass == artifacts.DataClassUnapprovedFileExcerpts {
			pending++
		}
	}
	return created, updated, pending
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

func runLifecycleFromStore(status string, pendingApprovals int) string {
	if pendingApprovals > 0 {
		return "blocked"
	}
	switch status {
	case "active":
		return "active"
	case "retained", "closed":
		return "completed"
	default:
		return "active"
	}
}
