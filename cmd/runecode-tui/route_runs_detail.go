package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type runEvidenceLinks struct {
	SessionID      string
	ApprovalIDs    []string
	ArtifactLabels []string
	AuditLabels    []string
}

func renderRunInspector(detail *brokerapi.RunDetail, presentation contentPresentationMode, document *longFormDocumentState) string {
	if detail == nil {
		return "  Select a run to review workflow status, outcome, and linked evidence."
	}
	if document == nil {
		fallback := newLongFormDocumentState()
		document = &fallback
	}
	summary := detail.Summary
	presentation = normalizePresentationMode(presentation)
	waitingStages := pendingApprovalStageCount(detail.StageSummaries)
	waitingRoles := waitingRoleCount(detail.RoleSummaries)
	links := buildRunEvidenceLinks(detail)
	content := runInspectorContent(summary, detail, waitingStages, waitingRoles, links, presentation)
	contentKind := runInspectorContentKind(presentation)
	ref := workbenchObjectRef{Kind: "run", ID: strings.TrimSpace(summary.RunID), WorkspaceID: strings.TrimSpace(summary.WorkspaceID)}
	document.SetDocument(ref, contentKind, "run details", content)
	return renderInspectorShell(inspectorShellSpec{
		Title:        "Run inspector",
		Summary:      runInspectorSummary(summary, detail),
		Identity:     runInspectorIdentity(summary, detail),
		Status:       runInspectorStatus(summary, detail),
		Badges:       []string{stateBadgeWithLabel("state", summary.LifecycleState), appTheme.InspectorHint.Render("authoritative vs advisory state shown")},
		References:   runInspectorReferences(detail, links),
		LocalActions: runInspectorLocalActions(),
		CopyActions:  runRouteCopyActions(detail),
		ModeTabs:     []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:   string(presentation),
		Document:     document,
	})
}

func runInspectorLocalActions() []routeActionItem {
	return []routeActionItem{
		{Label: "jump:session", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeChat}}},
		{Label: "jump:approvals", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeApprovals}}},
		{Label: "jump:artifacts", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeArtifacts}}},
		{Label: "jump:audit", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}},
		{Label: "copy:run_id"},
		{Label: "copy:session_id"},
	}
}

func runInspectorReferences(detail *brokerapi.RunDetail, links runEvidenceLinks) []inspectorReference {
	if detail == nil {
		return nil
	}
	refs := []inspectorReference{}
	if sessionID := strings.TrimSpace(links.SessionID); sessionID != "" {
		refs = append(refs, inspectorReference{Label: "session", Items: []inspectorReferenceItem{{Label: sessionID, Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "session", RouteID: routeChat, SessionID: sessionID}}}}})
	}
	refs = append(refs,
		inspectorReference{Label: "approvals", Items: mapReferenceIDs(links.ApprovalIDs, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: id}}
		})},
		inspectorReference{Label: "artifacts", Items: mapReferenceIDs(links.ArtifactLabels, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeArtifacts}}
		})},
		inspectorReference{Label: "audit", Items: mapReferenceIDs(links.AuditLabels, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}
		})},
	)
	return refs
}

func runInspectorReferenceActions(detail *brokerapi.RunDetail) []routeActionItem {
	if detail == nil {
		return nil
	}
	refs := runInspectorReferences(detail, buildRunEvidenceLinks(detail))
	out := make([]routeActionItem, 0, 12)
	for _, ref := range refs {
		for _, item := range ref.Items {
			if strings.TrimSpace(item.Label) == "" {
				continue
			}
			out = append(out, routeActionItem{Label: ref.Label + ":" + item.Label, Action: item.Action})
		}
	}
	return out
}

func pendingApprovalStageCount(stages []brokerapi.RunStageSummary) int {
	count := 0
	for _, stage := range stages {
		if stage.PendingApprovalCount > 0 {
			count++
		}
	}
	return count
}

func waitingRoleCount(roles []brokerapi.RunRoleSummary) int {
	count := 0
	for _, role := range roles {
		if role.WaitReasonCode != "" {
			count++
		}
	}
	return count
}

