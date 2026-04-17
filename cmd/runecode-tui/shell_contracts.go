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
	Focused    routeRegionFocus
	Breakpoint shellBreakpoint
	Regions    routeShellRegions
	Render     routeShellRenderPreferences
}

type routeShellRenderPreferences struct {
	PreferredPresentation contentPresentationMode
	ThemePreset           themePreset
}

type routeShellRegions struct {
	Main      routeRegionDimensions
	Inspector routeRegionDimensions
	Bottom    routeRegionDimensions
	Status    routeRegionDimensions
	Sidebar   routeRegionDimensions
}

type routeRegionDimensions struct {
	Width  int
	Height int
}

type routeRegionFocus string

const (
	routeRegionMain      routeRegionFocus = "main"
	routeRegionInspector routeRegionFocus = "inspector"
	routeRegionOverlay   routeRegionFocus = "overlay"
)

type routeCopyAction struct {
	ID    string
	Label string
	Text  string
}

type routeSurface struct {
	Regions      routeSurfaceRegions
	Chrome       routeSurfaceChrome
	Actions      routeSurfaceActions
	Capabilities routeSurfaceCapabilities
}

type routeSurfaceCapabilities struct {
	Inspector routeInspectorCapability
}

type routeInspectorCapability struct {
	Supported bool
	Enabled   bool
}

type routeSurfaceRegions struct {
	Main      routeSurfaceRegion
	Inspector routeSurfaceRegion
	Bottom    routeSurfaceRegion
	Status    routeSurfaceRegion
}

type routeSurfaceRegion struct {
	Title string
	Body  string
}

type routeSurfaceChrome struct {
	Breadcrumbs []string
}

type routeSurfaceActions struct {
	ModeTabs         []string
	ActiveTab        string
	CopyActions      []routeCopyAction
	ReferenceActions []routeActionItem
	LocalActions     []routeActionItem
}

type routeActionItem struct {
	Label  string
	Action paletteActionMsg
}

type shellObjectLocation struct {
	RouteID routeID
	Object  workbenchObjectRef
}

type shellWorkbenchLocation struct {
	Primary   shellObjectLocation
	Inspector *shellObjectLocation
}

type routeLoadState string

const (
	routeLoadStateReady   routeLoadState = "ready"
	routeLoadStateLoading routeLoadState = "loading"
	routeLoadStateEmpty   routeLoadState = "empty"
	routeLoadStateError   routeLoadState = "error"
)

func combineLegacyRouteView(surface routeSurface) string {
	parts := []string{strings.TrimSpace(surface.Regions.Main.Body)}
	if strings.TrimSpace(surface.Regions.Inspector.Body) != "" {
		parts = append(parts, tableHeader("Inspector"), strings.TrimSpace(surface.Regions.Inspector.Body))
	}
	if strings.TrimSpace(surface.Regions.Bottom.Body) != "" {
		parts = append(parts, strings.TrimSpace(surface.Regions.Bottom.Body))
	}
	if strings.TrimSpace(surface.Regions.Status.Body) != "" {
		parts = append(parts, "Status: "+strings.TrimSpace(surface.Regions.Status.Body))
	}
	return compactLines(parts...)
}

func routeRegionWidth(region routeRegionDimensions, fallback int) int {
	if region.Width > 0 {
		return region.Width
	}
	return fallback
}

func routeRegionHeight(region routeRegionDimensions, fallback int) int {
	if region.Height > 0 {
		return region.Height
	}
	return fallback
}
