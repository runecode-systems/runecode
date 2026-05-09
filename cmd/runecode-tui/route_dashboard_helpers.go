package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type dashboardExecutiveSummary struct {
	State      routeLoadState
	Title      string
	Message    string
	Reason     string
	NextAction string
}

type dashboardSnapshot struct {
	PrimaryRun          brokerapi.RunSummary
	PendingApprovals    int
	BlockedRuns         int
	DegradedSignals     int
	ActiveRuns          int
	RunRuntimeDegraded  int
	SetupBlocked        bool
	SetupNeedsAttention bool
	AuditDegraded       bool
	NoWork              bool
	Executive           dashboardExecutiveSummary
}

func buildDashboardSnapshot(data dashboardData) dashboardSnapshot {
	snapshot := dashboardSnapshot{
		PrimaryRun:          primaryDashboardRun(data.runs),
		PendingApprovals:    pendingApprovalCount(data.runs, data.approvals),
		BlockedRuns:         dashboardBlockedRunCount(data.runs),
		ActiveRuns:          dashboardActiveRunCount(data.runs),
		RunRuntimeDegraded:  dashboardRunRuntimeDegradedCount(data.runs),
		SetupBlocked:        dashboardProjectSubstrateBlocked(data.project),
		SetupNeedsAttention: dashboardProjectSubstrateNeedsAttention(data.project),
		AuditDegraded:       dashboardAuditDegraded(data.audit),
	}
	snapshot.DegradedSignals = snapshot.RunRuntimeDegraded
	if snapshot.AuditDegraded {
		snapshot.DegradedSignals++
	}
	if !data.readiness.Ready {
		snapshot.DegradedSignals++
	}
	snapshot.NoWork = len(data.runs) == 0 && len(data.approvals) == 0
	snapshot.Executive = buildDashboardExecutiveSummary(data, snapshot)
	return snapshot
}

