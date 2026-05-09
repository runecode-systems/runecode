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
		return "unavailable here because promotion approvals must be completed in the promotion flow to preserve exact binding"
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
	return "follow the linked workflow-specific route because resolve is unavailable here"
}

func approvalFollowUpRoute(summary brokerapi.ApprovalSummary) string {
	if strings.TrimSpace(summary.BoundScope.RunID) != "" {
		return "Artifacts → Audit"
	}
	return "Audit"
}

func approvalDisplayLabel(summary brokerapi.ApprovalSummary) string {
	action := strings.TrimSpace(summary.BoundScope.ActionKind)
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
	trigger := strings.TrimSpace(summary.ApprovalTriggerCode)
	if trigger == "" {
		return "review pending"
	}
	return "reason=" + approvalUIValue(trigger)
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
