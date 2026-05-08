package brokerapi

import (
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) runSummaries(order string) ([]RunSummary, error) {
	runStatus := s.RunStatuses()
	verification := s.runAuditVerificationOrFallback()
	projectContextIdentity := strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
	byRun := buildRunRecordIndex(s.List(), runStatus)
	pendingByRun := pendingApprovalCountByRun(s.listApprovals())
	summaries := make([]RunSummary, 0, len(byRun))
	for runID, records := range byRun {
		runnerAdvisory, _ := s.RunnerAdvisory(runID)
		summaries = append(summaries, s.buildRunSummary(runID, projectContextIdentity, records, runStatus[runID], pendingByRun[runID], verification, s.RuntimeFacts(runID), runnerAdvisory))
	}
	sortRunSummaries(summaries, order)
	return summaries, nil
}

func (s *Service) runAuditVerificationOrFallback() AuditVerificationSurface {
	verification, err := s.LatestAuditVerificationSurface(20)
	if err == nil {
		return verification
	}
	if s.store != nil && s.store.HasAuditEvents() {
		return AuditVerificationSurface{Summary: trustpolicy.DerivedRunAuditVerificationSummary{
			CryptographicallyValid: false,
			HistoricallyAdmissible: false,
			CurrentlyDegraded:      true,
			IntegrityStatus:        trustpolicy.AuditVerificationStatusDegraded,
			AnchoringStatus:        trustpolicy.AuditVerificationStatusDegraded,
			StoragePostureStatus:   trustpolicy.AuditVerificationStatusOK,
			SegmentLifecycleStatus: trustpolicy.AuditVerificationStatusOK,
			DegradedReasons:        []string{"audit_verification_unavailable"},
		}}
	}
	return AuditVerificationSurface{Summary: trustpolicy.DerivedRunAuditVerificationSummary{
		CryptographicallyValid: false,
		HistoricallyAdmissible: false,
		CurrentlyDegraded:      true,
		IntegrityStatus:        trustpolicy.AuditVerificationStatusFailed,
		AnchoringStatus:        trustpolicy.AuditVerificationStatusFailed,
		StoragePostureStatus:   trustpolicy.AuditVerificationStatusFailed,
		SegmentLifecycleStatus: trustpolicy.AuditVerificationStatusFailed,
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

func (s *Service) buildRunSummary(runID string, projectContextIdentityDigest string, records []artifacts.ArtifactRecord, status string, pending int, verification AuditVerificationSurface, runtimeFacts launcherbackend.RuntimeFactsSnapshot, runnerAdvisory artifacts.RunnerAdvisoryState) RunSummary {
	created, updated := runRecordTiming(records)
	state := runLifecycleFromStore(status, pending, len(records) > 0, runnerAdvisory, runtimeFacts)
	projection := s.resolveRunSummaryProjection(runID, records, pending)
	summary := newRunSummary(
		runID,
		projectContextIdentityDigest,
		created,
		updated,
		state,
		pending,
		projection,
		verification,
		runtimeFacts,
	)
	finalizeRunSummaryTerminalState(&summary, state, updated)
	return summary
}

func normalizedRunSummaryPosture(runtimeFacts launcherbackend.RuntimeFactsSnapshot) (string, string, string) {
	receipt := runtimeFacts.LaunchReceipt.Normalized()
	isolationAssuranceLevel := strings.TrimSpace(receipt.IsolationAssuranceLevel)
	if isolationAssuranceLevel == "" || isolationAssuranceLevel == launcherbackend.IsolationAssuranceUnknown {
		isolationAssuranceLevel = launcherbackend.IsolationAssuranceUnknown
	}
	provisioningPosture := strings.TrimSpace(receipt.ProvisioningPosture)
	if provisioningPosture == "" {
		provisioningPosture = launcherbackend.ProvisioningPostureUnknown
	}
	return receipt.BackendKind, isolationAssuranceLevel, provisioningPosture
}

func finalizeRunSummaryTerminalState(summary *RunSummary, state string, updated time.Time) {
	if state == "blocked" {
		summary.BlockingReasonCode = "pending_approval"
	}
	if state == "completed" || state == "failed" || state == "cancelled" {
		summary.FinishedAt = updated.UTC().Format(time.RFC3339)
	}
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
