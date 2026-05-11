//go:build runecode_tui_snapshot

package main

import (
	"strings"
	"time"

	"github.com/runecode-systems/runecode/internal/brokerapi"
	"github.com/runecode-systems/runecode/internal/trustpolicy"
)

type snapshotScenarioState struct {
	Name       string
	RouteID    routeID
	Surface    routeModel
	Sessions   []brokerapi.SessionSummary
	Watch      shellWatchTransportLoadedMsg
	Theme      themePreset
	Focus      focusArea
	Toast      string
	Viewport   snapshotViewportPreset
	Width      int
	Height     int
	Prepare    func(*shellModel)
	PostRender func(*shellModel)
}

func resolveSnapshotScenarios(name string) ([]snapshotScenarioState, error) {
	scenarios := allSnapshotScenarios()
	switch strings.TrimSpace(name) {
	case "", "all":
		return scenarios, nil
	case "dashboard":
		return dashboardSnapshotScenarios(), nil
	case "action-center":
		return []snapshotScenarioState{buildActionCenterSnapshot()}, nil
	default:
		for _, scenario := range scenarios {
			if scenario.Name == name {
				return []snapshotScenarioState{scenario}, nil
			}
		}
		return nil, usageErrorf("unknown snapshot scenario %q", name)
	}
}

func resolveSnapshotBundleScenarios(name string) ([]snapshotScenarioState, error) {
	bundle, err := resolveSnapshotBundle(name)
	if err != nil {
		return nil, err
	}
	out := make([]snapshotScenarioState, 0, len(bundle.Scenarios))
	for _, name := range bundle.Scenarios {
		scenarios, err := resolveSnapshotScenarios(name)
		if err != nil {
			return nil, err
		}
		out = append(out, scenarios...)
	}
	return out, nil
}

func renderSnapshotScenario(state snapshotScenarioState, cfg tuiSnapshotConfig) (string, routeID, error) {
	snapshotRenderMu.Lock()
	defer snapshotRenderMu.Unlock()

	return withSnapshotColorProfile(func() (string, routeID, error) {
		forceMemoryWorkbenchState = true
		defer func() { forceMemoryWorkbenchState = false }()
		theme := cfg.theme
		if theme == "" {
			theme = state.Theme
		}
		width, height := snapshotScenarioDimensions(state, cfg)
		appTheme = newTheme(theme)
		m := newShellModel()
		m.width = width
		m.height = height
		m.themePreset = theme
		m.preferredMode = presentationRendered
		m.location.Primary = shellObjectLocation{RouteID: state.RouteID, Object: workbenchObjectRef{Kind: "route", ID: string(state.RouteID)}}
		m.nav.SelectByRouteID(state.RouteID)
		m.routeModels[state.RouteID] = state.Surface
		m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: append([]brokerapi.SessionSummary(nil), state.Sessions...)})
		m.applyWatchTransport(state.Watch)
		m.publishWatchStateToRoutes()
		if strings.TrimSpace(state.Toast) != "" {
			m.toasts.Push(toastInfo, state.Toast)
		}
		if state.Prepare != nil {
			state.Prepare(&m)
		}
		m.setFocus(state.Focus)
		m.syncSidebarCursorToLocation()
		if state.PostRender != nil {
			state.PostRender(&m)
		}
		return m.View(), state.RouteID, nil
	})
}

func snapshotScenarioViewport(state snapshotScenarioState, cfg tuiSnapshotConfig) snapshotViewportPreset {
	if state.Viewport != "" {
		return state.Viewport
	}
	return cfg.viewport
}

func snapshotScenarioDimensions(state snapshotScenarioState, cfg tuiSnapshotConfig) (int, int) {
	width := cfg.width
	height := cfg.height
	if state.Viewport != "" {
		viewport, err := resolveSnapshotViewportPreset(state.Viewport)
		if err == nil {
			if cfg.widthFromViewport {
				width = viewport.Width
			}
			if cfg.heightFromViewport {
				height = viewport.Height
			}
		}
	}
	if state.Width > 0 {
		width = state.Width
	}
	if state.Height > 0 {
		height = state.Height
	}
	return width, height
}

