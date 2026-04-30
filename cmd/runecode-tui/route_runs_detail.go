package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderRunInspector(detail *brokerapi.RunDetail, presentation contentPresentationMode, document *longFormDocumentState) string {
	if detail == nil {
		return "  Select a run and press enter to load detail."
	}
	if document == nil {
		fallback := newLongFormDocumentState()
		document = &fallback
	}
	summary := detail.Summary
	presentation = normalizePresentationMode(presentation)
	waitingStages := pendingApprovalStageCount(detail.StageSummaries)
	waitingRoles := waitingRoleCount(detail.RoleSummaries)
	content := runInspectorContent(summary, detail, waitingStages, waitingRoles, presentation)
	contentKind := runInspectorContentKind(presentation)
	ref := workbenchObjectRef{Kind: "run", ID: strings.TrimSpace(summary.RunID), WorkspaceID: strings.TrimSpace(summary.WorkspaceID)}
	document.SetDocument(ref, contentKind, "run details", content)
	return renderInspectorShell(inspectorShellSpec{
		Title:    "Run inspector",
		Summary:  fmt.Sprintf("run=%s lifecycle=%s pending_approvals=%d", summary.RunID, valueOrNA(summary.LifecycleState), summary.PendingApprovalCount),
		Identity: fmt.Sprintf("run=%s backend=%s", summary.RunID, valueOrNA(summary.BackendKind)),
		Status:   fmt.Sprintf("lifecycle=%s blocked=%t", valueOrNA(summary.LifecycleState), detail.Coordination.Blocked),
		Badges:   []string{stateBadgeWithLabel("state", summary.LifecycleState), appTheme.InspectorHint.Render("authoritative vs advisory state shown")},
		References: []inspectorReference{{Label: "approvals", Items: mapReferenceIDs(detail.PendingApprovalIDs, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: id}}
		})}},
		LocalActions: runInspectorLocalActions(),
		CopyActions:  runRouteCopyActions(detail),
		ModeTabs:     []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:   string(presentation),
		Document:     document,
	})
}

func runInspectorLocalActions() []routeActionItem {
	return []routeActionItem{
		{Label: "jump:approvals", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeApprovals}}},
		{Label: "jump:artifacts", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeArtifacts}}},
		{Label: "jump:audit", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}},
		{Label: "copy:run_id"},
	}
}

