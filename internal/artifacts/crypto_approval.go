package artifacts

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func verifySignedApprovalDecision(req PromotionRequest, trustedVerifiers []trustpolicy.VerifierRecord) error {
	if req.ApprovalDecision == nil {
		return ErrApprovalArtifactRequired
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
	return nil
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
