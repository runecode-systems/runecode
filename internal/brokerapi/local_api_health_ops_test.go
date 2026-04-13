package brokerapi

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
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

func seedSecretsReadinessState(t *testing.T, root string) {
	t.Helper()
	svc, err := secretsd.Open(root)
	if err != nil {
		t.Fatalf("secretsd.Open returned error: %v", err)
	}
	if _, err := svc.ImportSecret("secrets/prod/db", strings.NewReader("db-secret")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	lease, err := svc.IssueLease(secretsd.IssueLeaseRequest{SecretRef: "secrets/prod/db", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 120})
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	if _, err := svc.RenewLease(secretsd.RenewLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 120}); err != nil {
		t.Fatalf("RenewLease returned error: %v", err)
	}
	if _, err := svc.RevokeLease(secretsd.RevokeLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", Reason: "operator"}); err != nil {
		t.Fatalf("RevokeLease returned error: %v", err)
	}
	_, err = svc.IssueLease(secretsd.IssueLeaseRequest{SecretRef: "secrets/prod/missing", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 120})
	if !errors.Is(err, secretsd.ErrAccessDenied) {
		t.Fatalf("IssueLease missing secret error = %v, want ErrAccessDenied", err)
	}
}

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) returned error: %v", dir, err)
	}
	return resolved
}

func TestHandleReadinessGetProjectsModelGatewayPostureFromAllowlist(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-model-gateway-posture"
	_ = putTrustedPolicyContextForRun(t, s, runID, false)

	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-model-gateway",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if !resp.Readiness.ModelGatewayReady {
		t.Fatal("model_gateway_ready = false, want true")
	}
	if resp.Readiness.ModelGatewayHealthState != "ok" {
		t.Fatalf("model_gateway_health_state = %q, want ok", resp.Readiness.ModelGatewayHealthState)
	}
	if resp.Readiness.ModelGatewayPosture == nil {
		t.Fatal("model_gateway_posture_projection = nil, want projection")
	}
	if resp.Readiness.ModelGatewayPosture.ProjectionKind != "broker_projected" {
		t.Fatalf("projection_kind = %q, want broker_projected", resp.Readiness.ModelGatewayPosture.ProjectionKind)
	}
	if resp.Readiness.ModelGatewayPosture.ConfigurationState != "configured" {
		t.Fatalf("configuration_state = %q, want configured", resp.Readiness.ModelGatewayPosture.ConfigurationState)
	}
	if resp.Readiness.ModelGatewayPosture.EgressPolicyPosture != "allowlist_only" {
		t.Fatalf("egress_policy_posture = %q, want allowlist_only", resp.Readiness.ModelGatewayPosture.EgressPolicyPosture)
	}
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(readinessGetResponse) returned error: %v", err)
	}
}

func TestHandleReadinessGetProjectsDenyByDefaultWithoutModelAllowlist(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})

	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-model-gateway-deny",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if !resp.Readiness.ModelGatewayReady {
		t.Fatal("model_gateway_ready = false, want true")
	}
	if resp.Readiness.ModelGatewayHealthState != "ok" {
		t.Fatalf("model_gateway_health_state = %q, want ok", resp.Readiness.ModelGatewayHealthState)
	}
	if resp.Readiness.ModelGatewayPosture == nil {
		t.Fatal("model_gateway_posture_projection = nil, want projection")
	}
	if resp.Readiness.ModelGatewayPosture.ConfigurationState != "not_configured" {
		t.Fatalf("configuration_state = %q, want not_configured", resp.Readiness.ModelGatewayPosture.ConfigurationState)
	}
	if resp.Readiness.ModelGatewayPosture.EgressPolicyPosture != "deny_by_default" {
		t.Fatalf("egress_policy_posture = %q, want deny_by_default", resp.Readiness.ModelGatewayPosture.EgressPolicyPosture)
	}
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(readinessGetResponse) returned error: %v", err)
	}
}

func TestHandleReadinessGetProjectsNeutralNotConfiguredWhenNoModelGatewayEntries(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-no-model-gateway-entry"
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{trustedDependencyFetchAllowlistEntry()})
	if digest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload); digest == "" {
		t.Fatal("putTrustedPolicyArtifact returned empty digest")
	}

	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-no-model-entry",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	assertModelGatewayNotConfigured(t, resp)
}