func runInspectorReferenceActions(detail *brokerapi.RunDetail) []routeActionItem {
	if detail == nil {
		return nil
	}
	items := mapReferenceIDs(detail.PendingApprovalIDs, func(id string) paletteActionMsg {
		return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: id}}
	})
	out := make([]routeActionItem, 0, len(items))
	for _, item := range items {
		out = append(out, routeActionItem{Label: "approval:" + item.Label, Action: item.Action})
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

func runInspectorContent(summary brokerapi.RunSummary, detail *brokerapi.RunDetail, waitingStages int, waitingRoles int, presentation contentPresentationMode) string {
	if presentation == presentationStructured {
		return compactLines(
			fmt.Sprintf("structured run detail: run=%s", summary.RunID),
			fmt.Sprintf("counts: authoritative=%d advisory=%d stages=%d roles=%d pending_approvals=%d", len(detail.AuthoritativeState), len(detail.AdvisoryState), len(detail.StageSummaries), len(detail.RoleSummaries), len(detail.PendingApprovalIDs)),
		)
	}
	if presentation == presentationRaw {
		return compactLines(
			fmt.Sprintf("raw summary run_id=%s lifecycle=%s backend_kind=%s", summary.RunID, summary.LifecycleState, summary.BackendKind),
			fmt.Sprintf("raw coordination blocked=%t wait_reason=%s mode=%s", detail.Coordination.Blocked, valueOrNA(detail.Coordination.WaitReasonCode), valueOrNA(detail.Coordination.CoordinationMode)),
			fmt.Sprintf("raw maps authoritative_keys=%d advisory_keys=%d", len(detail.AuthoritativeState), len(detail.AdvisoryState)),
		)
	}
	attestationPosture, attestationReasons := attestationPostureFromState(detail.AuthoritativeState)
	return compactLines(
		fmt.Sprintf("backend_kind=%s", summary.BackendKind),
		"Runtime isolation assurance (authoritative): "+renderRuntimeIsolationCue(summary.BackendKind, summary.IsolationAssuranceLevel),
		"Provisioning/binding posture (authoritative): "+renderProvisioningPostureCue(summary.ProvisioningPosture),
		"Attestation posture (authoritative): "+renderAttestationPostureCue(attestationPosture, attestationReasons),
		"Verifier class (authoritative): "+renderAuthoritativeVerifierClassCue(detail.AuthoritativeState),
		"Supported runtime requirements (authoritative): "+renderSupportedRuntimeRequirementsCue(detail.AuthoritativeState),
		"Reduced-assurance posture (authoritative): "+renderReducedAssurancePostureCue(detail.AuthoritativeState),
		"Audit posture (authoritative): "+renderAuditPostureCue(summary.AuditIntegrityStatus, summary.AuditAnchoringStatus, summary.AuditCurrentlyDegraded),
		fmt.Sprintf("Approval profile (authoritative): %s", renderApprovalProfileCue(summary.ApprovalProfile)),
		fmt.Sprintf("Authoritative broker state (control-plane truth): %d keys | Advisory state (non-authoritative runner hints): %d keys %s", len(detail.AuthoritativeState), len(detail.AdvisoryState), renderAdvisoryStateCue(detail.AdvisoryState)),
		fmt.Sprintf("Coordination summary: blocked=%t wait_reason=%s mode=%s locks=%d conflicts=%d", detail.Coordination.Blocked, detail.Coordination.WaitReasonCode, detail.Coordination.CoordinationMode, detail.Coordination.LockCount, detail.Coordination.ConflictCount),
		fmt.Sprintf("Blocking cue: %s (reason=%s)", renderBlockingStateCue(detail.Coordination.Blocked, detail.Coordination.WaitReasonCode), valueOrNA(detail.Coordination.WaitReasonCode)),
		fmt.Sprintf("Stage summaries: %d total, %d with pending approvals", len(detail.StageSummaries), waitingStages),
		fmt.Sprintf("Role summaries: %d total, %d reporting coordination waits", len(detail.RoleSummaries), waitingRoles),
		fmt.Sprintf("Pending approvals=%d active manifests=%d policy refs=%d", len(detail.PendingApprovalIDs), len(detail.ActiveManifestHashes), len(detail.LatestPolicyDecisionRefs)),
	)
}

func attestationPostureFromState(state map[string]any) (string, []string) {
	posture, _ := state["attestation_posture"].(string)
	reasonsAny, ok := state["attestation_reason_codes"].([]any)
	if !ok {
		reasons, _ := state["attestation_reason_codes"].([]string)
		return posture, reasons
	}
	reasons := make([]string, 0, len(reasonsAny))
	for _, value := range reasonsAny {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			reasons = append(reasons, s)
		}
	}
	return posture, reasons
}

func runInspectorContentKind(presentation contentPresentationMode) inspectorContentKind {
	if presentation == presentationRaw {
		return inspectorContentRaw
	}
	return inspectorContentStructured
}

func runRouteCopyActions(detail *brokerapi.RunDetail) []routeCopyAction {
	if detail == nil {
		return nil
	}
	summary := detail.Summary
	raw := compactLines(
		fmt.Sprintf("run_id=%s", summary.RunID),
		fmt.Sprintf("workspace_id=%s", summary.WorkspaceID),
		fmt.Sprintf("lifecycle=%s", summary.LifecycleState),
		fmt.Sprintf("backend_kind=%s", summary.BackendKind),
	)
	return compactCopyActions([]routeCopyAction{
		{ID: "run_id", Label: "run id", Text: summary.RunID},
		{ID: "workspace_id", Label: "workspace id", Text: summary.WorkspaceID},
		{ID: "raw_block", Label: "raw block", Text: raw},
	})
}

func (m runsRouteModel) activeSummary() brokerapi.RunSummary {
	if m.active != nil {
		return m.active.Summary
	}
	if len(m.runs) > 0 {
		return m.runs[m.selected]
	}
	return brokerapi.RunSummary{}
}

func (m *runsRouteModel) syncDetailDocument() {
	if m.active == nil {
		m.detailDoc.SetDocument(workbenchObjectRef{Kind: "run", ID: "none"}, inspectorContentStructured, "run details", "")
		return
	}
	summary := m.active.Summary
	presentation := normalizePresentationMode(m.presentation)
	waitingStages := pendingApprovalStageCount(m.active.StageSummaries)
	waitingRoles := waitingRoleCount(m.active.RoleSummaries)
	content := runInspectorContent(summary, m.active, waitingStages, waitingRoles, presentation)
	kind := runInspectorContentKind(presentation)
	ref := workbenchObjectRef{Kind: "run", ID: strings.TrimSpace(summary.RunID), WorkspaceID: strings.TrimSpace(summary.WorkspaceID)}
	m.detailDoc.SetDocument(ref, kind, "run details", content)
}
