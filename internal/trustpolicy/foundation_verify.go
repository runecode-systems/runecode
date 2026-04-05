package trustpolicy

import (
	"crypto/ed25519"
	"fmt"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const (
	EnvelopeSchemaID      = "runecode.protocol.v0.SignedObjectEnvelope"
	EnvelopeSchemaVersion = "0.2.0"
	SignatureInputProfile = "rfc8785_jcs_detached_payload"
	KeyIDProfile          = "key_sha256"

	VerifierSchemaID      = "runecode.protocol.v0.VerifierRecord"
	VerifierSchemaVersion = "0.1.0"
)

type VerifierRegistry struct {
	byIdentity map[string]VerifierRecord
}

func NewVerifierRegistry(records []VerifierRecord) (*VerifierRegistry, error) {
	registry := &VerifierRegistry{byIdentity: map[string]VerifierRecord{}}
	for index := range records {
		record := records[index]
		identity, err := normalizeVerifierRecord(record)
		if err != nil {
			return nil, fmt.Errorf("verifier[%d]: %w", index, err)
		}
		if _, exists := registry.byIdentity[identity]; exists {
			return nil, fmt.Errorf("duplicate verifier identity %q", identity)
		}
		registry.byIdentity[identity] = record
	}
	return registry, nil
}

func (r *VerifierRegistry) Resolve(signature SignatureBlock) (VerifierRecord, error) {
	identity, err := signatureVerifierIdentity(signature)
	if err != nil {
		return VerifierRecord{}, err
	}
	record, ok := r.byIdentity[identity]
	if !ok {
		return VerifierRecord{}, fmt.Errorf("verifier not found for %q", identity)
	}
	if record.Status != "active" {
		return VerifierRecord{}, fmt.Errorf("verifier status %q is not admissible", record.Status)
	}
	return record, nil
}

type EnvelopeVerificationOptions struct {
	RequirePayloadSchemaMatch bool
	ExpectedPayloadSchemaID   string
	ExpectedPayloadVersion    string
}

func VerifySignedEnvelope(envelope SignedObjectEnvelope, registry *VerifierRegistry, options EnvelopeVerificationOptions) error {
	if err := validateSignedEnvelopeMetadata(envelope); err != nil {
		return err
	}
	if err := validateSignedEnvelopePayloadSchema(envelope, options); err != nil {
		return err
	}
	canonicalPayload, err := canonicalizePayload(envelope.Payload)
	if err != nil {
		return err
	}
	verifier, signatureBytes, err := resolveSignedEnvelopeVerificationMaterial(envelope, registry)
	if err != nil {
		return err
	}
	return verifySignedEnvelopeEd25519(verifier, canonicalPayload, signatureBytes)
}

func validateSignedEnvelopeMetadata(envelope SignedObjectEnvelope) error {
	if envelope.SchemaID != EnvelopeSchemaID {
		return fmt.Errorf("unexpected envelope schema_id %q", envelope.SchemaID)
	}
	if envelope.SchemaVersion != EnvelopeSchemaVersion {
		return fmt.Errorf("unexpected envelope schema_version %q", envelope.SchemaVersion)
	}
	if envelope.SignatureInput != SignatureInputProfile {
		return fmt.Errorf("unsupported signature_input %q", envelope.SignatureInput)
	}
	if len(envelope.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}
	return nil
}

func validateSignedEnvelopePayloadSchema(envelope SignedObjectEnvelope, options EnvelopeVerificationOptions) error {
	if options.RequirePayloadSchemaMatch {
		if envelope.PayloadSchemaID == "" || envelope.PayloadSchemaVersion == "" {
			return fmt.Errorf("payload schema identity is required")
		}
		if options.ExpectedPayloadSchemaID != "" && envelope.PayloadSchemaID != options.ExpectedPayloadSchemaID {
			return fmt.Errorf("payload_schema_id %q does not match expected %q", envelope.PayloadSchemaID, options.ExpectedPayloadSchemaID)
		}
		if options.ExpectedPayloadVersion != "" && envelope.PayloadSchemaVersion != options.ExpectedPayloadVersion {
			return fmt.Errorf("payload_schema_version %q does not match expected %q", envelope.PayloadSchemaVersion, options.ExpectedPayloadVersion)
		}
	}
	return nil
}

func resolveSignedEnvelopeVerificationMaterial(envelope SignedObjectEnvelope, registry *VerifierRegistry) (VerifierRecord, []byte, error) {
	verifier, err := registry.Resolve(envelope.Signature)
	if err != nil {
		return VerifierRecord{}, nil, err
	}
	signatureBytes, err := envelope.Signature.SignatureBytes()
	if err != nil {
		return VerifierRecord{}, nil, err
	}
	return verifier, signatureBytes, nil
}

func verifySignedEnvelopeEd25519(verifier VerifierRecord, canonicalPayload []byte, signatureBytes []byte) error {
	publicKey, err := verifier.PublicKey.DecodedBytes()
	if err != nil {
		return err
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("ed25519 public key must be %d bytes", ed25519.PublicKeySize)
	}
	if !ed25519.Verify(publicKey, canonicalPayload, signatureBytes) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func canonicalizePayload(payload []byte) ([]byte, error) {
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		return nil, fmt.Errorf("payload canonicalization failed: %w", err)
	}
	return canonical, nil
}
