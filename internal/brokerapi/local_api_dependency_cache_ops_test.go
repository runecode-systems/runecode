package brokerapi

import "testing"

func requireDependencyCacheEnsureResult(t *testing.T, label string, resp DependencyCacheEnsureResponse, errResp *ErrorResponse, cacheOutcome string, registryRequests int) {
	t.Helper()
	if errResp != nil {
		t.Fatalf("%s HandleDependencyCacheEnsure error: %+v", label, errResp)
	}
	if resp.CacheOutcome != cacheOutcome {
		t.Fatalf("%s cache_outcome = %q, want %s", label, resp.CacheOutcome, cacheOutcome)
	}
	if resp.RegistryRequestCount != registryRequests {
		t.Fatalf("%s registry_request_count = %d, want %d", label, resp.RegistryRequestCount, registryRequests)
	}
}

func requireDependencyCacheEnsureAuditHit(t *testing.T, events []map[string]interface{}) {
	t.Helper()
	if len(events) < 2 {
		t.Fatalf("dependency_cache_ensure events = %d, want >= 2", len(events))
	}
	last := events[len(events)-1]
	if got, _ := last["gateway_role_kind"].(string); got != "dependency-fetch" {
		t.Fatalf("gateway_role_kind = %q, want dependency-fetch", got)
	}
	if got, _ := last["operation"].(string); got != "fetch_dependency" {
		t.Fatalf("operation = %q, want fetch_dependency", got)
	}
	if got, _ := last["audit_outcome"].(string); got != "succeeded" {
		t.Fatalf("audit_outcome = %q, want succeeded", got)
	}
	if got, _ := last["cache_outcome"].(string); got != "hit_exact" {
		t.Fatalf("audit cache_outcome = %q, want hit_exact", got)
	}
	bindings, ok := last["request_bindings"].([]any)
	if !ok || len(bindings) != 1 {
		t.Fatalf("request_bindings = %#v, want one entry", last["request_bindings"])
	}
	binding, ok := bindings[0].(map[string]any)
	if !ok {
		t.Fatalf("request_bindings[0] type = %T, want map[string]any", bindings[0])
	}
	if bound, _ := binding["request_payload_hash_bound"].(bool); !bound {
		t.Fatalf("request_payload_hash_bound = %v, want true", binding["request_payload_hash_bound"])
	}
}
