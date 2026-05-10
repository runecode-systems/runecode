package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const (
	shellMediumMinWidth    = 90
	shellWideMinWidth      = 130
	emergencyQuitArmWindow = 1500 * time.Millisecond
)

var forceMemoryWorkbenchState bool

type shellOverlayID string

const (
	overlayIDQuickJump   shellOverlayID = "quick-jump"
	overlayIDSessions    shellOverlayID = "session-switcher"
	overlayIDSidebar     shellOverlayID = "sidebar-drawer"
	overlayIDInspector   shellOverlayID = "inspector-sheet"
	overlayIDLeader      shellOverlayID = "leader-which-key"
	overlayIDQuitConfirm shellOverlayID = "quit-confirm"
)

type focusArea int

const (
	focusNav focusArea = iota
	focusContent
	focusInspector
	focusPalette
)

func (f focusArea) Label() string {
	switch f {
	case focusNav:
		return "sidebar"
	case focusContent:
		return "main"
	case focusInspector:
		return "inspector"
	case focusPalette:
		return "overlay"
	default:
		return "unknown"
	}
}

type sessionWorkspaceLoadedMsg struct {
	sessions []brokerapi.SessionSummary
	err      error
}

type shellEmergencyQuitTimeoutMsg struct {
	token uint64
}

type shellModel struct {
	quitting bool
	width    int
	height   int

	keys     shellKeyMap
	routes   []routeDefinition
	nav      primaryNavModel
	palette  paletteModel
	sessions sessionSwitcherModel
	focus    focusArea
	client   localBrokerClient

	routeModels map[routeID]routeModel
	location    shellWorkbenchLocation
	history     []shellWorkbenchLocation

	focusManager   shellFocusManager
	overlayManager shellOverlayManager
	commands       shellCommandRegistry
	actions        shellActionGraph
	clipboard      shellClipboardService
	workbench      shellWorkbenchStateStore
	workbenchScope string
	toasts         shellToastService
	objectIndex    shellDiscoverabilityIndex
	paletteCache   []paletteEntry

	sidebarVisible  bool
	inspectorOn     bool
	themePreset     themePreset
	preferredMode   contentPresentationMode
	sidebarRatio    float64
	inspectorRatio  float64
	sidebarFolded   bool
	inspectorFolded bool
	narrowSidebarOn bool
	narrowInspectOn bool
	overlays        []shellOverlayID
	overlayReturn   focusArea

	sessionItems     []brokerapi.SessionSummary
	sidebarCursor    int
	sessionSelected  int
	activeSessionID  string
	sessionLoadError string
	sessionLoading   bool
	pinnedSessions   map[string]struct{}
	recentSessions   []string
	lastSessionByWS  map[string]string
	recentObjects    []workbenchObjectRef
	sessionWorkspace map[string]string
	viewedActivity   map[string]string
	watch            shellWatchManager
	activityFrame    int
	selectionMode    bool
	copyActionIndex  int

	leader                  shellLeaderState
	leaderBindingsSignature string
	leaderKeyConfig         string
	leaderKeyInvalid        string
	commandMode             shellCommandModeState
	emergencyQuit           shellEmergencyQuitState
	quitConfirm             shellQuitConfirmState
	overlayFrameCache       *shellOverlayFrameCache
}

type shellOverlayFrameCache struct {
	overlay shellOverlayID
	width   int
	height  int
	frame   string
}

type shellEmergencyQuitState struct {
	pending bool
	token   uint64
}

type shellQuitConfirmState struct {
	active bool
	reason string
}

func (m shellModel) Init() tea.Cmd {
	return tea.Batch(m.activateCurrentRouteCmd(), m.loadSessionWorkspaceCmd(), m.loadObjectIndexCmd(), m.startWatchPollCmd(), m.mouseCaptureCmd())
}

