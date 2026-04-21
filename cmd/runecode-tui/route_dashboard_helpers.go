package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

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

func summarizeRunWatchEvents(events []brokerapi.RunWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "run_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Run != nil {
			s.lastSubject = event.Run.RunID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "run_watch_snapshot":
			s.snapshotCount++
		case "run_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func renderLiveActivityFeed(feed []shellLiveActivityEntry) string {
	if len(feed) == 0 {
		return "  feed: waiting for shell watch manager"
	}
	lines := make([]string, 0, len(feed)+1)
	lines = append(lines, "  feed:")
	for i := len(feed) - 1; i >= 0; i-- {
		e := feed[i]
		lines = append(lines, fmt.Sprintf("    %s event=%s subject=%s status=%s", infoBadge(sanitizeUIText(e.Family)), valueOrNA(sanitizeUIText(e.EventType)), valueOrNA(sanitizeUIText(e.Subject)), valueOrNA(sanitizeUIText(e.Status))))
	}
	return strings.Join(lines, "\n")
}

func summarizeApprovalWatchEvents(events []brokerapi.ApprovalWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "approval_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Approval != nil {
			s.lastSubject = event.Approval.ApprovalID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "approval_watch_snapshot":
			s.snapshotCount++
		case "approval_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func summarizeSessionWatchEvents(events []brokerapi.SessionWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "session_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Session != nil {
			s.lastSubject = event.Session.Identity.SessionID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "session_watch_snapshot":
			s.snapshotCount++
		case "session_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func renderWatchFamilySummary(summary watchFamilySummary) string {
	return fmt.Sprintf(
		"  %s\n    totals events=%d snapshot=%d upsert=%d terminal=%d errors=%d\n    last_event=%s subject=%s status=%s",
		infoBadge(sanitizeUIText(summary.family)),
		summary.eventCount,
		summary.snapshotCount,
		summary.upsertCount,
		summary.terminalCount,
		summary.errorCount,
		valueOrNA(sanitizeUIText(summary.lastEventType)),
		valueOrNA(sanitizeUIText(summary.lastSubject)),
		valueOrNA(sanitizeUIText(summary.lastStatus)),
	)
}

func pendingApprovalCount(runs []brokerapi.RunSummary, approvals []brokerapi.ApprovalSummary) int {
	total := 0
	for _, run := range runs {
		total += run.PendingApprovalCount
	}
	if total > 0 {
		return total
	}
	return len(approvals)
}

func renderRunHighlights(runs []brokerapi.RunSummary) string {
	if len(runs) == 0 {
		return "  No runs returned"
	}
	first := runs[0]
	return fmt.Sprintf("  Latest run %s %s backend=%s isolation=%s approvals=%d", sanitizeUIText(first.RunID), stateBadgeWithLabel("state", first.LifecycleState), sanitizeUIText(first.BackendKind), sanitizeUIText(first.IsolationAssuranceLevel), first.PendingApprovalCount)
}

func renderApprovalHighlights(approvals []brokerapi.ApprovalSummary) string {
	if len(approvals) == 0 {
		return "  No pending approvals"
	}
	first := approvals[0]
	return fmt.Sprintf("  Approval %s %s trigger=%s run=%s", sanitizeUIText(first.ApprovalID), stateBadgeWithLabel("status", first.Status), sanitizeUIText(first.ApprovalTriggerCode), valueOrNA(sanitizeUIText(first.BoundScope.RunID)))
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
		alerts = append(alerts, provisioningDegradedBadge("ALERT_TOFU_PROVISIONING")+" TOFU isolate key provisioning in effect")
	}
	if strings.ToLower(strings.TrimSpace(run.IsolationAssuranceLevel)) == "unknown" || strings.ToLower(strings.TrimSpace(run.IsolationAssuranceLevel)) == "unavailable" {
		alerts = append(alerts, dangerBadge("ALERT_RUNTIME_POSTURE_UNAVAILABLE")+" authoritative runtime isolation posture degraded/unavailable")
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
		return "Project substrate: unavailable"
	}
	return fmt.Sprintf(
		"Project substrate: validation=%s compatibility=%s normal_operation_allowed=%t",
		sanitizeUIText(summary.ValidationState),
		sanitizeUIText(summary.CompatibilityPosture),
		summary.NormalOperationAllowed,
	)
}

func renderDashboardProjectSubstrateGuidance(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	parts := []string{}
	if strings.TrimSpace(posture.BlockedExplanation) != "" {
		parts = append(parts, "Project substrate block: "+sanitizeUIText(posture.BlockedExplanation))
	}
	if len(posture.RemediationGuidance) > 0 {
		parts = append(parts, "Project substrate remediation: "+joinCSVWithWrapHint(posture.RemediationGuidance))
	}
	if strings.TrimSpace(posture.InitPreview.Status) != "" {
		parts = append(parts, fmt.Sprintf("Project substrate init=%s", posture.InitPreview.Status))
	}
	if strings.TrimSpace(posture.UpgradePreview.Status) != "" {
		parts = append(parts, fmt.Sprintf("Project substrate upgrade=%s", posture.UpgradePreview.Status))
	}
	if len(parts) == 0 {
		return "Project substrate guidance: none"
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
