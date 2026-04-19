package brokerapi

import (
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestEvaluateModelGatewayInvokeFailsClosedWhenMetadataMissing(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	binding := llmExecutionBinding{
		RequestHash:    trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		DestinationRef: "model.example.com/v1/chat/completions",
	}
	_, errResp := service.evaluateModelGatewayInvoke("req-llm-policy", "run-1", binding, llmOutcomeSucceeded)
	if errResp == nil {
		t.Fatal("expected fail-closed error response")
	}
	if errResp.Error.Code != "gateway_failure" {
		t.Fatalf("error code = %q, want gateway_failure", errResp.Error.Code)
	}
	if !strings.Contains(errResp.Error.Message, "lease_id") {
		t.Fatalf("error message = %q, want lease_id reference", errResp.Error.Message)
	}
}

func TestEmitModelGatewayAuditFailsClosedWhenMetadataMissing(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	err := service.emitModelGatewayAudit("run-1", policyengine.PolicyDecision{}, llmOutcomeSucceeded, llmExecutionBinding{})
	if err == nil {
		t.Fatal("expected fail-closed error")
	}
	if !strings.Contains(err.Error(), "llm execution metadata unavailable") {
		t.Fatalf("error = %q, want metadata unavailable message", err.Error())
	}
}

func TestEmitModelGatewayAuditPropagatesAllowlistMatch(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-llm-audit-match"
	putTrustedModelGatewayContextForRun(t, service, runID, []any{trustedModelGatewayAllowlistEntry()})
	now := time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)
	binding := llmExecutionBinding{
		RequestHash:    trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		ResponseHash:   trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)},
		LeaseID:        "lease-1",
		DestinationRef: "model.example.com/",
		StartedAt:      now,
		CompletedAt:    now.Add(time.Second),
		OutboundBytes:  120,
	}
	err := service.emitModelGatewayAudit(runID, policyengine.PolicyDecision{}, llmOutcomeSucceeded, binding)
	if err != nil {
		t.Fatalf("emitModelGatewayAudit returned error: %v", err)
	}
	found := requireLatestModelEgressAuditDetails(t, service)
	if got, _ := found["matched_allowlist_entry_id"].(string); got != "model_default" {
		t.Fatalf("matched_allowlist_entry_id = %q, want model_default", got)
	}
	if got, _ := found["matched_allowlist_ref"].(string); got == "" || !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("matched_allowlist_ref = %v, want sha256 digest", found["matched_allowlist_ref"])
	}
}

func TestDigestFromIdentityOrNilRejectsInvalidIdentity(t *testing.T) {
	_, err := digestFromIdentityOrNil("sha256:not-a-valid-digest")
	if err == nil {
		t.Fatal("expected digest parse error")
	}
}

func TestValidateLLMExecutionBindingForAuditAcceptsCompleteBinding(t *testing.T) {
	now := time.Now().UTC()
	binding := llmExecutionBinding{
		RequestHash:    trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		LeaseID:        "lease-1",
		DestinationRef: "model.example.com/v1/chat/completions",
		StartedAt:      now,
		CompletedAt:    now.Add(time.Millisecond),
		OutboundBytes:  42,
	}
	if err := validateLLMExecutionBindingForAudit(binding); err != nil {
		t.Fatalf("validateLLMExecutionBindingForAudit returned error: %v", err)
	}
}

func TestValidateLLMExecutionBindingForPolicyRejectsMissingRequestHash(t *testing.T) {
	now := time.Now().UTC()
	binding := llmExecutionBinding{
		RequestHash:   trustpolicy.Digest{},
		LeaseID:       "lease-1",
		StartedAt:     now,
		CompletedAt:   now,
		OutboundBytes: 42,
	}
	err := validateLLMExecutionBindingForPolicy(binding)
	if err == nil {
		t.Fatal("expected missing request_hash error")
	}
	if !strings.Contains(err.Error(), "request_hash") {
		t.Fatalf("error = %q, want request_hash reference", err.Error())
	}
}

func TestValidateLLMExecutionBindingForPolicyAllowsMissingTimingAndBytes(t *testing.T) {
	binding := llmExecutionBinding{
		RequestHash:    trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		DestinationRef: "model.example.com/v1/chat/completions",
		LeaseID:        "lease-1",
	}
	if err := validateLLMExecutionBindingForPolicy(binding); err != nil {
		t.Fatalf("validateLLMExecutionBindingForPolicy returned error: %v", err)
	}
}

