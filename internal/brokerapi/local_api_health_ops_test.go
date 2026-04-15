package brokerapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleReadinessGetProjectsHealthySecretsPosture(t *testing.T) {
	secretsRoot := filepath.Join(canonicalTempDir(t), "secrets-state")
	t.Setenv("RUNE_SECRETS_STATE_ROOT", secretsRoot)
	seedSecretsReadinessState(t, secretsRoot)

	s := newBrokerAPIServiceForTests(t, APIConfig{})
	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-secrets-healthy",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if !resp.Readiness.SecretsReady {
		t.Fatal("secrets_ready = false, want true")
	}
	if resp.Readiness.SecretsHealthState != "ok" {
		t.Fatalf("secrets_health_state = %q, want ok", resp.Readiness.SecretsHealthState)
	}
	if resp.Readiness.SecretsOperationalMetrics == nil {
		t.Fatal("secrets_operational_metrics = nil, want metrics")
	}
	assertSecretsOperationalMetrics(t, resp.Readiness.SecretsOperationalMetrics, SecretsOperationalMetrics{
		LeaseIssueCount:  1,
		LeaseRenewCount:  1,
		LeaseRevokeCount: 1,
		LeaseDeniedCount: 1,
		ActiveLeaseCount: 0,
	})
	if resp.Readiness.SecretsStoragePosture != nil {
		t.Fatal("secrets_storage_posture present, want nil until encrypted custody posture is projected")
	}
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(readinessGetResponse) returned error: %v", err)
	}
}

func assertSecretsOperationalMetrics(t *testing.T, got *SecretsOperationalMetrics, want SecretsOperationalMetrics) {
	t.Helper()
	if got.LeaseIssueCount != want.LeaseIssueCount {
		t.Fatalf("lease_issue_count = %d, want %d", got.LeaseIssueCount, want.LeaseIssueCount)
	}
	if got.LeaseRenewCount != want.LeaseRenewCount {
		t.Fatalf("lease_renew_count = %d, want %d", got.LeaseRenewCount, want.LeaseRenewCount)
	}
	if got.LeaseRevokeCount != want.LeaseRevokeCount {
		t.Fatalf("lease_revoke_count = %d, want %d", got.LeaseRevokeCount, want.LeaseRevokeCount)
	}
	if got.LeaseDeniedCount != want.LeaseDeniedCount {
		t.Fatalf("lease_denied_count = %d, want %d", got.LeaseDeniedCount, want.LeaseDeniedCount)
	}
	if got.ActiveLeaseCount != want.ActiveLeaseCount {
		t.Fatalf("active_lease_count = %d, want %d", got.ActiveLeaseCount, want.ActiveLeaseCount)
	}
}

func TestHandleReadinessGetSecretsUnavailableFailsClosed(t *testing.T) {
	root := t.TempDir()
	blockedRoot := filepath.Join(root, "blocked-root")
	if err := os.WriteFile(blockedRoot, []byte("not-a-directory"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("RUNE_SECRETS_STATE_ROOT", blockedRoot)

	s := newBrokerAPIServiceForTests(t, APIConfig{})
	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-secrets-unavailable",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if resp.Readiness.SecretsReady {
		t.Fatal("secrets_ready = true, want false")
	}
	if resp.Readiness.SecretsHealthState != "degraded" {
		t.Fatalf("secrets_health_state = %q, want degraded", resp.Readiness.SecretsHealthState)
	}
	if resp.Readiness.SecretsOperationalMetrics != nil {
		t.Fatal("secrets_operational_metrics present, want nil")
	}
	if resp.Readiness.SecretsStoragePosture != nil {
		t.Fatal("secrets_storage_posture present, want nil")
	}
	if resp.Readiness.Ready {
		t.Fatal("ready = true, want false when secrets unavailable")
	}
	if !resp.Readiness.ModelGatewayReady || resp.Readiness.ModelGatewayHealthState != "ok" {
		t.Fatalf("model_gateway readiness regression: ready=%t health=%q", resp.Readiness.ModelGatewayReady, resp.Readiness.ModelGatewayHealthState)
	}
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(readinessGetResponse) returned error: %v", err)
	}
}

func TestHandleReadinessGetSecretsInvalidStateFailsClosedAndDegraded(t *testing.T) {
	secretsRoot := filepath.Join(canonicalTempDir(t), "secrets-invalid")
	if err := os.MkdirAll(filepath.Join(secretsRoot, "secrets"), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsRoot, "state.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("RUNE_SECRETS_STATE_ROOT", secretsRoot)

	s := newBrokerAPIServiceForTests(t, APIConfig{})
	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-secrets-invalid",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if resp.Readiness.SecretsReady {
		t.Fatal("secrets_ready = true, want false")
	}
	if resp.Readiness.SecretsHealthState != "degraded" {
		t.Fatalf("secrets_health_state = %q, want degraded", resp.Readiness.SecretsHealthState)
	}
	if resp.Readiness.Ready {
		t.Fatal("ready = true, want false when secrets state invalid")
	}
}
