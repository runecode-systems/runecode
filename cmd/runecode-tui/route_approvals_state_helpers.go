package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func approvalPrimaryStateBadge(summary brokerapi.ApprovalSummary) string {
	return workflowApprovalBadge(summary, brokerapi.ApprovalDetail{})
}

func approvalSupportBadge(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	state := workflowApprovalState(summary, detail)
	if state == "resolved" {
		return successBadge("RESOLVED")
	}
	if state == "expired" {
		return dangerBadge("EXPIRED")
	}
	if state == "denied" {
		return dangerBadge("DENIED")
	}
	if approvalResolveSupported(summary, detail) {
		return successBadge("RESOLVE_SUPPORTED")
	}
	return dangerBadge("RESOLVE_UNAVAILABLE")
}

func workflowApprovalBadge(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	switch workflowApprovalState(summary, detail) {
	case "resolved":
		return successBadge("READY_TO_CONTINUE")
	case "expired":
		return dangerBadge("EXPIRED")
	case "denied":
		return dangerBadge("DENIED")
	case "unsupported":
		return dangerBadge("RESOLVE_UNAVAILABLE")
	case "approval required":
		return approvalRequiredBadge("APPROVAL_REQUIRED")
	default:
		return warnBadge("PENDING_REVIEW")
	}
}

func workflowApprovalState(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	status := strings.ToLower(strings.TrimSpace(summary.Status))
	lifecycle := approvalLifecycleState(detail)
	if lifecycle == "expired" || status == "expired" {
		return "expired"
	}
	if lifecycle == "approved" || lifecycle == "consumed" || status == "approved" || status == "consumed" || status == "resolved" {
		return "resolved"
	}
	if lifecycle == "denied" || status == "denied" {
		return "denied"
	}
	if lifecycle == "pending" || status == "pending" {
		return "approval required"
	}
	if strings.TrimSpace(summary.ApprovalID) == "" {
		return "pending"
	}
	return valueOrNA(status)
}

func approvalLifecycleState(detail brokerapi.ApprovalDetail) string {
	return strings.ToLower(strings.TrimSpace(detail.LifecycleDetail.LifecycleState))
}

func boolWord(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

func approvalDisplayState(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	state := workflowApprovalState(summary, detail)
	if state == "approval required" {
		return "approval required"
	}
	return state
}

func approvalPrimaryReason(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	switch workflowApprovalState(summary, detail) {
	case "expired":
		return "the previous approval window has expired, so the blocked action cannot continue until a fresh broker approval exists"
	case "resolved":
		return "the broker has already recorded a decision for this gate"
	case "denied":
		return "the broker recorded a denied decision for this gate; workflow progress requires a fresh approved gate before continuing"
	default:
		policy := strings.TrimSpace(detail.PolicyReasonCode)
		trigger := strings.TrimSpace(summary.ApprovalTriggerCode)
		if approvalResolveSupported(summary, detail) {
			return fmt.Sprintf("approval required before %s can continue; broker validation still requires operator confirmation", approvalDisplayLabel(summary))
		}
		if policy != "" {
			return fmt.Sprintf("policy requires operator review before %s can continue", approvalDisplayLabel(summary))
		}
		if trigger != "" {
			return fmt.Sprintf("workflow is waiting on an approval gate triggered by %s", trigger)
		}
		return "workflow is blocked on an operator approval"
	}
}

func approvalResolveBlockedReason(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	state := workflowApprovalState(summary, detail)
	switch state {
	case "resolved":
		return "not available because the broker already recorded a final decision"
	case "expired":
		return "not available because the approval expired and broker validation requires a fresh approval object"
	}
	if approvalResolveSupported(summary, detail) {
		return "available after review-first evidence checks"
	}
	switch strings.TrimSpace(summary.BoundScope.ActionKind) {
	case "promotion":
		return "promotion approvals must stay in the promotion flow so exact promotion binding remains intact"
	default:
		return "broker validation does not expose a supported typed resolve path for this approval kind in the current TUI"
	}
}

func approvalBoundScopeCue(summary brokerapi.ApprovalSummary) string {
	parts := []string{}
	if run := strings.TrimSpace(summary.BoundScope.RunID); run != "" {
		parts = append(parts, "run "+run)
	}
	if stage := strings.TrimSpace(summary.BoundScope.StageID); stage != "" {
		parts = append(parts, "stage "+stage)
	}
	if action := strings.TrimSpace(summary.BoundScope.ActionKind); action != "" {
		parts = append(parts, humanizeExecutionToken(action))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " • ")
}

func approvalExactObjectAction(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	scope := detail.BlockedWorkScope
	if strings.TrimSpace(scope.ActionKind) == "" {
		scope.WorkspaceID = summary.BoundScope.WorkspaceID
		scope.RunID = summary.BoundScope.RunID
		scope.StageID = summary.BoundScope.StageID
		scope.StepID = summary.BoundScope.StepID
		scope.RoleInstanceID = summary.BoundScope.RoleInstanceID
		scope.ActionKind = summary.BoundScope.ActionKind
	}
	parts := []string{}
	if run := strings.TrimSpace(valueOrBlank(scope.RunID)); run != "" {
		parts = append(parts, "run "+run)
	}
	if stage := strings.TrimSpace(valueOrBlank(scope.StageID)); stage != "" {
		parts = append(parts, "stage "+stage)
	}
	if step := strings.TrimSpace(valueOrBlank(scope.StepID)); step != "" {
		parts = append(parts, "step "+step)
	}
	if role := strings.TrimSpace(valueOrBlank(scope.RoleInstanceID)); role != "" {
		parts = append(parts, "role "+role)
	}
	action := valueOrNA(strings.TrimSpace(scope.ActionKind))
	if len(parts) == 0 {
		return humanizeExecutionToken(action)
	}
	return strings.Join(parts, " • ") + " • " + humanizeExecutionToken(action)
}
