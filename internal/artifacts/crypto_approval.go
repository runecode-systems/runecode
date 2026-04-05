package artifacts

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func verifySignedApprovalDecision(req PromotionRequest, trustedVerifiers []trustpolicy.VerifierRecord) error {
	if req.ApprovalRequest == nil {
		return ErrApprovalRequestArtifactRequired
	}
	if req.ApprovalDecision == nil {
		return ErrApprovalArtifactRequired
	}
	requestEnvelope, requestPayload, err := verifiedApprovalRequest(req, trustedVerifiers)
	if err != nil {
		return err
	}
	decision, err := verifiedApprovalDecision(req, trustedVerifiers)
	if err != nil {
		return err
	}
	if err := trustpolicy.ValidateApprovalDecisionEvidence(decision); err != nil {
		return errors.Join(ErrApprovalVerificationFailed, err)
	}
	if decision.DecisionOutcome != "approve" {
		return errors.Join(ErrApprovalVerificationFailed, errors.New("approval decision outcome is not approve"))
	}
	if decision.Approver.PrincipalID == "" {
		return errors.Join(ErrApprovalVerificationFailed, errors.New("approval decision approver principal is required"))
	}
	if req.Approver != "" && req.Approver != decision.Approver.PrincipalID {
		return errors.Join(ErrApprovalVerificationFailed, errors.New("promotion approver does not match approval decision approver"))
	}
	if err := validateApprovalBinding(req, requestEnvelope.Payload, requestPayload, decision); err != nil {
		return errors.Join(ErrApprovalVerificationFailed, err)
	}
	return nil
}

func verifiedApprovalRequest(req PromotionRequest, trustedVerifiers []trustpolicy.VerifierRecord) (trustpolicy.SignedObjectEnvelope, map[string]any, error) {
	verifiers, err := resolveTrustedApprovalVerifiersForEnvelope(*req.ApprovalRequest, trustedVerifiers)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, nil, err
	}
	registry, err := verifierRegistry(verifiers)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, nil, errors.Join(ErrApprovalVerificationFailed, err)
	}
	if err := trustpolicy.VerifySignedEnvelope(*req.ApprovalRequest, registry, trustpolicy.EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   trustpolicy.ApprovalRequestSchemaID,
		ExpectedPayloadVersion:    trustpolicy.ApprovalRequestSchemaVersion,
	}); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, nil, errors.Join(ErrApprovalVerificationFailed, err)
	}
	if err := validateObjectPayloadAgainstSchema(req.ApprovalRequest.Payload, "objects/ApprovalRequest.schema.json"); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, nil, errors.Join(ErrApprovalVerificationFailed, err)
	}
	payload, err := decodeObjectPayload(req.ApprovalRequest.Payload)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, nil, errors.Join(ErrApprovalVerificationFailed, err)
	}
	return *req.ApprovalRequest, payload, nil
}

func verifiedApprovalDecision(req PromotionRequest, trustedVerifiers []trustpolicy.VerifierRecord) (trustpolicy.ApprovalDecision, error) {
	verifiers, err := resolveTrustedApprovalVerifiersForEnvelope(*req.ApprovalDecision, trustedVerifiers)
	if err != nil {
		return trustpolicy.ApprovalDecision{}, err
	}
	registry, err := verifierRegistry(verifiers)
	if err != nil {
		return trustpolicy.ApprovalDecision{}, errors.Join(ErrApprovalVerificationFailed, err)
	}
	if err := verifyApprovalDecisionEnvelope(*req.ApprovalDecision, registry); err != nil {
		return trustpolicy.ApprovalDecision{}, errors.Join(ErrApprovalVerificationFailed, err)
	}
	if err := validateApprovalDecisionPayload(req.ApprovalDecision.Payload); err != nil {
		return trustpolicy.ApprovalDecision{}, errors.Join(ErrApprovalVerificationFailed, err)
	}
	decision, err := decodeApprovalDecision(req.ApprovalDecision.Payload)
	if err != nil {
		return trustpolicy.ApprovalDecision{}, errors.Join(ErrApprovalVerificationFailed, err)
	}
	return decision, nil
}

func verifierRegistry(verifiers []trustpolicy.VerifierRecord) (*trustpolicy.VerifierRegistry, error) {
	return trustpolicy.NewVerifierRegistry(verifiers)
}

func verifyApprovalDecisionEnvelope(envelope trustpolicy.SignedObjectEnvelope, registry *trustpolicy.VerifierRegistry) error {
	return trustpolicy.VerifySignedEnvelope(envelope, registry, trustpolicy.EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   trustpolicy.ApprovalDecisionSchemaID,
		ExpectedPayloadVersion:    trustpolicy.ApprovalDecisionSchemaVersion,
	})
}

func validateApprovalDecisionPayload(payload []byte) error {
	return validateObjectPayloadAgainstSchema(payload, "objects/ApprovalDecision.schema.json")
}

func decodeApprovalDecision(payload []byte) (trustpolicy.ApprovalDecision, error) {
	decision := trustpolicy.ApprovalDecision{}
	if err := json.Unmarshal(payload, &decision); err != nil {
		return trustpolicy.ApprovalDecision{}, err
	}
	return decision, nil
}

