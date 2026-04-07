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

func TestVerifySignedApprovalDecisionRejectsVerifierOwnerMismatch(t *testing.T) {
	req, verifiers, err := signedPromotionRequestForTests("human-1")
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	verifiers[0].OwnerPrincipal.InstanceID = "different-session"
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

func TestVerifySignedApprovalDecisionRejectsVerifierWithWrongPurpose(t *testing.T) {
	req, verifiers, err := signedPromotionRequestForTests("human-1")
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	verifiers[0].LogicalPurpose = "audit_anchor"
	err = verifySignedApprovalDecision(req, verifiers)
	if !errors.Is(err, ErrVerifierNotFound) {
		t.Fatalf("verifySignedApprovalDecision error = %v, want ErrVerifierNotFound", err)
	}
}

func TestVerifySignedApprovalDecisionRejectsVerifierWithWrongScope(t *testing.T) {
	req, verifiers, err := signedPromotionRequestForTests("human-1")
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	verifiers[0].LogicalScope = "deployment"
	err = verifySignedApprovalDecision(req, verifiers)
	if !errors.Is(err, ErrVerifierNotFound) {
		t.Fatalf("verifySignedApprovalDecision error = %v, want ErrVerifierNotFound", err)
	}
}

func TestVerifySignedApprovalDecisionRejectsPayloadUnknownFields(t *testing.T) {
	signer, err := newApprovalSignerFixture()
	if err != nil {
		t.Fatalf("newApprovalSignerFixture returned error: %v", err)
	}
	req, verifiers, err := signedPromotionRequestForTestsWithSigner("human-1", signer)
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	payloadMap := map[string]any{}
	if err := json.Unmarshal(req.ApprovalDecision.Payload, &payloadMap); err != nil {
		t.Fatalf("Unmarshal payload returned error: %v", err)
	}
	payloadMap["unknown_field"] = "should-fail-closed"
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		t.Fatalf("Marshal payload returned error: %v", err)
	}
	signedEnvelope, err := signedEnvelopeForPayload(payloadBytes, signer)
	if err != nil {
		t.Fatalf("signedEnvelopeForPayload returned error: %v", err)
	}
	req.ApprovalDecision = signedEnvelope
	err = verifySignedApprovalDecision(req, verifiers)
	if !errors.Is(err, ErrApprovalVerificationFailed) {
		t.Fatalf("verifySignedApprovalDecision error = %v, want ErrApprovalVerificationFailed", err)
	}
}

func TestVerifySignedApprovalDecisionRejectsMissingApprovalRequest(t *testing.T) {
	req, verifiers, err := signedPromotionRequestForTests("human-1")
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	req.ApprovalRequest = nil
	err = verifySignedApprovalDecision(req, verifiers)
	if !errors.Is(err, ErrApprovalRequestArtifactRequired) {
		t.Fatalf("verifySignedApprovalDecision error = %v, want ErrApprovalRequestArtifactRequired", err)
	}
}

func TestVerifySignedApprovalDecisionRejectsApprovalRequestBindingMismatch(t *testing.T) {
	req, verifiers, err := signedPromotionRequestForTests("human-1")
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests returned error: %v", err)
	}
	req.Commit = "def456"
	err = verifySignedApprovalDecision(req, verifiers)
	if !errors.Is(err, ErrApprovalVerificationFailed) {
		t.Fatalf("verifySignedApprovalDecision error = %v, want ErrApprovalVerificationFailed", err)
	}
}

func TestPromotionActionRequestHashEscapesDelimiterLikeValues(t *testing.T) {
	reqA := PromotionRequest{
		UnapprovedDigest:     testDigestValueForApprovalTests("a"),
		Approver:             "human|ops",
		RepoPath:             "repo/file.txt",
		Commit:               "abc123",
		ExtractorToolVersion: "tool-v1",
	}
	reqB := PromotionRequest{
		UnapprovedDigest:     testDigestValueForApprovalTests("a"),
		Approver:             "ops",
		RepoPath:             "repo/file.txt|human",
		Commit:               "abc123",
		ExtractorToolVersion: "tool-v1",
	}
	hashA, err := promotionActionRequestHash(reqA)
	if err != nil {
		t.Fatalf("promotionActionRequestHash(reqA) error: %v", err)
	}
	hashB, err := promotionActionRequestHash(reqB)
	if err != nil {
		t.Fatalf("promotionActionRequestHash(reqB) error: %v", err)
	}
	if hashA == hashB {
		t.Fatalf("promotionActionRequestHash collision: %q", hashA)
	}
}

func signedPromotionRequestForTests(approver string) (PromotionRequest, []trustpolicy.VerifierRecord, error) {
	return signedPromotionRequestForInputs(testDigestValueForApprovalTests("c"), approver, "a", "b", "tool-v1")
}

func signedPromotionRequestForInputs(unapprovedDigest string, approver string, repoPath string, commit string, extractorVersion string) (PromotionRequest, []trustpolicy.VerifierRecord, error) {
	signer, err := newApprovalSignerFixture()
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	request := PromotionRequest{
		UnapprovedDigest:     unapprovedDigest,
		Approver:             approver,
		RepoPath:             repoPath,
		Commit:               commit,
		ExtractorToolVersion: extractorVersion,
		FullContentVisible:   true,
	}
	return signedPromotionRequestFixtureWithSigner(request, signer)
}

type approvalSignerFixture struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	keyIDValue string
}

func newApprovalSignerFixture() (approvalSignerFixture, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return approvalSignerFixture{}, err
	}
	return approvalSignerFixture{
		publicKey:  publicKey,
		privateKey: privateKey,
		keyIDValue: strings.TrimPrefix(digestBytes(publicKey), "sha256:"),
	}, nil
}