func buildRunEvidenceLinks(detail *brokerapi.RunDetail) runEvidenceLinks {
	if detail == nil {
		return runEvidenceLinks{}
	}
	return runEvidenceLinks{
		SessionID:      strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "session_id")),
		ApprovalIDs:    append([]string(nil), detail.PendingApprovalIDs...),
		ArtifactLabels: runArtifactReferenceLabels(detail),
		AuditLabels:    runAuditReferenceLabels(detail),
	}
}

func runInspectorContent(summary brokerapi.RunSummary, detail *brokerapi.RunDetail, waitingStages int, waitingRoles int, links runEvidenceLinks, presentation contentPresentationMode) string {
	if presentation == presentationStructured {
		return compactLines(
			fmt.Sprintf("structured run detail: run=%s workflow=%s status=%s", summary.RunID, valueOrNA(summary.WorkflowKind), valueOrNA(summary.LifecycleState)),
			fmt.Sprintf("counts: authoritative=%d advisory=%d stages=%d roles=%d pending_approvals=%d artifact_classes=%d", len(detail.AuthoritativeState), len(detail.AdvisoryState), len(detail.StageSummaries), len(detail.RoleSummaries), len(detail.PendingApprovalIDs), len(detail.ArtifactCountsByClass)),
		)
	}
	if presentation == presentationRaw {
		return compactLines(
			fmt.Sprintf("raw summary run_id=%s lifecycle=%s backend_kind=%s workflow_kind=%s", summary.RunID, summary.LifecycleState, summary.BackendKind, valueOrNA(summary.WorkflowKind)),
			fmt.Sprintf("raw coordination blocked=%t wait_reason=%s mode=%s", detail.Coordination.Blocked, valueOrNA(detail.Coordination.WaitReasonCode), valueOrNA(detail.Coordination.CoordinationMode)),
			fmt.Sprintf("raw maps authoritative_keys=%d advisory_keys=%d artifact_classes=%d", len(detail.AuthoritativeState), len(detail.AdvisoryState), len(detail.ArtifactCountsByClass)),
		)
	}
	attestationPosture, attestationReasons := attestationPostureFromState(detail.AuthoritativeState)
	return compactLines(
		runOutcomeHeadline(summary, detail),
		fmt.Sprintf("Workflow operation: %s", runWorkflowOperation(summary, detail)),
		fmt.Sprintf("Runtime backend: backend_kind=%s", valueOrNA(summary.BackendKind)),
		muted("Structured/raw modes expose workflow hashes, coordination internals, approval profile, manifests, policy refs, and broker map counts."),
		fmt.Sprintf("Plan authority: %s", runPlanAuthoritySummary(summary, detail)),
		fmt.Sprintf("Runner/reporting posture: %s", runReportingPostureSummary(detail)),
		fmt.Sprintf("Blocked/failure reason: %s", runBlockingReasonSummary(summary, detail)),
		fmt.Sprintf("Evidence links: %s", runEvidenceSummary(detail)),
		fmt.Sprintf("Linked approvals: %s", runJoinedOrNA(links.ApprovalIDs)),
		fmt.Sprintf("Artifacts trail: %s", runJoinedOrNA(links.ArtifactLabels)),
		fmt.Sprintf("Audit evidence: %s", runJoinedOrNA(links.AuditLabels)),
		fmt.Sprintf("Navigation cues: session=%s approvals=%d artifacts=%d audit=%d", valueOrNA(links.SessionID), len(links.ApprovalIDs), runArtifactCount(detail), len(links.AuditLabels)),
		"Runtime assurance: "+renderRuntimeIsolationCue(summary.BackendKind, summary.IsolationAssuranceLevel),
		"Provisioning posture: "+renderProvisioningPostureCue(summary.ProvisioningPosture),
		"Attestation truth: "+renderAttestationPostureCue(attestationPosture, attestationReasons)+"; "+renderRuntimeAttestationTruthfulnessCue(detail.AuthoritativeState),
		"Audit posture: "+renderAuditPostureCue(summary.AuditIntegrityStatus, summary.AuditAnchoringStatus, summary.AuditCurrentlyDegraded),
		fmt.Sprintf("Work summary: %d stage(s), %d with approval waits, %d waiting role(s), %d pending approval(s).", len(detail.StageSummaries), waitingStages, waitingRoles, len(detail.PendingApprovalIDs)),
	)
}

