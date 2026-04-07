package brokerapi

import (
	"fmt"
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
	verification := s.runAuditVerificationOrFallback()
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
	manifestHashes := activeManifestHashes(summary, artifactsForRun)
	policyRefs := latestPolicyDecisionRefs(artifactsForRun)
	stageSummaries := []RunStageSummary{buildRunStageSummary(summary, artifactsForRun, pendingIDs)}
	roleSummaries := buildRunRoleSummaries(summary, artifactsForRun)
	authoritativeState := map[string]any{
		"source":                 "broker_store",
		"status":                 summary.LifecycleState,
		"artifact_count":         len(artifactsForRun),
		"pending_approval_count": len(pendingIDs),
	}
	advisoryState := map[string]any{
		"source":           "runner_advisory",
		"available":        false,
		"advisory_version": "0",
	}
	return RunDetail{
		SchemaID:                 "runecode.protocol.v0.RunDetail",
		SchemaVersion:            "0.1.0",
		Summary:                  summary,
		StageSummaries:           stageSummaries,
		RoleSummaries:            roleSummaries,
		Coordination:             buildRunCoordinationSummary(summary),
		AuditSummary:             verification.Summary,
		ArtifactCountsByClass:    classCount,
		PendingApprovalIDs:       pendingIDs,
		ActiveManifestHashes:     manifestHashes,
		LatestPolicyDecisionRefs: policyRefs,
		AuthoritativeState:       authoritativeState,
		AdvisoryState:            advisoryState,
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

func buildRunRoleSummaries(summary RunSummary, artifactsForRun []artifacts.ArtifactRecord) []RunRoleSummary {
	counts := map[string]int{}
	for _, record := range artifactsForRun {
		role := strings.TrimSpace(record.CreatedByRole)
		if role == "" {
			role = "unknown"
		}
		counts[role]++
	}
	if len(counts) == 0 {
		return []RunRoleSummary{{
			SchemaID:        "runecode.protocol.v0.RunRoleSummary",
			SchemaVersion:   "0.1.0",
			RoleInstanceID:  runRoleInstanceID("workspace"),
			RoleKind:        "workspace",
			LifecycleState:  summary.LifecycleState,
			ActiveItemCount: 0,
		}}
	}
	roles := make([]string, 0, len(counts))
	for role := range counts {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	out := make([]RunRoleSummary, 0, len(roles))
	for _, role := range roles {
		out = append(out, RunRoleSummary{
			SchemaID:        "runecode.protocol.v0.RunRoleSummary",
			SchemaVersion:   "0.1.0",
			RoleInstanceID:  runRoleInstanceID(role),
			RoleKind:        role,
			LifecycleState:  summary.LifecycleState,
			ActiveItemCount: counts[role],
		})
	}
	return out
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

func activeManifestHashes(summary RunSummary, artifactsForRun []artifacts.ArtifactRecord) []string {
	values := make([]string, 0, len(artifactsForRun)+1)
	for _, record := range artifactsForRun {
		values = append(values, record.Reference.ProvenanceReceiptHash)
	}
	values = append(values, summary.WorkflowDefinitionHash)
	unique := uniqueSortedDigests(values)
	if len(unique) > 0 {
		return unique
	}
	return []string{shaDigestIdentity("manifest:" + summary.RunID)}
}

func latestPolicyDecisionRefs(artifactsForRun []artifacts.ArtifactRecord) []string {
	refs := make([]string, 0, len(artifactsForRun))
	for _, record := range artifactsForRun {
		if record.ApprovalDecisionHash == "" {
			continue
		}
		refs = append(refs, fmt.Sprintf("approval_decision:%s", record.ApprovalDecisionHash))
	}
	if len(refs) == 0 {
		return []string{}
	}
	sort.Strings(refs)
	if len(refs) > 256 {
		return refs[len(refs)-256:]
	}
	return refs
}
