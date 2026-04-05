package trustpolicy

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestVerifySignedEnvelopeAcceptsValidPayload(t *testing.T) {
	registry, envelope := signedEnvelopeFixtureForTests(t)
	err := VerifySignedEnvelope(envelope, registry, EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   "runecode.protocol.v0.Error",
		ExpectedPayloadVersion:    "0.3.0",
	})
	if err != nil {
		t.Fatalf("VerifySignedEnvelope returned error: %v", err)
	}
}

func TestVerifySignedEnvelopeFailsClosedOnSchemaMismatch(t *testing.T) {
	registry, envelope := signedEnvelopeFixtureForTests(t)
	err := VerifySignedEnvelope(envelope, registry, EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   "runecode.protocol.v0.ApprovalRequest",
		ExpectedPayloadVersion:    "0.3.0",
	})
	if err == nil {
		t.Fatal("VerifySignedEnvelope expected payload schema mismatch error")
	}
}

func signedEnvelopeFixtureForTests(t *testing.T) (*VerifierRegistry, SignedObjectEnvelope) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	payload := []byte(`{"schema_id":"runecode.protocol.v0.Error","schema_version":"0.3.0","code":"unknown_schema_id","category":"validation","retryable":false,"message":"example"}`)
	signature := signPayloadForTests(t, privateKey, payload)
	keyIDValue := sha256Hex(publicKey)
	record := verifierRecordFixtureForTests(publicKey, keyIDValue)
	registry, err := NewVerifierRegistry([]VerifierRecord{record})
	if err != nil {
		t.Fatalf("NewVerifierRegistry returned error: %v", err)
	}
	envelope := SignedObjectEnvelope{
		SchemaID:             EnvelopeSchemaID,
		SchemaVersion:        EnvelopeSchemaVersion,
		PayloadSchemaID:      "runecode.protocol.v0.Error",
		PayloadSchemaVersion: "0.3.0",
		Payload:              payload,
		SignatureInput:       SignatureInputProfile,
		Signature: SignatureBlock{
			Alg:        "ed25519",
			KeyID:      KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}
	return registry, envelope
}

func signPayloadForTests(t *testing.T, privateKey ed25519.PrivateKey, payload []byte) []byte {
	t.Helper()
	canonicalPayload, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}
	return ed25519.Sign(privateKey, canonicalPayload)
}

func verifierRecordFixtureForTests(publicKey ed25519.PublicKey, keyIDValue string) VerifierRecord {
	return VerifierRecord{
		SchemaID:               VerifierSchemaID,
		SchemaVersion:          VerifierSchemaVersion,
		KeyID:                  KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "approval_authority",
		LogicalScope:           "user",
		OwnerPrincipal:         PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: "alice", InstanceID: "approver-session"},
		KeyProtectionPosture:   "hardware_backed",
		IdentityBindingPosture: "attested",
		PresenceMode:           "hardware_touch",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
}

func TestValidateApprovalDecisionEvidenceRejectsWeakHighestAssurance(t *testing.T) {
	decision := ApprovalDecision{
		SchemaID:               ApprovalDecisionSchemaID,
		SchemaVersion:          ApprovalDecisionSchemaVersion,
		ApprovalRequestHash:    Digest{HashAlg: "sha256", Hash: hexNibbleDigest("a")},
		Approver:               PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: "alice", InstanceID: "approval-session"},
		DecisionOutcome:        "approve",
		ApprovalAssuranceLevel: "hardware_backed",
		PresenceMode:           "none",
		KeyProtectionPosture:   "hardware_backed",
		IdentityBindingPosture: "attested",
		DecidedAt:              "2026-03-13T12:05:00Z",
		ConsumptionPosture:     "single_use",
	}

	err := ValidateApprovalDecisionEvidence(decision)
	if err == nil {
		t.Fatal("ValidateApprovalDecisionEvidence expected fail-closed error")
	}
}

func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func hexNibbleDigest(nibble string) string {
	if len(nibble) != 1 {
		panic("hexNibbleDigest requires one nibble")
	}
	output := ""
	for len(output) < 64 {
		output += nibble
	}
	return output
}
