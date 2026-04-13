package brokerapi

import (
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) projectModelGatewayPostureForReadiness() (bool, string, *ModelGatewayPostureProjection) {
	posture := defaultModelGatewayPostureProjection()
	runtime := policyRuntime{service: s}
	catalog, err := runtime.trustedPolicyCatalog()
	if err != nil {
		return false, "failed", nil
	}
	return projectModelGatewayPostureFromCatalog(runtime, catalog, posture)
}

func projectModelGatewayPostureFromCatalog(runtime policyRuntime, catalog trustedPolicyCatalog, posture *ModelGatewayPostureProjection) (bool, string, *ModelGatewayPostureProjection) {
	for _, record := range catalog.byKind[artifacts.TrustedContractImportKindPolicyAllowlist] {
		allowlist, unmarshalErr := decodePolicyAllowlist(runtime, record)
		if unmarshalErr != nil {
			return false, "degraded", nil
		}
		if allowlistHasModelGatewayEntry(allowlist) {
			posture.ConfigurationState = "configured"
			posture.EgressPolicyPosture = "allowlist_only"
			return true, "ok", posture
		}
	}
	return true, "ok", posture
}

func defaultModelGatewayPostureProjection() *ModelGatewayPostureProjection {
	return &ModelGatewayPostureProjection{
		SchemaID:             "runecode.protocol.v0.ModelGatewayPostureProjection",
		SchemaVersion:        "0.1.0",
		ProjectionKind:       "broker_projected",
		GatewayRoleKind:      "model-gateway",
		DestinationScopeKind: "gateway_destination",
		ConfigurationState:   "not_configured",
		EgressPolicyPosture:  "deny_by_default",
		SurfaceChannel:       "broker_local_api",
	}
}

func decodePolicyAllowlist(runtime policyRuntime, record artifacts.ArtifactRecord) (policyengine.PolicyAllowlist, error) {
	manifestInput, readErr := runtime.readManifestInput(record)
	if readErr != nil {
		return policyengine.PolicyAllowlist{}, readErr
	}
	if computedDigest := artifacts.DigestBytes(manifestInput.Payload); computedDigest != record.Reference.Digest {
		return policyengine.PolicyAllowlist{}, fmt.Errorf("trusted allowlist payload digest mismatch: expected %s got %s", record.Reference.Digest, computedDigest)
	}
	allowlist := policyengine.PolicyAllowlist{}
	if unmarshalErr := json.Unmarshal(manifestInput.Payload, &allowlist); unmarshalErr != nil {
		return policyengine.PolicyAllowlist{}, unmarshalErr
	}
	return allowlist, nil
}

func allowlistHasModelGatewayEntry(allowlist policyengine.PolicyAllowlist) bool {
	for _, entry := range allowlist.Entries {
		if isModelGatewayConfiguredEntry(entry) {
			return true
		}
	}
	return false
}

func isModelGatewayConfiguredEntry(entry policyengine.GatewayScopeRule) bool {
	if entry.ScopeKind != "gateway_destination" {
		return false
	}
	if entry.GatewayRoleKind != "model-gateway" {
		return false
	}
	if entry.Destination.DescriptorKind != "model_endpoint" {
		return false
	}
	for _, operation := range entry.PermittedOperations {
		if operation == "invoke_model" {
			return true
		}
	}
	return false
}
