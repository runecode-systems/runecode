package brokerapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type trustedPolicyContextDigests struct {
	roleDigest      string
	runDigest       string
	stageDigest     string
	allowlistDigest string
	ruleSetDigest   string
	controlRunID    string
}

func decisionAuditDetailsByDigest(t *testing.T, s *Service, digest string) (map[string]interface{}, bool) {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Type != "policy_decision_recorded" {
			continue
		}
		value, _ := event.Details["policy_decision_digest"].(string)
		if value == digest {
			return event.Details, true
		}
	}
	return nil, false
}

func putTrustedPolicyContextForRun(t *testing.T, s *Service, runID string, withRuleSet bool) trustedPolicyContextDigests {
	t.Helper()
	verifier, privateKey := newSignedContextVerifierFixture(t)
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	allowlistPayload := trustedPolicyAllowlistPayload(t)
	allowlistDigest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload)
	rolePayload := trustedRoleManifestPayload(t, verifier, privateKey, runID, allowlistDigest)
	roleDigest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRoleManifest, rolePayload)
	runPayload := trustedCapabilityManifestPayload(t, verifier, privateKey, runID, "", "run", allowlistDigest)
	runDigest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRunCapability, runPayload)
	stagePayload := trustedCapabilityManifestPayload(t, verifier, privateKey, runID, "artifact_flow", "stage", allowlistDigest)
	stageDigest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindStageCapability, stagePayload)
	ruleSetDigest := maybePutTrustedRuleSet(t, s, runID, withRuleSet)
	return trustedPolicyContextDigests{roleDigest: roleDigest, runDigest: runDigest, stageDigest: stageDigest, allowlistDigest: allowlistDigest, ruleSetDigest: ruleSetDigest, controlRunID: instanceControlRunIDForTests("launcher-instance-1")}
}

func instanceControlRunIDForTests(targetInstanceID string) string {
	return "instance-control:" + strings.TrimSpace(targetInstanceID)
}

func trustedPolicyAllowlistPayload(t *testing.T) []byte {
	t.Helper()
	return trustedPolicyAllowlistPayloadWithEntries(t, []any{trustedModelGatewayAllowlistEntry()})
}

func trustedPolicyAllowlistPayloadWithEntries(t *testing.T, entries []any) []byte {
	t.Helper()
	return mustJSONBytes(t, map[string]any{
		"schema_id":       "runecode.protocol.v0.PolicyAllowlist",
		"schema_version":  "0.1.0",
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": "runecode.protocol.v0.GatewayScopeRule",
		"entries":         entries,
	})
}

func trustedModelGatewayAllowlistEntry() map[string]any {
	return map[string]any{
		"schema_id":                   "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":              "0.1.0",
		"scope_kind":                  "gateway_destination",
		"gateway_role_kind":           "model-gateway",
		"destination":                 trustedModelGatewayDestination(),
		"permitted_operations":        []any{"invoke_model"},
		"allowed_egress_data_classes": []any{"spec_text"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16777216,
	}
}

func trustedModelGatewayDestination() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.DestinationDescriptor",
		"schema_version":           "0.1.0",
		"descriptor_kind":          "model_endpoint",
		"canonical_host":           "model.example.com",
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}

func trustedRoleManifestPayload(t *testing.T, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey, runID, allowlistDigest string) []byte {
	t.Helper()
	return signedPayloadForTrustedContext(t, map[string]any{"schema_id": "runecode.protocol.v0.RoleManifest", "schema_version": "0.2.0", "principal": signedContextPrincipal("workspace", "workspace-edit", runID, ""), "role_family": "workspace", "role_kind": "workspace-edit", "approval_profile": "moderate", "capability_opt_ins": []any{artifactReadCapabilityID}, "allowlist_refs": []any{digestObject(allowlistDigest)}}, verifier, privateKey)
}

func trustedCapabilityManifestPayload(t *testing.T, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey, runID, stageID, scope, allowlistDigest string) []byte {
	t.Helper()
	payload := map[string]any{"schema_id": "runecode.protocol.v0.CapabilityManifest", "schema_version": "0.2.0", "principal": signedContextPrincipal("workspace", "workspace-edit", runID, stageID), "manifest_scope": scope, "run_id": runID, "approval_profile": "moderate", "capability_opt_ins": []any{artifactReadCapabilityID}, "allowlist_refs": []any{digestObject(allowlistDigest)}}
	if strings.TrimSpace(stageID) != "" {
		payload["stage_id"] = stageID
	}
	return signedPayloadForTrustedContext(t, payload, verifier, privateKey)
}

