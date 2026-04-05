package artifacts

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestVerifySignedApprovalDecisionAcceptsSignedApproval(t *testing.T) {
	req, verifiers, err := signedPromotionRequestForTests("human-1")
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	if err := verifySignedApprovalDecision(req, verifiers); err != nil {
		t.Fatalf("verifySignedApprovalDecision returned error: %v", err)
	}
}

func TestVerifySignedApprovalDecisionRejectsApproverMismatch(t *testing.T) {
	req, verifiers, err := signedPromotionRequestForTests("human-1")
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	req.Approver = "human-2"
	err = verifySignedApprovalDecision(req, verifiers)
	if !errors.Is(err, ErrApprovalVerificationFailed) {
		t.Fatalf("verifySignedApprovalDecision error = %v, want ErrApprovalVerificationFailed", err)
	}
}

func TestVerifySignedApprovalDecisionRejectsUnknownTrustedVerifier(t *testing.T) {
	req, _, err := signedPromotionRequestForTests("human-1")
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	err = verifySignedApprovalDecision(req, []trustpolicy.VerifierRecord{})
	if !errors.Is(err, ErrVerifierNotFound) {
		t.Fatalf("verifySignedApprovalDecision error = %v, want ErrVerifierNotFound", err)
	}
}

func signedPromotionRequestForTests(approver string) (PromotionRequest, []trustpolicy.VerifierRecord, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	decision := approvalDecisionFixtureForTests(approver)
	payload, err := json.Marshal(decision)
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	signature := ed25519.Sign(privateKey, canonical)
	keyIDValue := strings.TrimPrefix(digestBytes(publicKey), "sha256:")
	verifiers := []trustpolicy.VerifierRecord{approvalVerifierFixtureForTests(approver, keyIDValue, publicKey)}

	return PromotionRequest{
		Approver:         approver,
		ApprovalDecision: approvalEnvelopeFixtureForTests(payload, keyIDValue, signature),
	}, verifiers, nil
}

func approvalDecisionFixtureForTests(approver string) map[string]any {
	return map[string]any{
		"schema_id":                trustpolicy.ApprovalDecisionSchemaID,
		"schema_version":           trustpolicy.ApprovalDecisionSchemaVersion,
		"approval_request_hash":    map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)},
		"approver":                 map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": approver, "instance_id": "approval-session"},
		"decision_outcome":         "approve",
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"key_protection_posture":   "hardware_backed",
		"identity_binding_posture": "attested",
		"approval_assertion_hash":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)},
		"decided_at":               "2026-03-13T12:05:00Z",
		"consumption_posture":      "single_use",
	}
}

func approvalVerifierFixtureForTests(approver string, keyIDValue string, publicKey []byte) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "approval_authority",
		LogicalScope:           "user",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: approver, InstanceID: "approval-session"},
		KeyProtectionPosture:   "hardware_backed",
		IdentityBindingPosture: "attested",
		PresenceMode:           "hardware_touch",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
}

func approvalEnvelopeFixtureForTests(payload []byte, keyIDValue string, signature []byte) *trustpolicy.SignedObjectEnvelope {
	return &trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.ApprovalDecisionSchemaID,
		PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion,
		Payload:              payload,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}
}
