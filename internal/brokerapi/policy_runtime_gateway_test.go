package brokerapi

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestPolicyRuntimeGatewayDeniesWhenDNSResolvesPrivateRange(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-gateway-private-range"
	putTrustedModelGatewayContextForRun(t, s, runID, []any{trustedModelGatewayAllowlistEntry()})
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"10.0.0.24"}}}

	decision, err := s.EvaluateAction(runID, trustedModelGatewayInvokeAction(t, "model.example.com", 1, "admission"))
	if err != nil {
		t.Fatalf("EvaluateAction returned error: %v", err)
	}
	if decision.DecisionOutcome != policyengine.DecisionDeny {
		t.Fatalf("decision_outcome = %q, want deny", decision.DecisionOutcome)
	}
	if got, _ := decision.Details["reason"].(string); got != "runtime_gateway_dns_rebinding_or_private_ip_blocked" {
		t.Fatalf("reason = %q, want runtime_gateway_dns_rebinding_or_private_ip_blocked", got)
	}
}

func TestPolicyRuntimeGatewayDeniesEscapingPathPrefixAtRuntime(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-gateway-path-prefix"
	entry := trustedModelGatewayAllowlistEntry()
	destination := entry["destination"].(map[string]any)
	destination["canonical_path_prefix"] = "/v1"
	putTrustedModelGatewayContextForRun(t, s, runID, []any{entry})
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"93.184.216.34"}}}

	action := trustedModelGatewayInvokeAction(t, "model.example.com", 1, "admission")
	action.ActionPayload["destination_ref"] = "model.example.com/v11/chat/completions"
	decision, err := s.EvaluateAction(runID, action)
	if err != nil {
		t.Fatalf("EvaluateAction returned error: %v", err)
	}
	if decision.DecisionOutcome != policyengine.DecisionDeny {
		t.Fatalf("decision_outcome = %q, want deny", decision.DecisionOutcome)
	}
}

func TestPolicyRuntimeGatewayQuotaAdmissionAndStreamEnforced(t *testing.T) {
	cfg := APIConfig{GatewayQuota: GatewayQuotaLimits{MaxRequestUnits: 1, MaxStreamedBytes: 1500}}
	s := newBrokerAPIServiceForTests(t, cfg)
	setupGatewayRuntimeTestContext(t, s, "run-gateway-quota")
	assertGatewayQuotaAdmissionEnforced(t, s, "run-gateway-quota")
	setupGatewayRuntimeTestContext(t, s, "run-gateway-stream-quota")
	assertGatewayStreamQuotaEnforced(t, s, "run-gateway-stream-quota")
}

func setupGatewayRuntimeTestContext(t *testing.T, s *Service, runID string) {
	t.Helper()
	putTrustedModelGatewayContextForRun(t, s, runID, []any{trustedModelGatewayAllowlistEntry()})
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"93.184.216.34"}}}
}

func assertGatewayQuotaAdmissionEnforced(t *testing.T, s *Service, runID string) {
	t.Helper()
	allowAdmission, err := s.EvaluateAction(runID, trustedModelGatewayInvokeAction(t, "model.example.com", 1, "admission"))
	if err != nil {
		t.Fatalf("EvaluateAction(admission allow) returned error: %v", err)
	}
	if allowAdmission.DecisionOutcome != policyengine.DecisionAllow {
		t.Fatalf("admission decision_outcome = %q, want allow", allowAdmission.DecisionOutcome)
	}

	denyAdmission, err := s.EvaluateAction(runID, trustedModelGatewayInvokeAction(t, "model.example.com", 1, "admission"))
	if err != nil {
		t.Fatalf("EvaluateAction(admission deny) returned error: %v", err)
	}
	if denyAdmission.DecisionOutcome != policyengine.DecisionDeny {
		t.Fatalf("admission deny decision_outcome = %q, want deny", denyAdmission.DecisionOutcome)
	}
	if got, _ := denyAdmission.Details["reason"].(string); got != "quota_admission_limit_exceeded_request_units" {
		t.Fatalf("admission deny reason = %q, want quota_admission_limit_exceeded_request_units", got)
	}
}

