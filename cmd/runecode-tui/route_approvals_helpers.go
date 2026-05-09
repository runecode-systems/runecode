package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderApprovalList(items []brokerapi.ApprovalSummary, selected int) string {
	if len(items) == 0 {
		return "  - no approvals"
	}
	line := ""
	for i, item := range items {
		marker := " "
		if i == selected {
			marker = ">"
		}
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s %s trigger=%s", marker, item.ApprovalID, stateBadgeWithLabel("status", item.Status), item.ApprovalTriggerCode)) + "\n"
		line += fmt.Sprintf("      bound scope: action=%s run=%s stage=%s step=%s role=%s\n", item.BoundScope.ActionKind, valueOrNA(item.BoundScope.RunID), valueOrNA(item.BoundScope.StageID), valueOrNA(item.BoundScope.StepID), valueOrNA(item.BoundScope.RoleInstanceID))
	}
	return line
}

func renderApprovalDirectoryItems(items []brokerapi.ApprovalSummary) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%s %s %s %s gate=%s", valueOrNA(item.ApprovalID), approvalDisplayLabel(item), approvalPrimaryStateBadge(item), approvalQueueReason(item), approvalBoundScopeCue(item)))
	}
	return out
}

func renderApprovalInspector(resp *brokerapi.ApprovalGetResponse, presentation contentPresentationMode, document *longFormDocumentState) string {
	if resp == nil {
		return "  Select an approval and press enter to load detail."
	}
	if document == nil {
		fallback := newLongFormDocumentState()
		document = &fallback
	}
	presentation = normalizePresentationMode(presentation)
	summary := resp.Approval
	detail := resp.ApprovalDetail
	lifecycleState := detail.LifecycleDetail.LifecycleState
	lifecycleFlags := renderApprovalLifecycleFlags(detail.LifecycleDetail)
	bindingLabel := approvalBindingLabel(detail.BindingKind)
	identity := detail.BoundIdentity
	boundScope := summary.BoundScope
	content := approvalInspectorContent(summary, detail, identity, boundScope, bindingLabel, lifecycleState, lifecycleFlags, presentation)
	contentKind := approvalInspectorContentKind(presentation)
	ref := workbenchObjectRef{Kind: "approval", ID: strings.TrimSpace(summary.ApprovalID), WorkspaceID: strings.TrimSpace(boundScope.WorkspaceID)}
	document.SetDocument(ref, contentKind, "approval details", content)
	return renderInspectorShell(inspectorShellSpec{
		Title:    "Approval inspector",
		Summary:  fmt.Sprintf("approval=%s state=%s reason=%s", summary.ApprovalID, approvalDisplayState(summary, detail), approvalPrimaryReason(summary, detail)),
		Identity: fmt.Sprintf("approval=%s run=%s action=%s", summary.ApprovalID, valueOrNA(boundScope.RunID), valueOrNA(boundScope.ActionKind)),
		Status:   fmt.Sprintf("workflow_state=%s resolve=%s", workflowApprovalState(summary, detail), approvalResolveStatus(summary, detail)),
		Badges:   []string{stateBadgeWithLabel("status", summary.Status), appTheme.InspectorHint.Render("policy/trigger/system cues are distinct")},
		References: []inspectorReference{
			{Label: "run", Items: mapReferenceIDs([]string{boundScope.RunID}, func(id string) paletteActionMsg {
				return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: id}}
			})},
			{Label: "approval evidence", Items: approvalEvidenceReferenceItems(resp)},
		},
		LocalActions: approvalInspectorLocalActions(),
		CopyActions:  approvalRouteCopyActions(resp),
		ModeTabs:     []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:   string(presentation),
		Document:     document,
	})
}

func approvalInspectorLocalActions() []routeActionItem {
	return []routeActionItem{
		{Label: "resolve:typed"},
		{Label: "jump:runs", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeRuns}}},
		{Label: "jump:artifacts", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeArtifacts}}},
		{Label: "jump:audit", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}},
		{Label: "copy:approval_id"},
	}
}

