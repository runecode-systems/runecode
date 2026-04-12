package main

import "fmt"

type routeID string

const (
	routeDashboard routeID = "dashboard"
	routeChat      routeID = "chat"
	routeRuns      routeID = "runs"
	routeApprovals routeID = "approvals"
	routeArtifacts routeID = "artifacts"
	routeAudit     routeID = "audit"
	routeStatus    routeID = "status"
)

type routeDefinition struct {
	ID          routeID
	Label       string
	Description string
	Index       int
}

func shellRoutes() []routeDefinition {
	return []routeDefinition{
		{ID: routeDashboard, Label: "Dashboard", Description: "System overview and safety posture", Index: 1},
		{ID: routeChat, Label: "Chat", Description: "Session transcript and operator interaction", Index: 2},
		{ID: routeRuns, Label: "Runs", Description: "Run list and run detail workbench", Index: 3},
		{ID: routeApprovals, Label: "Approvals", Description: "Approval queue and decision center", Index: 4},
		{ID: routeArtifacts, Label: "Artifacts", Description: "Artifact browsing and drill-down", Index: 5},
		{ID: routeAudit, Label: "Audit", Description: "Audit timeline and verification posture", Index: 6},
		{ID: routeStatus, Label: "Status", Description: "Broker readiness and subsystem posture", Index: 7},
	}
}

func routeByQuickJumpKey(key string, routes []routeDefinition) (routeDefinition, bool) {
	for _, r := range routes {
		if key == fmt.Sprintf("%d", r.Index) {
			return r, true
		}
	}
	return routeDefinition{}, false
}