func assertGatewayStreamQuotaEnforced(t *testing.T, s *Service, runID string) {
	t.Helper()
	allowStream, err := s.EvaluateAction(runID, trustedModelGatewayInvokeAction(t, "model.example.com", 0, "stream"))
	if err != nil {
		t.Fatalf("EvaluateAction(stream allow) returned error: %v", err)
	}
	if allowStream.DecisionOutcome != policyengine.DecisionAllow {
		t.Fatalf("stream allow decision_outcome = %q, want allow", allowStream.DecisionOutcome)
	}
	denyStream, err := s.EvaluateAction(runID, trustedModelGatewayInvokeAction(t, "model.example.com", 0, "stream"))
	if err != nil {
		t.Fatalf("EvaluateAction(stream deny) returned error: %v", err)
	}
	if denyStream.DecisionOutcome != policyengine.DecisionDeny {
		t.Fatalf("stream deny decision_outcome = %q, want deny", denyStream.DecisionOutcome)
	}
	if got, _ := denyStream.Details["reason"].(string); got != "quota_stream_limit_exceeded_streamed_bytes" {
		t.Fatalf("stream deny reason = %q, want quota_stream_limit_exceeded_streamed_bytes", got)
	}
}

func TestPolicyRuntimeGatewayDeniesNegativeQuotaMeterValues(t *testing.T) {
	backend := newGatewayQuotaBackend()
	minusOne := int64(-1)
	reason, _, blocked := backend.evaluateAndApply("quota-key", gatewayQuotaContextPayload{
		QuotaProfileKind: "hybrid",
		Phase:            "admission",
		Meters: gatewayQuotaMetersPayload{
			RequestUnits: &minusOne,
		},
	})
	if !blocked {
		t.Fatal("blocked = false, want true")
	}
	if reason != "invalid_quota_meter_negative" {
		t.Fatalf("reason = %q, want invalid_quota_meter_negative", reason)
	}
}

func TestPolicyRuntimeGatewayStreamQuotaNotEnforcedWhenDisabled(t *testing.T) {
	backend := newGatewayQuotaBackend()
	backend.setLimits(GatewayQuotaLimits{MaxStreamedBytes: 100})
	streamed := int64(1000)
	reason, _, blocked := backend.evaluateAndApply("quota-key", gatewayQuotaContextPayload{
		QuotaProfileKind:    "hybrid",
		Phase:               "stream",
		EnforceDuringStream: false,
		Meters: gatewayQuotaMetersPayload{
			StreamedBytes: &streamed,
		},
	})
	if blocked {
		t.Fatalf("blocked = true, want false (reason=%q)", reason)
	}
}

func TestPolicyRuntimeGatewayConcurrencySupportsRelease(t *testing.T) {
	backend := newGatewayQuotaBackend()
	backend.setLimits(GatewayQuotaLimits{MaxConcurrencyUnits: 1})
	one := int64(1)
	minusOne := int64(-1)
	if reason, _, blocked := backend.evaluateAndApply("quota-key", gatewayQuotaContextPayload{QuotaProfileKind: "hybrid", Phase: "admission", Meters: gatewayQuotaMetersPayload{ConcurrencyUnits: &one}}); blocked {
		t.Fatalf("initial acquire blocked with reason %q", reason)
	}
	if reason, _, blocked := backend.evaluateAndApply("quota-key", gatewayQuotaContextPayload{QuotaProfileKind: "hybrid", Phase: "admission", Meters: gatewayQuotaMetersPayload{ConcurrencyUnits: &minusOne}}); blocked {
		t.Fatalf("release blocked with reason %q", reason)
	}
	if reason, _, blocked := backend.evaluateAndApply("quota-key", gatewayQuotaContextPayload{QuotaProfileKind: "hybrid", Phase: "admission", Meters: gatewayQuotaMetersPayload{ConcurrencyUnits: &one}}); blocked {
		t.Fatalf("reacquire blocked with reason %q", reason)
	}
	if got := backend.state["quota-key"].ConcurrencyUnits; got != 1 {
		t.Fatalf("concurrency_units = %d, want 1", got)
	}
}

