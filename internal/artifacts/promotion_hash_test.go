package artifacts

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestPromotionDecisionHashCanonicalizesEnvelopeJSON(t *testing.T) {
	envelopeA, envelopeB, err := promotionDecisionHashEnvelopesForTests()
	if err != nil {
		t.Fatalf("promotionDecisionHashEnvelopesForTests error: %v", err)
	}
	hashA, err := promotionDecisionHash(PromotionRequest{ApprovalDecision: &envelopeA})
	if err != nil {
		t.Fatalf("promotionDecisionHash envelopeA error: %v", err)
	}
	hashB, err := promotionDecisionHash(PromotionRequest{ApprovalDecision: &envelopeB})
	if err != nil {
		t.Fatalf("promotionDecisionHash envelopeB error: %v", err)
	}
	if hashA != hashB {
		t.Fatalf("canonical decision hash mismatch: %q != %q", hashA, hashB)
	}
}

func promotionDecisionHashEnvelopesForTests() (trustpolicy.SignedObjectEnvelope, trustpolicy.SignedObjectEnvelope, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.SignedObjectEnvelope{}, err
	}
	payload := []byte(`{"schema_id":"runecode.protocol.v0.ApprovalDecision","schema_version":"0.3.0"}`)
	canonicalPayload, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.SignedObjectEnvelope{}, err
	}
	signature := ed25519.Sign(privateKey, canonicalPayload)
	keyID := strings.TrimPrefix(digestBytes(publicKey), "sha256:")
	envelopeA := trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.ApprovalDecisionSchemaID,
		PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion,
		Payload:              payload,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyID, Signature: base64.StdEncoding.EncodeToString(signature)},
	}
	envelopeB := trustpolicy.SignedObjectEnvelope{}
	envelopeBJSON := `{"schema_version":"0.2.0","schema_id":"` + trustpolicy.EnvelopeSchemaID + `","payload_schema_id":"` + trustpolicy.ApprovalDecisionSchemaID + `","payload_schema_version":"` + trustpolicy.ApprovalDecisionSchemaVersion + `","payload":` + string(payload) + `,"signature_input":"` + trustpolicy.SignatureInputProfile + `","signature":{"key_id":"` + trustpolicy.KeyIDProfile + `","alg":"ed25519","signature":"` + base64.StdEncoding.EncodeToString(signature) + `","key_id_value":"` + keyID + `"}}`
	if err := json.Unmarshal([]byte(envelopeBJSON), &envelopeB); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, trustpolicy.SignedObjectEnvelope{}, err
	}
	return envelopeA, envelopeB, nil
}
