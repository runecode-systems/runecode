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
	BadgeLabel string
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
		reasons = append(reasons, fmt.Sprintf("%s %s blocked or waiting", dashboardCountPhrase(snapshot.BlockedRuns, "workflow", "workflows"), dashboardIsAre(snapshot.BlockedRuns)))
	}
	return dashboardExecutiveSummary{State: routeLoadStateBlocked, Title: "Blocked", BadgeLabel: "Action needed", Message: "Workflow progress is blocked.", Reason: strings.Join(reasons, "; "), NextAction: "Open Action Center and clear the blocker."}
}

func dashboardDegradedSummary(data dashboardData, snapshot dashboardSnapshot) dashboardExecutiveSummary {
	subjects := []string{}
	if !data.readiness.Ready {
		subjects = append(subjects, "broker readiness")
	}
	if snapshot.AuditDegraded {
		subjects = append(subjects, "evidence posture")
	}
	if snapshot.RunRuntimeDegraded > 0 {
		subjects = append(subjects, dashboardCountPhrase(snapshot.RunRuntimeDegraded, "runtime signal", "runtime signals"))
	}
	reason := "broker posture needs review"
	if len(subjects) > 0 {
		reason = fmt.Sprintf("%s %s review", dashboardJoinPhrases(subjects), dashboardNeedsNeed(len(subjects)))
	}
	return dashboardExecutiveSummary{State: routeLoadStateDegraded, Title: "Degraded", BadgeLabel: "Review", Message: "Evidence or runtime posture needs review.", Reason: reason, NextAction: "Open Action Center, then confirm Audit or Runs."}
}

func dashboardAttentionSummary(data dashboardData, snapshot dashboardSnapshot) dashboardExecutiveSummary {
	reasons := []string{}
	if snapshot.PendingApprovals > 0 {
		reasons = append(reasons, fmt.Sprintf("%s %s waiting", dashboardCountPhrase(snapshot.PendingApprovals, "approval", "approvals"), dashboardIsAre(snapshot.PendingApprovals)))
	}
	if snapshot.SetupNeedsAttention {
		reasons = append(reasons, dashboardProjectSubstrateAttentionReason(data.project))
	}
	return dashboardExecutiveSummary{State: routeLoadStateApprovalRequired, Title: "Needs attention", BadgeLabel: "Approval", Message: "Operator follow-up is waiting.", Reason: strings.Join(reasons, "; "), NextAction: "Open Action Center for the exact follow-up."}
}

func dashboardActiveWorkSummary(activeRuns int) dashboardExecutiveSummary {
	return dashboardExecutiveSummary{State: routeLoadStateReady, Title: "Active work", Message: "Workflow work is active.", Reason: fmt.Sprintf("%s %s active", dashboardCountPhrase(activeRuns, "workflow", "workflows"), dashboardIsAre(activeRuns)), NextAction: "Use Action Center or open Runs for detail."}
}

func renderDashboardHero(snapshot dashboardSnapshot, width int) string {
	exec := snapshot.Executive
	lines := []string{
		wrapDashboardLine("Message: "+exec.Message, width),
		wrapDashboardLine(dashboardReasonLabel(exec)+": "+exec.Reason, width),
		wrapDashboardLine("Next: "+exec.NextAction, width),
	}
	if cue := dashboardHeroRouteCue(snapshot); cue != "" {
		lines = append(lines, muted("Route: "+cue))
	}
	badge := dashboardStateBadge(exec)
	return renderProductCard(productCardSpec{Tone: dashboardCardTone(snapshot), Title: exec.Title, Badge: badge, Lines: lines})
}

func dashboardReasonLabel(exec dashboardExecutiveSummary) string {
	if exec.State == routeLoadStateBlocked || exec.State == routeLoadStateDegraded {
		return "Why"
	}
	return "Reason"
}

func dashboardStateBadge(exec dashboardExecutiveSummary) string {
	label := strings.TrimSpace(exec.BadgeLabel)
	if label == "" {
		return ""
	}
	if strings.EqualFold(label, strings.TrimSpace(exec.Title)) {
		return ""
	}
	return toneBadge(stateCardTone(exec.State), strings.ToUpper(label))
}

