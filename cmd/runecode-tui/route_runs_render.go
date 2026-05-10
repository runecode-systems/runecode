package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderRunList(runs []brokerapi.RunSummary, selected int) string {
	if len(runs) == 0 {
		return "  - no runs"
	}
	line := ""
	for i, run := range runs {
		marker := " "
		if i == selected {
			marker = ">"
		}
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s %s approvals=%d", marker, run.RunID, stateBadgeWithLabel("state", run.LifecycleState), run.PendingApprovalCount)) + "\n"
		line += fmt.Sprintf("      %s | %s | %s | %s | approval_profile=%s\n", fmt.Sprintf("backend_kind=%s", valueOrNA(run.BackendKind)), renderRuntimeIsolationCue(run.BackendKind, run.IsolationAssuranceLevel), renderProvisioningPostureCue(run.ProvisioningPosture), renderAuditPostureCue(run.AuditIntegrityStatus, run.AuditAnchoringStatus, run.AuditCurrentlyDegraded), valueOrNA(run.ApprovalProfile))
	}
	return line
}

func renderRunDirectoryItems(runs []brokerapi.RunSummary) []string {
	items := make([]string, 0, len(runs))
	for _, run := range runs {
		items = append(items, strings.TrimSpace(strings.Join([]string{
			run.RunID,
			postureBadge(run.LifecycleState),
			runDirectoryWorkflowCue(run),
			runDirectoryAttentionCue(run),
			runDirectoryEvidenceCue(run),
		}, " • ")))
	}
	return items
}

func runStateCard(detail *brokerapi.RunDetail) stateCardSpec {
	if detail == nil {
		return stateCardSpec{
			State:       routeLoadStateEmpty,
			Title:       "Run overview",
			Message:     "Choose a run to review what happened and what needs attention.",
			Reason:      "The directory keeps each run tied to its approvals, artifacts, and audit trail.",
			NextAction:  "Select a run from the directory to open its operator summary.",
			ShortcutCue: "j/k move • enter review",
			RouteCue:    "Approvals / Artifacts / Audit",
		}
	}
	state := routeLoadStateReady
	if detail.Coordination.Blocked {
		state = routeLoadStateBlocked
	}
	if len(detail.PendingApprovalIDs) > 0 {
		state = routeLoadStateApprovalRequired
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(detail.Summary.LifecycleState)), "fail") {
		state = routeLoadStateDegraded
	}
	if strings.EqualFold(strings.TrimSpace(detail.Summary.LifecycleState), "completed") {
		state = routeLoadStateCompleted
	}
	return stateCardSpec{State: state, Title: "Run overview", Message: runOverviewMessage(detail), Reason: runOverviewReason(detail), NextAction: runOverviewNextAction(detail), ShortcutCue: "enter review • i inspector", RouteCue: "Chat / Approvals / Artifacts / Audit", EvidenceCue: runOverviewEvidenceCue(detail)}
}

func renderRunMainPaneCues(detail *brokerapi.RunDetail, summary brokerapi.RunSummary, width int) string {
	if strings.TrimSpace(summary.RunID) == "" {
		return ""
	}
	cues := []string{}
	if attention := strings.TrimSpace(runDirectoryAttentionCue(summary)); attention != "" {
		cues = append(cues, attention)
	}
	if runtime := strings.TrimSpace(runMainPaneRuntimeCue(summary)); runtime != "" {
		cues = append(cues, runtime)
	}
	if provisioning := strings.TrimSpace(runMainPaneProvisioningCue(summary)); provisioning != "" {
		cues = append(cues, provisioning)
	}
	if audit := strings.TrimSpace(runMainPaneAuditCue(summary)); audit != "" {
		cues = append(cues, audit)
	}
	if detail != nil {
		if evidence := strings.TrimSpace(runOverviewEvidenceCue(detail)); evidence != "" {
			cues = append(cues, evidence)
		}
	}
	if len(cues) == 0 {
		return ""
	}
	return compactLines(tableHeader("Safety and evidence"), wrapPartsByWidth(cues, " • ", width))
}

func runDirectoryWorkflowCue(summary brokerapi.RunSummary) string {
	workflow := strings.TrimSpace(summary.WorkflowKind)
	if workflow == "" {
		workflow = strings.TrimSpace(summary.CurrentStageID)
	}
	if workflow == "" {
		return "Run activity"
	}
	return humanizeRunLabel(workflow)
}

func runDirectoryAttentionCue(summary brokerapi.RunSummary) string {
	switch {
	case summary.PendingApprovalCount > 0:
		return countNoun(summary.PendingApprovalCount, "approval waiting", "approvals waiting")
	case strings.Contains(strings.ToLower(strings.TrimSpace(summary.LifecycleState)), "fail"):
		return "needs operator review"
	case strings.EqualFold(strings.TrimSpace(summary.LifecycleState), "blocked"):
		return "operator follow-up needed"
	case summary.RuntimePostureDegraded:
		return "runtime posture needs review"
	default:
		return "progress healthy"
	}
}

func runDirectoryEvidenceCue(summary brokerapi.RunSummary) string {
	if summary.AuditCurrentlyDegraded || !strings.EqualFold(strings.TrimSpace(summary.AuditAnchoringStatus), "ok") || !strings.EqualFold(strings.TrimSpace(summary.AuditIntegrityStatus), "ok") {
		return "audit trail needs review"
	}
	return "audit trail available"
}

func runMainPaneRuntimeCue(summary brokerapi.RunSummary) string {
	nBackend := strings.ToLower(strings.TrimSpace(summary.BackendKind))
	nIsolation := strings.ToLower(strings.TrimSpace(summary.IsolationAssuranceLevel))
	switch {
	case summary.RuntimePostureDegraded || nIsolation == "degraded" || nIsolation == "unavailable" || nIsolation == "unknown":
		return "runtime posture needs review"
	case nBackend == "container" || nIsolation == "reduced":
		return "runtime uses reduced assurance"
	case nIsolation == "sandboxed" || nIsolation == "isolated" || nIsolation == "microvm":
		return "runtime is isolated"
	default:
		return "runtime posture reported"
	}
}

func runMainPaneProvisioningCue(summary brokerapi.RunSummary) string {
	switch strings.ToLower(strings.TrimSpace(summary.ProvisioningPosture)) {
	case "ok", "trusted", "bound", "attested":
		return "provisioning evidence is in good standing"
	case "tofu", "degraded", "unavailable", "unknown":
		return "provisioning evidence needs review"
	default:
		return "provisioning posture reported"
	}
}

func runMainPaneAuditCue(summary brokerapi.RunSummary) string {
	if summary.AuditCurrentlyDegraded || !strings.EqualFold(strings.TrimSpace(summary.AuditAnchoringStatus), "ok") || !strings.EqualFold(strings.TrimSpace(summary.AuditIntegrityStatus), "ok") {
		return "audit trail needs review"
	}
	return "audit trail looks healthy"
}
