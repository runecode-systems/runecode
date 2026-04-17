package main

import "strings"

type shellBreakpoint string

const (
	shellBreakpointNarrow shellBreakpoint = "narrow"
	shellBreakpointMedium shellBreakpoint = "medium"
	shellBreakpointWide   shellBreakpoint = "wide"
)

type routeShellContext struct {
	Width      int
	Height     int
	Focus      focusArea
	Breakpoint shellBreakpoint
}

type routeCopyAction struct {
	ID    string
	Label string
	Text  string
}

type routeSurface struct {
	Main           string
	Inspector      string
	BottomStrip    string
	Status         string
	Breadcrumbs    []string
	MainTitle      string
	InspectorTitle string
	ModeTabs       []string
	ActiveTab      string
	CopyActions    []routeCopyAction
}

type routeLoadState string

const (
	routeLoadStateReady   routeLoadState = "ready"
	routeLoadStateLoading routeLoadState = "loading"
	routeLoadStateEmpty   routeLoadState = "empty"
	routeLoadStateError   routeLoadState = "error"
)

func combineLegacyRouteView(surface routeSurface) string {
	parts := []string{strings.TrimSpace(surface.Main)}
	if strings.TrimSpace(surface.Inspector) != "" {
		parts = append(parts, tableHeader("Inspector"), strings.TrimSpace(surface.Inspector))
	}
	if strings.TrimSpace(surface.BottomStrip) != "" {
		parts = append(parts, strings.TrimSpace(surface.BottomStrip))
	}
	if strings.TrimSpace(surface.Status) != "" {
		parts = append(parts, "Status: "+strings.TrimSpace(surface.Status))
	}
	return compactLines(parts...)
}