func TestValidateLLMExecutionBindingForPolicyRejectsUnavailableLeaseSentinel(t *testing.T) {
	binding := llmExecutionBinding{
		RequestHash:    trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		DestinationRef: "model.example.com/v1/chat/completions",
		LeaseID:        llmLeaseIDUnavailableSentinel,
	}
	err := validateLLMExecutionBindingForPolicy(binding)
	if err == nil {
		t.Fatal("expected unavailable lease_id validation error")
	}
	if !strings.Contains(err.Error(), "destination_ref") && !strings.Contains(err.Error(), "lease_id") {
		t.Fatalf("error = %q, want destination_ref or lease_id validation", err.Error())
	}
}

func TestValidateLLMExecutionBindingForAuditRejectsZeroOutboundBytes(t *testing.T) {
	now := time.Now().UTC()
	binding := llmExecutionBinding{
		RequestHash:    trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		LeaseID:        "lease-1",
		DestinationRef: "model.example.com/v1/chat/completions",
		StartedAt:      now,
		CompletedAt:    now.Add(time.Millisecond),
		OutboundBytes:  0,
	}
	err := validateLLMExecutionBindingForAudit(binding)
	if err == nil {
		t.Fatal("expected outbound byte count validation error")
	}
	if !strings.Contains(err.Error(), "outbound byte count missing") {
		t.Fatalf("error = %q, want outbound byte count missing", err.Error())
	}
}

func TestLLMGatewayPolicyHelpersUseBindingDestinationAndTiming(t *testing.T) {
	now := time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)
	destinationRef := "model.example.com/v1/chat/completions"
	binding := llmExecutionBinding{
		RequestHash:    trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		ResponseHash:   trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)},
		LeaseID:        "lease-1",
		DestinationRef: destinationRef,
		StartedAt:      now,
		CompletedAt:    now.Add(time.Second),
		OutboundBytes:  256,
	}
	action := llmGatewayEgressAction(binding, llmOutcomeSucceeded)
	if got := action.ActionPayload["destination_ref"]; got != "model.example.com/v1/chat/completions" {
		t.Fatalf("destination_ref = %v, want model.example.com/v1/chat/completions", got)
	}
	auditContext, ok := action.ActionPayload["audit_context"].(map[string]any)
	if !ok {
		t.Fatalf("audit_context type = %T, want map[string]any", action.ActionPayload["audit_context"])
	}
	if got := auditContext["started_at"]; got != "2026-04-12T10:00:00Z" {
		t.Fatalf("started_at = %v, want 2026-04-12T10:00:00Z", got)
	}
	if got := auditContext["completed_at"]; got != "2026-04-12T10:00:01Z" {
		t.Fatalf("completed_at = %v, want 2026-04-12T10:00:01Z", got)
	}
	if got := int64FromAny(auditContext["outbound_bytes"]); got != 256 {
		t.Fatalf("outbound_bytes = %v, want 256", auditContext["outbound_bytes"])
	}
}

func TestLLMGatewayPolicyHelpersOmitZeroTimestamps(t *testing.T) {
	destinationRef := "model.example.com/v1/chat/completions"
	binding := llmExecutionBinding{
		RequestHash:    trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)},
		ResponseHash:   trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)},
		LeaseID:        "lease-1",
		DestinationRef: destinationRef,
		OutboundBytes:  0,
	}
	action := llmGatewayEgressAction(binding, llmOutcomeSucceeded)
	auditContext, ok := action.ActionPayload["audit_context"].(map[string]any)
	if !ok {
		t.Fatalf("audit_context type = %T, want map[string]any", action.ActionPayload["audit_context"])
	}
	if got := auditContext["started_at"]; got != "" {
		t.Fatalf("started_at = %v, want empty", got)
	}
	if got := auditContext["completed_at"]; got != "" {
		t.Fatalf("completed_at = %v, want empty", got)
	}
}

func TestResolveLLMDestinationRefFromAllowlistsPrefersModelGatewayInvokeEntry(t *testing.T) {
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{trustedModelGatewayAllowlistEntry()})
	ref, err := resolveLLMDestinationRefFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload, ExpectedHash: artifacts.DigestBytes(allowlistPayload)}})
	if err != nil {
		t.Fatalf("resolveLLMDestinationRefFromAllowlists returned error: %v", err)
	}
	if ref != "model.example.com/" {
		t.Fatalf("destination_ref = %q, want model.example.com/", ref)
	}
}

func TestResolveLLMDestinationFromAllowlistsReturnsAllowlistMatch(t *testing.T) {
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{trustedModelGatewayAllowlistEntry()})
	expectedHash := artifacts.DigestBytes(allowlistPayload)
	ref, match, err := resolveLLMDestinationFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload, ExpectedHash: expectedHash}})
	if err != nil {
		t.Fatalf("resolveLLMDestinationFromAllowlists returned error: %v", err)
	}
	if ref != "model.example.com/" {
		t.Fatalf("destination_ref = %q, want model.example.com/", ref)
	}
	if match.AllowlistRef != expectedHash {
		t.Fatalf("allowlist_ref = %q, want %q", match.AllowlistRef, expectedHash)
	}
	if match.EntryID != "model_default" {
		t.Fatalf("entry_id = %q, want model_default", match.EntryID)
	}
}

