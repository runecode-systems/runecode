//go:build runecode_tui_snapshot

package main

import (
	"fmt"
	"slices"
	"strings"
)

type snapshotAuditBundle string

const (
	snapshotBundleNone          snapshotAuditBundle = ""
	snapshotBundleFullAudit     snapshotAuditBundle = "full-audit"
	snapshotBundleDashboard     snapshotAuditBundle = "dashboard-audit"
	snapshotBundleActionCenter  snapshotAuditBundle = "action-center-audit"
	snapshotBundleRuns          snapshotAuditBundle = "runs-audit"
	snapshotBundleApprovals     snapshotAuditBundle = "approvals-audit"
	snapshotBundleAudit         snapshotAuditBundle = "audit-route-audit"
	snapshotBundleStatus        snapshotAuditBundle = "status-audit"
	snapshotBundleSetup         snapshotAuditBundle = "setup-audit"
	snapshotBundleChat          snapshotAuditBundle = "chat-audit"
	snapshotBundleShell         snapshotAuditBundle = "shell-overlays-audit"
	snapshotBundleNarrowMobile  snapshotAuditBundle = "shell-narrow-mobile-audit"
	snapshotBundleNarrowCompact snapshotAuditBundle = "shell-narrow-compact-audit"
)

type snapshotCoverage struct {
	Bundle    string   `json:"bundle,omitempty"`
	Routes    []string `json:"routes,omitempty"`
	Viewports []string `json:"viewports,omitempty"`
	Scenarios []string `json:"scenarios,omitempty"`
}

type snapshotBundleSpec struct {
	Name      snapshotAuditBundle
	Scenarios []string
	Routes    []routeID
}

func normalizeSnapshotBundle(name string) snapshotAuditBundle {
	return snapshotAuditBundle(strings.TrimSpace(name))
}

func resolveSnapshotBundle(name string) (snapshotBundleSpec, error) {
	switch normalizeSnapshotBundle(name) {
	case snapshotBundleFullAudit:
		return snapshotBundleSpec{
			Name: snapshotBundleFullAudit,
			Scenarios: []string{
				"dashboard-healthy-empty",
				"dashboard-approval-waiting",
				"dashboard-blocked",
				"dashboard-degraded",
				"dashboard-toast-info",
				"command-mode-draft",
				"command-mode-error",
				"overlay-command-palette",
				"overlay-session-switcher",
				"quit-confirm-chat-compose",
				"quit-confirm-command-entry",
				"quit-confirm-provider-secret",
				"dashboard-focus-nav",
				"leader-help-root",
				"chat-active-session",
				"chat-compose-open",
				"runs-active-detail",
				"runs-focus-inspector",
				"runs-raw-detail",
				"approvals-pending-detail",
				"approvals-structured-detail",
				"action-center-triage",
				"artifacts-evidence-detail",
				"artifacts-structured-detail",
				"audit-degraded-detail",
				"audit-raw-detail",
				"status-ready-overview",
				"status-degraded-overview",
				"model-providers-credential-needed",
				"model-providers-secret-entry",
				"git-setup-identity-needed",
				"git-setup-provider-linked",
				"git-remote-approval-ready",
				"git-remote-executed",
				"git-remote-fail-closed",
			},
			Routes: []routeID{routeDashboard, routeChat, routeRuns, routeApprovals, routeAction, routeArtifacts, routeAudit, routeStatus, routeProviders, routeGitSetup, routeGitRemote},
		}, nil
	case snapshotBundleDashboard:
		return snapshotBundleSpec{Name: snapshotBundleDashboard, Scenarios: []string{"dashboard-healthy-empty", "dashboard-approval-waiting", "dashboard-blocked", "dashboard-degraded", "dashboard-toast-info", "dashboard-focus-nav"}, Routes: []routeID{routeDashboard}}, nil
	case snapshotBundleActionCenter:
		return snapshotBundleSpec{Name: snapshotBundleActionCenter, Scenarios: []string{"action-center-triage"}, Routes: []routeID{routeAction}}, nil
	case snapshotBundleRuns:
		return snapshotBundleSpec{Name: snapshotBundleRuns, Scenarios: []string{"runs-active-detail", "runs-focus-inspector", "runs-raw-detail", "artifacts-evidence-detail", "artifacts-structured-detail"}, Routes: []routeID{routeRuns, routeArtifacts}}, nil
	case snapshotBundleApprovals:
		return snapshotBundleSpec{Name: snapshotBundleApprovals, Scenarios: []string{"approvals-pending-detail", "approvals-structured-detail"}, Routes: []routeID{routeApprovals}}, nil
	case snapshotBundleAudit:
		return snapshotBundleSpec{Name: snapshotBundleAudit, Scenarios: []string{"audit-degraded-detail", "audit-raw-detail"}, Routes: []routeID{routeAudit}}, nil
	case snapshotBundleStatus:
		return snapshotBundleSpec{Name: snapshotBundleStatus, Scenarios: []string{"status-ready-overview", "status-degraded-overview"}, Routes: []routeID{routeStatus}}, nil
	case snapshotBundleSetup:
		return snapshotBundleSpec{Name: snapshotBundleSetup, Scenarios: []string{"model-providers-credential-needed", "model-providers-secret-entry", "git-setup-identity-needed", "git-setup-provider-linked", "git-remote-approval-ready", "git-remote-executed", "git-remote-fail-closed"}, Routes: []routeID{routeProviders, routeGitSetup, routeGitRemote}}, nil
	case snapshotBundleChat:
		return snapshotBundleSpec{Name: snapshotBundleChat, Scenarios: []string{"chat-active-session", "chat-compose-open"}, Routes: []routeID{routeChat}}, nil
	case snapshotBundleShell:
		return snapshotBundleSpec{Name: snapshotBundleShell, Scenarios: []string{"command-mode-draft", "command-mode-error", "overlay-command-palette", "overlay-session-switcher", "quit-confirm-chat-compose", "quit-confirm-command-entry", "quit-confirm-provider-secret", "leader-help-root"}, Routes: []routeID{routeDashboard, routeChat, routeProviders}}, nil
	case snapshotBundleNarrowMobile:
		return snapshotBundleSpec{Name: snapshotBundleNarrowMobile, Scenarios: []string{"narrow-sidebar-overlay", "narrow-inspector-overlay"}, Routes: []routeID{routeDashboard, routeRuns}}, nil
	case snapshotBundleNarrowCompact:
		return snapshotBundleSpec{Name: snapshotBundleNarrowCompact, Scenarios: []string{"narrow-sidebar-overlay-compact", "narrow-inspector-overlay-compact"}, Routes: []routeID{routeDashboard, routeRuns}}, nil
	default:
		return snapshotBundleSpec{}, fmt.Errorf("unknown snapshot bundle %q", name)
	}
}

func snapshotCoverageForBundle(bundle snapshotAuditBundle, scenarios []snapshotScenarioState, viewport snapshotViewportPreset) snapshotCoverage {
	routeSet := make(map[string]struct{}, len(scenarios))
	scenarioNames := make([]string, 0, len(scenarios))
	for _, scenario := range scenarios {
		routeSet[string(scenario.RouteID)] = struct{}{}
		scenarioNames = append(scenarioNames, scenario.Name)
	}
	routes := make([]string, 0, len(routeSet))
	for route := range routeSet {
		routes = append(routes, route)
	}
	slices.Sort(routes)
	slices.Sort(scenarioNames)
	return snapshotCoverage{Bundle: string(bundle), Routes: routes, Viewports: []string{string(viewport)}, Scenarios: scenarioNames}
}