func (m shellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.prepareOverlayFrameCache(msg)
	m = m.disarmEmergencyQuitOnNormalInteraction(msg)
	updated, cmd, handled := m.handleQuitMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleKeyboardOwnershipMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleQuitShortcutMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleWindowSize(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleOverlayMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	updated, cmd, handled = m.handleShellMessage(msg)
	m = updated.(shellModel)
	if handled {
		return updated, cmd
	}
	return m.updateActiveRoute(msg)
}

func (m shellModel) activateCurrentRouteCmd() tea.Cmd {
	active := m.currentRouteID()
	activeSessionID := m.activeSessionID
	inspectorVisible := m.inspectorOn
	preferredMode := normalizePresentationMode(m.preferredMode)
	return func() tea.Msg {
		return routeActivatedMsg{RouteID: active, ActiveSessionID: activeSessionID, InspectorVisible: inspectorVisible, InspectorSet: true, PreferredMode: preferredMode}
	}
}

func (m shellModel) updateActiveRoute(msg tea.Msg) (tea.Model, tea.Cmd) {
	active := m.routeModels[m.currentRouteID()]
	if active == nil {
		return m, nil
	}
	updated, cmd := active.Update(msg)
	m.routeModels[m.currentRouteID()] = updated
	return m, cmd
}

func (m *shellModel) publishShellPreferencesToCurrentRoute() {
	activeID := m.currentRouteID()
	active := m.routeModels[activeID]
	if active == nil {
		return
	}
	updated, _ := active.Update(routeShellPreferencesMsg{RouteID: activeID, InspectorVisible: m.inspectorOn, PreferredMode: normalizePresentationMode(m.preferredMode)})
	m.routeModels[activeID] = updated
}

func (m shellModel) activeShellSurface() routeSurface {
	surface, _ := m.activeShellSurfacePlan()
	return surface
}

func (m shellModel) activeShellSurfacePlan() (routeSurface, shellLayoutPlan) {
	active := m.routeModels[m.currentRouteID()]
	if active == nil {
		surface := routeSurface{
			Regions: routeSurfaceRegions{
				Main: routeSurfaceRegion{Body: "Route not available"},
			},
			Capabilities: routeSurfaceCapabilities{},
			Chrome:       routeSurfaceChrome{Breadcrumbs: []string{"Home", string(m.currentRouteID())}},
		}
		return surface, m.planShellLayout(surface)
	}
	baseCtx := routeShellContext{Width: m.width, Height: m.availableShellHeight(), Focus: m.focus, Focused: m.focusedRouteRegion(), Breakpoint: m.breakpoint(), Render: routeShellRenderPreferences{PreferredPresentation: normalizePresentationMode(m.preferredMode), ThemePreset: normalizeThemePreset(m.themePreset)}}
	surface := active.ShellSurface(baseCtx)
	layout := m.planShellLayout(surface)
	ctx := baseCtx
	ctx.Regions = layout.Regions
	ctx.Breakpoint = layout.Breakpoint
	surface = m.withLocationChrome(active.ShellSurface(ctx))
	return surface, layout
}

func (m shellModel) availableShellHeight() int {
	_, viewportHeight := normalizedShellViewport(m.width, m.height)
	if viewportHeight <= 0 {
		return viewportHeight
	}
	if overlayHeight := m.activeOverlayHeight(viewportHeight); overlayHeight > 0 {
		available := viewportHeight - overlayHeight
		if available < 1 {
			return 1
		}
		return available
	}
	return viewportHeight
}

func (m shellModel) focusTraversalLayout() shellLayoutPlan {
	active := m.routeModels[m.currentRouteID()]
	if active == nil {
		return m.planShellLayout(routeSurface{})
	}
	ctx := routeShellContext{Width: m.width, Height: m.height, Focus: m.focus, Focused: m.focusedRouteRegion(), Breakpoint: m.breakpoint(), Render: routeShellRenderPreferences{PreferredPresentation: normalizePresentationMode(m.preferredMode), ThemePreset: normalizeThemePreset(m.themePreset)}}
	return m.planShellLayout(active.ShellSurface(ctx))
}

func (m shellModel) activeOverlayID() shellOverlayID {
	switch {
	case m.palette.IsOpen():
		return overlayIDQuickJump
	case m.sessions.IsOpen():
		return overlayIDSessions
	case m.leader.Active():
		return overlayIDLeader
	case m.quitConfirm.active:
		return overlayIDQuitConfirm
	case m.narrowSidebarOn && m.breakpoint() == shellBreakpointNarrow:
		return overlayIDSidebar
	case m.narrowInspectOn && m.breakpoint() == shellBreakpointNarrow:
		return overlayIDInspector
	default:
		return ""
	}
}

func (m shellModel) overlayFrameCacheable() bool {
	switch m.activeOverlayID() {
	case overlayIDQuickJump, overlayIDSessions, overlayIDLeader, overlayIDQuitConfirm:
		return true
	default:
		return false
	}
}

func (m *shellModel) invalidateOverlayFrameCache() {
	if m.overlayFrameCache == nil {
		return
	}
	*m.overlayFrameCache = shellOverlayFrameCache{}
}

func (m *shellModel) invalidatePaletteCache() {
	m.paletteCache = nil
}

func (m shellModel) paletteImmediateEntries() []paletteEntry {
	return m.buildPaletteCommandEntries()
}

func (m shellModel) loadPaletteEntriesCmd(request uint64) tea.Cmd {
	commands := append([]paletteEntry(nil), m.buildPaletteCommandEntries()...)
	indexSnapshot := m.objectIndex.clone()
	activeSurfaceEntries := append([]paletteEntry(nil), m.buildActiveSurfacePaletteEntries()...)
	actionCenterEntries := append([]paletteEntry(nil), m.buildActionCenterPaletteEntries()...)
	return func() tea.Msg {
		entries := make([]paletteEntry, 0, len(commands)+len(activeSurfaceEntries)+len(actionCenterEntries)+64)
		entries = append(entries, commands...)
		entries = append(entries, activeSurfaceEntries...)
		entries = append(entries, actionCenterEntries...)
		entries = append(entries, buildPaletteDiscoverabilityEntries(indexSnapshot, len(entries)+1)...)
		return shellPaletteEntriesLoadedMsg{request: request, entries: entries}
	}
}

func (m shellModel) loadPaletteFilterCmd(request uint64) tea.Cmd {
	entriesVersion := m.palette.entriesVersion
	query := m.palette.query
	normalizedEntries := append([]string(nil), m.palette.normalizedEntries...)
	priorNeedle := m.palette.appliedNeedle
	priorMatches := append([]int(nil), m.palette.matchIndexes...)
	return func() tea.Msg {
		return buildPaletteFilterResult(request, entriesVersion, query, normalizedEntries, priorNeedle, priorMatches)
	}
}

func (m shellModel) focusedRouteRegion() routeRegionFocus {
	if m.palette.IsOpen() || m.sessions.IsOpen() || m.leader.Active() || m.quitConfirm.active {
		return routeRegionOverlay
	}
	if m.narrowInspectOn {
		return routeRegionInspector
	}
	if m.focus == focusInspector {
		return routeRegionInspector
	}
	return routeRegionMain
}

func (m shellModel) breakpoint() shellBreakpoint {
	return shellBreakpointForWidth(m.width)
}

func shellBreakpointForWidth(width int) shellBreakpoint {
	if width <= 0 {
		return shellBreakpointWide
	}
	if width < shellMediumMinWidth {
		return shellBreakpointNarrow
	}
	if width < shellWideMinWidth {
		return shellBreakpointMedium
	}
	return shellBreakpointWide
}

func (m shellModel) effectiveSidebarVisible() bool {
	if m.breakpoint() == shellBreakpointNarrow {
		return false
	}
	if m.sidebarFolded {
		return false
	}
	return m.sidebarVisible
}

func (m shellModel) navigationSurfaceVisible() bool {
	if m.effectiveSidebarVisible() {
		return true
	}
	return m.breakpoint() == shellBreakpointNarrow && m.narrowSidebarOn
}

func (m shellModel) shouldShowInspector(surface routeSurface) bool {
	return m.planShellLayout(surface).InspectorVisible
}

func (m shellModel) mouseCaptureCmd() tea.Cmd {
	if m.selectionMode {
		return tea.DisableMouse
	}
	return tea.EnableMouseCellMotion
}
