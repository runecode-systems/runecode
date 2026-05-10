//go:build runecode_tui_snapshot

package main

import (
	"fmt"
	"slices"
	"strings"
)

type snapshotAuditBundle string

const (
	snapshotBundleNone         snapshotAuditBundle = ""
	snapshotBundleFullAudit    snapshotAuditBundle = "full-audit"
	snapshotBundleDashboard    snapshotAuditBundle = "dashboard-audit"
	snapshotBundleActionCenter snapshotAuditBundle = "action-center-audit"
	snapshotBundleRuns         snapshotAuditBundle = "runs-audit"
	snapshotBundleApprovals    snapshotAuditBundle = "approvals-audit"
	snapshotBundleAudit        snapshotAuditBundle = "audit-route-audit"
	snapshotBundleStatus       snapshotAuditBundle = "status-audit"
	snapshotBundleSetup        snapshotAuditBundle = "setup-audit"
	snapshotBundleChat         snapshotAuditBundle = "chat-audit"
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
				"leader-help-root",
				"chat-active-session",
				"runs-active-detail",
				"approvals-pending-detail",
				"action-center-triage",
				"artifacts-evidence-detail",
				"audit-degraded-detail",
				"status-ready-overview",
				"model-providers-credential-needed",
				"git-setup-identity-needed",
				"git-remote-approval-ready",
			},
			Routes: []routeID{routeDashboard, routeChat, routeRuns, routeApprovals, routeAction, routeArtifacts, routeAudit, routeStatus, routeProviders, routeGitSetup, routeGitRemote},
		}, nil
	case snapshotBundleDashboard:
		return snapshotBundleSpec{Name: snapshotBundleDashboard, Scenarios: []string{"dashboard-healthy-empty", "dashboard-approval-waiting", "dashboard-blocked", "dashboard-degraded", "leader-help-root"}, Routes: []routeID{routeDashboard}}, nil
	case snapshotBundleActionCenter:
		return snapshotBundleSpec{Name: snapshotBundleActionCenter, Scenarios: []string{"action-center-triage"}, Routes: []routeID{routeAction}}, nil
	case snapshotBundleRuns:
		return snapshotBundleSpec{Name: snapshotBundleRuns, Scenarios: []string{"runs-active-detail", "artifacts-evidence-detail"}, Routes: []routeID{routeRuns, routeArtifacts}}, nil
	case snapshotBundleApprovals:
		return snapshotBundleSpec{Name: snapshotBundleApprovals, Scenarios: []string{"approvals-pending-detail"}, Routes: []routeID{routeApprovals}}, nil
	case snapshotBundleAudit:
		return snapshotBundleSpec{Name: snapshotBundleAudit, Scenarios: []string{"audit-degraded-detail"}, Routes: []routeID{routeAudit}}, nil
	case snapshotBundleStatus:
		return snapshotBundleSpec{Name: snapshotBundleStatus, Scenarios: []string{"status-ready-overview"}, Routes: []routeID{routeStatus}}, nil
	case snapshotBundleSetup:
		return snapshotBundleSpec{Name: snapshotBundleSetup, Scenarios: []string{"model-providers-credential-needed", "git-setup-identity-needed", "git-remote-approval-ready"}, Routes: []routeID{routeProviders, routeGitSetup, routeGitRemote}}, nil
	case snapshotBundleChat:
		return snapshotBundleSpec{Name: snapshotBundleChat, Scenarios: []string{"chat-active-session"}, Routes: []routeID{routeChat}}, nil
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
