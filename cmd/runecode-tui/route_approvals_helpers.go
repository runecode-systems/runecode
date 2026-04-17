package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func validateApprovalResolveInput(resp brokerapi.ApprovalGetResponse) error {
	approvalID := strings.TrimSpace(resp.Approval.ApprovalID)
	if approvalID == "" {
		return fmt.Errorf("approval detail missing approval_id")
	}
	if resp.SignedApprovalRequest == nil || resp.SignedApprovalDecision == nil {
		return fmt.Errorf("approval resolve requires signed approval request and decision envelopes")
	}
	boundScope := resp.Approval.BoundScope
	actionKind := strings.TrimSpace(boundScope.ActionKind)
	switch actionKind {
	case "backend_posture_change":
		selection := resp.ApprovalDetail.BackendPostureSelection
		if selection == nil {
			return fmt.Errorf("approval resolve requires typed backend posture selection detail")
		}
		if strings.TrimSpace(selection.TargetInstanceID) == "" || strings.TrimSpace(selection.TargetBackendKind) == "" {
			return fmt.Errorf("approval resolve requires backend posture target instance and backend kind")
		}
	case "promotion":
		return fmt.Errorf("promotion approvals must be resolved via promote-excerpt to preserve exact promotion binding")
	default:
		return fmt.Errorf("approval resolve does not support this action kind")
	}
	return nil
}

func approvalResolveRequestFromDetail(resp brokerapi.ApprovalGetResponse) (brokerapi.ApprovalResolveRequest, error) {
	if err := validateApprovalResolveInput(resp); err != nil {
		return brokerapi.ApprovalResolveRequest{}, err
	}
	summary := resp.Approval
	boundScope := summary.BoundScope
	if strings.TrimSpace(boundScope.SchemaID) == "" {
		boundScope.SchemaID = "runecode.protocol.v0.ApprovalBoundScope"
	}
	if strings.TrimSpace(boundScope.SchemaVersion) == "" {
		boundScope.SchemaVersion = "0.1.0"
	}
	resolveReq := brokerapi.ApprovalResolveRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion: localAPISchemaVersion,
		ApprovalID:    strings.TrimSpace(summary.ApprovalID),
		BoundScope:    boundScope,
		ResolutionDetails: brokerapi.ApprovalResolveDetails{
			SchemaID:      "runecode.protocol.v0.ApprovalResolveDetails",
			SchemaVersion: "0.1.0",
		},
		SignedApprovalRequest:  *resp.SignedApprovalRequest,
		SignedApprovalDecision: *resp.SignedApprovalDecision,
	}
	if strings.TrimSpace(boundScope.ActionKind) == "backend_posture_change" && resp.ApprovalDetail.BackendPostureSelection != nil {
		selection := resp.ApprovalDetail.BackendPostureSelection
		resolveReq.ResolutionDetails.BackendPostureSelection = &brokerapi.ApprovalResolveBackendPostureSelectionDetail{
			SchemaID:          "runecode.protocol.v0.ApprovalResolveBackendPostureSelectionDetail",
			SchemaVersion:     "0.1.0",
			TargetInstanceID:  strings.TrimSpace(selection.TargetInstanceID),
			TargetBackendKind: strings.TrimSpace(selection.TargetBackendKind),
		}
	}
	return resolveReq, nil
}

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
		out = append(out, fmt.Sprintf("%s %s trigger=%s", item.ApprovalID, stateBadgeWithLabel("status", item.Status), item.ApprovalTriggerCode))
	}
	return out
}