func defaultSnapshotSessions() []brokerapi.SessionSummary {
	return []brokerapi.SessionSummary{
		{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Draft change proposal", LinkedRunCount: 1, LinkedApprovalCount: 1, TurnCount: 2},
		{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "waiting", LastActivityPreview: "Review blocked work", LinkedRunCount: 1, LinkedApprovalCount: 0, TurnCount: 3},
	}
}

func buildHealthyEmptySnapshot() snapshotScenarioState {
	watch := snapshotWatchHealthyEmpty()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: true, RecoveryComplete: true},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project: brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{
				SchemaID:               "runecode.protocol.v0.ProjectSubstratePostureSummary",
				ValidationState:        "valid",
				CompatibilityPosture:   "supported",
				NormalOperationAllowed: true,
			}},
			live:  watchProjection(watch).Live,
			audit: brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "ok", CurrentlyDegraded: false}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-healthy-empty", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildApprovalWaitingSnapshot() snapshotScenarioState {
	watch := snapshotWatchApprovalWaiting()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: true, RecoveryComplete: true},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported_with_upgrade_available", NormalOperationAllowed: true}},
			runs:      []brokerapi.RunSummary{{RunID: "run-1", WorkspaceID: "ws-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", PendingApprovalCount: 1, ProvisioningPosture: "attested", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok"}},
			approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-1", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-1", StageID: "stage-2", ActionKind: "promotion"}}},
			live:      watchProjection(watch).Live,
			audit:     brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "ok", CurrentlyDegraded: false}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-approval-waiting", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildBlockedSnapshot() snapshotScenarioState {
	watch := snapshotWatchBlocked()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: true, RecoveryComplete: true},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported", NormalOperationAllowed: true}},
			runs:      []brokerapi.RunSummary{{RunID: "run-blocked", WorkspaceID: "ws-1", LifecycleState: "blocked", BlockingReasonCode: "approval_wait", PendingApprovalCount: 1, BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", ProvisioningPosture: "attested", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok"}},
			approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-blocked", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-blocked", StageID: "stage-1", ActionKind: "promotion"}}},
			live:      watchProjection(watch).Live,
			audit:     brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "ok", CurrentlyDegraded: false}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-blocked", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildDegradedSnapshot() snapshotScenarioState {
	watch := snapshotWatchDegraded()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: false, RecoveryComplete: false},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported_with_upgrade_available", NormalOperationAllowed: true}},
			runs:      []brokerapi.RunSummary{{RunID: "run-degraded", WorkspaceID: "ws-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "unavailable", RuntimePostureDegraded: true, ProvisioningPosture: "tofu", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "degraded", AuditCurrentlyDegraded: true}},
			live:      watchProjection(watch).Live,
			audit:     brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "degraded", CurrentlyDegraded: true, FindingCount: 2}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-degraded", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildToastSnapshot() snapshotScenarioState {
	watch := snapshotWatchBlocked()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: true, RecoveryComplete: true},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported", NormalOperationAllowed: true}},
			runs:      []brokerapi.RunSummary{{RunID: "run-blocked", WorkspaceID: "ws-1", LifecycleState: "blocked", BlockingReasonCode: "approval_wait", PendingApprovalCount: 1, BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", ProvisioningPosture: "attested", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok"}},
			approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-blocked", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-blocked", StageID: "stage-1", ActionKind: "promotion"}}},
			live:      watchProjection(watch).Live,
			audit:     brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "ok", CurrentlyDegraded: false}},
		},
	}
	return snapshotScenarioState{Name: "dashboard-toast-info", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent, Toast: "Sidebar visibility changed."}
}

func buildCommandModeDraftSnapshot() snapshotScenarioState {
	return snapshotScenarioState{Name: "command-mode-draft", RouteID: routeDashboard, Surface: buildBlockedSnapshot().Surface, Sessions: defaultSnapshotSessions(), Watch: snapshotWatchBlocked(), Theme: themePresetDark, Focus: focusContent, Prepare: func(m *shellModel) {
		m.commandMode = m.commandMode.Open().Append("open status")
	}}
}

func buildCommandModeErrorSnapshot() snapshotScenarioState {
	return snapshotScenarioState{Name: "command-mode-error", RouteID: routeDashboard, Surface: buildBlockedSnapshot().Surface, Sessions: defaultSnapshotSessions(), Watch: snapshotWatchBlocked(), Theme: themePresetDark, Focus: focusContent, Prepare: func(m *shellModel) {
		m.commandMode = m.commandMode.SetError("unknown command shell.unknown")
	}}
}

func buildCommandPaletteSnapshot() snapshotScenarioState {
	return snapshotScenarioState{Name: "overlay-command-palette", RouteID: routeDashboard, Surface: buildBlockedSnapshot().Surface, Sessions: defaultSnapshotSessions(), Watch: snapshotWatchBlocked(), Theme: themePresetDark, Focus: focusPalette, Prepare: func(m *shellModel) {
		openPaletteOverlay(m)
		m.palette.query = "focus"
		if updated, request, ok := m.palette.BeginFilterRefresh(); ok {
			if applied, ok := updated.ApplyFilterRefresh(buildPaletteFilterResult(request, updated.entriesVersion, updated.query, updated.normalizedEntries, updated.appliedNeedle, updated.matchIndexes)); ok {
				m.palette = applied
			}
		}
		m.syncOverlayStack()
	}}
}

