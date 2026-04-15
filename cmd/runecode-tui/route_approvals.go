package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type approvalsLoadedMsg struct {
	items  []brokerapi.ApprovalSummary
	detail *brokerapi.ApprovalGetResponse
	err    error
	seq    uint64
}

type approvalResolvedMsg struct {
	approvalID string
	result     string
	err        error
}

type approvalsRouteModel struct {
	def         routeDefinition
	client      localBrokerClient
	loading     bool
	resolving   bool
	errText     string
	statusText  string
	items       []brokerapi.ApprovalSummary
	selected    int
	active      *brokerapi.ApprovalGetResponse
	inspectorOn bool
	loadSeq     uint64
}

func newApprovalsRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return approvalsRouteModel{def: def, client: client, inspectorOn: true}
}

func (m approvalsRouteModel) ID() routeID { return m.def.ID }

func (m approvalsRouteModel) Title() string { return m.def.Label }

func (m approvalsRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		return m.reload()
	case tea.KeyMsg:
		return m.handleKey(typed)
	case approvalsLoadedMsg:
		if typed.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		if typed.err != nil {
			m.errText = safeUIErrorText(typed.err)
			return m, nil
		}
		m.errText = ""
		m.items = typed.items
		if m.selected >= len(m.items) {
			m.selected = 0
		}
		m.active = typed.detail
		return m, nil
	case approvalResolvedMsg:
		m.resolving = false
		if typed.err != nil {
			m.errText = safeUIErrorText(typed.err)
			m.statusText = ""
			return m, nil
		}
		m.errText = ""
		m.statusText = fmt.Sprintf("Approval %s resolved via typed ApprovalResolve (%s).", typed.approvalID, valueOrNA(typed.result))
		m.loading = true
		m.loadSeq++
		return m, m.loadCmd("", m.loadSeq)
	default:
		return m, nil
	}
}

func (m approvalsRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return "Loading approvals from broker approval contracts..."
	}
	if m.resolving {
		return "Resolving approval via typed broker ApprovalResolve..."
	}
	if m.errText != "" {
		return compactLines("Approvals", "Load failed: "+m.errText, "Press r to retry.")
	}
	body := []string{
		sectionTitle("Approvals") + " " + focusBadge(focus),
		renderApprovalSafetyStrip(m.active),
		renderApprovalFlowPath(m.active),
		tableHeader("Approval queue"),
		renderApprovalList(m.items, m.selected),
	}
	if m.statusText != "" {
		body = append(body, "Status: "+m.statusText)
	}
	if m.inspectorOn {
		body = append(body, tableHeader("Inspector")+" "+appTheme.InspectorHint.Render("(policy/trigger/system cues are distinct)"))
		body = append(body, renderApprovalInspector(m.active))
	}
	body = append(body, keyHint("Route keys: j/k move, enter load detail, a resolve current approval where supported, i toggle inspector, r reload"))
	return compactLines(body...)
}

func (m approvalsRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	switch key.String() {
	case "r":
		return m.reload()
	case "a":
		return m.handleResolveKey()
	case "i":
		m.inspectorOn = !m.inspectorOn
		return m, nil
	case "j", "down":
		if len(m.items) == 0 {
			return m, nil
		}
		m.selected = (m.selected + 1) % len(m.items)
		return m, nil
	case "k", "up":
		if len(m.items) == 0 {
			return m, nil
		}
		m.selected--
		if m.selected < 0 {
			m.selected = len(m.items) - 1
		}
		return m, nil
	case "enter":
		if len(m.items) == 0 {
			return m, nil
		}
		m.loading = true
		m.errText = ""
		m.statusText = ""
		m.loadSeq++
		return m, m.loadCmd(m.items[m.selected].ApprovalID, m.loadSeq)
	default:
		return m, nil
	}
}

func (m approvalsRouteModel) handleResolveKey() (routeModel, tea.Cmd) {
	if m.active == nil {
		m.statusText = "Load an approval detail first (enter), then press a to resolve."
		return m, nil
	}
	if err := validateApprovalResolveInput(*m.active); err != nil {
		m.resolving = false
		m.errText = ""
		m.statusText = safeUIErrorText(err)
		return m, nil
	}
	m.resolving = true
	m.errText = ""
	m.statusText = ""
	return m, m.resolveCmd(*m.active)
}

func (m approvalsRouteModel) reload() (routeModel, tea.Cmd) {
	m.loading = true
	m.errText = ""
	m.statusText = ""
	m.loadSeq++
	target := ""
	if m.selected >= 0 && m.selected < len(m.items) {
		target = m.items[m.selected].ApprovalID
	}
	return m, m.loadCmd(target, m.loadSeq)
}

