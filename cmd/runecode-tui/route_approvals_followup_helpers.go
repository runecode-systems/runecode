package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func approvalEffectSummary(detail brokerapi.ApprovalDetail) string {
	summary := strings.TrimSpace(detail.WhatChangesIfApproved.Summary)
	effect := strings.TrimSpace(detail.WhatChangesIfApproved.EffectKind)
	if summary == "" && effect == "" {
		return "broker-owned next step is not described in approval detail"
	}
	if summary == "" {
		return fmt.Sprintf("effect=%s", effect)
	}
	if effect == "" {
		return summary
	}
	return fmt.Sprintf("%s (effect=%s)", summary, effect)
}

func approvalResolveSummary(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	state := workflowApprovalState(summary, detail)
	if state == "resolved" {
		return "already resolved; broker recorded the decision and follow-on workflow state can continue when refreshed"
	}
	if state == "expired" {
		return "unavailable because this approval expired; a fresh broker approval is required before the blocked action can continue"
	}
	if approvalResolveSupported(summary, detail) {
		return "available from this route after reviewing the evidence"
	}
	switch strings.TrimSpace(summary.BoundScope.ActionKind) {
	case "promotion":
		return "continue in the promotion flow after review"
	default:
		return "unavailable here because this approval type does not have a supported typed resolve path in the current TUI"
	}
}

func approvalResolveSupported(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) bool {
	selection := detail.BackendPostureSelection
	return strings.TrimSpace(summary.BoundScope.ActionKind) == "backend_posture_change" && selection != nil && strings.TrimSpace(selection.TargetInstanceID) != "" && strings.TrimSpace(selection.TargetBackendKind) != ""
}

func approvalResolveStatus(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	state := workflowApprovalState(summary, detail)
	if state == "resolved" {
		return "resolved"
	}
	if state == "expired" {
		return "expired"
	}
	if state == "denied" {
		return "denied"
	}
	if approvalLifecycleState(detail) != "pending" {
		return "approval-required"
	}
	if approvalResolveSupported(summary, detail) {
		return "supported"
	}
	return "unsupported"
}

func approvalReviewFirst(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	if digest := strings.TrimSpace(detail.BoundIdentity.SummaryPreviewDigest); digest != "" {
		return "summary preview artifact " + shortIdentity(digest) + " (copy raw digest from Artifacts if needed)"
	}
	if digest := strings.TrimSpace(detail.BoundIdentity.DiffDigest); digest != "" {
		return "diff artifact " + shortIdentity(digest) + " (copy raw digest from Artifacts if needed)"
	}
	if digest := strings.TrimSpace(detail.BoundIdentity.ArtifactSetDigest); digest != "" {
		return "artifact set " + shortIdentity(digest) + " (copy raw digest from Artifacts if needed)"
	}
	if run := strings.TrimSpace(summary.BoundScope.RunID); run != "" {
		return "run evidence for " + run
	}
	return "the linked broker evidence"
}

func approvalOperatorReason(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	switch workflowApprovalState(summary, detail) {
	case "expired":
		return "the previous approval window expired, so the blocked action needs a fresh approval before it can continue"
	case "resolved":
		return "the broker already recorded a decision for this approval"
	case "denied":
		return "the broker recorded a denied decision, so workflow progress needs a new approved gate"
	default:
		if approvalResolveSupported(summary, detail) {
			return fmt.Sprintf("operator review is still required before %s can continue", approvalDisplayLabel(summary))
		}
		return fmt.Sprintf("operator review is required before %s can continue", approvalDisplayLabel(summary))
	}
}

func approvalWorkScopeSummary(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
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
	if action := strings.TrimSpace(scope.ActionKind); action != "" {
		parts = append(parts, humanizeExecutionToken(action))
	}
	if len(parts) == 0 {
		return "linked approval scope"
	}
	return strings.Join(parts, " • ")
}

func approvalEvidenceGuidance(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	first := approvalReviewFirst(summary, detail)
	if strings.TrimSpace(summary.BoundScope.RunID) != "" {
		return fmt.Sprintf("Evidence trail: review %s, then open Audit for policy and verification context.", first)
	}
	return fmt.Sprintf("Evidence trail: review %s, then continue in Audit for the linked decision record.", first)
}

func approvalDecisionGuidance(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	switch workflowApprovalState(summary, detail) {
	case "resolved":
		return "Decision path: this approval is already settled; refresh to confirm the workflow continues."
	case "expired":
		return "Decision path: this approval expired; request a fresh approval before continuing."
	case "denied":
		return "Decision path: this approval was denied; the workflow needs a new approved gate before continuing."
	}
	if approvalResolveSupported(summary, detail) {
		return "Decision path: after review, you can resolve this approval here."
	}
	return fmt.Sprintf("Decision path: review here, then continue in %s for the final workflow-specific step.", approvalFollowUpRoute(summary))
}

func approvalAuditLinkSummary(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	if run := strings.TrimSpace(summary.BoundScope.RunID); run != "" {
		return "run " + run
	}
	if digest := strings.TrimSpace(detail.BoundIdentity.PolicyDecisionHash); digest != "" {
		return "policy decision " + shortIdentity(digest)
	}
	return "linked approval records"
}

func approvalNextAction(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail) string {
	if approvalResolveSupported(summary, detail) {
		return "resolve it here if the evidence supports the requested change"
	}
	return "continue in the linked route after review"
}

func approvalFollowUpRoute(summary brokerapi.ApprovalSummary) string {
	if strings.TrimSpace(summary.BoundScope.RunID) != "" {
		return "Artifacts → Audit"
	}
	return "Audit"
}

func approvalDisplayLabel(summary brokerapi.ApprovalSummary) string {
	action := humanizeExecutionToken(summary.BoundScope.ActionKind)
	if run := strings.TrimSpace(summary.BoundScope.RunID); run != "" {
		if action != "" {
			return fmt.Sprintf("%s for %s", approvalUIValue(action), approvalUIValue(run))
		}
		return "approval for " + approvalUIValue(run)
	}
	if action != "" {
		return approvalUIValue(action)
	}
	return approvalUIValue(summary.ApprovalID)
}

func approvalQueueReason(summary brokerapi.ApprovalSummary) string {
	if run := strings.TrimSpace(summary.BoundScope.RunID); run != "" {
		return "Review evidence for " + approvalUIValue(run)
	}
	if action := strings.TrimSpace(summary.BoundScope.ActionKind); action != "" {
		return "Review the requested " + approvalUIValue(humanizeExecutionToken(action))
	}
	return "Review linked evidence"
}

func shortIdentity(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 20 {
		return valueOrNA(value)
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 2 && len(parts[1]) > 12 {
		return parts[0] + ":" + parts[1][:12]
	}
	return value[:20]
}

func valueOrBlank(value string) string {
	return strings.TrimSpace(value)
}