func TestResolveLLMDestinationFromAllowlistsFailsClosedOnExpectedHashMismatch(t *testing.T) {
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{trustedModelGatewayAllowlistEntry()})
	_, _, err := resolveLLMDestinationFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload, ExpectedHash: "sha256:" + strings.Repeat("f", 64)}})
	if err == nil {
		t.Fatal("expected hash mismatch error")
	}
	if !strings.Contains(err.Error(), "trusted allowlist payload hash mismatch") {
		t.Fatalf("error = %q, want trusted allowlist payload hash mismatch", err.Error())
	}
}

func TestResolveLLMDestinationRefFromAllowlistsRejectsUnhardenedEntry(t *testing.T) {
	entry := trustedModelGatewayAllowlistEntry()
	destination, _ := entry["destination"].(map[string]any)
	destination["tls_required"] = false
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{entry})
	_, err := resolveLLMDestinationRefFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload, ExpectedHash: artifacts.DigestBytes(allowlistPayload)}})
	if err == nil {
		t.Fatal("expected destination resolution error for unhardened entry")
	}
	if !strings.Contains(err.Error(), "trusted model gateway destination unavailable") {
		t.Fatalf("error = %q, want trusted model gateway destination unavailable", err.Error())
	}
}

func TestResolveLLMDestinationRefFromAllowlistsRejectsPathTraversalPrefix(t *testing.T) {
	entry := trustedModelGatewayAllowlistEntry()
	destination, _ := entry["destination"].(map[string]any)
	destination["canonical_path_prefix"] = "/v1/../../admin"
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{entry})
	_, err := resolveLLMDestinationRefFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload, ExpectedHash: artifacts.DigestBytes(allowlistPayload)}})
	if err == nil {
		t.Fatal("expected destination resolution error for traversal path prefix")
	}
	if !strings.Contains(err.Error(), "trusted model gateway destination unavailable") {
		t.Fatalf("error = %q, want trusted model gateway destination unavailable", err.Error())
	}
}

func TestResolveLLMDestinationRefFromAllowlistsRejectsEncodedPathTraversalPrefix(t *testing.T) {
	entry := trustedModelGatewayAllowlistEntry()
	destination, _ := entry["destination"].(map[string]any)
	destination["canonical_path_prefix"] = "/v1/%2e%2e/admin"
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{entry})
	_, err := resolveLLMDestinationRefFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload, ExpectedHash: artifacts.DigestBytes(allowlistPayload)}})
	if err == nil {
		t.Fatal("expected destination resolution error for encoded traversal path prefix")
	}
	if !strings.Contains(err.Error(), "trusted model gateway destination unavailable") {
		t.Fatalf("error = %q, want trusted model gateway destination unavailable", err.Error())
	}
}

func TestResolveLLMDestinationRefFromAllowlistsUsesCanonicalPathPrefix(t *testing.T) {
	entry := trustedModelGatewayAllowlistEntry()
	destination, _ := entry["destination"].(map[string]any)
	destination["canonical_path_prefix"] = "/v1"
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{entry})
	ref, err := resolveLLMDestinationRefFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload, ExpectedHash: artifacts.DigestBytes(allowlistPayload)}})
	if err != nil {
		t.Fatalf("resolveLLMDestinationRefFromAllowlists returned error: %v", err)
	}
	if ref != "model.example.com/v1" {
		t.Fatalf("destination_ref = %q, want model.example.com/v1", ref)
	}
}

func TestResolveLLMDestinationRefFromAllowlistsErrorsWhenMissingEntry(t *testing.T) {
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{trustedDependencyFetchAllowlistEntry()})
	_, err := resolveLLMDestinationRefFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload, ExpectedHash: artifacts.DigestBytes(allowlistPayload)}})
	if err == nil {
		t.Fatal("expected destination resolution error for missing model invoke entry")
	}
	if !strings.Contains(err.Error(), "trusted model gateway destination unavailable") {
		t.Fatalf("error = %q, want trusted model gateway destination unavailable", err.Error())
	}
}

func TestResolveLLMDestinationRefFromAllowlistsFailsClosedOnMissingExpectedHash(t *testing.T) {
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{trustedModelGatewayAllowlistEntry()})
	_, err := resolveLLMDestinationRefFromAllowlists([]policyengine.ManifestInput{{Payload: allowlistPayload}})
	if err == nil {
		t.Fatal("expected missing expected hash error")
	}
	if !strings.Contains(err.Error(), "trusted allowlist expected hash missing") {
		t.Fatalf("error = %q, want trusted allowlist expected hash missing", err.Error())
	}
}

func int64FromAny(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}