func decodeObjectPayload(payload []byte) (map[string]any, error) {
	value := map[string]any{}
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func resolveTrustedApprovalVerifiersForEnvelope(envelope trustpolicy.SignedObjectEnvelope, trustedVerifiers []trustpolicy.VerifierRecord) ([]trustpolicy.VerifierRecord, error) {
	if len(trustedVerifiers) == 0 {
		return nil, errors.Join(ErrApprovalVerificationFailed, ErrVerifierNotFound)
	}
	wantKeyID, wantKeyIDValue := envelopeVerifierIdentity(envelope)
	for _, record := range trustedVerifiers {
		if !matchesApprovalVerifier(record, wantKeyID, wantKeyIDValue) {
			continue
		}
		if record.Status != "active" {
			return nil, errors.Join(ErrApprovalVerificationFailed, fmt.Errorf("trusted verifier %s is not active", record.KeyIDValue))
		}
		publicKeyBytes, err := record.PublicKey.DecodedBytes()
		if err != nil {
			return nil, errors.Join(ErrApprovalVerificationFailed, err)
		}
		reencoded := base64.StdEncoding.EncodeToString(publicKeyBytes)
		verified := record
		verified.PublicKey.Value = reencoded
		return []trustpolicy.VerifierRecord{verified}, nil
	}
	return nil, errors.Join(ErrApprovalVerificationFailed, ErrVerifierNotFound)
}

func envelopeVerifierIdentity(envelope trustpolicy.SignedObjectEnvelope) (string, string) {
	return envelope.Signature.KeyID, envelope.Signature.KeyIDValue
}

func matchesApprovalVerifier(record trustpolicy.VerifierRecord, wantKeyID string, wantKeyIDValue string) bool {
	if record.LogicalPurpose != "approval_authority" {
		return false
	}
	if record.LogicalScope != "user" {
		return false
	}
	if record.KeyID != wantKeyID {
		return false
	}
	if record.KeyIDValue != wantKeyIDValue {
		return false
	}
	return true
}

func validateApprovalBinding(req PromotionRequest, requestPayloadBytes []byte, requestPayload map[string]any, decision trustpolicy.ApprovalDecision) error {
	if err := validateApprovalRequestPayloadBinding(req, requestPayload); err != nil {
		return err
	}
	expectedHash, err := canonicalPayloadDigest(requestPayloadBytes)
	if err != nil {
		return err
	}
	actualHash, err := decision.ApprovalRequestHash.Identity()
	if err != nil {
		return fmt.Errorf("approval_request_hash: %w", err)
	}
	if actualHash != expectedHash {
		return fmt.Errorf("approval_request_hash %q does not match expected request binding %q", actualHash, expectedHash)
	}
	return nil
}

func validateApprovalRequestPayloadBinding(req PromotionRequest, payload map[string]any) error {
	actionRequestHash, err := digestIdentityField(payload, "action_request_hash")
	if err != nil {
		return fmt.Errorf("action_request_hash: %w", err)
	}
	expectedActionRequestHash, err := promotionActionRequestHash(req)
	if err != nil {
		return fmt.Errorf("action_request_hash: %w", err)
	}
	if actionRequestHash != expectedActionRequestHash {
		return fmt.Errorf("action_request_hash %q does not match expected promotion action hash %q", actionRequestHash, expectedActionRequestHash)
	}
	relevantArtifactHashes, err := digestIdentityArrayField(payload, "relevant_artifact_hashes")
	if err != nil {
		return fmt.Errorf("relevant_artifact_hashes: %w", err)
	}
	if len(relevantArtifactHashes) != 1 || relevantArtifactHashes[0] != req.UnapprovedDigest {
		return fmt.Errorf("approval request must bind exactly the promoted source digest %q", req.UnapprovedDigest)
	}
	return nil
}

func canonicalPayloadDigest(payload []byte) (string, error) {
	b, err := canonicalizeJSONBytes(payload)
	if err != nil {
		return "", err
	}
	return digestBytes(b), nil
}

func digestIdentityField(object map[string]any, key string) (string, error) {
	raw, ok := object[key]
	if !ok {
		return "", fmt.Errorf("missing key %q", key)
	}
	digest, ok := raw.(map[string]any)
	if !ok {
		return "", fmt.Errorf("key %q has type %T, want digest object", key, raw)
	}
	hashAlg, ok := digest["hash_alg"].(string)
	if !ok {
		return "", fmt.Errorf("key %q hash_alg missing or invalid", key)
	}
	hash, ok := digest["hash"].(string)
	if !ok {
		return "", fmt.Errorf("key %q hash missing or invalid", key)
	}
	return hashAlg + ":" + hash, nil
}

func digestIdentityArrayField(object map[string]any, key string) ([]string, error) {
	raw, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing key %q", key)
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("key %q has type %T, want []any", key, raw)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		digest, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("key %q contains non-object digest", key)
		}
		hashAlg, ok := digest["hash_alg"].(string)
		if !ok {
			return nil, fmt.Errorf("key %q hash_alg missing or invalid", key)
		}
		hash, ok := digest["hash"].(string)
		if !ok {
			return nil, fmt.Errorf("key %q hash missing or invalid", key)
		}
		out = append(out, hashAlg+":"+hash)
	}
	sort.Strings(out)
	return out, nil
}
