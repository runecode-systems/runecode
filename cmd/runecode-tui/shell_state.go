package main

import "strings"

func (m shellModel) activeLocalEntryStateReason() (string, bool) {
	if m.commandMode.Active() {
		return "command entry", true
	}
	active := m.routeModels[m.currentRouteID()]
	switch typed := active.(type) {
	case chatRouteModel:
		if typed.composeOn {
			return "chat compose", true
		}
	case providerSetupRouteModel:
		if typed.entryActive {
			return "provider secret entry", true
		}
	}
	switch m.keyboardOwnership() {
	case routeKeyboardOwnershipTextEntry:
		return "local text entry", true
	case routeKeyboardOwnershipExclusiveLocalCapture:
		return "local captured entry", true
	default:
		return "", false
	}
}

func (m shellModel) keyboardOwnership() routeKeyboardOwnership {
	active := m.routeModels[m.currentRouteID()]
	ownershipModel, ok := active.(routeKeyboardOwnershipModel)
	if !ok {
		return routeKeyboardOwnershipNormal
	}
	ownership := ownershipModel.KeyboardOwnership()
	switch ownership {
	case routeKeyboardOwnershipTextEntry, routeKeyboardOwnershipExclusiveLocalCapture:
		return ownership
	default:
		return routeKeyboardOwnershipNormal
	}
}

func (m shellModel) shellPowerKeysAllowed() bool {
	return m.keyboardOwnership() == routeKeyboardOwnershipNormal
}

func (m shellModel) shellFocusTraversalAllowed() bool {
	return m.keyboardOwnership() != routeKeyboardOwnershipExclusiveLocalCapture
}

func (m shellModel) currentRouteID() routeID {
	rid := m.location.Primary.RouteID
	if rid == "" {
		return routeChat
	}
	return rid
}

func (m shellModel) currentLocation() shellWorkbenchLocation {
	loc := m.location
	if loc.Primary.RouteID == "" {
		loc.Primary.RouteID = routeChat
		if loc.Primary.Object.Kind == "" || loc.Primary.Object.ID == "" {
			loc.Primary.Object = workbenchObjectRef{Kind: "route", ID: string(routeChat)}
		}
	}
	if loc.Primary.Object.Kind == "" || loc.Primary.Object.ID == "" {
		loc.Primary.Object = workbenchObjectRef{Kind: "route", ID: string(loc.Primary.RouteID)}
	}
	return loc
}

func (m shellModel) withLocationChrome(surface routeSurface) routeSurface {
	breadcrumbs := []string{"Home", m.routeLabel(m.currentRouteID())}
	loc := m.currentLocation()
	if ref := strings.TrimSpace(loc.Primary.Object.ID); ref != "" && strings.TrimSpace(strings.ToLower(loc.Primary.Object.Kind)) != "route" {
		breadcrumbs = append(breadcrumbs, ref)
	}
	if loc.Inspector != nil {
		if ref := strings.TrimSpace(loc.Inspector.Object.ID); ref != "" {
			breadcrumbs = append(breadcrumbs, "Inspect", ref)
		}
	}
	if len(surface.Chrome.Breadcrumbs) > len(breadcrumbs) {
		breadcrumbs = surface.Chrome.Breadcrumbs
	}
	surface.Chrome.Breadcrumbs = breadcrumbs
	return surface
}
