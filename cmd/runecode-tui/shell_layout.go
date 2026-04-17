package main

type shellLayoutPlan struct {
	Breakpoint        shellBreakpoint
	Regions           routeShellRegions
	NavigationVisible bool
	InspectorVisible  bool
}

func (m shellModel) planShellLayout(surface routeSurface) shellLayoutPlan {
	viewportWidth, viewportHeight := normalizedShellViewport(m.width, m.height)
	breakpoint := m.breakpoint()
	navigationVisible := m.effectiveSidebarVisible()
	inspectorVisible := m.shellInspectorVisible(surface, breakpoint)
	regions := m.planShellRegions(viewportWidth, viewportHeight, breakpoint, navigationVisible, inspectorVisible)
	return shellLayoutPlan{
		Breakpoint:        breakpoint,
		NavigationVisible: navigationVisible,
		InspectorVisible:  inspectorVisible,
		Regions:           regions,
	}
}

func normalizedShellViewport(width, height int) (int, int) {
	viewportWidth := nonNegativeDimension(width)
	viewportHeight := nonNegativeDimension(height)
	if viewportWidth <= 0 {
		viewportWidth = 120
	}
	if viewportHeight <= 0 {
		viewportHeight = 40
	}
	return viewportWidth, viewportHeight
}

func (m shellModel) planShellRegions(viewportWidth int, viewportHeight int, breakpoint shellBreakpoint, navigationVisible bool, inspectorVisible bool) routeShellRegions {
	mainMinWidth := minimumMainPaneWidth(breakpoint)
	sidebarWidth, inspectorWidth := m.planSecondaryPaneWidths(viewportWidth, mainMinWidth, navigationVisible, inspectorVisible)
	mainWidth := viewportWidth - sidebarWidth - inspectorWidth
	if mainWidth < mainMinWidth {
		mainWidth = mainMinWidth
	}
	if mainWidth > viewportWidth {
		mainWidth = viewportWidth
	}

	mainHeight := viewportHeight - 12
	if mainHeight < 1 {
		mainHeight = 1
	}
	regions := routeShellRegions{
		Main:      routeRegionDimensions{Width: mainWidth, Height: mainHeight},
		Inspector: routeRegionDimensions{Width: inspectorWidth, Height: mainHeight},
		Bottom:    routeRegionDimensions{Width: viewportWidth, Height: 3},
		Status:    routeRegionDimensions{Width: viewportWidth, Height: 1},
		Sidebar:   routeRegionDimensions{Width: sidebarWidth, Height: mainHeight},
	}
	if !navigationVisible {
		regions.Sidebar = routeRegionDimensions{}
	}
	if !inspectorVisible {
		regions.Inspector = routeRegionDimensions{}
	}
	return regions
}

func (m shellModel) planSecondaryPaneWidths(viewportWidth int, mainMinWidth int, navigationVisible bool, inspectorVisible bool) (int, int) {
	availableSecondaryWidth := viewportWidth - mainMinWidth
	if availableSecondaryWidth < 0 {
		availableSecondaryWidth = 0
	}
	sidebarWidth := 0
	if navigationVisible {
		sidebarWidth = paneWidthForRatio(availableSecondaryWidth, m.sidebarRatio, 20, availableSecondaryWidth)
	}
	inspectorWidth := 0
	if inspectorVisible {
		remainingSecondaryWidth := availableSecondaryWidth - sidebarWidth
		if remainingSecondaryWidth < 0 {
			remainingSecondaryWidth = 0
		}
		inspectorWidth = paneWidthForRatio(remainingSecondaryWidth, m.inspectorRatio, 24, remainingSecondaryWidth)
	}
	return sidebarWidth, inspectorWidth
}

func minimumMainPaneWidth(breakpoint shellBreakpoint) int {
	switch breakpoint {
	case shellBreakpointNarrow:
		return 40
	case shellBreakpointMedium:
		return 48
	default:
		return 56
	}
}

func (m shellModel) shellInspectorVisible(surface routeSurface, breakpoint shellBreakpoint) bool {
	if breakpoint != shellBreakpointWide {
		return false
	}
	if !m.inspectorOn || m.inspectorFolded {
		return false
	}
	return routeInspectorAvailable(surface)
}

func routeInspectorAvailable(surface routeSurface) bool {
	return surface.Capabilities.Inspector.Supported && surface.Capabilities.Inspector.Enabled
}

func paneWidthForRatio(totalWidth int, ratio float64, minWidth int, maxWidth int) int {
	if totalWidth <= 0 {
		return 0
	}
	if maxWidth > 0 && maxWidth < minWidth {
		maxWidth = minWidth
	}
	width := int(float64(totalWidth) * clampPaneRatio(ratio))
	if width < minWidth {
		width = minWidth
	}
	if maxWidth > 0 && width > maxWidth {
		width = maxWidth
	}
	if width > totalWidth-1 {
		width = totalWidth - 1
	}
	if width < 0 {
		return 0
	}
	return width
}

func nonNegativeDimension(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