func maybePutTrustedRuleSet(t *testing.T, s *Service, runID string, withRuleSet bool) string {
	t.Helper()
	if !withRuleSet {
		return ""
	}
	ruleSetPayload := map[string]any{"schema_id": "runecode.protocol.v0.PolicyRuleSet", "schema_version": "0.1.0", "rules": []any{map[string]any{"rule_id": "allow-artifact-read", "effect": "allow", "action_kind": "artifact_read", "capability_id": artifactReadCapabilityID, "reason_code": "allow_manifest_opt_in", "details_schema_id": "runecode.protocol.details.policy.allow.v0"}}}
	return putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyRuleSet, mustJSONBytes(t, ruleSetPayload))
}

func putTrustedPolicyArtifact(t *testing.T, s *Service, runID, kind string, payload []byte) string {
	t.Helper()
	provenance := "sha256:" + strings.Repeat("1", 64)
	return putTrustedPolicyArtifactWithProvenance(t, s, runID, kind, payload, provenance)
}

func putTrustedPolicyArtifactWithProvenance(t *testing.T, s *Service, runID, kind string, payload []byte, provenance string) string {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{Payload: payload, ContentType: "application/json", DataClass: artifacts.DataClassAuditVerificationReport, ProvenanceReceiptHash: provenance, CreatedByRole: "broker", TrustedSource: true, RunID: runID})
	if err != nil {
		t.Fatalf("Put trusted policy artifact returned error: %v", err)
	}
	err = s.AppendTrustedAuditEvent(artifacts.TrustedContractImportAuditEventType, "brokerapi", map[string]interface{}{artifacts.TrustedContractImportKindDetailKey: kind, artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest, artifacts.TrustedContractImportProvenanceDetailKey: provenance})
	if err != nil {
		t.Fatalf("AppendTrustedAuditEvent returned error: %v", err)
	}
	return ref.Digest
}

func signedPayloadForTrustedContext(t *testing.T, payload map[string]any, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) []byte {
	t.Helper()
	payload["signatures"] = []any{}
	canonicalWithoutSignatures := canonicalSignedContextPayload(t, payload)
	sig := ed25519.Sign(privateKey, canonicalWithoutSignatures)
	payload["signatures"] = []any{map[string]any{"alg": "ed25519", "key_id": verifier.KeyID, "key_id_value": verifier.KeyIDValue, "signature": base64.StdEncoding.EncodeToString(sig)}}
	return mustJSONBytes(t, payload)
}

func canonicalSignedContextPayload(t *testing.T, payload map[string]any) []byte {
	t.Helper()
	clone := map[string]any{}
	for k, v := range payload {
		clone[k] = v
	}
	delete(clone, "signatures")
	b := mustJSONBytes(t, clone)
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		t.Fatalf("canonicalize payload returned error: %v", err)
	}
	return canonical
}

func signedContextPrincipal(roleFamily, roleKind, runID, stageID string) map[string]any {
	p := map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "role_instance", "principal_id": "brokerapi", "instance_id": "brokerapi-1", "role_family": roleFamily, "role_kind": roleKind}
	if strings.TrimSpace(runID) != "" {
		p["run_id"] = runID
	}
	if strings.TrimSpace(stageID) != "" {
		p["stage_id"] = stageID
	}
	return p
}

func digestObject(identity string) map[string]any {
	return map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(identity, "sha256:")}
}

func newSignedContextVerifierFixture(t *testing.T) (trustpolicy.VerifierRecord, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	sum := sha256Digest(publicKey)
	keyIDValue := hex.EncodeToString(sum)
	return trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "isolate_session_identity", LogicalScope: "session", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "brokerapi", InstanceID: "brokerapi-1"}, KeyProtectionPosture: "os_keystore", IdentityBindingPosture: "attested", PresenceMode: "os_confirmation", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}, privateKey
}

func mustJSONBytes(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	return b
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func toStringSlice(v any) []string {
	raw, ok := v.([]string)
	if ok {
		return raw
	}
	rawAny, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rawAny))
	for _, item := range rawAny {
		if asString, ok := item.(string); ok {
			out = append(out, asString)
		}
	}
	return out
}

func sha256Digest(input []byte) []byte {
	sum := sha256.Sum256(input)
	return sum[:]
}