func trustedDependencyFetchAllowlistEntry() map[string]any {
	return map[string]any{
		"schema_id":                   "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":              "0.1.0",
		"scope_kind":                  "gateway_destination",
		"gateway_role_kind":           "dependency-fetch",
		"destination":                 trustedModelGatewayDestination(),
		"permitted_operations":        []any{"fetch_dependency"},
		"allowed_egress_data_classes": []any{"spec_text"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16777216,
	}
}

func assertModelGatewayNotConfigured(t *testing.T, resp ReadinessGetResponse) {
	t.Helper()
	if !resp.Readiness.ModelGatewayReady {
		t.Fatal("model_gateway_ready = false, want true")
	}
	if resp.Readiness.ModelGatewayHealthState != "ok" {
		t.Fatalf("model_gateway_health_state = %q, want ok", resp.Readiness.ModelGatewayHealthState)
	}
	if resp.Readiness.ModelGatewayPosture == nil {
		t.Fatal("model_gateway_posture_projection = nil, want projection")
	}
	if resp.Readiness.ModelGatewayPosture.ConfigurationState != "not_configured" {
		t.Fatalf("configuration_state = %q, want not_configured", resp.Readiness.ModelGatewayPosture.ConfigurationState)
	}
	if got := resp.Readiness.ModelGatewayPosture.EgressPolicyPosture; got != "deny_by_default" {
		t.Fatalf("egress_policy_posture = %q, want deny_by_default", got)
	}
}

func TestHandleReadinessGetModelGatewayAllowlistDecodeFailureIsDegraded(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-model-gateway-degraded"
	badAllowlistPayload := mustJSONBytes(t, map[string]any{
		"schema_id":       "runecode.protocol.v0.PolicyAllowlist",
		"schema_version":  "0.1.0",
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": "runecode.protocol.v0.GatewayScopeRule",
		"entries":         "not-an-array",
	})
	if digest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, badAllowlistPayload); digest == "" {
		t.Fatal("putTrustedPolicyArtifact returned empty digest")
	}

	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-degraded",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	if resp.Readiness.ModelGatewayHealthState != "degraded" {
		t.Fatalf("model_gateway_health_state = %q, want degraded", resp.Readiness.ModelGatewayHealthState)
	}
	if resp.Readiness.ModelGatewayReady {
		t.Fatal("model_gateway_ready = true, want false for degraded catalog")
	}
	if resp.Readiness.Ready {
		t.Fatal("ready = true, want false for degraded model gateway posture")
	}
	if resp.Readiness.ModelGatewayPosture != nil {
		payload, marshalErr := json.Marshal(resp.Readiness.ModelGatewayPosture)
		if marshalErr != nil {
			t.Fatalf("json.Marshal posture returned error: %v", marshalErr)
		}
		t.Fatalf("model_gateway_posture_projection = %s, want nil", string(payload))
	}
}

func TestHandleReadinessGetRoleAgnosticEntryDoesNotMarkModelGatewayConfigured(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-model-gateway-role-agnostic"
	entry := trustedModelGatewayAllowlistEntry()
	delete(entry, "gateway_role_kind")
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{entry})
	if digest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload); digest == "" {
		t.Fatal("putTrustedPolicyArtifact returned empty digest")
	}

	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-role-agnostic",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	assertModelGatewayNotConfigured(t, resp)
}

func TestHandleAuditVerificationGetAppliesDefaultViewLimit(t *testing.T) {
	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}

	resp, errResp := service.HandleAuditVerificationGet(context.Background(), AuditVerificationGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditVerificationGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-verify-clamped",
		ViewLimit:     0,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditVerificationGet returned error: %+v", errResp)
	}
	if len(resp.Views) > 50 {
		t.Fatalf("views length = %d, want <= 50", len(resp.Views))
	}
	if err := service.validateResponse(resp, auditVerificationGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditVerificationGetResponse) returned error: %v", err)
	}
}

