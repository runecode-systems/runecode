//go:build runecode_devseed

package brokerapi

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type devManualVerificationMaterial struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	keyIDValue string
}

func devManualVerificationSignerMaterial() devManualVerificationMaterial {
	seed := sha256.Sum256([]byte("runecode.dev-manual-seed.audit-signing.v1"))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	publicKey := privateKey.Public().(ed25519.PublicKey)
	keyID := sha256.Sum256(publicKey)
	return devManualVerificationMaterial{privateKey: privateKey, publicKey: publicKey, keyIDValue: hex.EncodeToString(keyID[:])}
}

func devManualVerifierRecord(material devManualVerificationMaterial) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             material.keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(material.publicKey)},
		LogicalPurpose:         "isolate_session_identity",
		LogicalScope:           "session",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
}

func devManualEventContractCatalog() trustpolicy.AuditEventContractCatalog {
	return trustpolicy.AuditEventContractCatalog{SchemaID: trustpolicy.AuditEventContractCatalogSchemaID, SchemaVersion: trustpolicy.AuditEventContractCatalogSchemaVersion, CatalogID: "audit_event_contract_v0", Entries: []trustpolicy.AuditEventContractCatalogEntry{{AuditEventType: "isolate_session_bound", AllowedPayloadSchemaIDs: []string{trustpolicy.IsolateSessionBoundPayloadSchemaID}, AllowedSignerPurposes: []string{"isolate_session_identity"}, AllowedSignerScopes: []string{"session"}, RequiredScopeFields: []string{"workspace_id", "run_id", "stage_id"}, RequiredCorrelationFields: []string{"session_id", "operation_id"}, RequireSubjectRef: true, AllowedSubjectRefRoles: []string{"binding_target"}, AllowedCauseRefRoles: []string{"session_cause"}, AllowedRelatedRefRoles: []string{"binding", "evidence", "receipt"}, RequireSignerEvidenceRefs: true, AllowedSignerEvidenceRefRoles: []string{"admissibility", "binding"}}}}
}

func devManualSignerEvidenceRefs(material devManualVerificationMaterial) []trustpolicy.AuditSignerEvidenceReference {
	return []trustpolicy.AuditSignerEvidenceReference{{
		Digest: trustpolicy.Digest{HashAlg: "sha256", Hash: stringsRepeat("f")},
		Evidence: trustpolicy.AuditSignerEvidence{
			SignerPurpose: "isolate_session_identity",
			SignerScope:   "session",
			SignerKey:     trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: material.keyIDValue, Signature: base64.StdEncoding.EncodeToString([]byte("sig"))},
			IsolateBinding: &trustpolicy.IsolateSessionBinding{
				RunID:                   devManualSeedRunID,
				IsolateID:               "isolate-manual-001",
				SessionID:               devManualSeedSessionID,
				SessionNonce:            "nonce-manual-001",
				ProvisioningMode:        "tofu",
				ImageDigest:             trustpolicy.Digest{HashAlg: "sha256", Hash: stringsRepeat("1")},
				ActiveManifestHash:      trustpolicy.Digest{HashAlg: "sha256", Hash: stringsRepeat("2")},
				HandshakeTranscriptHash: trustpolicy.Digest{HashAlg: "sha256", Hash: stringsRepeat("3")},
				KeyID:                   trustpolicy.KeyIDProfile,
				KeyIDValue:              material.keyIDValue,
				IdentityBindingPosture:  "tofu",
			},
		},
	}}
}

func devManualSignedEnvelope(payloadSchemaID, payloadSchemaVersion string, payload any, material devManualVerificationMaterial) (trustpolicy.SignedObjectEnvelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	signature := ed25519.Sign(material.privateKey, canonicalPayload)
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      payloadSchemaID,
		PayloadSchemaVersion: payloadSchemaVersion,
		Payload:              payloadBytes,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: material.keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}, nil
}

func stringsRepeat(ch string) string {
	return strings.Repeat(ch, 64)
}
