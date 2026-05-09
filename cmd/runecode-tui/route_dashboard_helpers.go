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
	return dangerBadge("AUDIT_VERIFICATION_UNAVAILABLE") + " audit verification unavailable; showing degraded fallback posture (" + auditErr + ")"
}

func renderDashboardNowBar(run brokerapi.RunSummary, approvalCount int, focusActive bool, width int) string {
	parts := []string{}
	if focusActive {
		parts = append(parts, successBadge("CONTENT_READY"))
	} else {
		parts = append(parts, neutralBadge("CONTENT_IDLE"))
	}
	if strings.TrimSpace(run.RunID) != "" {
		parts = append(parts, fmt.Sprintf("run=%s", sanitizeUIText(run.RunID)))
		parts = append(parts, stateBadgeWithLabel("state", run.LifecycleState))
		parts = append(parts, stateBadgeWithLabel("backend", sanitizeUIText(run.BackendKind)))
	}
	if approvalCount > 0 {
		parts = append(parts, approvalRequiredBadge(fmt.Sprintf("PENDING_APPROVALS=%d", approvalCount)))
	} else {
		parts = append(parts, successBadge("PENDING_APPROVALS=0"))
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
		reasons = append(reasons, fmt.Sprintf("%d run(s) report blocked or waiting workflow posture", snapshot.BlockedRuns))
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
	return dashboardExecutiveSummary{State: routeLoadStateReady, Title: "Active work", Message: "Workflow work is active.", Reason: fmt.Sprintf("%d active run(s) are visible on the current broker surfaces.", activeRuns), NextAction: "Use Action Center or open Runs for detail."}
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
	return "High-value counts: " + wrapPartsByWidth(parts, " | ", width)
}

func renderDashboardNextActions(data dashboardData, snapshot dashboardSnapshot) string {
	if snapshot.SetupBlocked {
		return "Next action: Open Action Center, follow the setup blocker, then use Status for broker-owned remediation steps."
	}
	if snapshot.PendingApprovals > 0 {
		return "Next action: Open Action Center and review the pending approval before workflow progress resumes."
	}
	if snapshot.DegradedSignals > 0 {
		return "Next action: Open Action Center for degraded evidence or runtime follow-up, then inspect Audit or Runs."
	}
	if strings.TrimSpace(snapshot.PrimaryRun.RunID) != "" {
		return fmt.Sprintf("Next action: Open Runs for %s or Action Center if new follow-up appears.", sanitizeUIText(snapshot.PrimaryRun.RunID))
	}
	return "Next action: Start a workflow from Chat, then return here for the executive overview."
}

func renderDashboardActionCenterCue(snapshot dashboardSnapshot) string {
	return fmt.Sprintf(
		"Action Center is the operator home for approvals, blocked work, degraded posture, setup follow-up, and waiting queues. Current follow-up: approvals=%d blocked_or_waiting=%d degraded=%d.",
		snapshot.PendingApprovals,
		snapshot.BlockedRuns,
		snapshot.DegradedSignals,
	)
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
		return fmt.Sprintf("project setup posture=%s", sanitizeUIText(compatibility))
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

func renderRunHighlights(runs []brokerapi.RunSummary) string {
	if len(runs) == 0 {
		return "Current work: no recent runs returned by the broker."
	}
	first := runs[0]
	return fmt.Sprintf("Current work: latest run %s is %s on %s runtime with %d pending approval(s).", sanitizeUIText(first.RunID), valueOrNA(sanitizeUIText(first.LifecycleState)), valueOrNA(sanitizeUIText(first.BackendKind)), first.PendingApprovalCount)
}

func renderApprovalHighlights(approvals []brokerapi.ApprovalSummary) string {
	if len(approvals) == 0 {
		return "Approval follow-up: no pending approvals returned by the broker."
	}
	first := approvals[0]
	return fmt.Sprintf("Approval follow-up: %s is %s for %s on run %s.", sanitizeUIText(first.ApprovalID), valueOrNA(sanitizeUIText(first.Status)), valueOrNA(sanitizeUIText(first.ApprovalTriggerCode)), valueOrNA(sanitizeUIText(first.BoundScope.RunID)))
}

func primaryDashboardRun(runs []brokerapi.RunSummary) brokerapi.RunSummary {
	if len(runs) == 0 {
		return brokerapi.RunSummary{}
	}
	return runs[0]
}

func renderDashboardSafetyAlerts(data dashboardData) string {
	alerts := []string{}
	run := primaryDashboardRun(data.runs)
	if strings.ToLower(strings.TrimSpace(run.ProvisioningPosture)) == "tofu" {
		alerts = append(alerts, dangerBadge("ALERT_TOFU_PROVISIONING")+" unsupported legacy TOFU provisioning posture detected")
	}
	if strings.ToLower(strings.TrimSpace(run.IsolationAssuranceLevel)) == "unknown" || strings.ToLower(strings.TrimSpace(run.IsolationAssuranceLevel)) == "unavailable" {
		alerts = append(alerts, dangerBadge("ALERT_RUNTIME_POSTURE_UNAVAILABLE")+" authoritative runtime isolation posture degraded/unavailable")
	}
	if run.RuntimePostureDegraded {
		alerts = append(alerts, reducedAssuranceBadge("ALERT_REDUCED_ASSURANCE_RUNTIME")+" reduced-assurance runtime posture is active")
	}
	if strings.ToLower(strings.TrimSpace(data.audit.Summary.AnchoringStatus)) == "degraded" || data.audit.Summary.CurrentlyDegraded {
		alerts = append(alerts, auditDegradedBadge("ALERT_AUDIT_UNANCHORED")+" audit posture unanchored/degraded")
	}
	if len(alerts) == 0 {
		return "Safety alerts: " + successBadge("NO_ACTIVE_DEGRADATION")
	}
	return "Safety alerts: " + strings.Join(alerts, " ")
}

func renderDashboardProjectSubstrateLine(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	summary := posture.PostureSummary
	if strings.TrimSpace(summary.SchemaID) == "" {
		return "Project setup: unavailable"
	}
	return fmt.Sprintf(
		"Project setup: validation=%s compatibility=%s normal_operation_allowed=%t",
		sanitizeUIText(summary.ValidationState),
		sanitizeUIText(summary.CompatibilityPosture),
		summary.NormalOperationAllowed,
	)
}

func renderDashboardProjectSubstrateGuidance(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	parts := []string{}
	if strings.TrimSpace(posture.BlockedExplanation) != "" {
		parts = append(parts, "Project setup block: "+sanitizeUIText(posture.BlockedExplanation))
	}
	if len(posture.RemediationGuidance) > 0 {
		parts = append(parts, "Project setup guidance:")
		parts = append(parts, joinCSVWithWrapHint(posture.RemediationGuidance))
	}
	if strings.TrimSpace(posture.InitPreview.Status) != "" {
		parts = append(parts, fmt.Sprintf("Project setup init preview=%s", posture.InitPreview.Status))
	}
	if strings.TrimSpace(posture.UpgradePreview.Status) != "" {
		parts = append(parts, fmt.Sprintf("Project setup upgrade preview=%s", posture.UpgradePreview.Status))
	}
	if len(parts) == 0 {
		return "Project setup guidance: none"
	}
	return strings.Join(parts, " | ")
}

func joinCSVWithWrapHint(values []string) string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	return strings.Join(clean, ", ")
}