func runInspectorSummary(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	return fmt.Sprintf("Run %s is %s with %d pending approval(s).", summary.RunID, valueOrNA(summary.LifecycleState), runPendingApprovalCount(summary, detail))
}

func runInspectorIdentity(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	return fmt.Sprintf("run=%s workspace=%s session=%s", summary.RunID, valueOrNA(summary.WorkspaceID), valueOrNA(authoritativeString(detail.AuthoritativeState, "session_id")))
}

func runInspectorStatus(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	return fmt.Sprintf("lifecycle=%s workflow=%s blocked=%t reason=%s", valueOrNA(summary.LifecycleState), runWorkflowOperation(summary, detail), detail.Coordination.Blocked, valueOrNA(runBlockingReasonCode(summary, detail)))
}

func runOutcomeHeadline(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	return fmt.Sprintf("Outcome summary: run %s is %s.", summary.RunID, valueOrNA(summary.LifecycleState))
}

func runOverviewMessage(detail *brokerapi.RunDetail) string {
	if detail == nil {
		return "No run selected."
	}
	summary := detail.Summary
	status := humanizeExecutionToken(summary.LifecycleState)
	if len(detail.PendingApprovalIDs) > 0 {
		return fmt.Sprintf("Run %s is %s and waiting on %s.", summary.RunID, status, countNoun(len(detail.PendingApprovalIDs), "approval", "approvals"))
	}
	if detail.Coordination.Blocked {
		return fmt.Sprintf("Run %s is %s and currently blocked.", summary.RunID, status)
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(summary.LifecycleState)), "fail") {
		return fmt.Sprintf("Run %s finished in a failed state.", summary.RunID)
	}
	if strings.EqualFold(strings.TrimSpace(summary.LifecycleState), "completed") {
		return fmt.Sprintf("Run %s completed successfully.", summary.RunID)
	}
	return fmt.Sprintf("Run %s is %s.", summary.RunID, status)
}

func runOverviewReason(detail *brokerapi.RunDetail) string {
	if detail == nil {
		return "n/a"
	}
	parts := []string{}
	if workflow := strings.TrimSpace(runWorkflowOperatorCue(detail.Summary, detail)); workflow != "" {
		parts = append(parts, workflow)
	}
	if blocked := strings.TrimSpace(runBlockingReasonSummary(detail.Summary, detail)); blocked != "" && blocked != "no blocking or failure reason reported" {
		parts = append(parts, blocked)
	}
	if progress := strings.TrimSpace(runOperatorProgressCue(detail)); progress != "" {
		parts = append(parts, progress)
	}
	if len(parts) == 0 {
		return "No additional operator context reported yet."
	}
	return strings.Join(parts, " • ")
}

func runOverviewNextAction(detail *brokerapi.RunDetail) string {
	if detail == nil {
		return "Select a run from the directory."
	}
	if len(detail.PendingApprovalIDs) > 0 {
		return fmt.Sprintf("Open Approvals to review %s, then return here for the updated result.", countNoun(len(detail.PendingApprovalIDs), "pending approval", "pending approvals"))
	}
	if detail.Coordination.Blocked {
		if sessionID := strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "session_id")); sessionID != "" {
			return fmt.Sprintf("Check session %s for operator context, then review linked artifacts or audit evidence if the block clears.", sessionID)
		}
		return "Review the linked session or evidence trail to clear the current block."
	}
	if runArtifactCount(detail) > 0 {
		return "Open Artifacts to review the resulting evidence, or Audit if you need the verification trail."
	}
	if runAuditReferenceCount(detail) > 0 {
		return "Open Audit to confirm the recorded evidence trail."
	}
	if sessionID := strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "session_id")); sessionID != "" {
		return fmt.Sprintf("Open Chat for session %s if you need the surrounding operator conversation.", sessionID)
	}
	return "Use the inspector for structured detail if you need the underlying control-plane fields."
}

