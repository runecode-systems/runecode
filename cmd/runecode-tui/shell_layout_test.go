package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type captureRouteModel struct {
	id      routeID
	title   string
	surface routeSurface
	ctx     routeShellContext
}

func (m *captureRouteModel) ID() routeID { return m.id }

func (m *captureRouteModel) Title() string { return m.title }

func (m *captureRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	_ = msg
	return m, nil
}

func (m *captureRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	_ = focus
	return "capture"
}

func (m *captureRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	m.ctx = ctx
	return m.surface
}

func TestShellLayoutPlannerUsesTypedInspectorCapabilityNotRenderedBody(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40

	surface := routeSurface{
		Regions: routeSurfaceRegions{
			Inspector: routeSurfaceRegion{Body: ""},
		},
		Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: true}},
	}

	plan := m.planShellLayout(surface)
	if !plan.InspectorVisible {
		t.Fatal("expected inspector visible from typed capability even with empty body")
	}

	surface.Capabilities.Inspector = routeInspectorCapability{Supported: false, Enabled: false}
	surface.Regions.Inspector.Body = "non-empty"
	plan = m.planShellLayout(surface)
	if plan.InspectorVisible {
		t.Fatal("expected inspector hidden when typed capability is unsupported")
	}
}

func TestActiveShellSurfaceProvidesPlannedRegionDimensionsAndRenderPreferences(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 44
	m.preferredMode = presentationStructured
	m.themePreset = themePresetDusk
	m.location.Primary = shellObjectLocation{RouteID: routeChat, Object: workbenchObjectRef{Kind: "route", ID: string(routeChat)}}

	route := &captureRouteModel{
		id:    routeChat,
		title: "Capture",
		surface: routeSurface{
			Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: true}},
		},
	}
	m.routeModels[routeChat] = route

	_ = m.activeShellSurface()
	updated := m.routeModels[routeChat].(*captureRouteModel)

	if updated.ctx.Regions.Main.Width <= 0 || updated.ctx.Regions.Main.Height <= 0 {
		t.Fatalf("expected planned non-zero main region, got %+v", updated.ctx.Regions.Main)
	}
	if updated.ctx.Render.PreferredPresentation != presentationStructured {
		t.Fatalf("expected preferred presentation %q, got %q", presentationStructured, updated.ctx.Render.PreferredPresentation)
	}
	if updated.ctx.Render.ThemePreset != themePresetDusk {
		t.Fatalf("expected theme preset %q, got %q", themePresetDusk, updated.ctx.Render.ThemePreset)
	}
}

func TestNarrowInspectorToggleBlockedWhenRouteHasNoInspectorCapability(t *testing.T) {
	m := newShellModel()
	m.width = 80
	m.location.Primary = shellObjectLocation{RouteID: routeDashboard, Object: workbenchObjectRef{Kind: "route", ID: string(routeDashboard)}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	shell := updated.(shellModel)
	if shell.narrowInspectOn {
		t.Fatal("expected narrow inspector overlay to remain closed")
	}
}

func TestShellLayoutPlannerClampsSecondaryPanesToViewportBudget(t *testing.T) {
	m := newShellModel()
	m.width = 90
	m.height = 40
	m.sidebarVisible = true
	m.inspectorOn = true
	m.sidebarFolded = false
	m.inspectorFolded = false
	m.sidebarRatio = 0.5
	m.inspectorRatio = 0.5

	surface := routeSurface{Capabilities: routeSurfaceCapabilities{Inspector: routeInspectorCapability{Supported: true, Enabled: true}}}
	plan := m.planShellLayout(surface)
	total := plan.Regions.Sidebar.Width + plan.Regions.Main.Width + plan.Regions.Inspector.Width
	if total > m.width {
		t.Fatalf("expected pane widths to fit viewport, got total=%d viewport=%d", total, m.width)
	}
	if plan.Regions.Main.Width < minimumMainPaneWidth(plan.Breakpoint) {
		t.Fatalf("expected main pane width >= minimum, got %d", plan.Regions.Main.Width)
	}
}

func TestShellLayoutPlannerBudgetsMainHeightFromShellChrome(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.height = 40

	plan := m.planShellLayout(routeSurface{})
	want := 40 - shellChromeReservedHeight() - 2
	if got := plan.Regions.Main.Height; got != want {
		t.Fatalf("expected main height=%d from viewport-shell chrome budget, got %d", want, got)
	}
	if got := plan.Regions.Bottom.Height; got != shellBottomStripHeight {
		t.Fatalf("expected bottom strip height=%d, got %d", shellBottomStripHeight, got)
	}
	if got := plan.Regions.Status.Height; got != shellStatusHeight {
		t.Fatalf("expected status height=%d, got %d", shellStatusHeight, got)
	}
}

func TestShellLayoutPlannerBudgetsModeTabsAsVerticalChrome(t *testing.T) {
	m := newShellModel()
	m.width = 120
	m.height = 40

	plan := m.planShellLayout(routeSurface{Actions: routeSurfaceActions{ModeTabs: []string{"rendered", "raw"}}})
	want := 40 - shellChromeReservedHeight() - 2 - 1
	if got := plan.Regions.Main.Height; got != want {
		t.Fatalf("expected mode tabs to reserve one line and main height=%d, got %d", want, got)
	}
}

func TestShellLayoutPlannerAccountsForPaneBorderHeight(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 32
	surface := m.activeShellSurface()
	plan := m.planShellLayout(surface)
	row := m.renderShellPanes(surface, plan)
	if got := lipgloss.Height(row); got > m.availableShellHeight()-shellBottomStripHeight-shellStatusHeight-shellFooterHeight {
		t.Fatalf("expected pane row to leave room for footer and lower chrome, got row height=%d", got)
	}
}
