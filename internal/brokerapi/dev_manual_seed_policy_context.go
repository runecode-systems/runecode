//go:build runecode_devseed

package brokerapi

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (s *Service) seedDevManualInstanceControlContext() error {
	instanceControlRunID := instanceControlRunIDForInstanceID(devManualSeedInstanceID)
	verifier, privateKey := devManualPolicyContextVerifier()
	if err := s.recordTrustedVerifierRecord(verifier); err != nil {
		return err
	}
	allowlistDigest, err := s.seedDevManualPolicyAllowlist(instanceControlRunID)
	if err != nil {
		return err
	}
	if err := s.seedDevManualExternalAnchorGatewayContext(instanceControlRunID, allowlistDigest, verifier, privateKey); err != nil {
		return err
	}
	if err := s.recordDevManualSignedTrustedContext(instanceControlRunID, artifacts.TrustedContractImportKindRoleManifest, devManualWorkspaceRoleManifestPayload(instanceControlRunID, allowlistDigest), verifier, privateKey); err != nil {
		return err
	}
	return s.recordDevManualSignedTrustedContext(instanceControlRunID, artifacts.TrustedContractImportKindRunCapability, devManualWorkspaceCapabilityManifestPayload(instanceControlRunID, allowlistDigest), verifier, privateKey)
}

func devManualPolicyContextVerifier() (trustpolicy.VerifierRecord, ed25519.PrivateKey) {
	seed := sha256.Sum256([]byte("runecode.dev-manual-seed.policy-context.v1"))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)
	keyIDValue := hex.EncodeToString(sha256DigestBytes(publicKey))
	return trustpolicy.VerifierRecord{
		SchemaID:      trustpolicy.VerifierSchemaID,
		SchemaVersion: trustpolicy.VerifierSchemaVersion,
		KeyID:         trustpolicy.KeyIDProfile,
		KeyIDValue:    keyIDValue,
		Alg:           "ed25519",
		PublicKey: trustpolicy.PublicKey{
			Encoding: "base64",
			Value:    base64.StdEncoding.EncodeToString(publicKey),
		},
		LogicalPurpose:         "isolate_session_identity",
		LogicalScope:           "session",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "brokerapi", InstanceID: "brokerapi-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              devManualSeedRecordedAtRFC3339,
		Status:                 "active",
	}, privateKey
}

func (s *Service) recordTrustedVerifierRecord(record trustpolicy.VerifierRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = s.recordTrustedPolicyContextArtifact("", artifacts.TrustedContractImportKindVerifierRecord, payload)
	return err
}

func (s *Service) recordTrustedPolicyContextArtifact(runID, kind string, payload []byte) (string, error) {
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassAuditVerificationReport,
		ProvenanceReceiptHash: digestWithByte("1"),
		CreatedByRole:         "broker",
		TrustedSource:         true,
		RunID:                 runID,
	})
	if err != nil {
		return "", err
	}
	if err := s.AppendTrustedAuditEvent(artifacts.TrustedContractImportAuditEventType, "brokerapi", map[string]any{
		artifacts.TrustedContractImportKindDetailKey:           kind,
		artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest,
		artifacts.TrustedContractImportProvenanceDetailKey:     digestWithByte("1"),
	}); err != nil {
		return "", err
	}
	return ref.Digest, nil
}

func devManualWorkspaceRoleManifestPayload(runID, allowlistDigest string) map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          devManualSignedContextPrincipal(runID, "workspace", "workspace-edit"),
		"role_family":        "workspace",
		"role_kind":          "workspace-edit",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_backend"},
		"allowlist_refs":     []any{digestObjectForDevSeed(allowlistDigest)},
	}
}

func devManualWorkspaceCapabilityManifestPayload(runID, allowlistDigest string) map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          devManualSignedContextPrincipal(runID, "workspace", "workspace-edit"),
		"manifest_scope":     "run",
		"run_id":             runID,
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_backend"},
		"allowlist_refs":     []any{digestObjectForDevSeed(allowlistDigest)},
	}
}

func devManualSignedContextPrincipal(runID, roleFamily, roleKind string) map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
		"schema_version": "0.2.0",
		"actor_kind":     "role_instance",
		"principal_id":   "brokerapi",
		"instance_id":    "brokerapi-1",
		"role_family":    roleFamily,
		"role_kind":      roleKind,
		"run_id":         runID,
	}
}

func (s *Service) recordDevManualSignedTrustedContext(runID, kind string, payload map[string]any, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) error {
	signedPayload, err := devManualSignedPayloadForTrustedContext(payload, verifier, privateKey)
	if err != nil {
		return err
	}
	_, err = s.recordTrustedPolicyContextArtifact(runID, kind, signedPayload)
	return err
}

func devManualSignedPayloadForTrustedContext(payload map[string]any, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) ([]byte, error) {
	payload["signatures"] = []any{}
	canonicalWithoutSignatures, err := devManualCanonicalSignedContextPayload(payload)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(privateKey, canonicalWithoutSignatures)
	payload["signatures"] = []any{map[string]any{"alg": "ed25519", "key_id": verifier.KeyID, "key_id_value": verifier.KeyIDValue, "signature": base64.StdEncoding.EncodeToString(sig)}}
	return json.Marshal(payload)
}

func devManualCanonicalSignedContextPayload(payload map[string]any) ([]byte, error) {
	clone := map[string]any{}
	for k, v := range payload {
		clone[k] = v
	}
	delete(clone, "signatures")
	b, err := json.Marshal(clone)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(b)
}

func mustJSONBytesForDevSeed(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return b
}

func sha256DigestBytes(input []byte) []byte {
	sum := sha256.Sum256(input)
	return sum[:]
}