func runOverviewEvidenceCue(detail *brokerapi.RunDetail) string {
	if detail == nil {
		return ""
	}
	parts := []string{}
	if approvals := len(detail.PendingApprovalIDs); approvals > 0 {
		parts = append(parts, countNoun(approvals, "approval link", "approval links"))
	}
	if artifacts := runArtifactCount(detail); artifacts > 0 {
		parts = append(parts, countNoun(artifacts, "artifact", "artifacts"))
	}
	if audits := runAuditReferenceCount(detail); audits > 0 {
		parts = append(parts, countNoun(audits, "audit reference", "audit references"))
	}
	if len(parts) == 0 {
		return "No linked evidence reported yet."
	}
	return strings.Join(parts, " • ")
}

func runWorkflowOperatorCue(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	workflow := strings.TrimSpace(summary.WorkflowKind)
	if workflow == "" {
		workflow = strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "workflow_kind"))
	}
	stage := strings.TrimSpace(summary.CurrentStageID)
	if stage == "" {
		stage = strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "workflow_operation"))
	}
	parts := []string{}
	if workflow != "" {
		parts = append(parts, humanizeRunLabel(workflow))
	}
	if stage != "" {
		parts = append(parts, "current stage "+humanizeRunLabel(stage))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " • ")
}

func runOperatorProgressCue(detail *brokerapi.RunDetail) string {
	if detail == nil {
		return ""
	}
	parts := []string{}
	if runRunnerActive(detail) {
		parts = append(parts, "runner reporting is active")
	}
	if checkpoint := strings.TrimSpace(runCheckpointCode(detail)); checkpoint != "" {
		parts = append(parts, "last checkpoint "+humanizeRunLabel(checkpoint))
	}
	return strings.Join(parts, " • ")
}

func humanizeRunLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	return value
}

func runWorkflowOperation(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	workflow := strings.TrimSpace(summary.WorkflowKind)
	if workflow == "" {
		workflow = strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "workflow_kind"))
	}
	operation := strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "workflow_operation"))
	if operation == "" {
		operation = strings.TrimSpace(summary.CurrentStageID)
	}
	if workflow != "" && operation != "" {
		return fmt.Sprintf("%s • stage=%s", workflow, operation)
	}
	if workflow != "" {
		return workflow
	}
	if operation != "" {
		return "stage=" + operation
	}
	return "n/a"
}

func runPendingApprovalCount(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) int {
	if detail != nil && len(detail.PendingApprovalIDs) > 0 {
		return len(detail.PendingApprovalIDs)
	}
	return summary.PendingApprovalCount
}

func runPlanAuthoritySummary(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	reason := strings.TrimSpace(authoritativeString(detail.AuthoritativeState, "workflow_projection_reason"))
	if strings.TrimSpace(summary.WorkflowDefinitionHash) != "" {
		return fmt.Sprintf("workflow definition hash %s with projection=%s", summary.WorkflowDefinitionHash, valueOrNA(reason))
	}
	if reason != "" {
		return fmt.Sprintf("projection=%s", reason)
	}
	return "plan authority not reported"
}

func runReportingPostureSummary(detail *brokerapi.RunDetail) string {
	parts := []string{}
	if runner := strings.TrimSpace(authoritativeString(detail.AdvisoryState, "runner")); runner != "" {
		parts = append(parts, "runner="+runner)
	}
	if checkpoint := strings.TrimSpace(runCheckpointCode(detail)); checkpoint != "" {
		parts = append(parts, "last checkpoint="+checkpoint)
	}
	if len(parts) == 0 {
		return "runner reporting has not produced advisory progress yet"
	}
	return strings.Join(parts, " • ")
}

func runJoinedOrNA(items []string) string {
	if len(items) == 0 {
		return "n/a"
	}
	return strings.Join(items, " • ")
}

func runBlockingReasonCode(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	if reason := strings.TrimSpace(summary.BlockingReasonCode); reason != "" {
		return reason
	}
	return strings.TrimSpace(detail.Coordination.WaitReasonCode)
}

func runBlockingReasonSummary(summary brokerapi.RunSummary, detail *brokerapi.RunDetail) string {
	reason := runBlockingReasonCode(summary, detail)
	if reason == "" && !detail.Coordination.Blocked && !strings.Contains(strings.ToLower(strings.TrimSpace(summary.LifecycleState)), "fail") {
		return "no blocking or failure reason reported"
	}
	if reason == "" {
		return "state reported without a reason code"
	}
	return humanizeExecutionToken(reason)
}
