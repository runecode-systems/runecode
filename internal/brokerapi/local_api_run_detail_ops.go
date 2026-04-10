package brokerapi

import (
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
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
	policyRefs := s.PolicyDecisionRefsForRun(runID)
	pendingIDs := runPendingApprovalIDs(s.listApprovals(), runID)
	verification := s.runAuditVerificationOrFallback()
	runtimeFacts := s.RuntimeFacts(runID)
	return buildRunDetail(summary, verification, artifactsForRun, classCount, pendingIDs, policyRefs, runtimeFacts), true, nil
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

func buildRunDetail(summary RunSummary, verification AuditVerificationSurface, artifactsForRun []artifacts.ArtifactRecord, classCount map[string]int, pendingIDs []string, policyRefs []string, runtimeFacts launcherbackend.RuntimeFactsSnapshot) RunDetail {
	manifestHashes := activeManifestHashes(artifactsForRun)
	stageSummaries := []RunStageSummary{buildRunStageSummary(summary, artifactsForRun, pendingIDs)}
	roleSummaries := buildRunRoleSummaries(summary, artifactsForRun)
	authoritativeState := buildAuthoritativeRunState(summary, artifactsForRun, pendingIDs, manifestHashes, runtimeFacts)
	advisoryState := buildAdvisoryRunState()
	return RunDetail{
		SchemaID:                 "runecode.protocol.v0.RunDetail",
		SchemaVersion:            "0.2.0",
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
			RoleFamily:      "workspace",
			RoleKind:        "workspace-edit",
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
		family, kind := normalizeRoleForSummary(role)
		out = append(out, RunRoleSummary{
			SchemaID:        "runecode.protocol.v0.RunRoleSummary",
			SchemaVersion:   "0.1.0",
			RoleInstanceID:  runRoleInstanceID(role),
			RoleFamily:      family,
			RoleKind:        kind,
			LifecycleState:  summary.LifecycleState,
			ActiveItemCount: counts[role],
		})
	}
	return out
}

func normalizeRoleForSummary(createdByRole string) (string, string) {
	switch strings.TrimSpace(createdByRole) {
	case "workspace-read", "workspace-edit", "workspace-test":
		return "workspace", strings.TrimSpace(createdByRole)
	case "workspace", "brokerapi", "unknown", "":
		return "workspace", "workspace-edit"
	case "model_gateway", "model-gateway":
		return "gateway", "model-gateway"
	case "auth_gateway", "auth-gateway":
		return "gateway", "auth-gateway"
	case "git_gateway", "git-gateway":
		return "gateway", "git-gateway"
	case "web_research", "web-research":
		return "gateway", "web-research"
	case "dependency_fetch", "dependency-fetch":
		return "gateway", "dependency-fetch"
	default:
		return "workspace", "workspace-edit"
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

func activeManifestHashes(artifactsForRun []artifacts.ArtifactRecord) []string {
	values := runProvenanceDigests(artifactsForRun)
	unique := uniqueSortedDigests(values)
	if len(unique) > 0 {
		if len(unique) > 64 {
			return unique[:64]
		}
		return unique
	}
	return []string{}
}