func buildSessionSwitcherSnapshot() snapshotScenarioState {
	return snapshotScenarioState{Name: "overlay-session-switcher", RouteID: routeDashboard, Surface: buildBlockedSnapshot().Surface, Sessions: defaultSnapshotSessions(), Watch: snapshotWatchBlocked(), Theme: themePresetDark, Focus: focusPalette, Prepare: func(m *shellModel) {
		openSessionSwitcherOverlay(m)
		m.sessions.query = "session"
		m.sessions.rebuildMatches()
		m.syncOverlayStack()
	}}
}

func buildQuitConfirmChatComposeSnapshot() snapshotScenarioState {
	chatState := buildChatSnapshot()
	return snapshotScenarioState{Name: "quit-confirm-chat-compose", RouteID: routeChat, Surface: chatState.Surface, Sessions: chatState.Sessions, Watch: chatState.Watch, Theme: themePresetDark, Focus: focusPalette, Prepare: func(m *shellModel) {
		chat := m.routeModels[routeChat].(chatRouteModel)
		chat.composeOn = true
		chat.composer.Focus()
		m.routeModels[routeChat] = chat
		m.beginOverlaySession()
		m.quitConfirm = shellQuitConfirmState{active: true, reason: "chat compose"}
		m.setFocus(focusPalette)
		m.syncOverlayStack()
	}}
}

func buildQuitConfirmCommandEntrySnapshot() snapshotScenarioState {
	return snapshotScenarioState{Name: "quit-confirm-command-entry", RouteID: routeDashboard, Surface: buildBlockedSnapshot().Surface, Sessions: defaultSnapshotSessions(), Watch: snapshotWatchBlocked(), Theme: themePresetDark, Focus: focusPalette, Prepare: func(m *shellModel) {
		m.commandMode = m.commandMode.Open().Append("open runs")
		m.beginOverlaySession()
		m.quitConfirm = shellQuitConfirmState{active: true, reason: "command entry"}
		m.setFocus(focusPalette)
		m.syncOverlayStack()
	}}
}

func buildQuitConfirmProviderSecretSnapshot() snapshotScenarioState {
	providerState := buildProviderSetupSnapshot()
	return snapshotScenarioState{Name: "quit-confirm-provider-secret", RouteID: routeProviders, Surface: providerState.Surface, Sessions: providerState.Sessions, Watch: providerState.Watch, Theme: themePresetDark, Focus: focusPalette, Prepare: func(m *shellModel) {
		provider := m.routeModels[routeProviders].(providerSetupRouteModel)
		provider.entryActive = true
		provider.secretRunes = []rune("secret-token")
		m.routeModels[routeProviders] = provider
		m.beginOverlaySession()
		m.quitConfirm = shellQuitConfirmState{active: true, reason: "provider secret entry"}
		m.setFocus(focusPalette)
		m.syncOverlayStack()
	}}
}

func buildNarrowSidebarOverlaySnapshot() snapshotScenarioState {
	return snapshotScenarioState{Name: "narrow-sidebar-overlay", RouteID: routeDashboard, Surface: buildBlockedSnapshot().Surface, Sessions: defaultSnapshotSessions(), Watch: snapshotWatchBlocked(), Theme: themePresetDark, Focus: focusPalette, Viewport: snapshotViewportMobile, Prepare: func(m *shellModel) {
		m.narrowSidebarOn = true
		m.setFocus(focusPalette)
		m.syncOverlayStack()
	}}
}

func buildNarrowInspectorOverlaySnapshot() snapshotScenarioState {
	runsState := buildRunsSnapshot()
	return snapshotScenarioState{Name: "narrow-inspector-overlay", RouteID: routeRuns, Surface: runsState.Surface, Sessions: runsState.Sessions, Watch: runsState.Watch, Theme: themePresetDark, Focus: focusPalette, Viewport: snapshotViewportMobile, Prepare: func(m *shellModel) {
		m.narrowInspectOn = true
		m.setFocus(focusPalette)
		m.syncOverlayStack()
	}}
}

func buildNarrowSidebarOverlayCompactSnapshot() snapshotScenarioState {
	state := buildNarrowSidebarOverlaySnapshot()
	state.Name = "narrow-sidebar-overlay-compact"
	state.Width = shellMediumMinWidth - 2
	state.Height = 36
	return state
}