func approvalInspectorReferenceActions(resp *brokerapi.ApprovalGetResponse) []routeActionItem {
	if resp == nil {
		return nil
	}
	bound := resp.Approval.BoundScope
	items := []routeActionItem{}
	for _, ref := range mapReferenceIDs([]string{bound.RunID}, func(id string) paletteActionMsg {
		return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: id}}
	}) {
		items = append(items, routeActionItem{Label: "run:" + ref.Label, Action: ref.Action})
	}
	for _, ref := range approvalEvidenceReferenceItems(resp) {
		items = append(items, routeActionItem{Label: "evidence:" + ref.Label, Action: ref.Action})
	}
	return items
}

func approvalEvidenceReferenceItems(resp *brokerapi.ApprovalGetResponse) []inspectorReferenceItem {
	if resp == nil {
		return nil
	}
	detail := resp.ApprovalDetail
	items := make([]inspectorReferenceItem, 0, 6)
	appendArtifact := func(label, digest string) {
		digest = strings.TrimSpace(digest)
		if digest == "" {
			return
		}
		items = append(items, inspectorReferenceItem{Label: label + " " + shortIdentity(digest), Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: digest}}})
	}
	appendArtifact("summary", detail.BoundIdentity.SummaryPreviewDigest)
	appendArtifact("diff", detail.BoundIdentity.DiffDigest)
	appendArtifact("artifact_set", detail.BoundIdentity.ArtifactSetDigest)
	if policy := strings.TrimSpace(detail.BoundIdentity.PolicyDecisionHash); policy != "" {
		items = append(items, inspectorReferenceItem{Label: "policy " + shortIdentity(policy), Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}})
	}
	if manifest := strings.TrimSpace(detail.BoundIdentity.ManifestHash); manifest != "" {
		items = append(items, inspectorReferenceItem{Label: "manifest " + shortIdentity(manifest), Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}})
	}
	return items
}

func approvalBindingLabel(bindingKind string) string {
	if bindingKind == "exact_action" {
		return "exact-action approval"
	}
	return "stage-sign-off approval"
}

func approvalInspectorContent(summary brokerapi.ApprovalSummary, detail brokerapi.ApprovalDetail, identity brokerapi.ApprovalBoundIdentity, boundScope brokerapi.ApprovalBoundScope, bindingLabel string, lifecycleState string, lifecycleFlags string, presentation contentPresentationMode) string {
	if presentation == presentationStructured {
		return compactLines(
			fmt.Sprintf("structured approval: id=%s", approvalUIValue(summary.ApprovalID)),
			fmt.Sprintf("state: workflow=%s resolve=%s lifecycle_flags=%s", approvalUIValue(workflowApprovalState(summary, detail)), approvalUIValue(approvalResolveStatus(summary, detail)), approvalUIValue(lifecycleFlags)),
			fmt.Sprintf("review_first=%s next_route=%s", approvalUIValue(approvalReviewFirst(summary, detail)), approvalUIValue(approvalFollowUpRoute(summary))),
			fmt.Sprintf("identity: request=%s decision=%s", approvalUIValue(identity.ApprovalRequestDigest), approvalUIValue(identity.ApprovalDecisionDigest)),
		)
	}
	if presentation == presentationRaw {
		return compactLines(
			fmt.Sprintf("raw approval_id=%s status=%s trigger=%s", approvalUIValue(summary.ApprovalID), approvalUIValue(summary.Status), approvalUIValue(summary.ApprovalTriggerCode)),
			fmt.Sprintf("raw bound_scope workspace=%s run=%s stage=%s action=%s", approvalUIValue(boundScope.WorkspaceID), approvalUIValue(boundScope.RunID), approvalUIValue(boundScope.StageID), approvalUIValue(boundScope.ActionKind)),
			fmt.Sprintf("raw lifecycle state=%s reason=%s stale=%t", approvalUIValue(detail.LifecycleDetail.LifecycleState), approvalUIValue(detail.LifecycleDetail.LifecycleReasonCode), detail.LifecycleDetail.Stale),
		)
	}
	return compactLines(
		fmt.Sprintf("Approval state: %s %s", approvalUIValue(approvalDisplayState(summary, detail)), approvalPrimaryStateBadge(summary)),
		fmt.Sprintf("Why this approval exists: %s", approvalUIValue(approvalPrimaryReason(summary, detail))),
		fmt.Sprintf("Exact gated object/action: %s", approvalUIValue(approvalExactObjectAction(summary, detail))),
		fmt.Sprintf("Review first: %s", approvalUIValue(approvalReviewFirst(summary, detail))),
		fmt.Sprintf("If approved next: %s", approvalUIValue(approvalEffectSummary(detail))),
		fmt.Sprintf("After approval route: %s", approvalUIValue(approvalFollowUpRoute(summary))),
		fmt.Sprintf("Resolve availability: %s", approvalUIValue(approvalResolveSummary(summary, detail))),
		fmt.Sprintf("Resolve unavailable because: %s", approvalUIValue(approvalResolveBlockedReason(summary, detail))),
		fmt.Sprintf("Workflow posture: %s; lifecycle=%s (%s)", approvalUIValue(workflowApprovalState(summary, detail)), approvalUIValue(lifecycleState), approvalUIValue(lifecycleFlags)),
		"Safety cue: approval policy, lifecycle, and execution/system errors remain distinct before any decision is accepted.",
		muted(fmt.Sprintf("Structured/raw modes expose %s, binding kind, trigger codes, policy reason codes, bound scope, request/decision digests, manifest, and policy decision hash.", approvalUIValue(bindingLabel))),
	)
}

