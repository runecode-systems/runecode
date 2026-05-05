package brokerapi

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func newPhase5DependencyService(repoRoot string) (*Service, func(), error) {
	storeRoot, err := os.MkdirTemp("", "runecode-phase5-deps-store-")
	if err != nil {
		return nil, nil, err
	}
	ledgerRoot, err := os.MkdirTemp("", "runecode-phase5-deps-ledger-")
	if err != nil {
		_ = os.RemoveAll(storeRoot)
		return nil, nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(storeRoot)
		_ = os.RemoveAll(ledgerRoot)
	}
	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repoRoot, DependencyFetch: DependencyFetchConfig{MaxParallelFetches: 8}})
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return service, cleanup, nil
}

func phase5DependencyAllowlistPayload() ([]byte, error) {
	return json.Marshal(map[string]any{
		"schema_id":       "runecode.protocol.v0.PolicyAllowlist",
		"schema_version":  "0.1.0",
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": "runecode.protocol.v0.GatewayScopeRule",
		"entries":         []any{phase5DependencyAllowlistEntry()},
	})
}

func phase5DependencyAllowlistEntry() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":    "0.1.0",
		"scope_kind":        "gateway_destination",
		"entry_id":          "dependency_default",
		"gateway_role_kind": "dependency-fetch",
		"destination": map[string]any{
			"schema_id":                "runecode.protocol.v0.DestinationDescriptor",
			"schema_version":           "0.1.0",
			"descriptor_kind":          "package_registry",
			"canonical_host":           "registry.npmjs.org",
			"canonical_path_prefix":    "/",
			"provider_or_namespace":    "npm",
			"tls_required":             true,
			"private_range_blocking":   "enforced",
			"dns_rebinding_protection": "enforced",
		},
		"permitted_operations":        []any{"fetch_dependency"},
		"allowed_egress_data_classes": []any{"dependency_resolved_payload"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16 << 20,
	}
}

func phase5DependencyRolePayload(runID, allowlistDigest string, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) ([]byte, error) {
	return phase5SignedPayloadForTrustedContext(map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          phase5SignedContextPrincipal(runID, "gateway", "dependency-fetch"),
		"role_family":        "gateway",
		"role_kind":          "dependency-fetch",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_gateway"},
		"allowlist_refs":     []any{phase5DigestObject(allowlistDigest)},
	}, verifier, privateKey)
}

func phase5DependencyRunPayload(runID, allowlistDigest string, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) ([]byte, error) {
	return phase5SignedPayloadForTrustedContext(map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          phase5SignedContextPrincipal(runID, "gateway", "dependency-fetch"),
		"manifest_scope":     "run",
		"run_id":             runID,
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_gateway"},
		"allowlist_refs":     []any{phase5DigestObject(allowlistDigest)},
	}, verifier, privateKey)
}