func buildNarrowInspectorOverlayCompactSnapshot() snapshotScenarioState {
	state := buildNarrowInspectorOverlaySnapshot()
	state.Name = "narrow-inspector-overlay-compact"
	state.Width = shellMediumMinWidth - 2
	state.Height = 36
	return state
}

func buildDashboardFocusNavSnapshot() snapshotScenarioState {
	blocked := buildBlockedSnapshot()
	blocked.Name = "dashboard-focus-nav"
	blocked.Focus = focusNav
	return blocked
}

func buildActionCenterSnapshot() snapshotScenarioState {
	watch := snapshotWatchActionCenter()
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	audit := brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "failed", AnchoringStatus: "degraded", CurrentlyDegraded: true, FindingCount: 3, HardFailures: []string{"anchor_receipt_invalid"}}}
	action := actionCenterRouteModel{
		def:         routeDefinition{ID: routeAction, Label: "Action Center"},
		now:         func() time.Time { return now },
		inspectorOn: true,
		family:      actionCenterFamilyApprovals,
		selected: map[actionCenterFamily]int{
			actionCenterFamilyApprovals: 0,
			actionCenterFamilyOps:       0,
			actionCenterFamilyBlocked:   0,
		},
		watch:       watchProjection(watch).Live,
		watchHealth: watchProjection(watch).Health,
		runs: []brokerapi.RunSummary{
			{RunID: "run-approval", WorkspaceID: "ws-1", LifecycleState: "waiting", PendingApprovalCount: 1, BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed"},
			{RunID: "run-degraded", WorkspaceID: "ws-1", LifecycleState: "active", RuntimePostureDegraded: true, BackendKind: "workspace", IsolationAssuranceLevel: "reduced", AuditAnchoringStatus: "degraded", AuditCurrentlyDegraded: true},
			{RunID: "run-blocked", WorkspaceID: "ws-1", LifecycleState: "blocked", BlockingReasonCode: "project_blocked", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed"},
		},
		approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-urgent", Status: "pending", ApprovalTriggerCode: "policy_gate", ExpiresAt: now.Add(45 * time.Minute).Format(time.RFC3339), BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-approval", StageID: "stage-2", ActionKind: "promotion"}}},
		project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "incompatible", CompatibilityPosture: "unsupported", NormalOperationAllowed: false}, BlockedExplanation: "project setup blocks normal operation", RemediationGuidance: []string{"review project setup", "apply supported substrate"}},
		audit:     &audit,
	}
	return snapshotScenarioState{Name: "action-center-triage", RouteID: routeAction, Surface: action, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusContent}
}

func buildLeaderHelpSnapshot() snapshotScenarioState {
	watch := snapshotWatchBlocked()
	dashboard := dashboardRouteModel{
		def: routeDefinition{ID: routeDashboard, Label: "Dashboard"},
		data: dashboardData{
			readiness: brokerapi.BrokerReadiness{Ready: true, RecoveryComplete: true},
			version:   brokerapi.BrokerVersionInfo{ProductVersion: "0.1.0"},
			project:   brokerapi.ProjectSubstratePostureGetResponse{PostureSummary: brokerapi.ProjectSubstratePostureSummary{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureSummary", ValidationState: "valid", CompatibilityPosture: "supported", NormalOperationAllowed: true}},
			runs:      []brokerapi.RunSummary{{RunID: "run-blocked", WorkspaceID: "ws-1", LifecycleState: "blocked", BlockingReasonCode: "approval_wait", PendingApprovalCount: 1, BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", ProvisioningPosture: "attested", AuditIntegrityStatus: "ok", AuditAnchoringStatus: "ok"}},
			approvals: []brokerapi.ApprovalSummary{{ApprovalID: "ap-blocked", Status: "pending", ApprovalTriggerCode: "policy_gate", BoundScope: brokerapi.ApprovalBoundScope{RunID: "run-blocked", StageID: "stage-1", ActionKind: "promotion"}}},
			live:      watchProjection(watch).Live,
			audit:     brokerapi.AuditVerificationGetResponse{Summary: trustpolicy.DerivedRunAuditVerificationSummary{IntegrityStatus: "ok", AnchoringStatus: "ok", CurrentlyDegraded: false}},
		},
	}
	return snapshotScenarioState{Name: "leader-help-root", RouteID: routeDashboard, Surface: dashboard, Sessions: defaultSnapshotSessions(), Watch: watch, Theme: themePresetDark, Focus: focusPalette, Prepare: func(m *shellModel) {
		m.beginOverlaySession()
		m.leader.Rebind(m.actions.leaderBindings(*m))
		m.leader.Start()
		m.setFocus(focusPalette)
		m.syncOverlayStack()
	}}
}
