package main

import (
	"fmt"
	"strings"
)

type routeID string

const (
	routeDashboard routeID = "dashboard"
	routeChat      routeID = "chat"
	routeRuns      routeID = "runs"
	routeApprovals routeID = "approvals"
	routeAction    routeID = "action-center"
	routeArtifacts routeID = "artifacts"
	routeAudit     routeID = "audit"
	routeStatus    routeID = "status"
	routeProviders routeID = "model-providers"
	routeGitSetup  routeID = "git-setup"
	routeGitRemote routeID = "git-remote-mutation"
)

type routeDefinition struct {
	ID           routeID
	Label        string
	Description  string
	Index        int
	QuickJumpKey string
}

func shellRoutes() []routeDefinition {
	return []routeDefinition{
		{ID: routeDashboard, Label: "Dashboard", Description: "System overview and safety posture", Index: 1, QuickJumpKey: "1"},
		{ID: routeChat, Label: "Chat", Description: "Session transcript and operator interaction", Index: 2, QuickJumpKey: "2"},
		{ID: routeRuns, Label: "Runs", Description: "Run list and run detail workbench", Index: 3, QuickJumpKey: "3"},
		{ID: routeApprovals, Label: "Approvals", Description: "Approval queue and decision center", Index: 4, QuickJumpKey: "4"},
		{ID: routeAction, Label: "Action Center", Description: "Interactive/operator-attention queues and blocked-work impact", Index: 5, QuickJumpKey: "5"},
		{ID: routeArtifacts, Label: "Artifacts", Description: "Artifact browsing and drill-down", Index: 6, QuickJumpKey: "6"},
		{ID: routeAudit, Label: "Audit", Description: "Audit timeline and verification posture", Index: 7, QuickJumpKey: "7"},
		{ID: routeStatus, Label: "Status", Description: "Broker readiness and subsystem posture", Index: 8, QuickJumpKey: "8"},
		{ID: routeProviders, Label: "Model Providers", Description: "Broker-owned direct credential setup", Index: 9, QuickJumpKey: "9"},
		{ID: routeGitSetup, Label: "Git Setup", Description: "Broker-owned git setup and auth posture", Index: 10, QuickJumpKey: "0"},
		{ID: routeGitRemote, Label: "Git Remote", Description: "Review prepared git remote mutations and execute", Index: 11, QuickJumpKey: "-"},
	}
}

func routeByQuickJumpKey(key string, routes []routeDefinition) (routeDefinition, bool) {
	for _, r := range routes {
		if key == routeQuickJumpKey(r) {
			return r, true
		}
	}
	return routeDefinition{}, false
}

func routeQuickJumpKey(route routeDefinition) string {
	if key := strings.TrimSpace(route.QuickJumpKey); key != "" {
		return key
	}
	return fmt.Sprintf("%d", route.Index)
}
