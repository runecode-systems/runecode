package main

import "fmt"

type navHitbox struct {
	RouteID routeID
	StartX  int
	EndX    int
}

type primaryNavModel struct {
	routes        []routeDefinition
	selectedIndex int
}

func newPrimaryNavModel(routes []routeDefinition) primaryNavModel {
	return primaryNavModel{routes: routes}
}

func (m primaryNavModel) Selected() routeDefinition {
	if len(m.routes) == 0 {
		return routeDefinition{}
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
	if m.selectedIndex >= len(m.routes) {
		m.selectedIndex = len(m.routes) - 1
	}
	return m.routes[m.selectedIndex]
}

func (m *primaryNavModel) SelectByRouteID(id routeID) {
	for i, r := range m.routes {
		if r.ID == id {
			m.selectedIndex = i
			return
		}
	}
}

func (m *primaryNavModel) MoveNext() {
	if len(m.routes) == 0 {
		return
	}
	m.selectedIndex = (m.selectedIndex + 1) % len(m.routes)
}

func (m *primaryNavModel) MovePrev() {
	if len(m.routes) == 0 {
		return
	}
	m.selectedIndex--
	if m.selectedIndex < 0 {
		m.selectedIndex = len(m.routes) - 1
	}
}

func (m primaryNavModel) Render(wide bool) (string, []navHitbox) {
	line := ""
	boxes := make([]navHitbox, 0, len(m.routes))
	currentX := 0
	for i, route := range m.routes {
		label := fmt.Sprintf("[%d %s]", route.Index, route.Label)
		if i == m.selectedIndex {
			label = ">" + label + "<"
		}
		if !wide {
			label = fmt.Sprintf("[%d] %s", route.Index, route.Label)
			if i == m.selectedIndex {
				label = ">" + label + "<"
			}
		}
		if line != "" {
			line += "  "
			currentX += 2
		}
		start := currentX
		line += label
		currentX += len(label)
		boxes = append(boxes, navHitbox{RouteID: route.ID, StartX: start, EndX: currentX - 1})
	}
	return line, boxes
}

func navRouteAtX(boxes []navHitbox, x int) (routeID, bool) {
	for _, box := range boxes {
		if x >= box.StartX && x <= box.EndX {
			return box.RouteID, true
		}
	}
	return "", false
}
