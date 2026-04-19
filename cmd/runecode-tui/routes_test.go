package main

import "testing"

func TestShellRoutesQuickJumpIndicesAreSequentialAndUnique(t *testing.T) {
	routes := shellRoutes()
	seen := map[int]routeID{}

	for i, route := range routes {
		expected := i + 1
		if route.Index != expected {
			t.Fatalf("route %q has index %d, want %d", route.ID, route.Index, expected)
		}
		if prior, ok := seen[route.Index]; ok {
			t.Fatalf("duplicate quick-jump index %d for routes %q and %q", route.Index, prior, route.ID)
		}
		seen[route.Index] = route.ID
	}
}

func TestShellRoutesQuickJumpKeysAreSingleStrokeAndUnique(t *testing.T) {
	routes := shellRoutes()
	seen := map[string]routeID{}

	for _, route := range routes {
		key := routeQuickJumpKey(route)
		if len(key) != 1 {
			t.Fatalf("route %q has quick-jump key %q, want single-stroke key", route.ID, key)
		}
		if prior, ok := seen[key]; ok {
			t.Fatalf("duplicate quick-jump key %q for routes %q and %q", key, prior, route.ID)
		}
		seen[key] = route.ID
	}
}

func TestRouteByQuickJumpKeyResolvesSingleStrokeRoutes(t *testing.T) {
	routes := shellRoutes()

	gitSetup, ok := routeByQuickJumpKey("0", routes)
	if !ok {
		t.Fatal("expected quick-jump key 0 to resolve")
	}
	if gitSetup.ID != routeGitSetup {
		t.Fatalf("quick-jump key 0 resolved %q, want %q", gitSetup.ID, routeGitSetup)
	}

	gitRemote, ok := routeByQuickJumpKey("-", routes)
	if !ok {
		t.Fatal("expected quick-jump key - to resolve")
	}
	if gitRemote.ID != routeGitRemote {
		t.Fatalf("quick-jump key - resolved %q, want %q", gitRemote.ID, routeGitRemote)
	}
}

func TestShellRoutesExposeExpectedQuickJumpKeys(t *testing.T) {
	routes := shellRoutes()
	keys := map[routeID]string{}
	for _, route := range routes {
		keys[route.ID] = routeQuickJumpKey(route)
	}
	if got := keys[routeProviders]; got != "9" {
		t.Fatalf("model providers quick-jump key = %q, want 9", got)
	}
	if got := keys[routeGitSetup]; got != "0" {
		t.Fatalf("git setup quick-jump key = %q, want 0", got)
	}
	if got := keys[routeGitRemote]; got != "-" {
		t.Fatalf("git remote quick-jump key = %q, want -", got)
	}
}