func TestPolicyRuntimeGatewayConcurrencyUnderflowDenied(t *testing.T) {
	backend := newGatewayQuotaBackend()
	minusOne := int64(-1)
	reason, _, blocked := backend.evaluateAndApply("quota-key", gatewayQuotaContextPayload{QuotaProfileKind: "hybrid", Phase: "admission", Meters: gatewayQuotaMetersPayload{ConcurrencyUnits: &minusOne}})
	if !blocked {
		t.Fatal("blocked = false, want true")
	}
	if reason != "invalid_quota_meter_underflow" {
		t.Fatalf("reason = %q, want invalid_quota_meter_underflow", reason)
	}
}

func TestPolicyRuntimeGatewayQuotaStatePrunedByRunCompletionAndTTL(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	now := time.Date(2026, time.April, 13, 21, 0, 0, 0, time.UTC)
	s.gatewayQuota.now = func() time.Time { return now }
	one := int64(1)
	if reason, _, blocked := s.gatewayQuota.evaluateAndApply("run-prune:model:model_endpoint:model.example.com/v1:hybrid", gatewayQuotaContextPayload{QuotaProfileKind: "hybrid", Phase: "admission", Meters: gatewayQuotaMetersPayload{RequestUnits: &one}}); blocked {
		t.Fatalf("evaluateAndApply blocked with reason %q", reason)
	}
	if len(s.gatewayQuota.state) != 1 {
		t.Fatalf("quota state len = %d, want 1", len(s.gatewayQuota.state))
	}
	if err := s.SetRunStatus("run-prune", "completed"); err != nil {
		t.Fatalf("SetRunStatus returned error: %v", err)
	}
	if len(s.gatewayQuota.state) != 0 {
		t.Fatalf("quota state len after releaseRun = %d, want 0", len(s.gatewayQuota.state))
	}
	if reason, _, blocked := s.gatewayQuota.evaluateAndApply("stale:model:model_endpoint:model.example.com/v1:hybrid", gatewayQuotaContextPayload{QuotaProfileKind: "hybrid", Phase: "admission", Meters: gatewayQuotaMetersPayload{RequestUnits: &one}}); blocked {
		t.Fatalf("evaluateAndApply stale setup blocked with reason %q", reason)
	}
	now = now.Add(gatewayQuotaStateTTL + time.Minute)
	if reason, _, blocked := s.gatewayQuota.evaluateAndApply("fresh:model:model_endpoint:model.example.com/v1:hybrid", gatewayQuotaContextPayload{QuotaProfileKind: "hybrid", Phase: "admission", Meters: gatewayQuotaMetersPayload{RequestUnits: &one}}); blocked {
		t.Fatalf("evaluateAndApply fresh blocked with reason %q", reason)
	}
	if _, ok := s.gatewayQuota.state["stale:model:model_endpoint:model.example.com/v1:hybrid"]; ok {
		t.Fatal("stale quota key still present after TTL prune")
	}
}

func TestPolicyRuntimeGatewayReleasesConcurrencyOnTerminalOutcome(t *testing.T) {
	backend := newGatewayQuotaBackend()
	s := &modelGatewayRuntime{quota: backend}
	one := int64(1)
	payload := gatewayActionPayloadRuntime{
		GatewayRoleKind: "model-gateway",
		DestinationKind: "model_endpoint",
		DestinationRef:  "model.example.com/v1/chat/completions",
		QuotaContext: &gatewayQuotaContextPayload{
			QuotaProfileKind: "hybrid",
			Phase:            "admission",
			Meters: gatewayQuotaMetersPayload{
				ConcurrencyUnits: &one,
			},
		},
		AuditContext: &gatewayAuditContextPayload{Outcome: "succeeded"},
	}
	key := runtimeQuotaStateKey("run-1", payload)
	if reason, _, blocked := backend.evaluateAndApply(key, *payload.QuotaContext); blocked {
		t.Fatalf("evaluateAndApply blocked with reason %q", reason)
	}
	if got := backend.state[key].ConcurrencyUnits; got != 1 {
		t.Fatalf("concurrency_units before release = %d, want 1", got)
	}
	s.releaseQuotaUsage("run-1", payload)
	if got := backend.state[key].ConcurrencyUnits; got != 0 {
		t.Fatalf("concurrency_units after release = %d, want 0", got)
	}
}