func (m approvalsRouteModel) resolveCmd(resp brokerapi.ApprovalGetResponse) tea.Cmd {
	return func() tea.Msg {
		resolveReq, err := approvalResolveRequestFromDetail(resp)
		if err != nil {
			return approvalResolvedMsg{approvalID: strings.TrimSpace(resp.Approval.ApprovalID), err: err}
		}
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resolveResp, err := m.client.ApprovalResolve(ctx, resolveReq)
		if err != nil {
			return approvalResolvedMsg{approvalID: strings.TrimSpace(resp.Approval.ApprovalID), err: err}
		}
		result := strings.TrimSpace(resolveResp.ResolutionReasonCode)
		if result == "" {
			result = strings.TrimSpace(resolveResp.ResolutionStatus)
		}
		if result == "" {
			result = "resolved"
		}
		return approvalResolvedMsg{approvalID: strings.TrimSpace(resp.Approval.ApprovalID), result: result}
	}
}

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

func (m approvalsRouteModel) loadCmd(approvalID string, seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		listResp, err := m.client.ApprovalList(ctx, 30)
		if err != nil {
			return approvalsLoadedMsg{err: err, seq: seq}
		}
		target := approvalID
		if target == "" && len(listResp.Approvals) > 0 {
			target = listResp.Approvals[0].ApprovalID
		}
		if target == "" {
			return approvalsLoadedMsg{items: listResp.Approvals, seq: seq}
		}
		getResp, err := m.client.ApprovalGet(ctx, target)
		if err != nil {
			return approvalsLoadedMsg{err: err, seq: seq}
		}
		return approvalsLoadedMsg{items: listResp.Approvals, detail: &getResp, seq: seq}
	}
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

func renderApprovalInspector(resp *brokerapi.ApprovalGetResponse) string {
	if resp == nil {
		return "  Select an approval and press enter to load detail."
	}
	summary := resp.Approval
	detail := resp.ApprovalDetail
	lifecycleState := detail.LifecycleDetail.LifecycleState
	lifecycleFlags := renderApprovalLifecycleFlags(detail.LifecycleDetail)
	bindingLabel := ""
	if detail.BindingKind == "exact_action" {
		bindingLabel = "exact-action approval"
	} else {
		bindingLabel = "stage-sign-off approval"
	}
	identity := detail.BoundIdentity
	boundScope := summary.BoundScope
	return compactLines(
		fmt.Sprintf("  Approval type: %s (binding_kind=%s) %s", bindingLabel, detail.BindingKind, infoBadge("type cue")),
		fmt.Sprintf("  Lifecycle state: %s (%s) %s", lifecycleState, lifecycleFlags, postureBadge(lifecycleState)),
		fmt.Sprintf("  Lifecycle reason code: %s", detail.LifecycleDetail.LifecycleReasonCode),
		fmt.Sprintf("  Policy reason code: %s %s", detail.PolicyReasonCode, warnBadge("policy cue")),
		fmt.Sprintf("  Approval trigger code: %s %s", summary.ApprovalTriggerCode, infoBadge("trigger cue")),
		fmt.Sprintf("  Distinct blocking semantics: trigger=%s cue=%s", summary.ApprovalTriggerCode, renderBlockingStateCue(true, summary.ApprovalTriggerCode)),
		"  Execution/system errors: shown as load failures above; not merged with policy/trigger codes. "+dangerBadge("system cue"),
		fmt.Sprintf("  What changes if approved: effect=%s summary=%s", detail.WhatChangesIfApproved.EffectKind, detail.WhatChangesIfApproved.Summary),
		fmt.Sprintf("  Blocked work scope: kind=%s action=%s run=%s stage=%s step=%s role=%s", detail.BlockedWorkScope.ScopeKind, detail.BlockedWorkScope.ActionKind, valueOrNA(detail.BlockedWorkScope.RunID), valueOrNA(detail.BlockedWorkScope.StageID), valueOrNA(detail.BlockedWorkScope.StepID), valueOrNA(detail.BlockedWorkScope.RoleInstanceID)),
		fmt.Sprintf("  Canonical bound identity: request=%s decision=%s manifest=%s policy_decision=%s", valueOrNA(identity.ApprovalRequestDigest), valueOrNA(identity.ApprovalDecisionDigest), valueOrNA(identity.ManifestHash), valueOrNA(identity.PolicyDecisionHash)),
		fmt.Sprintf("  Exact bound scope: workspace=%s run=%s stage=%s step=%s role=%s action=%s", valueOrNA(boundScope.WorkspaceID), valueOrNA(boundScope.RunID), valueOrNA(boundScope.StageID), valueOrNA(boundScope.StepID), valueOrNA(boundScope.RoleInstanceID), valueOrNA(boundScope.ActionKind)),
	)
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
