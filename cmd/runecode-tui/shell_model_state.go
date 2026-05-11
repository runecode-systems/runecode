package main

import (
	"os"
	"strings"
)

func newShellModel() shellModel {
	return newShellModelWithWorkbenchStore(nil)
}

func newShellModelWithWorkbenchStore(store shellWorkbenchStateStore) shellModel {
	routes := shellRoutes()
	models := newRouteModels(routes)
	defaultRoute := routeChat
	commands := defaultShellCommandRegistry()
	actions := newShellActionGraph(routes, commands)
	leaderBindings := actions.leaderBindings(shellModel{})
	workbench := resolveShellWorkbenchStore(store)
	scope := logicalBrokerTargetKey()
	primeShellWorkbenchState(workbench, scope, defaultRoute)
	appTheme = newTheme(themePresetDark)
	m := newShellModelState(routes, models, commands, actions, leaderBindings, workbench, scope, defaultRoute)
	_ = m.setLeaderKey(m.leaderKeyConfig)
	m.restoreWorkbenchState()
	m.syncSidebarCursorToLocation()
	return m
}

func resolveShellWorkbenchStore(store shellWorkbenchStateStore) shellWorkbenchStateStore {
	if store != nil {
		return store
	}
	binaryPath := strings.ToLower(strings.TrimSpace(os.Args[0]))
	if forceMemoryWorkbenchState || strings.HasSuffix(binaryPath, ".test") || strings.HasSuffix(binaryPath, ".test.exe") {
		return &memoryWorkbenchStateStore{}
	}
	return newDefaultWorkbenchStateStore()
}

func primeShellWorkbenchState(workbench shellWorkbenchStateStore, scope string, defaultRoute routeID) {
	initialState := workbenchLocalState{SidebarVisible: true, InspectorVisible: true, InspectorMode: presentationRendered, ThemePreset: themePresetDark, LastRouteID: defaultRoute, ViewedActivity: map[string]string{}, LastSessionByWS: map[string]string{}, SidebarPaneRatio: 0.22, InspectorPaneRatio: 0.30}
	if existing := workbench.Read(scope); isZeroWorkbenchState(existing) {
		workbench.Write(scope, initialState)
	}
}

func newShellModelState(routes []routeDefinition, models map[routeID]routeModel, commands shellCommandRegistry, actions shellActionGraph, leaderBindings []shellLeaderBinding, workbench shellWorkbenchStateStore, scope string, defaultRoute routeID) shellModel {
	return shellModel{
		keys:           defaultShellKeyMap(),
		routes:         routes,
		nav:            newPrimaryNavModel(routes),
		palette:        newPaletteModel(nil),
		sessions:       newSessionSwitcherModel(),
		focus:          focusNav,
		client:         newLocalBrokerClient(),
		focusManager:   newShellFocusManager(focusNav),
		overlayManager: shellOverlayManager{},
		commands:       commands,
		actions:        actions,
		clipboard:      newShellClipboardService(),
		workbench:      workbench,
		workbenchScope: scope,
		toasts:         newShellToastService(),
		routeModels:    models,
		location: shellWorkbenchLocation{
			Primary: shellObjectLocation{RouteID: defaultRoute, Object: workbenchObjectRef{Kind: "route", ID: string(defaultRoute)}},
		},
		sidebarVisible:          true,
		inspectorOn:             true,
		themePreset:             themePresetDark,
		preferredMode:           presentationRendered,
		sidebarRatio:            0.22,
		inspectorRatio:          0.30,
		sessionLoading:          true,
		pinnedSessions:          map[string]struct{}{},
		lastSessionByWS:         map[string]string{},
		recentObjects:           nil,
		sessionWorkspace:        map[string]string{},
		viewedActivity:          map[string]string{},
		watch:                   newShellWatchManager(),
		objectIndex:             newShellDiscoverabilityIndex(routes),
		overlayReturn:           focusContent,
		leader:                  newShellLeaderState(leaderBindings),
		leaderBindingsSignature: shellLeaderBindingsSignature(leaderBindings),
		leaderKeyConfig:         "space",
		overlayFrameCache:       &shellOverlayFrameCache{},
		paletteCache:            nil,
	}
}
