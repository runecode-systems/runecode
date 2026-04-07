package brokerapi

import (
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) runDetail(runID string) (RunDetail, bool, error) {
	summaries, err := s.runSummaries("updated_at_desc")
	if err != nil {
		return RunDetail{}, false, err
	}
	summary, found := findRunSummary(summaries, runID)
	if !found {
		return RunDetail{}, false, nil
	}
	artifactsForRun, classCount := runArtifactsAndClassCount(s.List(), runID)
	pendingIDs := runPendingApprovalIDs(s.listApprovals(), runID)
	verification, _ := s.LatestAuditVerificationSurface(20)
	return buildRunDetail(summary, verification, artifactsForRun, classCount, pendingIDs), true, nil
}

func findRunSummary(summaries []RunSummary, runID string) (RunSummary, bool) {
	for _, item := range summaries {
		if item.RunID == runID {
			return item, true
		}
	}
	return RunSummary{}, false
}

func runArtifactsAndClassCount(all []artifacts.ArtifactRecord, runID string) ([]artifacts.ArtifactRecord, map[string]int) {
	artifactsForRun := make([]artifacts.ArtifactRecord, 0)
	classCount := map[string]int{}
	for _, rec := range all {
		if rec.RunID != runID {
			continue
		}
		artifactsForRun = append(artifactsForRun, rec)
		classCount[string(rec.Reference.DataClass)]++
	}
	return artifactsForRun, classCount
}

func runPendingApprovalIDs(approvals []ApprovalSummary, runID string) []string {
	pendingIDs := make([]string, 0)
	for _, approval := range approvals {
		if approval.Status == "pending" && approval.BoundScope.RunID == runID {
			pendingIDs = append(pendingIDs, approval.ApprovalID)
		}
	}
	sort.Strings(pendingIDs)
	return pendingIDs
}

func buildRunDetail(summary RunSummary, verification AuditVerificationSurface, artifactsForRun []artifacts.ArtifactRecord, classCount map[string]int, pendingIDs []string) RunDetail {
	return RunDetail{
		SchemaID:                 "runecode.protocol.v0.RunDetail",
		SchemaVersion:            "0.1.0",
		Summary:                  summary,
		StageSummaries:           []RunStageSummary{buildRunStageSummary(summary, artifactsForRun, pendingIDs)},
		RoleSummaries:            []RunRoleSummary{buildRunRoleSummary(summary, artifactsForRun)},
		Coordination:             buildRunCoordinationSummary(summary),
		AuditSummary:             verification.Summary,
		ArtifactCountsByClass:    classCount,
		PendingApprovalIDs:       pendingIDs,
		ActiveManifestHashes:     []string{"sha256:" + strings.Repeat("0", 64)},
		LatestPolicyDecisionRefs: []string{},
		AuthoritativeState:       map[string]any{"source": "broker_store", "status": summary.LifecycleState},
		AdvisoryState:            map[string]any{"source": "runner_advisory", "available": false},
	}
}

func buildRunStageSummary(summary RunSummary, artifactsForRun []artifacts.ArtifactRecord, pendingIDs []string) RunStageSummary {
	return RunStageSummary{
		SchemaID:             "runecode.protocol.v0.RunStageSummary",
		SchemaVersion:        "0.1.0",
		StageID:              "artifact_flow",
		LifecycleState:       summary.LifecycleState,
		StartedAt:            summary.StartedAt,
		FinishedAt:           summary.FinishedAt,
		PendingApprovalCount: len(pendingIDs),
		ArtifactCount:        len(artifactsForRun),
	}
}

func buildRunRoleSummary(summary RunSummary, artifactsForRun []artifacts.ArtifactRecord) RunRoleSummary {
	return RunRoleSummary{
		SchemaID:        "runecode.protocol.v0.RunRoleSummary",
		SchemaVersion:   "0.1.0",
		RoleInstanceID:  "workspace-1",
		RoleKind:        "workspace",
		LifecycleState:  summary.LifecycleState,
		ActiveItemCount: len(artifactsForRun),
	}
}

func buildRunCoordinationSummary(summary RunSummary) RunCoordinationSummary {
	return RunCoordinationSummary{
		SchemaID:         "runecode.protocol.v0.RunCoordinationSummary",
		SchemaVersion:    "0.1.0",
		Blocked:          summary.LifecycleState == "blocked",
		WaitReasonCode:   summary.BlockingReasonCode,
		LockCount:        0,
		ConflictCount:    0,
		CoordinationMode: "single_broker_queue",
	}
}