func approvalInspectorContentKind(presentation contentPresentationMode) inspectorContentKind {
	if presentation == presentationRaw {
		return inspectorContentRaw
	}
	return inspectorContentStructured
}

func approvalRouteCopyActions(resp *brokerapi.ApprovalGetResponse) []routeCopyAction {
	if resp == nil {
		return nil
	}
	summary := resp.Approval
	bound := summary.BoundScope
	raw := compactLines(
		fmt.Sprintf("approval_id=%s", approvalUIValue(summary.ApprovalID)),
		fmt.Sprintf("status=%s", approvalUIValue(summary.Status)),
		fmt.Sprintf("trigger=%s", approvalUIValue(summary.ApprovalTriggerCode)),
		fmt.Sprintf("run_id=%s", approvalUIValue(bound.RunID)),
		fmt.Sprintf("stage_id=%s", approvalUIValue(bound.StageID)),
		fmt.Sprintf("action_kind=%s", approvalUIValue(bound.ActionKind)),
	)
	return compactCopyActions([]routeCopyAction{
		{ID: "approval_id", Label: "approval id", Text: approvalUIValue(summary.ApprovalID)},
		{ID: "run_id", Label: "bound run id", Text: approvalUIValue(bound.RunID)},
		{ID: "request_digest", Label: "request digest", Text: approvalUIValue(resp.ApprovalDetail.BoundIdentity.ApprovalRequestDigest)},
		{ID: "decision_digest", Label: "decision digest", Text: approvalUIValue(resp.ApprovalDetail.BoundIdentity.ApprovalDecisionDigest)},
		{ID: "raw_block", Label: "raw block", Text: sanitizeUIText(raw)},
	})
}

func approvalUIValue(value string) string {
	return valueOrNA(sanitizeUIText(value))
}

func renderApprovalLifecycleFlags(detail brokerapi.ApprovalLifecycleDetail) string {
	flags := []string{}
	if detail.Stale {
		flags = append(flags, "stale")
	}
	if detail.SupersededByApprovalID != "" {
		flags = append(flags, "superseded")
	}
	switch detail.LifecycleState {
	case "expired":
		flags = append(flags, "expired")
	case "consumed":
		flags = append(flags, "consumed")
	case "approved":
		flags = append(flags, "approved")
	case "denied":
		flags = append(flags, "denied")
	}
	if len(flags) == 0 {
		return "active"
	}
	return joinCSV(flags)
}

