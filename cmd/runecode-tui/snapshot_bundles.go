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
	bundle, ok := snapshotBundleSpecs()[normalizeSnapshotBundle(name)]
	if !ok {
		return snapshotBundleSpec{}, fmt.Errorf("unknown snapshot bundle %q", name)
	}
	return bundle, nil
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
