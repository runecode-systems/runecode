package policyengine

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func verifyContextSignatures(payload []byte, schemaID, schemaVersion string, registry *trustpolicy.VerifierRegistry, required bool) ([]string, error) {
	signatures, err := extractSignatureBlocks(payload)
	if err != nil {
		return nil, err
	}
	if out, done, err := verifySignaturePreconditions(signatures, schemaID, registry, required); done {
		return out, err
	}
	canonicalPayload, err := canonicalPayloadWithoutSignatures(payload)
	if err != nil {
		return nil, err
	}
	return verifySignatureSet(signatures, canonicalPayload, schemaID, schemaVersion, registry)
}

func verifySignaturePreconditions(signatures []trustpolicy.SignatureBlock, schemaID string, registry *trustpolicy.VerifierRegistry, required bool) ([]string, bool, error) {
	if len(signatures) == 0 {
		if required {
			return nil, true, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("%s payload missing signatures for required signed context verification", schemaID)}
		}
		return []string{}, true, nil
	}
	if registry == nil {
		if required {
			return nil, true, &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: "signed context verification requested but verifier registry unavailable"}
		}
		return []string{}, true, nil
	}
	return nil, false, nil
}

func verifySignatureSet(signatures []trustpolicy.SignatureBlock, canonicalPayload []byte, schemaID, schemaVersion string, registry *trustpolicy.VerifierRegistry) ([]string, error) {
	signerIDs := make([]string, 0, len(signatures))
	for idx := range signatures {
		signerID, err := verifySingleContextSignature(idx, signatures[idx], canonicalPayload, schemaID, schemaVersion, registry)
		if err != nil {
			return nil, err
		}
		signerIDs = append(signerIDs, signerID)
	}
	return sortedUnique(signerIDs), nil
}

func verifySingleContextSignature(idx int, signature trustpolicy.SignatureBlock, canonicalPayload []byte, schemaID, schemaVersion string, registry *trustpolicy.VerifierRegistry) (string, error) {
	verifier, resolveErr := registry.Resolve(signature)
	if resolveErr != nil {
		return "", &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("signature[%d] verifier resolution failed: %v", idx, resolveErr)}
	}
	sigBytes, decodeErr := signature.SignatureBytes()
	if decodeErr != nil {
		return "", &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("signature[%d] decode failed: %v", idx, decodeErr)}
	}
	pub, pubErr := verifier.PublicKey.DecodedBytes()
	if pubErr != nil {
		return "", &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("signature[%d] verifier key decode failed: %v", idx, pubErr)}
	}
	if len(pub) != ed25519.PublicKeySize {
		return "", &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("signature[%d] verifier key length invalid", idx)}
	}
	if !ed25519.Verify(pub, canonicalPayload, sigBytes) {
		return "", &EvaluationError{Code: ErrCodeBrokerLimitPolicyReject, Category: "policy", Retryable: false, Message: fmt.Sprintf("signature[%d] verification failed for %s@%s", idx, schemaID, schemaVersion)}
	}
	return signature.KeyID + ":" + signature.KeyIDValue, nil
}

func extractSignatureBlocks(payload []byte) ([]trustpolicy.SignatureBlock, error) {
	root := map[string]any{}
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("decode signed context payload: %v", err)}
	}
	raw, ok := root["signatures"]
	if !ok {
		return []trustpolicy.SignatureBlock{}, nil
	}
	rawSlice, ok := raw.([]any)
	if !ok {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: "signatures field must be an array"}
	}
	if len(rawSlice) == 0 {
		return []trustpolicy.SignatureBlock{}, nil
	}
	b, err := json.Marshal(rawSlice)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("marshal signatures field: %v", err)}
	}
	out := []trustpolicy.SignatureBlock{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("decode signatures field: %v", err)}
	}
	return out, nil
}

func canonicalPayloadWithoutSignatures(payload []byte) ([]byte, error) {
	root := map[string]any{}
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("decode signed context payload: %v", err)}
	}
	delete(root, "signatures")
	b, err := json.Marshal(root)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("marshal payload without signatures: %v", err)}
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return nil, &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: fmt.Sprintf("canonicalize payload without signatures: %v", err)}
	}
	return canonical, nil
}