func valueOrNA(value string) string {
	if value == "" {
		return "n/a"
	}
	return value
}

func joinCSV(items []string) string {
	line := ""
	for i, item := range items {
		if i > 0 {
			line += ","
		}
		line += item
	}
	return line
}

func renderApprovalSafetyStrip(resp *brokerapi.ApprovalGetResponse) string {
	if resp == nil {
		return tableHeader("Approval safety strip") + " " + neutralBadge("NO_ACTIVE_APPROVAL")
	}
	s := resp.Approval
	d := resp.ApprovalDetail
	stateCue := renderBlockingStateCue(true, d.PolicyReasonCode)
	triggerCue := renderBlockingStateCue(true, s.ApprovalTriggerCode)
	return compactLines(
		tableHeader("Approval posture")+" "+approvalPrimaryStateBadge(s)+" "+approvalSupportBadge(s, d),
		fmt.Sprintf("Needs attention because %s", approvalPrimaryReason(s, d)),
		fmt.Sprintf("Broker cues: policy_reason_code=%s %s | approval_trigger_code=%s %s", approvalUIValue(d.PolicyReasonCode), stateCue, approvalUIValue(s.ApprovalTriggerCode), triggerCue),
	)
}

func renderApprovalFlowPath(resp *brokerapi.ApprovalGetResponse) string {
	if resp == nil {
		return "Evidence path: run waits for approval -> review evidence -> resolve where supported -> blocked workflow continues when broker state advances"
	}
	s := resp.Approval
	return fmt.Sprintf("Evidence path: %s -> artifacts (%s) -> audit (%s) -> verification posture (Audit) -> anchor/export actions where available", approvalUIValue(approvalDisplayLabel(s)), approvalUIValue(approvalReviewFirst(s, resp.ApprovalDetail)), approvalUIValue(approvalAuditLinkSummary(s, resp.ApprovalDetail)))
}

func renderApprovalOverviewCard(resp *brokerapi.ApprovalGetResponse) string {
	if resp == nil {
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateWaiting, Title: "Approval review", Message: "Select an approval to see the blocked workflow, review order, and next step.", Reason: "No active approval detail is loaded yet.", NextAction: "Move through the queue and press enter to inspect the broker-owned approval detail.", ShortcutCue: "enter", RouteCue: "Approvals"})
	}
	summary := resp.Approval
	detail := resp.ApprovalDetail
	state := routeLoadStateReady
	switch workflowApprovalState(summary, detail) {
	case "approval required", "pending":
		state = routeLoadStateApprovalRequired
	case "expired", "unsupported":
		state = routeLoadStateBlocked
	case "resolved":
		state = routeLoadStateCompleted
	}
	return renderStateCardSpec(stateCardSpec{
		State:       state,
		Title:       "Approval review",
		Message:     fmt.Sprintf("%s needs attention for %s.", approvalDisplayLabel(summary), approvalDisplayState(summary, detail)),
		Reason:      approvalPrimaryReason(summary, detail),
		NextAction:  fmt.Sprintf("Review %s first, then %s.", approvalReviewFirst(summary, detail), approvalNextAction(summary, detail)),
		ShortcutCue: "enter / a",
		RouteCue:    approvalFollowUpRoute(summary),
	})
}

func renderApprovalReviewPlan(resp *brokerapi.ApprovalGetResponse) string {
	if resp == nil {
		return "Review plan: load an approval to see what is blocked, what to review first, and where to continue the evidence trail."
	}
	summary := resp.Approval
	detail := resp.ApprovalDetail
	return compactLines(
		tableHeader("Review plan"),
		fmt.Sprintf("1. Inspect %s.", approvalReviewFirst(summary, detail)),
		fmt.Sprintf("2. Confirm gated scope: %s.", approvalExactObjectAction(summary, detail)),
		fmt.Sprintf("3. Continue to %s for the audit + verification trail.", approvalFollowUpRoute(summary)),
		fmt.Sprintf("4. Resolve only when broker validation allows it; current status=%s.", approvalResolveStatus(summary, detail)),
	)
}