func TestPolicyRuntimeGatewayDoesNotReleaseConcurrencyForInProgressStream(t *testing.T) {
	backend := newGatewayQuotaBackend()
	s := &modelGatewayRuntime{quota: backend}
	one := int64(1)
	payload := gatewayActionPayloadRuntime{
		GatewayRoleKind: "model-gateway",
		DestinationKind: "model_endpoint",
		DestinationRef:  "model.example.com/v1/chat/completions",
		QuotaContext: &gatewayQuotaContextPayload{
			QuotaProfileKind: "hybrid",
			Phase:            "stream",
			Meters: gatewayQuotaMetersPayload{
				ConcurrencyUnits: &one,
			},
		},
		AuditContext: &gatewayAuditContextPayload{Outcome: "streaming_in_progress"},
	}
	key := runtimeQuotaStateKey("run-2", payload)
	if reason, _, blocked := backend.evaluateAndApply(key, *payload.QuotaContext); blocked {
		t.Fatalf("evaluateAndApply blocked with reason %q", reason)
	}
	s.releaseQuotaUsage("run-2", payload)
	if got := backend.state[key].ConcurrencyUnits; got != 1 {
		t.Fatalf("concurrency_units after no-release outcome = %d, want 1", got)
	}
}

func TestPolicyRuntimeGatewayEmitsAuditWithCanonicalBindings(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-gateway-audit"
	putTrustedModelGatewayContextForRun(t, s, runID, []any{trustedModelGatewayAllowlistEntry()})
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"model.example.com": {"93.184.216.34"}}}

	decision, err := s.EvaluateAction(runID, trustedModelGatewayInvokeAction(t, "model.example.com", 1, "admission"))
	if err != nil {
		t.Fatalf("EvaluateAction returned error: %v", err)
	}
	if decision.DecisionOutcome != policyengine.DecisionAllow {
		t.Fatalf("decision_outcome = %q, want allow", decision.DecisionOutcome)
	}
	found := requireLatestModelEgressAuditDetails(t, s)
	assertCanonicalModelEgressBindings(t, found)
}

func requireLatestModelEgressAuditDetails(t *testing.T, s *Service) map[string]interface{} {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	var found map[string]interface{}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == "model_egress" {
			found = events[i].Details
			break
		}
	}
	if found == nil {
		t.Fatal("model_egress audit event not found")
	}
	return found
}

func assertCanonicalModelEgressBindings(t *testing.T, found map[string]interface{}) {
	t.Helper()
	if bound, _ := found["request_payload_hash_bound"].(bool); !bound {
		t.Fatalf("request_payload_hash_bound = %v, want true", found["request_payload_hash_bound"])
	}
	if got, _ := found["lease_id"].(string); got != "lease-model-1" {
		t.Fatalf("lease_id = %q, want lease-model-1", got)
	}
	if got, _ := found["policy_decision_hash"].(string); got == "" || !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("policy_decision_hash = %v, want sha256 digest", found["policy_decision_hash"])
	}
	if got, _ := found["matched_allowlist_entry_id"].(string); got != "model_default" {
		t.Fatalf("matched_allowlist_entry_id = %q, want model_default", got)
	}
	if got, _ := found["matched_allowlist_ref"].(string); got == "" || !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("matched_allowlist_ref = %v, want sha256 digest", found["matched_allowlist_ref"])
	}
}