func putPhase5TrustedDependencyContext(service *Service, runID string) error {
	verifier, privateKey, err := phase5VerifierFixture()
	if err != nil {
		return err
	}
	if err := phase5PutTrustedVerifierRecord(service, verifier); err != nil {
		return err
	}
	allowlistPayload, err := phase5DependencyAllowlistPayload()
	if err != nil {
		return err
	}
	allowlistDigest, err := phase5PutTrustedPolicyArtifact(service, runID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload)
	if err != nil {
		return err
	}
	rolePayload, err := phase5DependencyRolePayload(runID, allowlistDigest, verifier, privateKey)
	if err != nil {
		return err
	}
	runPayload, err := phase5DependencyRunPayload(runID, allowlistDigest, verifier, privateKey)
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

func phase5DependencyFetchRegistryRequest(requestID, runID, pkg string) DependencyFetchRegistryRequest {
	dep := DependencyFetchRequestObject{
		SchemaID:      "runecode.protocol.v0.DependencyFetchRequest",
		SchemaVersion: "0.1.0",
		RequestKind:   "package_version_fetch",
		RegistryIdentity: policyengine.DestinationDescriptor{
			SchemaID:               "runecode.protocol.v0.DestinationDescriptor",
			SchemaVersion:          "0.1.0",
			DescriptorKind:         "package_registry",
			CanonicalHost:          "registry.npmjs.org",
			CanonicalPathPrefix:    "/",
			ProviderOrNamespace:    "npm",
			TLSRequired:            true,
			PrivateRangeBlocking:   "enforced",
			DNSRebindingProtection: "enforced",
		},
		Ecosystem:      "npm",
		PackageName:    "pkg-" + pkg,
		PackageVersion: "1.0.0",
	}
	hash, _ := canonicalDependencyRequestIdentity(dep)
	digest, _ := digestFromIdentity(hash)
	return DependencyFetchRegistryRequest{SchemaID: "runecode.protocol.v0.DependencyFetchRegistryRequest", SchemaVersion: "0.1.0", RequestID: requestID, RunID: runID, DependencyRequest: dep, RequestHash: digest}
}

func phase5DependencyEnsureRequest(requestID, runID, pkg string) DependencyCacheEnsureRequest {
	depReq := phase5DependencyFetchRegistryRequest(requestID+"-single", runID, pkg).DependencyRequest
	batch := DependencyFetchBatchRequestObject{
		SchemaID:            "runecode.protocol.v0.DependencyFetchBatchRequest",
		SchemaVersion:       "0.1.0",
		LockfileKind:        "generic_lock",
		LockfileDigest:      mustDigestObjectFromIdentity(artifacts.DigestBytes([]byte("lock:" + pkg))),
		RequestSetHash:      mustDigestObjectFromIdentity(artifacts.DigestBytes([]byte("request-set:" + pkg))),
		DependencyRequests:  []DependencyFetchRequestObject{depReq},
		BatchRequestID:      "batch-" + pkg,
		LockfileLocatorHint: "deps.lock",
	}
	return DependencyCacheEnsureRequest{SchemaID: "runecode.protocol.v0.DependencyCacheEnsureRequest", SchemaVersion: "0.1.0", RequestID: requestID, RunID: runID, BatchRequest: batch}
}

func phase5DependencyHandoffRequest(requestID, pkg, consumerRole string) DependencyCacheHandoffRequest {
	dep := phase5DependencyFetchRegistryRequest(requestID+"-single", "run-deps-phase5", pkg).DependencyRequest
	hash, _ := canonicalDependencyRequestIdentity(dep)
	return DependencyCacheHandoffRequest{SchemaID: "runecode.protocol.v0.DependencyCacheHandoffRequest", SchemaVersion: "0.1.0", RequestID: requestID, RequestDigest: mustDigestObjectFromIdentity(hash), ConsumerRole: consumerRole}
}

func phase5VerifierFixture() (trustpolicy.VerifierRecord, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return trustpolicy.VerifierRecord{}, nil, err
	}
	sum := sha256.Sum256(publicKey)
	keyIDValue := hex.EncodeToString(sum[:])
	return trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "isolate_session_identity", LogicalScope: "session", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "brokerapi", InstanceID: "brokerapi-1"}, KeyProtectionPosture: "os_keystore", IdentityBindingPosture: "attested", PresenceMode: "os_confirmation", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}, privateKey, nil
}

func phase5PutTrustedVerifierRecord(service *Service, record trustpolicy.VerifierRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = phase5PutTrustedPolicyArtifact(service, "", artifacts.TrustedContractImportKindVerifierRecord, payload)
	return err
}

func phase5PutTrustedPolicyArtifact(service *Service, runID, kind string, payload []byte) (string, error) {
	provenance := "sha256:" + strings.Repeat("1", 64)
	ref, err := service.Put(artifacts.PutRequest{Payload: payload, ContentType: "application/json", DataClass: artifacts.DataClassAuditVerificationReport, ProvenanceReceiptHash: provenance, CreatedByRole: "broker", TrustedSource: true, RunID: runID})
	if err != nil {
		return "", err
	}
	details := map[string]interface{}{
		artifacts.TrustedContractImportKindDetailKey:           kind,
		artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest,
		artifacts.TrustedContractImportProvenanceDetailKey:     provenance,
	}
	if err := service.AppendTrustedAuditEvent(artifacts.TrustedContractImportAuditEventType, "brokerapi", details); err != nil {
		return "", err
	}
	return ref.Digest, nil
}

func phase5SignedPayloadForTrustedContext(payload map[string]any, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) ([]byte, error) {
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

func phase5SignedContextPrincipal(runID, roleFamily, roleKind string) map[string]any {
	return map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "role_instance", "principal_id": "brokerapi", "instance_id": "brokerapi-1", "role_family": roleFamily, "role_kind": roleKind, "run_id": runID}
}

func phase5DigestObject(identity string) map[string]any {
	return map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(identity, "sha256:")}
}
