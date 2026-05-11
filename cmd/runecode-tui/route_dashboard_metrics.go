package main

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func dashboardCountPhrase(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func dashboardIsAre(count int) string {
	if count == 1 {
		return "is"
	}
	return "are"
}

func dashboardNeedsNeed(count int) string {
	if count == 1 {
		return "needs"
	}
	return "need"
}

func dashboardJoinPhrases(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	if len(parts) == 2 {
		return parts[0] + " and " + parts[1]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
}

func dashboardProjectSubstrateAttentionReason(posture brokerapi.ProjectSubstratePostureGetResponse) string {
	if len(posture.RemediationGuidance) > 0 {
		return "setup guidance is available"
	}
	compatibility := strings.ToLower(strings.TrimSpace(posture.PostureSummary.CompatibilityPosture))
	if strings.Contains(compatibility, "upgrade") {
		return "setup guidance is available"
	}
	return dashboardProjectSubstrateReason(posture)
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
	return !posture.PostureSummary.NormalOperationAllowed
}

func dashboardProjectSubstrateNeedsAttention(posture brokerapi.ProjectSubstratePostureGetResponse) bool {
	if strings.TrimSpace(posture.PostureSummary.SchemaID) == "" {
		return false
	}
	if dashboardProjectSubstrateBlocked(posture) || len(posture.RemediationGuidance) > 0 {
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