func putTrustedModelGatewayContextForRun(t *testing.T, s *Service, runID string, allowlistEntries []any) {
	t.Helper()
	verifier, privateKey := newSignedContextVerifierFixture(t)
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	allowlistDigest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, trustedPolicyAllowlistPayloadWithEntries(t, allowlistEntries))
	rolePayload := signedPayloadForTrustedContext(t, map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          signedContextPrincipal("gateway", "model-gateway", runID, ""),
		"role_family":        "gateway",
		"role_kind":          "model-gateway",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_gateway"},
		"allowlist_refs":     []any{digestObject(allowlistDigest)},
	}, verifier, privateKey)
	runPayload := signedPayloadForTrustedContext(t, map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          signedContextPrincipal("gateway", "model-gateway", runID, ""),
		"manifest_scope":     "run",
		"run_id":             runID,
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_gateway"},
		"allowlist_refs":     []any{digestObject(allowlistDigest)},
	}, verifier, privateKey)
	putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRoleManifest, rolePayload)
	putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRunCapability, runPayload)
}

func trustedModelGatewayInvokeAction(t *testing.T, host string, requestUnits int64, phase string) policyengine.ActionRequest {
	t.Helper()
	payloadHash := mustDigestIdentityForE2E(t, "sha256:"+strings.Repeat("1", 64))
	policyHash := mustDigestIdentityForE2E(t, "sha256:"+strings.Repeat("2", 64))
	timeout := 30
	quotaInput := gatewayQuotaContextForAction(requestUnits, phase)
	return policyengine.NewGatewayEgressAction(policyengine.GatewayEgressActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID:           "cap_gateway",
			RelevantArtifactHashes: []trustpolicy.Digest{payloadHash},
			Actor: policyengine.ActionActor{
				ActorKind:  "role_instance",
				RoleFamily: "gateway",
				RoleKind:   "model-gateway",
			},
		},
		GatewayRoleKind: "model-gateway",
		DestinationKind: "model_endpoint",
		DestinationRef:  host + "/v1/chat/completions",
		EgressDataClass: "spec_text",
		Operation:       "invoke_model",
		TimeoutSeconds:  &timeout,
		PayloadHash:     &payloadHash,
		AuditContext: &policyengine.GatewayAuditContextInput{
			OutboundBytes:      120,
			StartedAt:          "2026-04-12T10:00:00Z",
			CompletedAt:        "2026-04-12T10:00:01Z",
			Outcome:            "admission_allowed",
			RequestHash:        &payloadHash,
			LeaseID:            "lease-model-1",
			PolicyDecisionHash: &policyHash,
		},
		QuotaContext: quotaInput,
	})
}

func gatewayQuotaContextForAction(requestUnits int64, phase string) *policyengine.GatewayQuotaContextInput {
	streamLimit := int64(1024)
	streamed := int64(800)
	meters := policyengine.GatewayQuotaMetersInput{InputTokens: int64Ptr(256), OutputTokens: int64Ptr(64), ConcurrencyUnits: int64Ptr(1), SpendMicros: int64Ptr(1000), EntitlementUnits: int64Ptr(1)}
	if requestUnits > 0 {
		meters.RequestUnits = &requestUnits
	}
	if phase == "stream" {
		meters.StreamedBytes = &streamed
	}
	return &policyengine.GatewayQuotaContextInput{
		QuotaProfileKind:    "hybrid",
		Phase:               phase,
		EnforceDuringStream: phase == "stream",
		StreamLimitBytes:    &streamLimit,
		Meters:              meters,
	}
}

type fakeResolver struct {
	hosts map[string][]string
}

func (r fakeResolver) LookupIP(_ context.Context, _ string, host string) ([]net.IP, error) {
	items := r.hosts[host]
	out := make([]net.IP, 0, len(items))
	for _, item := range items {
		out = append(out, net.ParseIP(item))
	}
	return out, nil
}

func int64Ptr(v int64) *int64 { return &v }