func degradedDashboardAuditFallback() brokerapi.AuditVerificationGetResponse {
	return brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{
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

func renderDashboardAuditFallbackNotice(auditErr string) string {
	if strings.TrimSpace(auditErr) == "" {
		return ""
	}
	return dangerBadge("Evidence verification unavailable") + " Audit verification is unavailable, so RuneCode is showing degraded evidence posture. Reason: " + auditErr
}

func renderDashboardNowBar(run brokerapi.RunSummary, approvalCount int, focusActive bool, width int) string {
	parts := []string{}
	if focusActive {
		parts = append(parts, successBadge("Overview focused"))
	} else {
		parts = append(parts, neutralBadge("Overview visible"))
	}
	if strings.TrimSpace(run.RunID) != "" {
		parts = append(parts, fmt.Sprintf("Current run %s", sanitizeUIText(run.RunID)))
		parts = append(parts, "state "+postureBadge(run.LifecycleState))
		parts = append(parts, "runtime "+postureBadge(sanitizeUIText(run.BackendKind)))
	}
	if approvalCount > 0 {
		parts = append(parts, approvalRequiredBadge(fmt.Sprintf("%d approval waiting", approvalCount)))
	} else {
		parts = append(parts, successBadge("No approvals waiting"))
	}
	return wrapPartsByWidth(parts, " ", width)
}

func buildDashboardExecutiveSummary(data dashboardData, snapshot dashboardSnapshot) dashboardExecutiveSummary {
	switch {
	case snapshot.SetupBlocked || snapshot.BlockedRuns > 0:
		return dashboardBlockedSummary(data, snapshot)
	case snapshot.DegradedSignals > 0:
		return dashboardDegradedSummary(data, snapshot)
	case snapshot.PendingApprovals > 0 || snapshot.SetupNeedsAttention:
		return dashboardAttentionSummary(data, snapshot)
	case snapshot.ActiveRuns > 0:
		return dashboardActiveWorkSummary(snapshot.ActiveRuns)
	case snapshot.NoWork:
		return dashboardNoWorkSummary()
	default:
		return defaultDashboardExecutiveSummary()
	}
}

func defaultDashboardExecutiveSummary() dashboardExecutiveSummary {
	return dashboardExecutiveSummary{State: routeLoadStateReady, Title: "Healthy", Message: "RuneCode is ready and no follow-up is waiting.", Reason: "Broker surfaces show normal operation.", NextAction: "Stay here for overview or open Action Center."}
}

func dashboardNoWorkSummary() dashboardExecutiveSummary {
	return dashboardExecutiveSummary{State: routeLoadStateEmpty, Title: "No work yet", Message: "RuneCode is ready, but no work has started yet.", Reason: "No recent runs or approvals are visible.", NextAction: "Start work from Chat, then use Action Center."}
}

func dashboardBlockedSummary(data dashboardData, snapshot dashboardSnapshot) dashboardExecutiveSummary {
	reasons := []string{}
	if snapshot.SetupBlocked {
		reasons = append(reasons, dashboardProjectSubstrateReason(data.project))
	}
	if snapshot.BlockedRuns > 0 {
		reasons = append(reasons, fmt.Sprintf("%d workflow(s) are blocked or waiting", snapshot.BlockedRuns))
	}
	return dashboardExecutiveSummary{State: routeLoadStateBlocked, Title: "Blocked", Message: "Workflow progress is blocked.", Reason: strings.Join(reasons, "; "), NextAction: "Open Action Center and clear the blocker."}
}

func dashboardDegradedSummary(data dashboardData, snapshot dashboardSnapshot) dashboardExecutiveSummary {
	reasons := []string{}
	if !data.readiness.Ready {
		reasons = append(reasons, "broker readiness is not fully settled")
	}
	if snapshot.AuditDegraded {
		reasons = append(reasons, "audit or runtime evidence needs review")
	}
	if snapshot.RunRuntimeDegraded > 0 {
		reasons = append(reasons, fmt.Sprintf("%d run(s) report degraded runtime posture", snapshot.RunRuntimeDegraded))
	}
	return dashboardExecutiveSummary{State: routeLoadStateDegraded, Title: "Degraded", Message: "Evidence or runtime posture needs review.", Reason: strings.Join(reasons, "; "), NextAction: "Open Action Center, then confirm Audit or Runs."}
}

func dashboardAttentionSummary(data dashboardData, snapshot dashboardSnapshot) dashboardExecutiveSummary {
	reasons := []string{}
	if snapshot.PendingApprovals > 0 {
		reasons = append(reasons, fmt.Sprintf("%d approval decision(s) are waiting", snapshot.PendingApprovals))
	}
	if snapshot.SetupNeedsAttention {
		reasons = append(reasons, dashboardProjectSubstrateReason(data.project))
	}
	return dashboardExecutiveSummary{State: routeLoadStateApprovalRequired, Title: "Needs attention", Message: "Operator follow-up is waiting.", Reason: strings.Join(reasons, "; "), NextAction: "Open Action Center for the exact follow-up."}
}

func dashboardActiveWorkSummary(activeRuns int) dashboardExecutiveSummary {
	return dashboardExecutiveSummary{State: routeLoadStateReady, Title: "Active work", Message: "Workflow work is active.", Reason: fmt.Sprintf("%d active workflow(s) are visible.", activeRuns), NextAction: "Use Action Center or open Runs for detail."}
}

func renderDashboardWorkflowPosture(snapshot dashboardSnapshot) string {
	parts := []string{}
	if snapshot.SetupBlocked {
		parts = append(parts, "normal product work is blocked until project setup is remediated")
	} else if snapshot.BlockedRuns > 0 {
		parts = append(parts, fmt.Sprintf("%d workflow(s) are blocked or waiting on follow-up", snapshot.BlockedRuns))
	} else if snapshot.ActiveRuns > 0 {
		parts = append(parts, fmt.Sprintf("%d workflow(s) are active", snapshot.ActiveRuns))
	} else {
		parts = append(parts, "no active workflow work is currently running")
	}
	if snapshot.PendingApprovals > 0 {
		parts = append(parts, fmt.Sprintf("%d approval decision(s) are waiting", snapshot.PendingApprovals))
	}
	if snapshot.SetupNeedsAttention && !snapshot.SetupBlocked {
		parts = append(parts, "project setup guidance is available")
	}
	if snapshot.AuditDegraded {
		parts = append(parts, "evidence posture needs review")
	}
	return "Workflow posture: " + strings.Join(parts, "; ") + "."
}

func renderDashboardHighValueCounts(snapshot dashboardSnapshot, width int) string {
	parts := []string{
		fmt.Sprintf("active work=%d", snapshot.ActiveRuns),
		fmt.Sprintf("approval follow-up=%d", snapshot.PendingApprovals),
		fmt.Sprintf("blocked or waiting=%d", snapshot.BlockedRuns),
		fmt.Sprintf("degraded cues=%d", snapshot.DegradedSignals),
	}
	return "At a glance: " + wrapPartsByWidth(parts, " | ", width)
}

func dashboardCardTone(snapshot dashboardSnapshot) visualTone {
	switch snapshot.Executive.State {
	case routeLoadStateBlocked, routeLoadStateError:
		return visualToneDanger
	case routeLoadStateApprovalRequired, routeLoadStateDegraded, routeLoadStateWaiting:
		return visualToneAttention
	case routeLoadStateReady:
		return visualToneSuccess
	default:
		return visualToneInfo
	}
}

func renderDashboardCurrentWork(data dashboardData, snapshot dashboardSnapshot) string {
	runID := strings.TrimSpace(snapshot.PrimaryRun.RunID)
	if runID == "" {
		if snapshot.NoWork {
			return "Current work: no active run is visible yet."
		}
		return "Current work: broker returned follow-up items, but no primary run is selected."
	}
	state := humanizeExecutionToken(snapshot.PrimaryRun.LifecycleState)
	parts := []string{fmt.Sprintf("Current work: %s is %s", sanitizeUIText(runID), state)}
	if snapshot.BlockedRuns > 0 || strings.TrimSpace(snapshot.PrimaryRun.BlockingReasonCode) != "" {
		parts = append(parts, "blocked follow-up is waiting")
	} else if snapshot.ActiveRuns > 0 {
		parts = append(parts, "work is active")
	}
	if snapshot.DegradedSignals > 0 {
		parts = append(parts, "evidence or runtime posture needs review")
	}
	if data.readiness.RecoveryComplete {
		parts = append(parts, "recovery is complete")
	}
	return strings.Join(parts, "; ") + "."
}

func renderDashboardApprovalCue(data dashboardData, snapshot dashboardSnapshot) string {
	if snapshot.PendingApprovals == 0 {
		return "Approvals: none waiting."
	}
	first := firstPendingApproval(data.approvals)
	if strings.TrimSpace(first.ApprovalID) == "" {
		return fmt.Sprintf("Approvals: %d decision(s) are waiting.", snapshot.PendingApprovals)
	}
	return fmt.Sprintf("Approvals: %d decision(s) waiting; next is %s for run %s.", snapshot.PendingApprovals, sanitizeUIText(first.ApprovalID), valueOrNA(sanitizeUIText(first.BoundScope.RunID)))
}

func dashboardMetricLines(snapshot dashboardSnapshot) []string {
	return []string{
		fmt.Sprintf("Active work: %d", snapshot.ActiveRuns),
		fmt.Sprintf("Approvals waiting: %d", snapshot.PendingApprovals),
		fmt.Sprintf("Blocked or waiting: %d", snapshot.BlockedRuns),
		fmt.Sprintf("Needs review: %d", snapshot.DegradedSignals),
	}
}

func renderDashboardDetailCue(snapshot dashboardSnapshot) string {
	if snapshot.BlockedRuns > 0 || snapshot.PendingApprovals > 0 || snapshot.DegradedSignals > 0 || snapshot.SetupNeedsAttention {
		return "Detail: Action Center explains blockers and degraded cues; Audit, Runs, and Status keep the raw proof."
	}
	return "Detail: use Runs, Audit, or Status when you need broker proof instead of the overview."
}

func firstPendingApproval(approvals []brokerapi.ApprovalSummary) brokerapi.ApprovalSummary {
	for _, approval := range approvals {
		if strings.EqualFold(strings.TrimSpace(approval.Status), "pending") || strings.TrimSpace(approval.Status) == "" {
			return approval
		}
	}
	if len(approvals) == 0 {
		return brokerapi.ApprovalSummary{}
	}
	return approvals[0]
}

func renderDashboardNextActions(data dashboardData, snapshot dashboardSnapshot) string {
	if snapshot.SetupBlocked {
		return "Open Action Center and clear the setup blocker, then use Status for broker-owned remediation steps."
	}
	if snapshot.PendingApprovals > 0 {
		return "Open Action Center and review the pending approval before workflow progress resumes."
	}
	if snapshot.DegradedSignals > 0 {
		return "Open Action Center for degraded evidence or runtime follow-up, then inspect Audit or Runs."
	}
	if strings.TrimSpace(snapshot.PrimaryRun.RunID) != "" {
		return fmt.Sprintf("Open Runs for %s or Action Center if new follow-up appears.", sanitizeUIText(snapshot.PrimaryRun.RunID))
	}
	return "Start a workflow from Chat, then return here for the executive overview."
}

func renderDashboardActionCenterCue(snapshot dashboardSnapshot) string {
	return fmt.Sprintf(
		"Action Center has the exact follow-up list: approvals=%d, blocked or waiting=%d, degraded cues=%d.",
		snapshot.PendingApprovals,
		snapshot.BlockedRuns,
		snapshot.DegradedSignals,
	)
}

func renderDashboardLiveActivitySummary(live dashboardLiveActivity) string {
	degraded := 0
	for _, family := range []watchFamilySummary{live.runWatch, live.approvalWatch, live.sessionWatch} {
		if family.errorCount > 0 || !strings.EqualFold(strings.TrimSpace(family.lastStatus), "ok") {
			degraded++
		}
	}
	if degraded > 0 {
		return fmt.Sprintf("Live updates need attention in %d stream(s); Action Center and Status have the details.", degraded)
	}
	if len(live.feed) > 0 {
		return fmt.Sprintf("Live updates are healthy; %d recent broker event(s) are visible below.", len(live.feed))
	}
	return "Live updates are quiet; RuneCode will show broker-owned activity here when work changes."
}

func dashboardActiveRunCount(runs []brokerapi.RunSummary) int {
	total := 0
	for _, run := range runs {
		state := strings.ToLower(strings.TrimSpace(run.LifecycleState))
		if state == "active" || state == "running" || strings.Contains(state, "in_progress") {
			total++
		}
	}
	return total
}

func dashboardBlockedRunCount(runs []brokerapi.RunSummary) int {
	total := 0
	for _, run := range runs {
		state := strings.ToLower(strings.TrimSpace(run.LifecycleState))
		if strings.Contains(state, "block") || strings.Contains(state, "wait") || strings.TrimSpace(run.BlockingReasonCode) != "" {
			total++
		}
	}
	return total
}

func dashboardRunRuntimeDegradedCount(runs []brokerapi.RunSummary) int {
	total := 0
	for _, run := range runs {
		if run.RuntimePostureDegraded || run.AuditCurrentlyDegraded {
			total++
		}
	}
	return total
}

func dashboardAuditDegraded(audit brokerapi.AuditVerificationGetResponse) bool {
	summary := audit.Summary
	return summary.CurrentlyDegraded || strings.EqualFold(strings.TrimSpace(summary.AnchoringStatus), "degraded") || strings.EqualFold(strings.TrimSpace(summary.AnchoringStatus), "failed") || strings.EqualFold(strings.TrimSpace(summary.IntegrityStatus), "failed")
}

func dashboardProjectSubstrateBlocked(posture brokerapi.ProjectSubstratePostureGetResponse) bool {
	if strings.TrimSpace(posture.PostureSummary.SchemaID) == "" {
		return false
	}
	if !posture.PostureSummary.NormalOperationAllowed {
		return true
	}
	return false
}

func dashboardProjectSubstrateNeedsAttention(posture brokerapi.ProjectSubstratePostureGetResponse) bool {
	if strings.TrimSpace(posture.PostureSummary.SchemaID) == "" {
		return false
	}
	if dashboardProjectSubstrateBlocked(posture) {
		return true
	}
	if len(posture.RemediationGuidance) > 0 {
		return true
	}
	compatibility := strings.ToLower(strings.TrimSpace(posture.PostureSummary.CompatibilityPosture))
	return compatibility != "" && compatibility != "supported" && compatibility != "ready"
}

func dashboardProjectSubstrateReason(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	if text := strings.TrimSpace(posture.BlockedExplanation); text != "" {
		return sanitizeUIText(text)
	}
	compatibility := strings.TrimSpace(posture.PostureSummary.CompatibilityPosture)
	if compatibility != "" {
		return fmt.Sprintf("project setup posture is %s", humanizeExecutionToken(compatibility))
	}
	if len(posture.RemediationGuidance) > 0 {
		return "project setup guidance is available"
	}
	return "project setup posture needs review"
}

func pendingApprovalCount(runs []brokerapi.RunSummary, approvals []brokerapi.ApprovalSummary) int {
	total := 0
	for _, run := range runs {
		total += run.PendingApprovalCount
	}
	if total > 0 {
		return total
	}
	pending := 0
	for _, approval := range approvals {
		if strings.EqualFold(strings.TrimSpace(approval.Status), "pending") {
			pending++
		}
	}
	return pending
}

func primaryDashboardRun(runs []brokerapi.RunSummary) brokerapi.RunSummary {
	if len(runs) == 0 {
		return brokerapi.RunSummary{}
	}
	return runs[0]
}