func renderApprovalInspector(resp *brokerapi.ApprovalGetResponse, presentation contentPresentationMode) string {
	if resp == nil {
		return "  Select an approval and press enter to load detail."
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
	return renderInspectorShell(inspectorShellSpec{
		Title:          "Approval inspector",
		Summary:        fmt.Sprintf("approval=%s status=%s trigger=%s", summary.ApprovalID, valueOrNA(summary.Status), valueOrNA(summary.ApprovalTriggerCode)),
		Identity:       fmt.Sprintf("approval=%s run=%s", summary.ApprovalID, valueOrNA(boundScope.RunID)),
		Status:         fmt.Sprintf("lifecycle=%s policy_reason=%s", lifecycleState, valueOrNA(detail.PolicyReasonCode)),
		Badges:         []string{stateBadgeWithLabel("status", summary.Status), appTheme.InspectorHint.Render("policy/trigger/system cues are distinct")},
		References:     []inspectorReference{{Label: "run", Items: []string{boundScope.RunID}}, {Label: "stage", Items: []string{boundScope.StageID}}},
		LocalActions:   []string{"resolve:typed", "jump:runs", "jump:audit", "copy:approval_id"},
		CopyActions:    approvalRouteCopyActions(resp),
		ModeTabs:       []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:     string(presentation),
		ContentKind:    contentKind,
		ContentLabel:   "approval details",
		Content:        content,
		ViewportWidth:  96,
		ViewportHeight: 14,
	})
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
			fmt.Sprintf("structured approval: id=%s", summary.ApprovalID),
			fmt.Sprintf("counts: lifecycle_flags=%s links=identity+scope", lifecycleFlags),
		)
	}
	if presentation == presentationRaw {
		return compactLines(
			fmt.Sprintf("raw approval_id=%s status=%s trigger=%s", summary.ApprovalID, summary.Status, summary.ApprovalTriggerCode),
			fmt.Sprintf("raw bound_scope workspace=%s run=%s stage=%s action=%s", valueOrNA(boundScope.WorkspaceID), valueOrNA(boundScope.RunID), valueOrNA(boundScope.StageID), valueOrNA(boundScope.ActionKind)),
			fmt.Sprintf("raw lifecycle state=%s reason=%s stale=%t", detail.LifecycleDetail.LifecycleState, valueOrNA(detail.LifecycleDetail.LifecycleReasonCode), detail.LifecycleDetail.Stale),
		)
	}
	return compactLines(
		fmt.Sprintf("Approval type: %s (binding_kind=%s) %s", bindingLabel, detail.BindingKind, infoBadge("type cue")),
		fmt.Sprintf("Lifecycle state: %s (%s) %s", lifecycleState, lifecycleFlags, postureBadge(lifecycleState)),
		fmt.Sprintf("Lifecycle reason code: %s", detail.LifecycleDetail.LifecycleReasonCode),
		fmt.Sprintf("Policy reason code: %s %s", detail.PolicyReasonCode, warnBadge("policy cue")),
		fmt.Sprintf("Approval trigger code: %s %s", summary.ApprovalTriggerCode, infoBadge("trigger cue")),
		fmt.Sprintf("Distinct blocking semantics: trigger=%s cue=%s", summary.ApprovalTriggerCode, renderBlockingStateCue(true, summary.ApprovalTriggerCode)),
		"Execution/system errors: shown as load failures above; not merged with policy/trigger codes. "+dangerBadge("system cue"),
		fmt.Sprintf("What changes if approved: effect=%s summary=%s", detail.WhatChangesIfApproved.EffectKind, detail.WhatChangesIfApproved.Summary),
		fmt.Sprintf("Blocked work scope: kind=%s action=%s run=%s stage=%s step=%s role=%s", detail.BlockedWorkScope.ScopeKind, detail.BlockedWorkScope.ActionKind, valueOrNA(detail.BlockedWorkScope.RunID), valueOrNA(detail.BlockedWorkScope.StageID), valueOrNA(detail.BlockedWorkScope.StepID), valueOrNA(detail.BlockedWorkScope.RoleInstanceID)),
		fmt.Sprintf("Canonical bound identity: request=%s decision=%s manifest=%s policy_decision=%s", valueOrNA(identity.ApprovalRequestDigest), valueOrNA(identity.ApprovalDecisionDigest), valueOrNA(identity.ManifestHash), valueOrNA(identity.PolicyDecisionHash)),
		fmt.Sprintf("Exact bound scope: workspace=%s run=%s stage=%s step=%s role=%s action=%s", valueOrNA(boundScope.WorkspaceID), valueOrNA(boundScope.RunID), valueOrNA(boundScope.StageID), valueOrNA(boundScope.StepID), valueOrNA(boundScope.RoleInstanceID), valueOrNA(boundScope.ActionKind)),
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
		fmt.Sprintf("approval_id=%s", summary.ApprovalID),
		fmt.Sprintf("status=%s", summary.Status),
		fmt.Sprintf("trigger=%s", summary.ApprovalTriggerCode),
		fmt.Sprintf("run_id=%s", bound.RunID),
		fmt.Sprintf("stage_id=%s", bound.StageID),
		fmt.Sprintf("action_kind=%s", bound.ActionKind),
	)
	return compactCopyActions([]routeCopyAction{
		{ID: "approval_id", Label: "approval id", Text: summary.ApprovalID},
		{ID: "run_id", Label: "bound run id", Text: bound.RunID},
		{ID: "raw_block", Label: "raw block", Text: raw},
	})
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
		tableHeader("Approval safety strip")+" "+approvalRequiredBadge("APPROVAL_REQUIRED")+" profile cues remain explicit",
		fmt.Sprintf("status=%s %s | policy_reason_code=%s %s | approval_trigger_code=%s %s", s.Status, postureBadge(s.Status), valueOrNA(d.PolicyReasonCode), stateCue, valueOrNA(s.ApprovalTriggerCode), triggerCue),
	)
}

func renderApprovalFlowPath(resp *brokerapi.ApprovalGetResponse) string {
	if resp == nil {
		return "Flow path: run -> approval -> typed resolve (shared exact-action path) -> run resumes (load an approval detail to inspect)"
	}
	s := resp.Approval
	workspace := valueOrNA(s.BoundScope.WorkspaceID)
	run := valueOrNA(s.BoundScope.RunID)
	stage := valueOrNA(s.BoundScope.StageID)
	action := valueOrNA(s.BoundScope.ActionKind)
	return fmt.Sprintf("Flow path: workspace=%s run=%s stage=%s action=%s -> approval=%s -> typed approval_resolve -> resume signal", workspace, run, stage, action, s.ApprovalID)
}