func signedPromotionRequestForTestsWithSigner(approver string, signer approvalSignerFixture) (PromotionRequest, []trustpolicy.VerifierRecord, error) {
	request := PromotionRequest{UnapprovedDigest: testDigestValueForApprovalTests("c"), Approver: approver, RepoPath: "a", Commit: "b", ExtractorToolVersion: "tool-v1", FullContentVisible: true}
	return signedPromotionRequestFixtureWithSigner(request, signer)
}

func signedPromotionRequestFixtureWithSigner(request PromotionRequest, signer approvalSignerFixture) (PromotionRequest, []trustpolicy.VerifierRecord, error) {
	requestPayloadMap := approvalRequestFixtureForTests(request)
	requestPayload, err := json.Marshal(requestPayloadMap)
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	canonicalRequest, err := jsoncanonicalizer.Transform(requestPayload)
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	requestSignature := ed25519.Sign(signer.privateKey, canonicalRequest)
	requestEnvelope := approvalRequestEnvelopeFixtureForTests(requestPayload, signer.keyIDValue, requestSignature)
	requestDigest, err := canonicalPayloadDigest(requestPayload)
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	decision := approvalDecisionFixtureForTests(request.Approver, requestDigest)
	decisionPayload, err := json.Marshal(decision)
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	canonicalDecision, err := jsoncanonicalizer.Transform(decisionPayload)
	if err != nil {
		return PromotionRequest{}, nil, err
	}
	decisionSignature := ed25519.Sign(signer.privateKey, canonicalDecision)
	verifiers := []trustpolicy.VerifierRecord{approvalVerifierFixtureForTests(request.Approver, signer.keyIDValue, signer.publicKey)}

	request.ApprovalRequest = requestEnvelope
	request.ApprovalDecision = approvalEnvelopeFixtureForTests(decisionPayload, signer.keyIDValue, decisionSignature)
	return request, verifiers, nil
}

func signedEnvelopeForPayload(payload []byte, signer approvalSignerFixture) (*trustpolicy.SignedObjectEnvelope, error) {
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		return nil, err
	}
	signature := ed25519.Sign(signer.privateKey, canonical)
	return approvalEnvelopeFixtureForTests(payload, signer.keyIDValue, signature), nil
}

func approvalDecisionFixtureForTests(approver string, requestDigest string) map[string]any {
	hashAlg, hash := splitDigestIdentityForTests(requestDigest)
	return map[string]any{
		"schema_id":                trustpolicy.ApprovalDecisionSchemaID,
		"schema_version":           trustpolicy.ApprovalDecisionSchemaVersion,
		"approval_request_hash":    map[string]any{"hash_alg": hashAlg, "hash": hash},
		"approver":                 map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": approver, "instance_id": "approval-session"},
		"decision_outcome":         "approve",
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"key_protection_posture":   "hardware_backed",
		"identity_binding_posture": "attested",
		"approval_assertion_hash":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)},
		"decided_at":               "2026-03-13T12:05:00Z",
		"consumption_posture":      "single_use",
		"signatures":               []any{approvalDecisionSignaturePlaceholderForTests()},
	}
}

func approvalRequestFixtureForTests(req PromotionRequest) map[string]any {
	actionRequestHash, err := promotionActionRequestHash(req)
	if err != nil {
		panic(err)
	}
	actionHashAlg, actionHash := splitDigestIdentityForTests(actionRequestHash)
	sourceHashAlg, sourceHash := splitDigestIdentityForTests(req.UnapprovedDigest)
	return map[string]any{
		"schema_id":                trustpolicy.ApprovalRequestSchemaID,
		"schema_version":           trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile":         "moderate",
		"requester":                map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-artifact-store"},
		"approval_trigger_code":    "excerpt_promotion",
		"manifest_hash":            map[string]any{"hash_alg": sourceHashAlg, "hash": sourceHash},
		"action_request_hash":      map[string]any{"hash_alg": actionHashAlg, "hash": actionHash},
		"relevant_artifact_hashes": []any{map[string]any{"hash_alg": sourceHashAlg, "hash": sourceHash}},
		"details_schema_id":        "runecode.protocol.details.approval.excerpt-promotion.v0",
		"details":                  map[string]any{"repo_path": req.RepoPath, "commit": req.Commit},
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"requested_at":             "2026-03-13T12:00:00Z",
		"expires_at":               "2026-03-13T12:30:00Z",
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      "Promote reviewed file excerpts for downstream use.",
		"signatures":               []any{approvalDecisionSignaturePlaceholderForTests()},
	}
}

func splitDigestIdentityForTests(identity string) (string, string) {
	parts := strings.SplitN(identity, ":", 2)
	if len(parts) != 2 {
		return "sha256", identity
	}
	return parts[0], parts[1]
}

func testDigestValueForApprovalTests(seed string) string {
	base := strings.Repeat(seed, 64)
	if len(base) > 64 {
		base = base[:64]
	}
	return "sha256:" + base[:64]
}

func approvalDecisionSignaturePlaceholderForTests() map[string]any {
	return map[string]any{
		"alg":          "ed25519",
		"key_id":       trustpolicy.KeyIDProfile,
		"key_id_value": strings.Repeat("a", 64),
		"signature":    "c2ln",
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

func approvalRequestEnvelopeFixtureForTests(payload []byte, keyIDValue string, signature []byte) *trustpolicy.SignedObjectEnvelope {
	return &trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.ApprovalRequestSchemaID,
		PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion,
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