func TestHandleReadinessGetModelGatewayEntryQualificationAuthRoleNotConfigured(t *testing.T) {
	entry := map[string]any{
		"schema_id":                   "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":              "0.1.0",
		"scope_kind":                  "gateway_destination",
		"gateway_role_kind":           "auth-gateway",
		"destination":                 trustedModelGatewayDestination(),
		"permitted_operations":        []any{"refresh_auth_token"},
		"allowed_egress_data_classes": []any{"spec_text"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16777216,
	}
	resp := readinessFromSingleAllowlistEntry(t, entry)
	assertModelGatewayNotConfigured(t, resp)
}

func TestHandleReadinessGetModelGatewayEntryQualificationMissingInvokeModelNotConfigured(t *testing.T) {
	entry := map[string]any{
		"schema_id":                   "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":              "0.1.0",
		"scope_kind":                  "gateway_destination",
		"gateway_role_kind":           "model-gateway",
		"destination":                 trustedModelGatewayDestination(),
		"permitted_operations":        []any{"change_allowlist"},
		"allowed_egress_data_classes": []any{"spec_text"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16777216,
	}
	resp := readinessFromSingleAllowlistEntry(t, entry)
	assertModelGatewayNotConfigured(t, resp)
}

func TestHandleReadinessGetModelGatewayEntryQualificationInvokeModelConfigured(t *testing.T) {
	resp := readinessFromSingleAllowlistEntry(t, trustedModelGatewayAllowlistEntry())
	if got := resp.Readiness.ModelGatewayPosture.ConfigurationState; got != "configured" {
		t.Fatalf("configuration_state = %q, want configured", got)
	}
	if got := resp.Readiness.ModelGatewayPosture.EgressPolicyPosture; got != "allowlist_only" {
		t.Fatalf("egress_policy_posture = %q, want allowlist_only", got)
	}
}

func readinessFromSingleAllowlistEntry(t *testing.T, entry map[string]any) ReadinessGetResponse {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-qualify"
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{entry})
	if digest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload); digest == "" {
		t.Fatal("putTrustedPolicyArtifact returned empty digest")
	}
	resp, errResp := s.HandleReadinessGet(context.Background(), ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-readiness-qualify",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleReadinessGet returned error: %+v", errResp)
	}
	return resp
}

func TestHandleAuditTimelineProjectsSchemaAlignedViews(t *testing.T) {
	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	resp, errResp := service.HandleAuditTimeline(context.Background(), AuditTimelineRequest{
		SchemaID:      "runecode.protocol.v0.AuditTimelineRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-timeline",
		Limit:         10,
		Order:         "operational_seq_desc",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditTimeline returned error: %+v", errResp)
	}
	if len(resp.Views) == 0 {
		t.Fatal("timeline views empty")
	}
	if resp.Views[0].Summary == "" {
		t.Fatal("timeline summary empty")
	}
	if len(resp.Views[0].LinkedReferences) == 0 {
		t.Fatal("timeline linked_references empty")
	}
	if err := service.validateResponse(resp, auditTimelineResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditTimelineResponse) returned error: %v", err)
	}
}

func TestHandleAuditTimelineCursorRoundTripSupportsShortEncodedValues(t *testing.T) {
	encoded, err := encodeCursor(pageCursor{Offset: 1})
	if err != nil {
		t.Fatalf("encodeCursor returned error: %v", err)
	}
	if len(encoded) >= 32 {
		t.Fatalf("encoded cursor length = %d, expected short cursor for regression coverage", len(encoded))
	}
	resp := AuditTimelineResponse{
		SchemaID:      "runecode.protocol.v0.AuditTimelineResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-cursor",
		Order:         "operational_seq_asc",
		Views: []AuditTimelineViewEntry{{
			RecordDigest: digestChar("a"),
			Summary:      "Audit record projected for timeline.",
		}},
		NextCursor: encoded,
	}
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	if err := s.validateResponse(resp, auditTimelineResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(short next_cursor) returned error: %v", err)
	}
}

func TestMergeVerificationStatusTreatsInfoAsOK(t *testing.T) {
	if got := mergeVerificationStatus("", trustpolicy.AuditVerificationSeverityInfo); got != "ok" {
		t.Fatalf("mergeVerificationStatus(info from empty) = %q, want ok", got)
	}
	if got := mergeVerificationStatus("degraded", trustpolicy.AuditVerificationSeverityInfo); got != "degraded" {
		t.Fatalf("mergeVerificationStatus(info from degraded) = %q, want degraded", got)
	}
	if got := mergeVerificationStatus("failed", trustpolicy.AuditVerificationSeverityInfo); got != "failed" {
		t.Fatalf("mergeVerificationStatus(info from failed) = %q, want failed", got)
	}
	if got := mergeVerificationStatus("", "future_unknown"); got != "degraded" {
		t.Fatalf("mergeVerificationStatus(unknown from empty) = %q, want degraded", got)
	}
}