func dashboardHeroRouteCue(snapshot dashboardSnapshot) string {
	if snapshot.PendingApprovals > 0 || snapshot.BlockedRuns > 0 || snapshot.DegradedSignals > 0 || snapshot.SetupNeedsAttention || snapshot.SetupBlocked {
		return "Action Center"
	}
	if snapshot.ActiveRuns > 0 {
		return "Action Center or Runs"
	}
	return ""
}

func renderDashboardWorkflowPosture(snapshot dashboardSnapshot) string {
	parts := []string{}
	if snapshot.SetupBlocked {
		parts = append(parts, "normal product work is blocked until project setup is remediated")
	} else if snapshot.BlockedRuns > 0 {
		parts = append(parts, fmt.Sprintf("%s %s blocked or waiting on follow-up", dashboardCountPhrase(snapshot.BlockedRuns, "workflow", "workflows"), dashboardIsAre(snapshot.BlockedRuns)))
	} else if snapshot.ActiveRuns > 0 {
		parts = append(parts, fmt.Sprintf("%s %s active", dashboardCountPhrase(snapshot.ActiveRuns, "workflow", "workflows"), dashboardIsAre(snapshot.ActiveRuns)))
	} else {
		parts = append(parts, "no active workflow work is currently running")
	}
	if snapshot.PendingApprovals > 0 {
		parts = append(parts, fmt.Sprintf("%s %s waiting", dashboardCountPhrase(snapshot.PendingApprovals, "approval", "approvals"), dashboardIsAre(snapshot.PendingApprovals)))
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
		return "Current work: follow-up is waiting, but no active run is highlighted yet."
	}
	state := humanizeExecutionToken(snapshot.PrimaryRun.LifecycleState)
	parts := []string{fmt.Sprintf("Current work: %s is %s", sanitizeUIText(runID), state)}
	if snapshot.BlockedRuns > 0 || strings.TrimSpace(snapshot.PrimaryRun.BlockingReasonCode) != "" {
		parts = append(parts, "blocked follow-up is waiting")
	} else if snapshot.ActiveRuns > 0 {
		parts = append(parts, "work is moving")
	}
	if snapshot.DegradedSignals > 0 {
		parts = append(parts, "review is still needed")
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
		return fmt.Sprintf("Approvals: %s %s waiting.", dashboardCountPhrase(snapshot.PendingApprovals, "approval", "approvals"), dashboardIsAre(snapshot.PendingApprovals))
	}
	return fmt.Sprintf("Approvals: %s %s waiting; next is %s for run %s.", dashboardCountPhrase(snapshot.PendingApprovals, "approval", "approvals"), dashboardIsAre(snapshot.PendingApprovals), sanitizeUIText(first.ApprovalID), valueOrNA(sanitizeUIText(first.BoundScope.RunID)))
}

func dashboardMetricStrip(snapshot dashboardSnapshot) string {
	parts := []string{
		fmt.Sprintf("Work %d", snapshot.ActiveRuns),
		fmt.Sprintf("Approvals %d", snapshot.PendingApprovals),
		fmt.Sprintf("Blocked %d", snapshot.BlockedRuns),
		fmt.Sprintf("Review %d", snapshot.DegradedSignals),
	}
	return strings.Join(parts, "  •  ")
}

func renderDashboardDetailCue(snapshot dashboardSnapshot) string {
	if snapshot.BlockedRuns > 0 || snapshot.PendingApprovals > 0 || snapshot.DegradedSignals > 0 || snapshot.SetupNeedsAttention {
		return "Evidence: Runs, Audit, and Status keep proof details."
	}
	return "Evidence: open Runs, Audit, or Status for broker proof."
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
	if snapshot.BlockedRuns > 0 {
		return "Open Action Center and clear the blocker."
	}
	if snapshot.PendingApprovals > 0 {
		return "Open Action Center and review the pending approval."
	}
	if snapshot.DegradedSignals > 0 {
		return "Open Action Center for the follow-up, then inspect Audit or Runs."
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
		return fmt.Sprintf("Live updates need attention in %s; Action Center and Status have the details.", dashboardCountPhrase(degraded, "stream", "streams"))
	}
	if len(live.feed) > 0 {
		return fmt.Sprintf("Live updates are healthy; %s visible below.", dashboardCountPhrase(len(live.feed), "recent broker event", "recent broker events"))
	}
	return "Live updates are quiet; RuneCode will show broker-owned activity here when work changes."
}
