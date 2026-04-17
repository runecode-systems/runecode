package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type routeModel interface {
	ID() routeID
	Title() string
	Update(msg tea.Msg) (routeModel, tea.Cmd)
	View(width, height int, focus focusArea) string
	ShellSurface(ctx routeShellContext) routeSurface
}

type routeActivatedMsg struct {
	RouteID         routeID
	ActiveSessionID string
}

type routeViewportScrollMsg struct {
	Region routeRegionFocus
	Delta  int
}

type routeViewportResizeMsg struct {
	Width  int
	Height int
}

type routeErrorModel struct {
	def routeDefinition
}

func (m routeErrorModel) ID() routeID {
	return m.def.ID
}

func (m routeErrorModel) Title() string {
	return m.def.Label
}

func (m routeErrorModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	_ = msg
	return m, nil
}

func (m routeErrorModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	focusHint := "inactive"
	if focus == focusContent {
		focusHint = "active"
	}
	return fmt.Sprintf(
		"%s\n\n%s\n\nRoute initialization failed.",
		m.def.Label,
		"Content focus: "+focusHint,
	)
}

func (m routeErrorModel) ShellSurface(ctx routeShellContext) routeSurface {
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main: routeSurfaceRegion{Body: m.View(ctx.Width, ctx.Height, ctx.Focus)},
		},
		Chrome: routeSurfaceChrome{Breadcrumbs: []string{"Home", m.def.Label}},
	}
}

func newRouteModels(defs []routeDefinition) map[routeID]routeModel {
	client := newLocalBrokerClient()
	models := make(map[routeID]routeModel, len(defs))
	for _, def := range defs {
		switch def.ID {
		case routeDashboard:
			models[def.ID] = newDashboardRouteModel(def, client)
		case routeChat:
			models[def.ID] = newChatRouteModel(def, client)
		case routeRuns:
			models[def.ID] = newRunsRouteModel(def, client)
		case routeApprovals:
			models[def.ID] = newApprovalsRouteModel(def, client)
		case routeAction:
			models[def.ID] = newActionCenterRouteModel(def, client)
		case routeArtifacts:
			models[def.ID] = newArtifactsRouteModel(def, client)
		case routeAudit:
			models[def.ID] = newAuditRouteModel(def, client)
		case routeStatus:
			models[def.ID] = newStatusRouteModel(def, client)
		default:
			models[def.ID] = routeErrorModel{def: def}
		}
	}
	return models
}

func compactLines(lines ...string) string {
	nonEmpty := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		nonEmpty = append(nonEmpty, trimmed)
	}
	return strings.Join(nonEmpty, "\n")
}

func safeUIErrorText(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "unknown_error"
	}
	return sanitizeUIText(remediateBrokerErrorText(text))
}

func remediateBrokerErrorText(text string) string {
	switch strings.TrimSpace(text) {
	case "local_ipc_dial_error":
		return "local broker IPC unavailable; start `runecode-broker serve-local` in another terminal, then press r to retry"
	case "local_ipc_config_error":
		return "local broker IPC is not configured on this machine; use Linux with a local runtime dir/socket and start broker with matching --runtime-dir/--socket-name, or run with an available local broker listener"
	default:
		return text
	}
}
