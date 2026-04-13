package brokerapi

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

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
