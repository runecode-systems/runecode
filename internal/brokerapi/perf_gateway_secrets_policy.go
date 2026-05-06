package brokerapi

import (
	"encoding/json"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func putPhase5TrustedModelGatewayContext(service *Service, runID string) error {
	verifier, privateKey, err := phase5VerifierFixture()
	if err != nil {
		return err
	}
	if err := phase5PutTrustedVerifierRecord(service, verifier); err != nil {
		return err
	}
	allowlistPayload, err := json.Marshal(phase5GatewayAllowlistPayload())
	if err != nil {
		return err
	}
	allowlistDigest, err := phase5PutTrustedPolicyArtifact(service, runID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload)
	if err != nil {
		return err
	}
	rolePayload, err := phase5SignedPayloadForTrustedContext(phase5GatewayRoleManifest(runID, allowlistDigest), verifier, privateKey)
	if err != nil {
		return err
	}
	runPayload, err := phase5SignedPayloadForTrustedContext(phase5GatewayRunCapability(runID, allowlistDigest), verifier, privateKey)
	if err != nil {
		return err
	}
	if _, err := phase5PutTrustedPolicyArtifact(service, runID, artifacts.TrustedContractImportKindRoleManifest, rolePayload); err != nil {
		return err
	}
	if _, err := phase5PutTrustedPolicyArtifact(service, runID, artifacts.TrustedContractImportKindRunCapability, runPayload); err != nil {
		return err
	}
	return nil
}

func phase5GatewayAllowlistPayload() map[string]any {
	return map[string]any{
		"schema_id":       "runecode.protocol.v0.PolicyAllowlist",
		"schema_version":  "0.1.0",
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": "runecode.protocol.v0.GatewayScopeRule",
		"entries": []any{map[string]any{
			"schema_id":                   "runecode.protocol.v0.GatewayScopeRule",
			"schema_version":              "0.1.0",
			"scope_kind":                  "gateway_destination",
			"entry_id":                    "model_default",
			"gateway_role_kind":           "model-gateway",
			"destination":                 phase5GatewayDestinationDescriptor(),
			"permitted_operations":        []any{"invoke_model"},
			"allowed_egress_data_classes": []any{"spec_text"},
			"redirect_posture":            "allowlist_only",
			"max_timeout_seconds":         120,
			"max_response_bytes":          16 << 20,
		}},
	}
}

func phase5GatewayDestinationDescriptor() map[string]any {
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

func phase5GatewayRoleManifest(runID string, allowlistDigest string) map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          phase5SignedContextPrincipal(runID, "gateway", "model-gateway"),
		"role_family":        "gateway",
		"role_kind":          "model-gateway",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_gateway"},
		"allowlist_refs":     []any{phase5DigestObject(allowlistDigest)},
	}
}

func phase5GatewayRunCapability(runID string, allowlistDigest string) map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          phase5SignedContextPrincipal(runID, "gateway", "model-gateway"),
		"manifest_scope":     "run",
		"run_id":             runID,
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_gateway"},
		"allowlist_refs":     []any{phase5DigestObject(allowlistDigest)},
	}
}
