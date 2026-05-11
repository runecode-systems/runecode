package brokerperf

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/brokerapi"
	"github.com/runecode-systems/runecode/internal/trustpolicy"
	"github.com/runecode-systems/runecode/third_party/jsoncanonicalizer"
)

func seedBackendPosturePolicyContext(service *brokerapi.Service) error {
	targetInstanceID := serviceCurrentInstanceID(service)
	if strings.TrimSpace(targetInstanceID) == "" {
		return fmt.Errorf("backend posture policy context target instance missing")
	}
	controlRunID := "instance-control:" + targetInstanceID
	verifier, privateKey, err := backendPosturePolicyContextVerifier()
	if err != nil {
		return err
	}
	if err := putTrustedVerifierRecordForService(service, verifier); err != nil {
		return err
	}
	allowlistDigest, err := persistBackendPosturePolicyAllowlist(service, controlRunID)
	if err != nil {
		return err
	}
	if err := persistBackendPostureRoleManifest(service, controlRunID, allowlistDigest, verifier, privateKey); err != nil {
		return err
	}
	return persistBackendPostureRunCapability(service, controlRunID, allowlistDigest, verifier, privateKey)
}

func persistBackendPosturePolicyAllowlist(service *brokerapi.Service, controlRunID string) (string, error) {
	allowlistPayload, err := json.Marshal(backendPosturePolicyAllowlistPayload())
	if err != nil {
		return "", err
	}
	return putTrustedPolicyArtifactForService(service, controlRunID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload)
}

func persistBackendPostureRoleManifest(service *brokerapi.Service, controlRunID, allowlistDigest string, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) error {
	payload, err := signedTrustedContextPayload(map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          backendPostureSignedContextPrincipal(controlRunID),
		"role_family":        "workspace",
		"role_kind":          "workspace-edit",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_backend"},
		"allowlist_refs":     []any{digestObjectFromIdentity(allowlistDigest)},
	}, verifier, privateKey)
	if err != nil {
		return err
	}
	_, err = putTrustedPolicyArtifactForService(service, controlRunID, artifacts.TrustedContractImportKindRoleManifest, payload)
	return err
}

func persistBackendPostureRunCapability(service *brokerapi.Service, controlRunID, allowlistDigest string, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) error {
	payload, err := signedTrustedContextPayload(map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          backendPostureSignedContextPrincipal(controlRunID),
		"manifest_scope":     "run",
		"run_id":             controlRunID,
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_backend"},
		"allowlist_refs":     []any{digestObjectFromIdentity(allowlistDigest)},
	}, verifier, privateKey)
	if err != nil {
		return err
	}
	_, err = putTrustedPolicyArtifactForService(service, controlRunID, artifacts.TrustedContractImportKindRunCapability, payload)
	return err
}

func backendPosturePolicyContextVerifier() (trustpolicy.VerifierRecord, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return trustpolicy.VerifierRecord{}, nil, err
	}
	keyIDValue := backendPostureKeyIDValue(publicKey)
	return trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "isolate_session_identity", LogicalScope: "session", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "brokerapi", InstanceID: "brokerapi-1"}, KeyProtectionPosture: "os_keystore", IdentityBindingPosture: "attested", PresenceMode: "os_confirmation", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}, privateKey, nil
}

func backendPosturePolicyAllowlistPayload() map[string]any {
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
			"destination":                 backendPostureGatewayDestinationDescriptor(),
			"permitted_operations":        []any{"invoke_model"},
			"allowed_egress_data_classes": []any{"spec_text"},
			"redirect_posture":            "allowlist_only",
			"max_timeout_seconds":         120,
			"max_response_bytes":          16 << 20,
		}},
	}
}

func backendPostureGatewayDestinationDescriptor() map[string]any {
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

func signedTrustedContextPayload(payload map[string]any, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) ([]byte, error) {
	payload["signatures"] = []any{}
	clone := map[string]any{}
	for k, v := range payload {
		clone[k] = v
	}
	delete(clone, "signatures")
	raw, err := json.Marshal(clone)
	if err != nil {
		return nil, err
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(privateKey, canonical)
	payload["signatures"] = []any{map[string]any{"alg": "ed25519", "key_id": verifier.KeyID, "key_id_value": verifier.KeyIDValue, "signature": base64.StdEncoding.EncodeToString(sig)}}
	return json.Marshal(payload)
}

func backendPostureSignedContextPrincipal(runID string) map[string]any {
	return map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "role_instance", "principal_id": "brokerapi", "instance_id": "brokerapi-1", "role_family": "workspace", "role_kind": "workspace-edit", "run_id": runID}
}

func digestObjectFromIdentity(identity string) map[string]any {
	return map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(identity, "sha256:")}
}

func putTrustedPolicyArtifactForService(service *brokerapi.Service, runID, kind string, payload []byte) (string, error) {
	provenance := "sha256:" + strings.Repeat("1", 64)
	ref, err := service.Put(artifacts.PutRequest{Payload: payload, ContentType: "application/json", DataClass: artifacts.DataClassAuditVerificationReport, ProvenanceReceiptHash: provenance, CreatedByRole: "broker", TrustedSource: true, RunID: strings.TrimSpace(runID)})
	if err != nil {
		return "", err
	}
	details := map[string]interface{}{artifacts.TrustedContractImportKindDetailKey: kind, artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest, artifacts.TrustedContractImportProvenanceDetailKey: provenance}
	if err := service.AppendTrustedAuditEvent(artifacts.TrustedContractImportAuditEventType, "brokerapi", details); err != nil {
		return "", err
	}
	return ref.Digest, nil
}
