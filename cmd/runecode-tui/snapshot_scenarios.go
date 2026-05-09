//go:build runecode_tui_snapshot

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type snapshotScenarioState struct {
	Name     string
	RouteID  routeID
	Surface  routeModel
	Sessions []brokerapi.SessionSummary
	Watch    shellWatchTransportLoadedMsg
	Theme    themePreset
	Focus    focusArea
}

func resolveSnapshotScenarios(name string) ([]snapshotScenarioState, error) {
	scenarios := []snapshotScenarioState{
		buildHealthyEmptySnapshot(),
		buildApprovalWaitingSnapshot(),
		buildBlockedSnapshot(),
		buildDegradedSnapshot(),
		buildActionCenterSnapshot(),
	}
	switch strings.TrimSpace(name) {
	case "", "all":
		return scenarios, nil
	case "dashboard":
		return scenarios[:4], nil
	case "action-center":
		return scenarios[4:], nil
	default:
		for _, scenario := range scenarios {
			if scenario.Name == name {
				return []snapshotScenarioState{scenario}, nil
			}
		}
		return nil, fmt.Errorf("unknown snapshot scenario %q", name)
	}
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
		appTheme = newTheme(theme)
		m := newShellModel()
		m.width = cfg.width
		m.height = cfg.height
		m.themePreset = theme
		m.preferredMode = presentationRendered
		m.location.Primary = shellObjectLocation{RouteID: state.RouteID, Object: workbenchObjectRef{Kind: "route", ID: string(state.RouteID)}}
		m.nav.SelectByRouteID(state.RouteID)
		m.routeModels[state.RouteID] = state.Surface
		m.applySessionWorkspaceLoaded(sessionWorkspaceLoadedMsg{sessions: append([]brokerapi.SessionSummary(nil), state.Sessions...)})
		m.applyWatchTransport(state.Watch)
		m.publishWatchStateToRoutes()
		m.setFocus(state.Focus)
		m.syncSidebarCursorToLocation()
		return m.View(), state.RouteID, nil
	})
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

func watchProjection(msg shellWatchTransportLoadedMsg) shellWatchProjectionState {
	manager := newShellWatchManager()
	manager.applyTransport(msg)
	return manager.projection
}

func snapshotWatchHealthyEmpty() shellWatchTransportLoadedMsg {
	return shellWatchTransportLoadedMsg{ObservedAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)}
}

func snapshotWatchApprovalWaiting() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 5, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-1", LifecycleState: "active", BackendKind: "workspace", IsolationAssuranceLevel: "sandboxed", PendingApprovalCount: 1}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-1", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Awaiting approval"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}

func snapshotWatchBlocked() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 10, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-blocked", LifecycleState: "blocked", BlockingReasonCode: "approval_wait", PendingApprovalCount: 1}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-blocked", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "blocked", LastActivityPreview: "Blocked on approval"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}

func snapshotWatchDegraded() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 15, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-degraded", LifecycleState: "active", RuntimePostureDegraded: true, AuditCurrentlyDegraded: true}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-quiet", Status: "consumed"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-1", WorkspaceID: "ws-1"}, Status: "degraded", LastActivityPreview: "Runtime posture degraded"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}, {EventType: "run_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}, {EventType: "session_watch_terminal", Seq: 2, Terminal: true, TerminalStatus: "completed"}}}}
}

func snapshotWatchActionCenter() shellWatchTransportLoadedMsg {
	now := time.Date(2026, 5, 9, 12, 20, 0, 0, time.UTC)
	run := brokerapi.RunSummary{RunID: "run-degraded", LifecycleState: "active", RuntimePostureDegraded: true}
	approval := brokerapi.ApprovalSummary{ApprovalID: "ap-urgent", Status: "pending"}
	session := brokerapi.SessionSummary{Identity: brokerapi.SessionIdentity{SessionID: "session-2", WorkspaceID: "ws-1"}, Status: "active", LastActivityPreview: "Triage action center"}
	return shellWatchTransportLoadedMsg{ObservedAt: now, Run: shellWatchRunTransportResult{Events: []brokerapi.RunWatchEvent{{EventType: "run_watch_snapshot", Seq: 1, Run: &run}}}, Approval: shellWatchApprovalTransportResult{Events: []brokerapi.ApprovalWatchEvent{{EventType: "approval_watch_snapshot", Seq: 1, Approval: &approval}}}, Session: shellWatchSessionTransportResult{Events: []brokerapi.SessionWatchEvent{{EventType: "session_watch_snapshot", Seq: 1, Session: &session}}}}
}
