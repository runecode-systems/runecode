package brokerapi

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestDependencyFetchAuditIncludesPolicyAndAllowlistLinkage(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-deps-policy-audit"
	allowlistDigest := putTrustedDependencyFetchContextForRun(t, s, runID)

	req := dependencyFetchRegistryRequestForTest("req-policy-audit", runID, "alpha")
	requestHash, err := req.RequestHash.Identity()
	if err != nil {
		t.Fatalf("request hash identity error: %v", err)
	}
	decision := buildDependencyFetchPolicyDecision(requestHash)
	if err := s.RecordPolicyDecision(runID, "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	decisionHash := decisionDigestIdentity(decision)

	_, errResp := s.HandleDependencyFetchRegistry(context.Background(), req, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}

	events := auditEventsByType(t, s, "dependency_registry_fetch")
	requireDependencyFetchPolicyAudit(t, events, decision, decisionHash, allowlistDigest)
}

func TestDependencyFetchRegistryAuthIsBrokerInternalAndRunnerSurfaceClean(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
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
	})
}
