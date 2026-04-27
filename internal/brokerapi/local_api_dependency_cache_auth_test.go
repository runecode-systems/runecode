package brokerapi

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDependencyFetchAuditIncludesPolicyAndAllowlistLinkage(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-deps-policy-audit"
	allowlistDigest := putTrustedDependencyFetchContextForRun(t, s, runID)

	req := dependencyFetchRegistryRequestForTest("req-policy-audit", runID, "alpha")

	_, errResp := s.HandleDependencyFetchRegistry(context.Background(), req, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}

	events := auditEventsByType(t, s, "dependency_registry_fetch")
	requireDependencyFetchPolicyAudit(t, events, allowlistDigest)
}

func TestDependencyFetchRegistryAuthIsBrokerInternalAndRunnerSurfaceClean(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedDependencyFetchContextForRun(t, s, "run-deps")
	s.SetDependencyRegistryAuthSourceForTests(fakeDependencyRegistryAuthSource{leaseID: "lease-sensitive", expiresAt: time.Now().Add(5 * time.Minute).UTC()})
	fetcher := &leaseRecordingFetcher{payload: "leased-payload"}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	resp, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-auth-guard", "run-deps", "auth"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}
	if got := fetcher.lastLeasePosture.Load(); got != string(dependencyRegistryAuthPosturePublicNoAuth) {
		t.Fatalf("lease posture seen by fetcher = %q, want %q", got, dependencyRegistryAuthPosturePublicNoAuth)
	}
	if got := fetcher.lastLeaseID.Load(); got != "lease-sensitive" {
		t.Fatalf("lease id seen by fetcher = %q, want lease-sensitive", got)
	}
	requireNoAuthMaterialInResponse(t, resp)
	requireNoAuthMaterialInAudit(t, auditEventsByType(t, s, "dependency_registry_fetch"))
}

func TestDependencyFetchRunnerFacingTypesDoNotExposeAuthMaterial(t *testing.T) {
	requireNoForbiddenAuthFields(t, []reflect.Type{
		reflect.TypeOf(DependencyFetchRequestObject{}),
		reflect.TypeOf(DependencyFetchBatchRequestObject{}),
		reflect.TypeOf(DependencyCacheEnsureRequest{}),
		reflect.TypeOf(DependencyCacheEnsureResponse{}),
		reflect.TypeOf(DependencyFetchRegistryRequest{}),
		reflect.TypeOf(DependencyFetchRegistryResponse{}),
		reflect.TypeOf(DependencyCacheHandoffRequest{}),
		reflect.TypeOf(DependencyCacheHandoffMetadata{}),
		reflect.TypeOf(DependencyCacheHandoffResponse{}),
	})
}

func TestDependencyFetchDeniedBeforeFetchWhenPolicyRejects(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-deps-policy-deny"
	denyEntry := trustedDependencyFetchAllowlistEntryForTests()
	denyEntry["permitted_operations"] = []any{"enable_dependency_fetch"}
	putTrustedDependencyFetchContextForRunWithAllowlistEntries(t, s, runID, []any{denyEntry})

	fetcher := &countingDependencyFetcher{payload: "should-not-fetch"}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	req := dependencyFetchRegistryRequestForTest("req-policy-deny", runID, "deny")

	_, errResp := s.HandleDependencyFetchRegistry(context.Background(), req, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleDependencyFetchRegistry succeeded, want policy denial")
	}
	if errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error code = %q, want broker_limit_policy_rejected", errResp.Error.Code)
	}
	if !strings.Contains(errResp.Error.Message, "decision outcome") {
		t.Fatalf("error message = %q, want decision outcome detail", errResp.Error.Message)
	}
	if got := fetcher.calls(); got != 0 {
		t.Fatalf("fetcher calls = %d, want 0 for pre-fetch denial", got)
	}
}

func TestDependencyFetchDeniedWhenPolicyContextUnavailableBeforeFetch(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	fetcher := &countingDependencyFetcher{payload: "should-not-fetch"}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	_, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-no-context", "run-deps-no-context", "ctx-missing"), RequestContext{})
	if errResp == nil {
		t.Fatal("HandleDependencyFetchRegistry succeeded, want policy context denial")
	}
	if errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error code = %q, want broker_limit_policy_rejected", errResp.Error.Code)
	}
	if got := fetcher.calls(); got != 0 {
		t.Fatalf("fetcher calls = %d, want 0 when policy context unavailable", got)
	}
}
